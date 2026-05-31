package pipeline

import (
	"bytes"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const genderBasicSrc = `package main

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
}`

const missingCaseSrc = `package main

type atom struct{}

type gender union {
	Male, Female atom
}

func main() {
	var g gender
	switch g.(union) {
	case Male:
	}
}`

const opaqueOnlySrc = `package main

func main() {}`

// TestTranspile is the primary table-driven pipeline test.
func TestTranspile(t *testing.T) {
	tests := []struct {
		name               string
		src                string
		wantErrContains    string
		wantOutputContains string
	}{
		{
			name:               "simple_atom",
			src:                genderBasicSrc,
			wantOutputContains: "isGender()",
		},
		{
			name: "full_example",
			src: func() string {
				b, _ := os.ReadFile(filepath.Join("..", "..", "testdata", "full_example.union.go"))
				return string(b)
			}(),
			wantOutputContains: "isDeskConfig()",
		},
		{
			name:            "missing_case_error",
			src:             missingCaseSrc,
			wantErrContains: "non-exhaustive",
		},
		{
			name:               "opaque_only",
			src:                opaqueOnlySrc,
			wantOutputContains: "func main()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := Transpile([]byte(tt.src), tt.name, &buf)

			if tt.wantErrContains != "" {
				if err == nil {
					t.Fatalf("want error containing %q, got nil", tt.wantErrContains)
				}
				if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("error %q missing %q", err.Error(), tt.wantErrContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("Transpile: %v", err)
			}
			if tt.wantOutputContains != "" && !strings.Contains(buf.String(), tt.wantOutputContains) {
				t.Errorf("output missing %q\n--- output ---\n%s", tt.wantOutputContains, buf.String())
			}
		})
	}
}

// TestTranspileFileNotFound verifies error on missing input file.
func TestTranspileFileNotFound(t *testing.T) {
	var buf bytes.Buffer
	err := TranspileFile("nonexistent_file_that_does_not_exist.union.go", &buf)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention filename: %v", err)
	}
}

// TestTranspileOutputIsValidGo verifies all testdata inputs produce parseable Go.
func TestTranspileOutputIsValidGo(t *testing.T) {
	inputs, err := filepath.Glob(filepath.Join("..", "..", "testdata", "*.union.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range inputs {
		t.Run(filepath.Base(input), func(t *testing.T) {
			var buf bytes.Buffer
			if err := TranspileFile(input, &buf); err != nil {
				t.Fatalf("TranspileFile: %v", err)
			}
			fset := token.NewFileSet()
			_, err := goparser.ParseFile(fset, "", buf.String(), goparser.AllErrors)
			if err != nil {
				t.Errorf("output is not valid Go: %v\n--- output ---\n%s", err, buf.String())
			}
		})
	}
}

// TestTranspileErrorFiles verifies all error testdata inputs are rejected.
func TestTranspileErrorFiles(t *testing.T) {
	inputs, err := filepath.Glob(filepath.Join("..", "..", "testdata", "errors", "*.union.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range inputs {
		t.Run(filepath.Base(input), func(t *testing.T) {
			var buf bytes.Buffer
			err := TranspileFile(input, &buf)
			if err == nil {
				t.Errorf("expected error from %s, got nil", input)
			}
		})
	}
}

// TestTranspilePreservesOpaqueText checks verbatim passthrough of non-union code.
func TestTranspilePreservesOpaqueText(t *testing.T) {
	src := "package main\n\nconst Answer = 42\n\nfunc main() {}"
	var buf bytes.Buffer
	if err := Transpile([]byte(src), "test", &buf); err != nil {
		t.Fatalf("Transpile: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "Answer") {
		t.Errorf("opaque constant not preserved\n%s", got)
	}
	if !strings.Contains(got, "42") {
		t.Errorf("opaque value not preserved\n%s", got)
	}
}

// TestTranspileMultipleUnions verifies both unions are emitted.
func TestTranspileMultipleUnions(t *testing.T) {
	src := `package main

type atom struct{}

type gender union { Male, Female atom }
type shape union { Circle, Square atom }
`
	var buf bytes.Buffer
	if err := Transpile([]byte(src), "test", &buf); err != nil {
		t.Fatalf("Transpile: %v", err)
	}
	got := buf.String()
	for _, want := range []string{"isGender()", "isShape()", "_genderMale", "_shapeCircle"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\n%s", want, got)
		}
	}
}

// FuzzTranspile ensures the pipeline never panics on arbitrary input.
func FuzzTranspile(f *testing.F) {
	seeds := []string{
		"package main",
		genderBasicSrc,
		opaqueOnlySrc,
		`type g union { A atom }` + "\nswitch x.(union) { case A:\ndefault: }",
		``,
		`type`,
		`switch .(union) {}`,
		`type g union {}`,
	}

	// Add testdata files as seeds.
	for _, p := range []string{
		"../../testdata/gender_basic.union.go",
		"../../testdata/full_example.union.go",
		"../../testdata/payload_only.union.go",
	} {
		if b, err := os.ReadFile(p); err == nil {
			seeds = append(seeds, string(b))
		}
	}

	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, src []byte) {
		var buf bytes.Buffer
		_ = Transpile(src, "fuzz", &buf)
	})
}
