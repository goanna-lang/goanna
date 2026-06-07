package parser

import (
	"os"
	"testing"
)

var parserBenchInputs = []struct {
	name string
	path string
}{
	{"gender_basic", "../testdata/gender_basic.goa"},
	{"payload_only", "../testdata/payload_only.goa"},
	{"full_example", "../testdata/full_example.goa"},
	{"crud_api", "../testdata/crud_api.goa"},
}

func BenchmarkParse(b *testing.B) {
	for _, tc := range parserBenchInputs {
		src, err := os.ReadFile(tc.path)
		if err != nil {
			b.Fatalf("read %s: %v", tc.path, err)
		}
		b.Run(tc.name, func(b *testing.B) {
			b.SetBytes(int64(len(src)))
			for b.Loop() {
				if _, err := Parse(src); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
