package lsp

import (
	"bytes"

	"github.com/goanna-lang/goanna/transpiler/ast"
	"github.com/goanna-lang/goanna/transpiler/emitter"
)

// SourceMap maps lines between .goa source and raw emitter output.
// All line numbers are 0-indexed. -1 means no direct mapping (inside an expansion).
type SourceMap struct {
	SrcToGen    []int // srcLine → genLine
	GenToSrc    []int // genLine → srcLine
	variantInfo map[int]variantInfo
}

// variantInfo holds column-aware forward mapping for a union variant source line.
// The generated target is the marker method line: func (TypeName) isUnion() {}
// where TypeName always starts at column 6 (after "func (").
type variantInfo struct {
	srcTypeCol int // column where the type name starts in the source line
}

const genTypeCol = 6 // len("func (") — where the type name sits in a marker method

// Build constructs a SourceMap from the AST and per-item emitter line ranges.
// srcBytes is the original .goa source.
func Build(srcBytes []byte, astFile *ast.File, itemRanges []emitter.ItemLineRange) *SourceMap {
	srcLineCount := bytes.Count(srcBytes, []byte{'\n'}) + 1

	genLineCount := 0
	for _, r := range itemRanges {
		if r.End > genLineCount {
			genLineCount = r.End
		}
	}
	genLineCount += 2 // safety margin

	sm := &SourceMap{
		SrcToGen:    make([]int, srcLineCount+1),
		GenToSrc:    make([]int, genLineCount+1),
		variantInfo: make(map[int]variantInfo),
	}
	for i := range sm.SrcToGen {
		sm.SrcToGen[i] = -1
	}
	for i := range sm.GenToSrc {
		sm.GenToSrc[i] = -1
	}

	srcLine := 0
	for idx, item := range astFile.Items {
		if idx >= len(itemRanges) {
			break
		}
		gr := itemRanges[idx]

		switch v := item.(type) {
		case ast.OpaqueChunk:
			n := bytes.Count([]byte(v.Text), []byte{'\n'})
			for j := 0; j < n; j++ {
				sl := srcLine + j
				gl := gr.Start + j
				if sl < len(sm.SrcToGen) && gl < len(sm.GenToSrc) {
					sm.SrcToGen[sl] = gl
					sm.GenToSrc[gl] = sl
				}
			}
			srcLine += n

		case ast.UnionDecl:
			declSrcStart := byteToLine(srcBytes, v.Line)
			declSrcEnd := byteToLine(srcBytes, v.EndOffset)

			if declSrcStart < len(sm.SrcToGen) && gr.Start < len(sm.GenToSrc) {
				sm.SrcToGen[declSrcStart] = gr.Start
				// Map every generated line back to the first source line so that
				// go-to-definition (e.g. on the interface line deep in the emitted
				// block) translates back correctly.
				for gl := gr.Start; gl < gr.End && gl < len(sm.GenToSrc); gl++ {
					sm.GenToSrc[gl] = declSrcStart
				}
			}

			// Map each variant group source line to its marker method generated line.
			// Emitter order: numAtomNames wrapper structs, then all marker methods in order.
			numAtomNames := 0
			for _, vg := range v.Variants {
				if vg.Type == "atom" {
					numAtomNames += len(vg.Names)
				}
			}

			expandedIdx := 0
			for i, vg := range v.Variants {
				srcVLine := declSrcStart + 1 + i
				if srcVLine >= len(sm.SrcToGen) {
					break
				}
				genMarkerLine := gr.Start + numAtomNames + expandedIdx
				sm.SrcToGen[srcVLine] = genMarkerLine
				sm.variantInfo[srcVLine] = variantInfo{
					srcTypeCol: typeNameCol(srcBytes, srcVLine),
				}
				expandedIdx += len(vg.Names)
			}

			srcLine = declSrcEnd

		case ast.UnionSwitch:
			swSrcStart := byteToLine(srcBytes, v.Line)
			swSrcEnd := byteToLine(srcBytes, v.EndOffset)
			n := swSrcEnd - swSrcStart
			genN := gr.End - gr.Start
			count := n
			if genN < count {
				count = genN
			}
			for j := 0; j < count; j++ {
				sl := swSrcStart + j
				gl := gr.Start + j
				if sl < len(sm.SrcToGen) && gl < len(sm.GenToSrc) {
					sm.SrcToGen[sl] = gl
					sm.GenToSrc[gl] = sl
				}
			}
			srcLine = swSrcEnd
		}
	}
	return sm
}

// typeNameCol returns the column (0-indexed) where the last whitespace-separated
// token starts on the given source line — that token is the type name in a variant group.
func typeNameCol(srcBytes []byte, lineNum int) int {
	cur := 0
	for i := 0; i < lineNum; i++ {
		idx := bytes.IndexByte(srcBytes[cur:], '\n')
		if idx < 0 {
			return 0
		}
		cur += idx + 1
	}
	end := bytes.IndexByte(srcBytes[cur:], '\n')
	var line []byte
	if end < 0 {
		line = srcBytes[cur:]
	} else {
		line = srcBytes[cur : cur+end]
	}
	// Trim trailing whitespace
	for len(line) > 0 && (line[len(line)-1] == ' ' || line[len(line)-1] == '\t') {
		line = line[:len(line)-1]
	}
	// Walk back to find start of last token
	i := len(line) - 1
	for i > 0 && line[i-1] != ' ' && line[i-1] != '\t' {
		i--
	}
	return i
}

// byteToLine counts the number of newlines before offset in data (= 0-indexed line number).
func byteToLine(data []byte, offset int) int {
	if offset > len(data) {
		offset = len(data)
	}
	return bytes.Count(data[:offset], []byte{'\n'})
}

// ToGenerated maps a source position to the corresponding generated position.
func (sm *SourceMap) ToGenerated(srcLine, srcCol int) (int, int) {
	if srcLine < 0 || srcLine >= len(sm.SrcToGen) {
		return srcLine, srcCol
	}
	g := sm.SrcToGen[srcLine]
	if g == -1 {
		return srcLine, srcCol
	}
	if info, ok := sm.variantInfo[srcLine]; ok {
		if srcCol >= info.srcTypeCol {
			return g, genTypeCol + (srcCol - info.srcTypeCol)
		}
		return g, genTypeCol
	}
	return g, srcCol
}

// ToSource maps a generated position to the corresponding source position.
func (sm *SourceMap) ToSource(genLine, genCol int) (int, int) {
	if genLine < 0 || genLine >= len(sm.GenToSrc) {
		return genLine, genCol
	}
	s := sm.GenToSrc[genLine]
	if s == -1 {
		return genLine, genCol
	}
	return s, genCol
}
