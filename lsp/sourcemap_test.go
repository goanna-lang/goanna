package lsp

import (
	"strings"
	"testing"

	"github.com/nahmanmate/gounion/emitter"
	"github.com/nahmanmate/gounion/parser"
	"github.com/nahmanmate/gounion/pipeline"
	"github.com/nahmanmate/gounion/resolver"
)

func TestBuildSourceMap_GenderBasic(t *testing.T) {
	src := []byte(`package main

type atom struct{}

type gender union {
	Male, Female atom
}

type person struct {
	name   string
	gender gender
}

func main() {
	greg := person{name: "Greg", gender: Male}
	switch greg.gender.(union) {
	case Male:
	case Female:
	default:
	}
}
`)

	astFile, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tbl, err := resolver.Build(astFile)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	_, ranges, err := emitter.EmitWithLineMap(astFile, tbl)
	if err != nil {
		t.Fatalf("emit: %v", err)
	}

	sm := Build(src, astFile, ranges)
	if sm == nil {
		t.Fatal("Build returned nil")
	}

	// Lines 0-2 ("package main", blank, "type atom struct{}") are opaque — must map 1:1.
	for _, srcLine := range []int{0, 1, 2, 3} {
		g, _ := sm.ToGenerated(srcLine, 0)
		if g != srcLine {
			t.Errorf("line %d: ToGenerated = %d, want %d", srcLine, g, srcLine)
		}
		s, _ := sm.ToSource(srcLine, 0)
		if s != srcLine {
			t.Errorf("line %d: ToSource = %d, want %d", srcLine, s, srcLine)
		}
	}

	// Line 4 is "type gender union {" — should map to the first gen line of the expansion.
	unionSrcLine := 4
	genLine, _ := sm.ToGenerated(unionSrcLine, 0)
	if genLine < 0 {
		t.Errorf("union decl src line %d: ToGenerated = %d, want >= 0", unionSrcLine, genLine)
	}
	// Back-translate the gen line should give a reasonable source line.
	srcBack, _ := sm.ToSource(genLine, 0)
	if srcBack < 0 {
		t.Errorf("gen line %d: ToSource = %d, want >= 0", genLine, srcBack)
	}

	t.Logf("SrcToGen[0..5] = %v", sm.SrcToGen[:min(6, len(sm.SrcToGen))])
	t.Logf("GenToSrc[0..5] = %v", sm.GenToSrc[:min(6, len(sm.GenToSrc))])
}

func TestTranspileForLSP_CheckErrors(t *testing.T) {
	src := []byte(`package main

type atom struct{}

type gender union {
	Male, Female atom
}

func main() {
	var g gender
	switch g.(union) {
	case Male:
	}
}
`)
	result, err := pipeline.TranspileForLSP(src, "test.union.go")
	if err != nil {
		t.Fatalf("unexpected parse/resolve error: %v", err)
	}
	if len(result.CheckErrors) == 0 {
		t.Error("expected exhaustiveness error, got none")
	}
	if result.Generated == nil {
		t.Error("Generated should be non-nil even when check errors exist")
	}
	if !strings.Contains(string(result.Generated), "isGender") {
		t.Error("Generated should contain marker method isGender")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
