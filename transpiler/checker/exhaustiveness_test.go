package checker

import (
	"strings"
	"testing"

	"github.com/nahmanmate/goanna/ast"
	"github.com/nahmanmate/goanna/resolver"
)

// genderTable is a shared SymbolTable used across checker tests.
func genderTable() *resolver.SymbolTable {
	return &resolver.SymbolTable{
		Unions: map[string][]resolver.Variant{
			"gender": {
				{Name: "Male", PayloadType: "atom", IsAtom: true},
				{Name: "Female", PayloadType: "atom", IsAtom: true},
			},
			"deskConfig": {
				{Name: "config1", PayloadType: "normalConfig"},
				{Name: "config2", PayloadType: "fixedConfig"},
				{Name: "config3", PayloadType: "strangeConfig"},
			},
		},
		VariantToUnion: map[string]string{
			"Male":    "gender",
			"Female":  "gender",
			"config1": "deskConfig",
			"config2": "deskConfig",
			"config3": "deskConfig",
		},
	}
}

func makeSwitch(subject, bindVar string, hasDefault bool, cases ...ast.UnionCase) ast.UnionSwitch {
	return ast.UnionSwitch{
		Subject:    subject,
		BindVar:    bindVar,
		HasDefault: hasDefault,
		Cases:      cases,
	}
}

func caseOf(names ...string) ast.UnionCase {
	return ast.UnionCase{VariantNames: names}
}

func fileWith(switches ...ast.UnionSwitch) *ast.File {
	items := make([]ast.Item, len(switches))
	for i, s := range switches {
		items[i] = s
	}
	return &ast.File{Items: items}
}

// TestCheck covers all exhaustiveness scenarios.
func TestCheck(t *testing.T) {
	tbl := genderTable()

	tests := []struct {
		name            string
		file            *ast.File
		wantErrCount    int
		wantErrContains []string // substrings that must appear in errors
	}{
		{
			name: "exhaustive_no_default",
			file: fileWith(makeSwitch(
				"greg.gender", "", false,
				caseOf("Male"),
				caseOf("Female"),
			)),
			wantErrCount: 0,
		},
		{
			name: "exhaustive_with_default_opts_out",
			file: fileWith(makeSwitch(
				"greg.gender", "", true,
				caseOf("Male"),
				// Female missing but default present — no error
			)),
			wantErrCount: 0,
		},
		{
			name: "missing_one_variant",
			file: fileWith(makeSwitch(
				"greg.gender", "", false,
				caseOf("Male"),
			)),
			wantErrCount:    1,
			wantErrContains: []string{"non-exhaustive", "Female"},
		},
		{
			name:            "missing_all_variants",
			file:            fileWith(makeSwitch("greg.gender", "", false)),
			wantErrCount:    1,
			wantErrContains: []string{"non-exhaustive", "Male"},
		},
		{
			name: "unknown_variant_in_case",
			file: fileWith(makeSwitch(
				"greg.gender", "", false,
				caseOf("Nonexistent"),
				caseOf("Female"),
			)),
			// Checker emits: (1) unknown variant error, (2) non-exhaustive (Male missing).
			wantErrCount:    2,
			wantErrContains: []string{"not a variant"},
		},
		{
			name: "multiple_switches_multiple_errors",
			file: fileWith(
				makeSwitch("greg.gender", "", false, caseOf("Male")),   // missing Female
				makeSwitch("greg.gender", "", false, caseOf("Female")), // missing Male
			),
			wantErrCount: 2,
		},
		{
			name: "non_union_subject_skipped",
			// Subject "notAUnion" is not in tbl.Unions and no case labels help resolve it.
			file: fileWith(makeSwitch(
				"notAUnion", "", false,
				caseOf("X"),
			)),
			wantErrCount: 0,
		},
		{
			name: "infer_union_from_case_label",
			// Subject "g" is not a union name, but cases contain "Male" → infer gender.
			file: fileWith(makeSwitch(
				"g", "", false,
				caseOf("Male"),
			)),
			wantErrCount:    1,
			wantErrContains: []string{"non-exhaustive", "Female"},
		},
		{
			name: "payload_union_exhaustive",
			file: fileWith(makeSwitch(
				"cfg", "", false,
				caseOf("config1"),
				caseOf("config2"),
				caseOf("config3"),
			)),
			wantErrCount: 0,
		},
		{
			name: "payload_union_missing",
			file: fileWith(makeSwitch(
				"cfg", "", false,
				caseOf("config1"),
			)),
			wantErrCount:    1,
			wantErrContains: []string{"non-exhaustive", "config2"},
		},
		{
			name: "default_body_no_error",
			file: fileWith(func() ast.UnionSwitch {
				sw := makeSwitch("greg.gender", "", true, caseOf("Male"))
				sw.DefaultBody = "_ = 0"
				return sw
			}()),
			wantErrCount: 0,
		},
		{
			name:         "empty_file_no_errors",
			file:         &ast.File{Items: []ast.Item{}},
			wantErrCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Check(tt.file, tbl)
			if len(errs) != tt.wantErrCount {
				t.Fatalf("Check: got %d errors, want %d: %v", len(errs), tt.wantErrCount, errs)
			}
			combined := ""
			for _, e := range errs {
				combined += e.Error() + "\n"
			}
			for _, sub := range tt.wantErrContains {
				if !strings.Contains(combined, sub) {
					t.Errorf("errors %q missing substring %q", combined, sub)
				}
			}
		})
	}
}

// TestInferUnionName covers the two-strategy inference logic.
func TestInferUnionName(t *testing.T) {
	tbl := genderTable()

	tests := []struct {
		name    string
		sw      ast.UnionSwitch
		wantRes string
	}{
		{
			name:    "tail_ident_match",
			sw:      makeSwitch("greg.gender", "", false, caseOf("Male")),
			wantRes: "gender",
		},
		{
			name:    "case_label_fallback",
			sw:      makeSwitch("g", "", false, caseOf("Male")),
			wantRes: "gender",
		},
		{
			name:    "no_match_returns_tail",
			sw:      makeSwitch("greg.unknown", "", false),
			wantRes: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InferUnionName(tt.sw, tbl)
			if got != tt.wantRes {
				t.Errorf("InferUnionName: got %q, want %q", got, tt.wantRes)
			}
		})
	}
}
