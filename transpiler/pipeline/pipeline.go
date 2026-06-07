package pipeline

import (
	"fmt"
	"go/format"
	"io"
	"os"
	"strings"

	"github.com/nahmanmate/goanna/transpiler/ast"
	"github.com/nahmanmate/goanna/transpiler/checker"
	"github.com/nahmanmate/goanna/transpiler/emitter"
	"github.com/nahmanmate/goanna/transpiler/parser"
	"github.com/nahmanmate/goanna/transpiler/resolver"
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

	checkErrs := checker.Check(file, tbl)
	if len(checkErrs) > 0 {
		var msgs []string
		for _, e := range checkErrs {
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

// TranspileResult holds everything the LSP proxy needs from one transpilation.
type TranspileResult struct {
	Generated   []byte // raw (not gofmt'd) Go source
	ASTFile     *ast.File
	SymTable    *resolver.SymbolTable
	ItemRanges  []emitter.ItemLineRange // line ranges in Generated, parallel to ASTFile.Items
	CheckErrors []*checker.CheckError   // non-nil = union semantic errors; Generated is still valid Go
}

// TranspileForLSP transpiles src without aborting on checker errors and without go/format.
// Returns non-nil error only on parse or resolve failure.
// The caller should still send Generated to gopls even if CheckErrors is non-empty.
func TranspileForLSP(src []byte, name string) (*TranspileResult, error) {
	file, err := parser.Parse(src)
	if err != nil {
		return nil, fmt.Errorf("%s: parse: %w", name, err)
	}

	tbl, err := resolver.Build(file)
	if err != nil {
		return nil, fmt.Errorf("%s: resolve: %w", name, err)
	}

	checkErrs := checker.Check(file, tbl)

	raw, ranges, emitErr := emitter.EmitWithLineMap(file, tbl)
	if emitErr != nil {
		checkErrs = append(checkErrs, &checker.CheckError{
			Kind:    "emit_error",
			Message: fmt.Sprintf("%s: emit: %s", name, emitErr.Error()),
		})
	}

	return &TranspileResult{
		Generated:   []byte(raw),
		ASTFile:     file,
		SymTable:    tbl,
		ItemRanges:  ranges,
		CheckErrors: checkErrs,
	}, nil
}
