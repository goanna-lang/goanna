package lsp

import (
	"strings"
	"testing"

	"github.com/nahmanmate/goanna/transpiler/emitter"
	"github.com/nahmanmate/goanna/transpiler/parser"
	"github.com/nahmanmate/goanna/transpiler/pipeline"
	"github.com/nahmanmate/goanna/transpiler/resolver"
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

func TestBuildSourceMap_PayloadVariantLines(t *testing.T) {
	src := []byte(`package main

type normalConfig struct{ r int }
type fixedConfig struct{ b int }
type strangeConfig struct{ g int }

type deskConfig union {
	config1 normalConfig
	config2 fixedConfig
	config3 strangeConfig
}
`)
	// Source lines (0-indexed):
	// 0: package main
	// 1: (blank)
	// 2: type normalConfig struct{ r int }
	// 3: type fixedConfig struct{ b int }
	// 4: type strangeConfig struct{ g int }
	// 5: (blank)
	// 6: type deskConfig union {
	// 7:     config1 normalConfig
	// 8:     config2 fixedConfig
	// 9:     config3 strangeConfig
	// 10: }

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

	// The union expands to (no atoms, so no wrapper structs):
	//   func (normalConfig) isDeskConfig() {}   ← genBase+0
	//   func (fixedConfig) isDeskConfig() {}    ← genBase+1
	//   func (strangeConfig) isDeskConfig() {}  ← genBase+2
	//   type deskConfig interface{...}          ← genBase+3
	genBase, _ := sm.ToGenerated(6, 0) // first line of union → genBase

	for i, want := range []struct {
		srcLine  int
		wantGen  int
		typeName string
	}{
		{7, genBase + 0, "normalConfig"},
		{8, genBase + 1, "fixedConfig"},
		{9, genBase + 2, "strangeConfig"},
	} {
		gotLine, _ := sm.ToGenerated(want.srcLine, 0)
		if gotLine != want.wantGen {
			t.Errorf("variant %d (%s): ToGenerated line = %d, want %d", i, want.typeName, gotLine, want.wantGen)
		}

		// Back-translate: every generated line in the block should map back to decl start.
		srcBack, _ := sm.ToSource(want.wantGen, 0)
		if srcBack != 6 {
			t.Errorf("variant %d (%s): ToSource(%d) = %d, want 6 (decl start)", i, want.typeName, want.wantGen, srcBack)
		}
	}

	// Column mapping: cursor on type name in source should land on type name in generated.
	// Source line 8 = "\tconfig2 fixedConfig"; "fixedConfig" starts at col 9 (tab=1 char + "config2 " = 9).
	srcTypeCol := typeNameCol(src, 8)
	// Cursor at srcTypeCol should map to genTypeCol (6 = len("func (")).
	_, gotCol := sm.ToGenerated(8, srcTypeCol)
	if gotCol != genTypeCol {
		t.Errorf("col mapping: cursor at srcTypeCol %d → genCol %d, want %d", srcTypeCol, gotCol, genTypeCol)
	}
	// Cursor one char into type name should shift by one.
	_, gotCol2 := sm.ToGenerated(8, srcTypeCol+3)
	if gotCol2 != genTypeCol+3 {
		t.Errorf("col mapping: cursor at srcTypeCol+3 → genCol %d, want %d", gotCol2, genTypeCol+3)
	}
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
	result, err := pipeline.TranspileForLSP(src, "test.goa")
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
