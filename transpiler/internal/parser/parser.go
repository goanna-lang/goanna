package parser

import (
	"fmt"
	"go/scanner"
	"go/token"
	"strings"
	"unicode"

	"github.com/nahmanmate/gounion/internal/ast"
)

// Parse converts .union.go source into an ast.File.
// It uses go/scanner for tokenisation then hand-parses the union-specific constructs.
func Parse(src []byte) (*ast.File, error) {
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))

	var errs scanner.ErrorList
	var s scanner.Scanner
	s.Init(file, src, func(pos token.Position, msg string) {
		errs = append(errs, &scanner.Error{Pos: pos, Msg: msg})
	}, scanner.ScanComments)

	tokens := scanAll(&s, file, src)
	p := &chunkParser{tokens: tokens, src: src}
	return p.parseFile()
}

type tok struct {
	pos    token.Pos
	kind   token.Token
	lit    string
	offset int // byte offset in source
}

func scanAll(s *scanner.Scanner, file *token.File, src []byte) []tok {
	var out []tok
	for {
		pos, kind, lit := s.Scan()
		if kind == token.ILLEGAL && lit == "" {
			break
		}
		offset := file.Offset(pos)
		if lit == "" {
			lit = kind.String()
		}
		out = append(out, tok{pos: pos, kind: kind, lit: lit, offset: offset})
		if kind == token.EOF {
			break
		}
	}
	return out
}

type chunkParser struct {
	tokens []tok
	src    []byte
	pos    int // current token index
	items  []ast.Item
}

func (p *chunkParser) peek() tok {
	if p.pos >= len(p.tokens) {
		return tok{kind: token.EOF}
	}
	return p.tokens[p.pos]
}

func (p *chunkParser) peekN(n int) tok {
	i := p.pos + n
	if i >= len(p.tokens) {
		return tok{kind: token.EOF}
	}
	return p.tokens[i]
}

func (p *chunkParser) consume() tok {
	t := p.peek()
	p.pos++
	return t
}

func (p *chunkParser) parseFile() (*ast.File, error) {
	var items []ast.Item
	chunkStart := 0

	for p.peek().kind != token.EOF {
		t := p.peek()

		// Detect: type <Ident> union {
		if t.kind == token.TYPE && p.isUnionDecl() {
			// flush preceding chunk
			if chunkEnd := t.offset; chunkEnd > chunkStart {
				items = append(items, ast.OpaqueChunk{Text: string(p.src[chunkStart:chunkEnd])})
			}
			decl, err := p.parseUnionDecl()
			if err != nil {
				return nil, err
			}
			items = append(items, decl)
			chunkStart = p.currentOffset()
			continue
		}

		// Detect: switch [v :=] <expr>.(union) {
		if t.kind == token.SWITCH && p.isUnionSwitch() {
			if chunkEnd := t.offset; chunkEnd > chunkStart {
				items = append(items, ast.OpaqueChunk{Text: string(p.src[chunkStart:chunkEnd])})
			}
			sw, err := p.parseUnionSwitch()
			if err != nil {
				return nil, err
			}
			items = append(items, sw)
			chunkStart = p.currentOffset()
			continue
		}

		p.consume()
	}

	// flush trailing chunk
	eofOffset := len(p.src)
	if chunkStart < eofOffset {
		items = append(items, ast.OpaqueChunk{Text: string(p.src[chunkStart:eofOffset])})
	}

	return &ast.File{Items: items}, nil
}

func (p *chunkParser) currentOffset() int {
	if p.pos >= len(p.tokens) {
		return len(p.src)
	}
	return p.tokens[p.pos].offset
}

// isUnionDecl checks if the current position starts "type <Ident> union {"
func (p *chunkParser) isUnionDecl() bool {
	// tokens: TYPE IDENT "union" LBRACE
	return p.peek().kind == token.TYPE &&
		p.peekN(1).kind == token.IDENT &&
		p.peekN(2).kind == token.IDENT && p.peekN(2).lit == "union" &&
		p.peekN(3).kind == token.LBRACE
}

// isUnionSwitch checks if current position starts a switch with .(union)
// Scans forward to find .(union) before the opening brace.
func (p *chunkParser) isUnionSwitch() bool {
	if p.peek().kind != token.SWITCH {
		return false
	}
	for i := p.pos + 1; i < len(p.tokens)-3; i++ {
		t := p.tokens[i]
		if t.kind == token.LBRACE {
			break
		}
		// look for . ( "union" )
		if t.kind == token.PERIOD &&
			p.tokens[i+1].kind == token.LPAREN &&
			p.tokens[i+2].kind == token.IDENT && p.tokens[i+2].lit == "union" &&
			p.tokens[i+3].kind == token.RPAREN {
			return true
		}
	}
	return false
}

func (p *chunkParser) parseUnionDecl() (ast.UnionDecl, error) {
	line := p.peek().offset
	p.consume() // type
	nameT := p.consume() // ident
	p.consume() // union
	p.consume() // {

	decl := ast.UnionDecl{Line: line, Name: nameT.lit}

	for p.peek().kind != token.RBRACE && p.peek().kind != token.EOF {
		// skip newlines/semicolons
		if p.peek().kind == token.SEMICOLON {
			p.consume()
			continue
		}
		vg, err := p.parseVariantGroup()
		if err != nil {
			return decl, err
		}
		decl.Variants = append(decl.Variants, vg)
	}

	if p.peek().kind == token.RBRACE {
		p.consume() // }
	}

	return decl, nil
}

func (p *chunkParser) parseVariantGroup() (ast.VariantGroup, error) {
	var names []string
	// collect comma-separated idents
	for {
		if p.peek().kind != token.IDENT {
			return ast.VariantGroup{}, fmt.Errorf("expected identifier, got %v", p.peek().lit)
		}
		names = append(names, p.consume().lit)
		if p.peek().kind == token.COMMA {
			p.consume()
		} else {
			break
		}
	}
	// type name
	if p.peek().kind != token.IDENT {
		return ast.VariantGroup{}, fmt.Errorf("expected type name, got %v", p.peek().lit)
	}
	typeName := p.consume().lit
	// optional trailing semicolon
	if p.peek().kind == token.SEMICOLON {
		p.consume()
	}
	return ast.VariantGroup{Names: names, Type: typeName}, nil
}

// parseUnionSwitch parses:
//
//	switch [v :=] <expr>.(union) { ... }
func (p *chunkParser) parseUnionSwitch() (ast.UnionSwitch, error) {
	sw := ast.UnionSwitch{Line: p.peek().offset}
	p.consume() // switch

	// Collect everything up to .(union) to extract subject and optional bind var.
	// Two forms:
	//   switch expr.(union)         — no bind
	//   switch v := expr.(union)    — bind
	var headerToks []tok
	for {
		t := p.peek()
		if t.kind == token.PERIOD &&
			p.peekN(1).kind == token.LPAREN &&
			p.peekN(2).kind == token.IDENT && p.peekN(2).lit == "union" &&
			p.peekN(3).kind == token.RPAREN {
			break
		}
		if t.kind == token.LBRACE || t.kind == token.EOF {
			return sw, fmt.Errorf("malformed union switch: no .(union) found")
		}
		headerToks = append(headerToks, p.consume())
	}
	p.consume() // .
	p.consume() // (
	p.consume() // union
	p.consume() // )

	// Parse header tokens to extract bindVar and subject.
	sw.BindVar, sw.Subject = extractBindAndSubject(headerToks)

	// Now parse the switch body.
	if p.peek().kind != token.LBRACE {
		return sw, fmt.Errorf("expected { after .(union)")
	}
	p.consume() // {

	for p.peek().kind != token.RBRACE && p.peek().kind != token.EOF {
		if p.peek().kind == token.SEMICOLON {
			p.consume()
			continue
		}
		if p.peek().kind == token.CASE {
			uc, err := p.parseUnionCase()
			if err != nil {
				return sw, err
			}
			sw.Cases = append(sw.Cases, uc)
			continue
		}
		if p.peek().kind == token.DEFAULT {
			sw.HasDefault = true
			p.consume() // default
			if p.peek().kind == token.COLON {
				p.consume()
			}
			body, err := p.parseCaseBody()
			if err != nil {
				return sw, err
			}
			sw.DefaultBody = body
			continue
		}
		// skip comments / semis inside switch body
		p.consume()
	}
	if p.peek().kind == token.RBRACE {
		p.consume()
	}

	return sw, nil
}

func (p *chunkParser) parseUnionCase() (ast.UnionCase, error) {
	p.consume() // case
	var names []string
	for {
		if p.peek().kind != token.IDENT {
			return ast.UnionCase{}, fmt.Errorf("expected variant name in case, got %v", p.peek().lit)
		}
		names = append(names, p.consume().lit)
		if p.peek().kind == token.COMMA {
			p.consume()
		} else {
			break
		}
	}
	if p.peek().kind == token.COLON {
		p.consume()
	}
	body, err := p.parseCaseBody()
	if err != nil {
		return ast.UnionCase{}, err
	}
	return ast.UnionCase{VariantNames: names, Body: body}, nil
}

// parseCaseBody reads verbatim source until the next case/default/} at the same brace depth.
func (p *chunkParser) parseCaseBody() (string, error) {
	if p.pos >= len(p.tokens) {
		return "", nil
	}
	start := p.currentOffset()
	depth := 0
	for {
		t := p.peek()
		if t.kind == token.EOF {
			break
		}
		if t.kind == token.LBRACE {
			depth++
		}
		if t.kind == token.RBRACE {
			if depth == 0 {
				break
			}
			depth--
		}
		if depth == 0 && (t.kind == token.CASE || t.kind == token.DEFAULT) {
			break
		}
		p.consume()
	}
	end := p.currentOffset()
	return strings.TrimSpace(string(p.src[start:end])), nil
}

// extractBindAndSubject parses header tokens of the form:
//
//	expr                     → bindVar="", subject="expr"
//	v := expr                → bindVar="v", subject="expr"
func extractBindAndSubject(toks []tok) (bindVar, subject string) {
	// look for :=
	for i, t := range toks {
		if t.kind == token.DEFINE {
			bindVar = strings.TrimSpace(joinLits(toks[:i]))
			subject = strings.TrimSpace(joinLits(toks[i+1:]))
			return
		}
	}
	subject = strings.TrimSpace(joinLits(toks))
	return
}

func joinLits(toks []tok) string {
	var b strings.Builder
	for i, t := range toks {
		if i > 0 && needSpace(toks[i-1], t) {
			b.WriteByte(' ')
		}
		b.WriteString(t.lit)
	}
	return b.String()
}

func needSpace(prev, cur tok) bool {
	if cur.kind == token.PERIOD || prev.kind == token.PERIOD {
		return false
	}
	if cur.kind == token.LPAREN || prev.kind == token.LPAREN {
		return false
	}
	if cur.kind == token.RPAREN || prev.kind == token.RPAREN {
		return false
	}
	// space between two idents or an ident and a keyword
	if isWordToken(prev) && isWordToken(cur) {
		return true
	}
	return false
}

func isWordToken(t tok) bool {
	return t.kind == token.IDENT || t.kind.IsKeyword() || isLiteral(t)
}

func isLiteral(t tok) bool {
	if len(t.lit) == 0 {
		return false
	}
	return unicode.IsLetter(rune(t.lit[0])) || unicode.IsDigit(rune(t.lit[0]))
}
