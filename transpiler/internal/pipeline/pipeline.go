package pipeline

import (
	"fmt"
	"go/format"
	"io"
	"os"
	"strings"

	"github.com/nahmanmate/gounion/internal/checker"
	"github.com/nahmanmate/gounion/internal/emitter"
	"github.com/nahmanmate/gounion/internal/parser"
	"github.com/nahmanmate/gounion/internal/resolver"
)

// TranspileFile reads inputPath, transpiles it, and writes valid Go to w.
func TranspileFile(inputPath string, w io.Writer) error {
	src, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", inputPath, err)
	}
	return Transpile(src, inputPath, w)
}

// Transpile transpiles src (labelled with name for error messages) and writes to w.
func Transpile(src []byte, name string, w io.Writer) error {
	file, err := parser.Parse(src)
	if err != nil {
		return fmt.Errorf("%s: parse: %w", name, err)
	}

	tbl, err := resolver.Build(file)
	if err != nil {
		return fmt.Errorf("%s: resolve: %w", name, err)
	}

	errs := checker.Check(file, tbl)
	if len(errs) > 0 {
		var msgs []string
		for _, e := range errs {
			msgs = append(msgs, e.Error())
		}
		return fmt.Errorf("%s: %s", name, strings.Join(msgs, "\n"))
	}

	raw, err := emitter.Emit(file, tbl)
	if err != nil {
		return fmt.Errorf("%s: emit: %w", name, err)
	}

	formatted, err := format.Source([]byte(raw))
	if err != nil {
		// Surface the unformatted source to aid debugging.
		return fmt.Errorf("%s: format: %w\n--- unformatted ---\n%s", name, err, raw)
	}

	_, err = w.Write(formatted)
	return err
}
