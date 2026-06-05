package emitter

import (
	"flag"
	"go/format"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nahmanmate/gounion/parser"
	"github.com/nahmanmate/gounion/resolver"
)

var updateGolden = flag.Bool("update", false, "update golden files")

func testdataPath(rel string) string {
	return filepath.Join("..", "testdata", rel)
}

func goldenPath(name string) string {
	return testdataPath(filepath.Join("golden", name))
}

// mustEmit runs parse → resolve → emit → format on src.
func mustEmit(t *testing.T, src []byte) string {
	t.Helper()
	f, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tbl, err := resolver.Build(f)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	raw, err := Emit(f, tbl)
	if err != nil {
		t.Fatalf("emit: %v", err)
	}
	formatted, err := format.Source([]byte(raw))
	if err != nil {
		t.Fatalf("format: %v\n--- raw ---\n%s", err, raw)
	}
	return string(formatted)
}

// TestEmitGolden compares emitter output against golden files.
// Run with -update to regenerate golden files.
func TestEmitGolden(t *testing.T) {
	cases := []struct {
		inputFile  string
		goldenFile string
	}{
		{"gender_basic.union.go", "gender_basic.go"},
		{"full_example.union.go", "full_example.go"},
		{"payload_only.union.go", "payload_only.go"},
	}

	for _, tc := range cases {
		t.Run(tc.goldenFile, func(t *testing.T) {
			src, err := os.ReadFile(testdataPath(tc.inputFile))
			if err != nil {
				t.Fatalf("read input: %v", err)
			}
			got := mustEmit(t, src)

			golden := goldenPath(tc.goldenFile)
			if *updateGolden {
				if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
					t.Fatalf("write golden: %v", err)
				}
				t.Logf("updated %s", golden)
				return
			}

			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("read golden (run with -update to create): %v", err)
			}
			if got != string(want) {
				t.Errorf("output mismatch\n--- want ---\n%s\n--- got ---\n%s", want, got)
			}
		})
	}
}

// TestEmitProducesValidGo verifies all testdata inputs produce parseable Go.
func TestEmitProducesValidGo(t *testing.T) {
	inputs, err := filepath.Glob(testdataPath("*.union.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range inputs {
		t.Run(filepath.Base(input), func(t *testing.T) {
			src, err := os.ReadFile(input)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			got := mustEmit(t, src)
			fset := token.NewFileSet()
			_, err = goparser.ParseFile(fset, "", got, goparser.AllErrors)
			if err != nil {
				t.Errorf("output is not valid Go: %v\n%s", err, got)
			}
		})
	}
}

// TestGoTypeName covers the internal type name mapping.
func TestGoTypeName(t *testing.T) {
	tests := []struct {
		unionName string
		variant   resolver.Variant
		want      string
	}{
		{
			unionName: "gender",
			variant:   resolver.Variant{Name: "Male", PayloadType: "atom", IsAtom: true},
			want:      "_genderMale",
		},
		{
			unionName: "gender",
			variant:   resolver.Variant{Name: "Female", PayloadType: "atom", IsAtom: true},
			want:      "_genderFemale",
		},
		{
			unionName: "deskConfig",
			variant:   resolver.Variant{Name: "config1", PayloadType: "normalConfig", IsAtom: false},
			want:      "normalConfig",
		},
		{
			unionName: "color",
			variant:   resolver.Variant{Name: "red", PayloadType: "redConfig", IsAtom: false},
			want:      "redConfig",
		},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := goTypeName(tt.unionName, tt.variant)
			if got != tt.want {
				t.Errorf("goTypeName(%q, %+v): got %q, want %q", tt.unionName, tt.variant, got, tt.want)
			}
		})
	}
}

// TestTitle covers the title-case helper.
func TestTitle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"gender", "Gender"},
		{"deskConfig", "DeskConfig"},
		{"G", "G"},
		{"", ""},
		{"a", "A"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := title(tt.input)
			if got != tt.want {
				t.Errorf("title(%q): got %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestEmitAtomUnionStructure checks emitted output contains expected declarations.
func TestEmitAtomUnionStructure(t *testing.T) {
	src := `type gender union { Male, Female atom }`
	got := mustEmit(t, []byte(src))

	checks := []string{
		"type _genderMale struct{}",
		"type _genderFemale struct{}",
		"func (_genderMale) isGender()",
		"func (_genderFemale) isGender()",
		"type gender interface",
		"isGender()",
		"var Male gender",
		"var Female gender",
	}
	for _, c := range checks {
		if !strings.Contains(got, c) {
			t.Errorf("output missing %q\n--- output ---\n%s", c, got)
		}
	}
}

// TestEmitPayloadUnionStructure checks emitted output for payload unions.
func TestEmitPayloadUnionStructure(t *testing.T) {
	src := "type deskConfig union {\n\tconfig1 normalConfig\n\tconfig2 fixedConfig\n}"
	got := mustEmit(t, []byte(src))

	checks := []string{
		"func (normalConfig) isDeskConfig()",
		"func (fixedConfig) isDeskConfig()",
		"type deskConfig interface",
		"isDeskConfig()",
	}
	for _, c := range checks {
		if !strings.Contains(got, c) {
			t.Errorf("output missing %q\n--- output ---\n%s", c, got)
		}
	}
	// Payload unions must NOT emit var declarations.
	if strings.Contains(got, "var config1") || strings.Contains(got, "var config2") {
		t.Errorf("payload union should not emit package-level vars\n%s", got)
	}
}

// TestEmitSwitchBare checks a union switch without binding variable.
func TestEmitSwitchBare(t *testing.T) {
	src := "package main\ntype gender union { Male, Female atom }\nfunc f(x gender) {\nswitch x.(union) {\ncase Male:\ncase Female:\ndefault:\n}}"
	got := mustEmit(t, []byte(src))

	if !strings.Contains(got, "switch x.(type)") {
		t.Errorf("missing type switch header\n%s", got)
	}
	if !strings.Contains(got, "case _genderMale:") {
		t.Errorf("missing case _genderMale\n%s", got)
	}
	if !strings.Contains(got, "case _genderFemale:") {
		t.Errorf("missing case _genderFemale\n%s", got)
	}
	if !strings.Contains(got, "default:") {
		t.Errorf("missing default\n%s", got)
	}
}

// TestEmitSwitchBinding checks a union switch with a binding variable.
func TestEmitSwitchBinding(t *testing.T) {
	src := "package main\ntype normalConfig struct{ people int }\ntype deskConfig union {\n\tconfig1 normalConfig\n}\nfunc f(x deskConfig) {\nswitch v := x.(union) {\ncase config1:\n_ = v.people\ndefault:\n}}"
	got := mustEmit(t, []byte(src))

	if !strings.Contains(got, "switch v := x.(type)") {
		t.Errorf("missing binding switch header\n%s", got)
	}
	if !strings.Contains(got, "case normalConfig:") {
		t.Errorf("missing case normalConfig\n%s", got)
	}
}
