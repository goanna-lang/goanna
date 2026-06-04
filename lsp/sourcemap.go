package lsp

import (
	"bytes"

	"github.com/nahmanmate/gounion/ast"
	"github.com/nahmanmate/gounion/emitter"
)

// SourceMap maps lines between .union.go source and raw emitter output.
// All line numbers are 0-indexed. -1 means no direct mapping (inside an expansion).
type SourceMap struct {
	SrcToGen []int // srcLine → genLine
	GenToSrc []int // genLine → srcLine
}

// Build constructs a SourceMap from the AST and per-item emitter line ranges.
// srcBytes is the original .union.go source.
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
		SrcToGen: make([]int, srcLineCount+1),
		GenToSrc: make([]int, genLineCount+1),
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
			// Map first src line of the block → first gen line.
			if declSrcStart < len(sm.SrcToGen) && gr.Start < len(sm.GenToSrc) {
				sm.SrcToGen[declSrcStart] = gr.Start
				sm.GenToSrc[gr.Start] = declSrcStart
			}
			srcLine = declSrcEnd + 1

		case ast.UnionSwitch:
			swSrcStart := byteToLine(srcBytes, v.Line)
			swSrcEnd := byteToLine(srcBytes, v.EndOffset)
			n := swSrcEnd - swSrcStart + 1
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
			srcLine = swSrcEnd + 1
		}
	}
	return sm
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
