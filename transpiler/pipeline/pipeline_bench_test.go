package pipeline

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

var benchInputs = []struct {
	name string
	path string
}{
	{"gender_basic", "../testdata/gender_basic.goa"},
	{"payload_only", "../testdata/payload_only.goa"},
	{"full_example", "../testdata/full_example.goa"},
	{"crud_api", "../testdata/crud_api.goa"},
}

func BenchmarkTranspile(b *testing.B) {
	for _, tc := range benchInputs {
		src, err := os.ReadFile(tc.path)
		if err != nil {
			b.Fatalf("read %s: %v", tc.path, err)
		}
		b.Run(tc.name, func(b *testing.B) {
			b.SetBytes(int64(len(src)))
			for b.Loop() {
				var buf bytes.Buffer
				if err := Transpile(src, tc.name, &buf); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkTranspileForLSP(b *testing.B) {
	for _, tc := range benchInputs {
		src, err := os.ReadFile(tc.path)
		if err != nil {
			b.Fatalf("read %s: %v", tc.path, err)
		}
		b.Run(tc.name, func(b *testing.B) {
			b.SetBytes(int64(len(src)))
			for b.Loop() {
				if _, err := TranspileForLSP(src, tc.name); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkTranspileAllFiles(b *testing.B) {
	inputs, err := filepath.Glob("../testdata/*.goa")
	if err != nil || len(inputs) == 0 {
		b.Fatal("no testdata files found")
	}
	srcs := make([][]byte, len(inputs))
	total := 0
	for i, path := range inputs {
		srcs[i], err = os.ReadFile(path)
		if err != nil {
			b.Fatalf("read %s: %v", path, err)
		}
		total += len(srcs[i])
	}
	b.SetBytes(int64(total))
	for b.Loop() {
		for i, src := range srcs {
			if err := Transpile(src, inputs[i], io.Discard); err != nil {
				b.Fatal(err)
			}
		}
	}
}

// BenchmarkTranspileScale shows sequential vs parallel crossover as file count grows.
// srcs are replicated (cycling through testdata) to reach fileCount.
func BenchmarkTranspileScale(b *testing.B) {
	inputs, err := filepath.Glob("../testdata/*.goa")
	if err != nil || len(inputs) == 0 {
		b.Fatal("no testdata files found")
	}
	base := make([][]byte, len(inputs))
	for i, path := range inputs {
		base[i], err = os.ReadFile(path)
		if err != nil {
			b.Fatalf("read %s: %v", path, err)
		}
	}

	for _, n := range []int{4, 8, 16, 32, 3000} {
		srcs := make([][]byte, n)
		total := 0
		for i := range srcs {
			srcs[i] = base[i%len(base)]
			total += len(srcs[i])
		}

		b.Run(fmt.Sprintf("seq/n=%d", n), func(b *testing.B) {
			b.SetBytes(int64(total))
			for b.Loop() {
				for _, src := range srcs {
					if err := Transpile(src, "bench", io.Discard); err != nil {
						b.Fatal(err)
					}
				}
			}
		})

		b.Run(fmt.Sprintf("par/n=%d", n), func(b *testing.B) {
			b.SetBytes(int64(total))
			for b.Loop() {
				type result struct{ err error }
				ch := make(chan result, n)
				for _, src := range srcs {
					go func(src []byte) {
						ch <- result{Transpile(src, "bench", io.Discard)}
					}(src)
				}
				for range srcs {
					if r := <-ch; r.err != nil {
						b.Fatal(r.err)
					}
				}
			}
		})
	}
}

func BenchmarkTranspileAllFilesParallel(b *testing.B) {
	inputs, err := filepath.Glob("../testdata/*.goa")
	if err != nil || len(inputs) == 0 {
		b.Fatal("no testdata files found")
	}
	srcs := make([][]byte, len(inputs))
	total := 0
	for i, path := range inputs {
		srcs[i], err = os.ReadFile(path)
		if err != nil {
			b.Fatalf("read %s: %v", path, err)
		}
		total += len(srcs[i])
	}
	b.SetBytes(int64(total))
	for b.Loop() {
		type result struct{ err error }
		ch := make(chan result, len(inputs))
		for i, src := range srcs {
			go func(src []byte, name string) {
				ch <- result{Transpile(src, name, io.Discard)}
			}(src, inputs[i])
		}
		for range inputs {
			if r := <-ch; r.err != nil {
				b.Fatal(r.err)
			}
		}
	}
}
