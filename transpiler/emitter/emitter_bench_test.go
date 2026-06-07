package emitter

import (
	"os"
	"testing"

	"github.com/nahmanmate/goanna/transpiler/ast"
	"github.com/nahmanmate/goanna/transpiler/parser"
	"github.com/nahmanmate/goanna/transpiler/resolver"
)

type emitterBenchCase struct {
	name string
	file *ast.File
	tbl  *resolver.SymbolTable
	size int
}

func loadEmitterBenchCases(b *testing.B) []emitterBenchCase {
	b.Helper()
	inputs := []struct{ name, path string }{
		{"gender_basic", "../testdata/gender_basic.goa"},
		{"payload_only", "../testdata/payload_only.goa"},
		{"full_example", "../testdata/full_example.goa"},
		{"crud_api", "../testdata/crud_api.goa"},
	}
	cases := make([]emitterBenchCase, 0, len(inputs))
	for _, tc := range inputs {
		src, err := os.ReadFile(tc.path)
		if err != nil {
			b.Fatalf("read %s: %v", tc.path, err)
		}
		file, err := parser.Parse(src)
		if err != nil {
			b.Fatalf("parse %s: %v", tc.path, err)
		}
		tbl, err := resolver.Build(file)
		if err != nil {
			b.Fatalf("resolve %s: %v", tc.path, err)
		}
		cases = append(cases, emitterBenchCase{name: tc.name, file: file, tbl: tbl, size: len(src)})
	}
	return cases
}

func BenchmarkEmit(b *testing.B) {
	for _, tc := range loadEmitterBenchCases(b) {
		b.Run(tc.name, func(b *testing.B) {
			b.SetBytes(int64(tc.size))
			for b.Loop() {
				if _, err := Emit(tc.file, tc.tbl); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkEmitWithLineMap(b *testing.B) {
	for _, tc := range loadEmitterBenchCases(b) {
		b.Run(tc.name, func(b *testing.B) {
			b.SetBytes(int64(tc.size))
			for b.Loop() {
				if _, _, err := EmitWithLineMap(tc.file, tc.tbl); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
