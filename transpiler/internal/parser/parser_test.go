package parser

import (
	"testing"

	"github.com/nahmanmate/gounion/internal/ast"
)

func mustParse(t *testing.T, src string) *ast.File {
	t.Helper()
	f, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse(%q): %v", src, err)
	}
	return f
}

func findDecls(f *ast.File) []ast.UnionDecl {
	var out []ast.UnionDecl
	for _, item := range f.Items {
		if d, ok := item.(ast.UnionDecl); ok {
			out = append(out, d)
		}
	}
	return out
}

func findSwitches(f *ast.File) []ast.UnionSwitch {
	var out []ast.UnionSwitch
	for _, item := range f.Items {
		if s, ok := item.(ast.UnionSwitch); ok {
			out = append(out, s)
		}
	}
	return out
}

func findChunks(f *ast.File) []ast.OpaqueChunk {
	var out []ast.OpaqueChunk
	for _, item := range f.Items {
		if c, ok := item.(ast.OpaqueChunk); ok {
			out = append(out, c)
		}
	}
	return out
}

// TestParseUnionDecl covers union declaration parsing.
func TestParseUnionDecl(t *testing.T) {
	tests := []struct {
		name        string
		src         string
		wantName    string
		wantGroups  int
		wantNames   []string // first group names
		wantType    string   // first group type
	}{
		{
			name:       "atom_single",
			src:        `type gender union { Male atom }`,
			wantName:   "gender",
			wantGroups: 1,
			wantNames:  []string{"Male"},
			wantType:   "atom",
		},
		{
			name:       "atom_multi",
			src:        `type gender union { Male, Female atom }`,
			wantName:   "gender",
			wantGroups: 1,
			wantNames:  []string{"Male", "Female"},
			wantType:   "atom",
		},
		{
			name:       "payload_single",
			src:        `type deskConfig union { config1 normalConfig }`,
			wantName:   "deskConfig",
			wantGroups: 1,
			wantNames:  []string{"config1"},
			wantType:   "normalConfig",
		},
		{
			name:       "multi_group",
			src:        "type shape union {\n\tcircle Circle\n\tsquare Square\n}",
			wantName:   "shape",
			wantGroups: 2,
			wantNames:  []string{"circle"},
			wantType:   "Circle",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := mustParse(t, tt.src)
			decls := findDecls(f)
			if len(decls) != 1 {
				t.Fatalf("want 1 UnionDecl, got %d", len(decls))
			}
			d := decls[0]
			if d.Name != tt.wantName {
				t.Errorf("Name: got %q, want %q", d.Name, tt.wantName)
			}
			if len(d.Variants) != tt.wantGroups {
				t.Fatalf("Variants: got %d groups, want %d", len(d.Variants), tt.wantGroups)
			}
			g := d.Variants[0]
			if len(g.Names) != len(tt.wantNames) {
				t.Fatalf("first group Names: got %v, want %v", g.Names, tt.wantNames)
			}
			for i, n := range tt.wantNames {
				if g.Names[i] != n {
					t.Errorf("Names[%d]: got %q, want %q", i, g.Names[i], n)
				}
			}
			if g.Type != tt.wantType {
				t.Errorf("Type: got %q, want %q", g.Type, tt.wantType)
			}
		})
	}
}

// TestParseMultipleUnions checks that multiple union declarations in one file are all captured.
func TestParseMultipleUnions(t *testing.T) {
	src := `type gender union { Male, Female atom }
type shape union { circle Circle }`
	f := mustParse(t, src)
	decls := findDecls(f)
	if len(decls) != 2 {
		t.Fatalf("want 2 UnionDecls, got %d", len(decls))
	}
	if decls[0].Name != "gender" {
		t.Errorf("decls[0].Name: got %q, want %q", decls[0].Name, "gender")
	}
	if decls[1].Name != "shape" {
		t.Errorf("decls[1].Name: got %q, want %q", decls[1].Name, "shape")
	}
}

// TestParseOpaquePassthrough verifies non-union source is preserved as OpaqueChunk.
func TestParseOpaquePassthrough(t *testing.T) {
	src := "package main\nfunc main(){}"
	f := mustParse(t, src)
	decls := findDecls(f)
	if len(decls) != 0 {
		t.Fatalf("want 0 UnionDecls, got %d", len(decls))
	}
	chunks := findChunks(f)
	if len(chunks) == 0 {
		t.Fatal("want at least 1 OpaqueChunk, got 0")
	}
}

// TestParseUnionWithOpaque verifies chunk before a union decl is captured separately.
func TestParseUnionWithOpaque(t *testing.T) {
	src := "var x int\ntype g union { A atom }"
	f := mustParse(t, src)
	if len(f.Items) < 2 {
		t.Fatalf("want at least 2 items, got %d", len(f.Items))
	}
	if _, ok := f.Items[0].(ast.OpaqueChunk); !ok {
		t.Errorf("Items[0]: want OpaqueChunk, got %T", f.Items[0])
	}
	if _, ok := f.Items[1].(ast.UnionDecl); !ok {
		t.Errorf("Items[1]: want UnionDecl, got %T", f.Items[1])
	}
}

// TestParseUnionSwitch covers switch statement parsing.
func TestParseUnionSwitch(t *testing.T) {
	tests := []struct {
		name           string
		src            string
		wantBindVar    string
		wantSubject    string
		wantCases      int
		wantHasDefault bool
		wantFirstCase  []string // variant names of first case
	}{
		{
			name:          "bare_switch",
			src:           "switch x.(union) { case A: }",
			wantBindVar:   "",
			wantSubject:   "x",
			wantCases:     1,
			wantFirstCase: []string{"A"},
		},
		{
			name:          "binding_switch",
			src:           "switch v := x.(union) { case A: }",
			wantBindVar:   "v",
			wantSubject:   "x",
			wantCases:     1,
			wantFirstCase: []string{"A"},
		},
		{
			name:           "with_default",
			src:            "switch x.(union) { case A:\ndefault: }",
			wantSubject:    "x",
			wantCases:      1,
			wantHasDefault: true,
		},
		{
			name:        "dotted_subject",
			src:         "switch a.b.c.(union) { case A: }",
			wantSubject: "a.b.c",
			wantCases:   1,
		},
		{
			name:          "multi_case_labels",
			src:           "switch x.(union) { case A, B: }",
			wantSubject:   "x",
			wantCases:     1,
			wantFirstCase: []string{"A", "B"},
		},
		{
			name:          "multiple_cases",
			src:           "switch x.(union) { case A:\ncase B: }",
			wantSubject:   "x",
			wantCases:     2,
			wantFirstCase: []string{"A"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := mustParse(t, tt.src)
			switches := findSwitches(f)
			if len(switches) != 1 {
				t.Fatalf("want 1 UnionSwitch, got %d", len(switches))
			}
			sw := switches[0]
			if sw.BindVar != tt.wantBindVar {
				t.Errorf("BindVar: got %q, want %q", sw.BindVar, tt.wantBindVar)
			}
			if sw.Subject != tt.wantSubject {
				t.Errorf("Subject: got %q, want %q", sw.Subject, tt.wantSubject)
			}
			if len(sw.Cases) != tt.wantCases {
				t.Fatalf("Cases: got %d, want %d", len(sw.Cases), tt.wantCases)
			}
			if sw.HasDefault != tt.wantHasDefault {
				t.Errorf("HasDefault: got %v, want %v", sw.HasDefault, tt.wantHasDefault)
			}
			if tt.wantFirstCase != nil && len(sw.Cases) > 0 {
				got := sw.Cases[0].VariantNames
				if len(got) != len(tt.wantFirstCase) {
					t.Fatalf("Cases[0].VariantNames: got %v, want %v", got, tt.wantFirstCase)
				}
				for i, n := range tt.wantFirstCase {
					if got[i] != n {
						t.Errorf("Cases[0].VariantNames[%d]: got %q, want %q", i, got[i], n)
					}
				}
			}
		})
	}
}

// TestParseCaseBody verifies verbatim body extraction, including nested braces.
func TestParseCaseBody(t *testing.T) {
	tests := []struct {
		name        string
		src         string
		isDefault   bool
		wantBody    string
	}{
		{
			name:      "simple_body",
			src:       "switch x.(union) { case A:\n_ = 1\ncase B: }",
			wantBody:  "_ = 1",
		},
		{
			name:      "nested_braces",
			src:       "switch x.(union) { case A:\nif true { _ = 1 }\ncase B: }",
			wantBody:  "if true { _ = 1 }",
		},
		{
			name:      "default_body",
			src:       "switch x.(union) { default:\n_ = 2 }",
			isDefault: true,
			wantBody:  "_ = 2",
		},
		{
			name:     "empty_body",
			src:      "switch x.(union) { case A:\ncase B: }",
			wantBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := mustParse(t, tt.src)
			switches := findSwitches(f)
			if len(switches) != 1 {
				t.Fatalf("want 1 switch, got %d", len(switches))
			}
			sw := switches[0]
			if tt.isDefault {
				if sw.DefaultBody != tt.wantBody {
					t.Errorf("DefaultBody: got %q, want %q", sw.DefaultBody, tt.wantBody)
				}
				return
			}
			if len(sw.Cases) == 0 {
				t.Fatal("no cases")
			}
			if sw.Cases[0].Body != tt.wantBody {
				t.Errorf("Cases[0].Body: got %q, want %q", sw.Cases[0].Body, tt.wantBody)
			}
		})
	}
}

// TestParseRegularSwitchIsOpaque ensures a non-union switch is not parsed as a UnionSwitch.
func TestParseRegularSwitchIsOpaque(t *testing.T) {
	src := "switch x { case 1: }"
	f := mustParse(t, src)
	switches := findSwitches(f)
	if len(switches) != 0 {
		t.Errorf("regular switch should not be parsed as UnionSwitch, got %d", len(switches))
	}
}

// FuzzParse ensures the parser never panics on arbitrary input.
func FuzzParse(f *testing.F) {
	seeds := []string{
		`package main`,
		`type g union { A atom }`,
		`switch x.(union) { case A: }`,
		"type g union { A, B atom }\nswitch x.(union) { case A:\ncase B: }",
		`type d union { c1 Config\n c2 OtherConfig }`,
		`switch v := x.y.(union) { case A:\n_ = v.field\ndefault: }`,
		``,
		`type`,
		`switch`,
		`.(union)`,
		`union`,
		`{{{`,
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, src []byte) {
		_, _ = Parse(src)
	})
}
