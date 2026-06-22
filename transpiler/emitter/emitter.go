package emitter

import (
	"fmt"
	"strings"

	"github.com/goanna-lang/goanna/transpiler/ast"
	"github.com/goanna-lang/goanna/transpiler/resolver"
)

// ItemLineRange records the 0-indexed line range an item occupies in emitter output.
// End is exclusive.
type ItemLineRange struct{ Start, End int }

// Emit converts an ast.File into Go source text.
// The result is not yet gofmt'd — call go/format on it afterward.
func Emit(file *ast.File, tbl *resolver.SymbolTable) (string, error) {
	src, _, err := EmitWithLineMap(file, tbl)
	return src, err
}

// EmitWithLineMap is like Emit but also returns per-item line ranges in the output.
// Ranges are 0-indexed and parallel to file.Items.
func EmitWithLineMap(file *ast.File, tbl *resolver.SymbolTable) (string, []ItemLineRange, error) {
	var b strings.Builder
	ranges := make([]ItemLineRange, len(file.Items))

	countLines := func(s string) int {
		n := strings.Count(s, "\n")
		return n
	}

	line := 0
	for i, item := range file.Items {
		start := line
		switch v := item.(type) {
		case ast.AtomDecl:
			// not emitted; goanna_types.go carries this per package
		case ast.OpaqueChunk:
			b.WriteString(v.Text)
			line += countLines(v.Text)
		case ast.UnionDecl:
			before := b.Len()
			if err := emitUnionDecl(&b, v, tbl); err != nil {
				return "", nil, err
			}
			line += countLines(b.String()[before:])
		case ast.UnionSwitch:
			before := b.Len()
			if err := emitUnionSwitch(&b, v, tbl); err != nil {
				return "", nil, err
			}
			line += countLines(b.String()[before:])
		}
		ranges[i] = ItemLineRange{Start: start, End: line}
	}

	return b.String(), ranges, nil
}

func emitUnionDecl(b *strings.Builder, decl ast.UnionDecl, tbl *resolver.SymbolTable) error {
	variants := tbl.Unions[decl.Name]
	markerMethod := "is" + title(decl.Name)

	// 1. For atom variants: emit unexported zero-size struct types.
	for _, v := range variants {
		if v.IsAtom {
			fmt.Fprintf(b, "type %s struct{}\n", goTypeName(decl.Name, v))
		}
	}

	// 2. Emit marker methods on the Go type (payload type for non-atom, generated type for atom).
	for _, v := range variants {
		fmt.Fprintf(b, "func (%s) %s() {}\n", goTypeName(decl.Name, v), markerMethod)
	}

	// 3. Sealed interface.
	fmt.Fprintf(b, "type %s interface{ %s() }\n", decl.Name, markerMethod)

	// 4. Package-level vars for atom variants (the public names).
	for _, v := range variants {
		if v.IsAtom {
			fmt.Fprintf(b, "var %s %s = %s{}\n", v.Name, decl.Name, goTypeName(decl.Name, v))
		}
	}

	return nil
}

func emitUnionSwitch(b *strings.Builder, sw ast.UnionSwitch, tbl *resolver.SymbolTable) error {
	unionName := inferUnionName(sw, tbl)

	// switch header
	if sw.BindVar != "" {
		fmt.Fprintf(b, "switch %s := %s.(type) {\n", sw.BindVar, sw.Subject)
	} else {
		fmt.Fprintf(b, "switch %s.(type) {\n", sw.Subject)
	}

	for _, uc := range sw.Cases {
		var typeNames []string
		for _, name := range uc.VariantNames {
			v, _, ok := tbl.LookupVariant(name)
			if !ok {
				return fmt.Errorf("unknown variant %q in switch on %s", name, unionName)
			}
			typeNames = append(typeNames, goTypeName(unionName, v))
		}
		fmt.Fprintf(b, "case %s:\n", strings.Join(typeNames, ", "))
		if uc.Body != "" {
			fmt.Fprintf(b, "%s\n", uc.Body)
		}
	}

	if sw.HasDefault {
		b.WriteString("default:\n")
		if sw.DefaultBody != "" {
			fmt.Fprintf(b, "%s\n", sw.DefaultBody)
		}
	}

	b.WriteString("}\n")
	return nil
}

func inferUnionName(sw ast.UnionSwitch, tbl *resolver.SymbolTable) string {
	tail := resolver.TailIdent(sw.Subject)
	if _, ok := tbl.Unions[tail]; ok {
		return tail
	}
	for _, uc := range sw.Cases {
		for _, name := range uc.VariantNames {
			if u, ok := tbl.VariantToUnion[name]; ok {
				return u
			}
		}
	}
	return tail
}

// goTypeName returns the Go type name used in type switches and method receivers.
// Atom variants get a generated private type; payload variants reuse the payload type directly.
func goTypeName(unionName string, v resolver.Variant) string {
	if v.IsAtom {
		return "_" + unionName + v.Name
	}
	return v.PayloadType
}

func title(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
