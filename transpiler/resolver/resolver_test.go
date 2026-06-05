package resolver

import (
	"testing"

	"github.com/nahmanmate/gounion/ast"
)

func makeFile(decls ...ast.UnionDecl) *ast.File {
	items := make([]ast.Item, len(decls))
	for i, d := range decls {
		items[i] = d
	}
	return &ast.File{Items: items}
}

func makeDecl(name string, groups ...ast.VariantGroup) ast.UnionDecl {
	return ast.UnionDecl{Name: name, Variants: groups}
}

func group(typ string, names ...string) ast.VariantGroup {
	return ast.VariantGroup{Names: names, Type: typ}
}

// TestBuild covers symbol table construction.
func TestBuild(t *testing.T) {
	tests := []struct {
		name               string
		file               *ast.File
		wantUnionNames     []string
		wantVariantCount   map[string]int  // union name → variant count
		wantIsAtom         map[string]bool // variant name → IsAtom
		wantVariantToUnion map[string]string
		wantErr            bool
	}{
		{
			name: "single_atom_union",
			file: makeFile(makeDecl(
				"gender",
				group("atom", "Male", "Female"),
			)),
			wantUnionNames:   []string{"gender"},
			wantVariantCount: map[string]int{"gender": 2},
			wantIsAtom: map[string]bool{
				"Male":   true,
				"Female": true,
			},
			wantVariantToUnion: map[string]string{
				"Male":   "gender",
				"Female": "gender",
			},
		},
		{
			name: "payload_union",
			file: makeFile(makeDecl(
				"deskConfig",
				group("normalConfig", "config1"),
				group("fixedConfig", "config2"),
			)),
			wantUnionNames:   []string{"deskConfig"},
			wantVariantCount: map[string]int{"deskConfig": 2},
			wantIsAtom: map[string]bool{
				"config1": false,
				"config2": false,
			},
			wantVariantToUnion: map[string]string{
				"config1": "deskConfig",
				"config2": "deskConfig",
			},
		},
		{
			name: "multi_union",
			file: makeFile(
				makeDecl("gender", group("atom", "Male", "Female")),
				makeDecl("shape", group("Circle", "circle"), group("Square", "square")),
			),
			wantUnionNames:   []string{"gender", "shape"},
			wantVariantCount: map[string]int{"gender": 2, "shape": 2},
			wantVariantToUnion: map[string]string{
				"Male":   "gender",
				"Female": "gender",
				"circle": "shape",
				"square": "shape",
			},
		},
		{
			name: "duplicate_variant_errors",
			file: makeFile(
				makeDecl("gender", group("atom", "Male", "Female")),
				makeDecl("shape", group("atom", "Male")), // Male clash
			),
			wantErr: true,
		},
		{
			name:             "empty_union",
			file:             makeFile(makeDecl("empty")),
			wantUnionNames:   []string{"empty"},
			wantVariantCount: map[string]int{"empty": 0},
		},
		{
			name:           "no_decls",
			file:           &ast.File{Items: []ast.Item{ast.OpaqueChunk{Text: "package main"}}},
			wantUnionNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tbl, err := Build(tt.file)
			if tt.wantErr {
				if err == nil {
					t.Fatal("want error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Build: %v", err)
			}

			for _, uname := range tt.wantUnionNames {
				variants, ok := tbl.Unions[uname]
				if !ok {
					t.Errorf("Unions[%q] missing", uname)
					continue
				}
				if wc, ok := tt.wantVariantCount[uname]; ok && len(variants) != wc {
					t.Errorf("Unions[%q]: got %d variants, want %d", uname, len(variants), wc)
				}
			}

			for vname, wantAtom := range tt.wantIsAtom {
				v, _, ok := tbl.LookupVariant(vname)
				if !ok {
					t.Errorf("LookupVariant(%q): not found", vname)
					continue
				}
				if v.IsAtom != wantAtom {
					t.Errorf("Variant %q IsAtom: got %v, want %v", vname, v.IsAtom, wantAtom)
				}
			}

			for vname, wantUnion := range tt.wantVariantToUnion {
				got, ok := tbl.VariantToUnion[vname]
				if !ok {
					t.Errorf("VariantToUnion[%q] missing", vname)
					continue
				}
				if got != wantUnion {
					t.Errorf("VariantToUnion[%q]: got %q, want %q", vname, got, wantUnion)
				}
			}
		})
	}
}

// TestTailIdent covers expression tail extraction.
func TestTailIdent(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"x", "x"},
		{"a.b", "b"},
		{"greg.deskConfig", "deskConfig"},
		{"a.b.c.d", "d"},
		{"", ""},
		{".", ""},
		{".foo", "foo"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := TailIdent(tt.input)
			if got != tt.want {
				t.Errorf("TailIdent(%q): got %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestLookupVariant covers variant lookup by name.
func TestLookupVariant(t *testing.T) {
	tbl := &SymbolTable{
		Unions: map[string][]Variant{
			"gender": {
				{Name: "Male", PayloadType: "atom", IsAtom: true},
				{Name: "Female", PayloadType: "atom", IsAtom: true},
			},
			"deskConfig": {
				{Name: "config1", PayloadType: "normalConfig", IsAtom: false},
			},
		},
		VariantToUnion: map[string]string{
			"Male":    "gender",
			"Female":  "gender",
			"config1": "deskConfig",
		},
	}

	tests := []struct {
		name      string
		lookup    string
		wantFound bool
		wantUnion string
		wantAtom  bool
	}{
		{"atom_variant", "Male", true, "gender", true},
		{"second_atom", "Female", true, "gender", true},
		{"payload_variant", "config1", true, "deskConfig", false},
		{"not_found", "Unknown", false, "", false},
		{"empty", "", false, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, union, ok := tbl.LookupVariant(tt.lookup)
			if ok != tt.wantFound {
				t.Fatalf("LookupVariant(%q): found=%v, want %v", tt.lookup, ok, tt.wantFound)
			}
			if !ok {
				return
			}
			if union != tt.wantUnion {
				t.Errorf("union: got %q, want %q", union, tt.wantUnion)
			}
			if v.IsAtom != tt.wantAtom {
				t.Errorf("IsAtom: got %v, want %v", v.IsAtom, tt.wantAtom)
			}
		})
	}
}
