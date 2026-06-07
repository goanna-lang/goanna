package lsp

import (
	"bytes"
	"encoding/json"

	"github.com/nahmanmate/goanna/transpiler/checker"
)

// Position is an LSP zero-indexed line/character position.
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range is an LSP source range.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Diagnostic is an LSP diagnostic item.
type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity"` // 1=error 2=warning
	Source   string `json:"source,omitempty"`
	Message  string `json:"message"`
}

// ByteOffsetToPosition converts a byte offset in src to a 0-indexed LSP Position.
func ByteOffsetToPosition(src []byte, offset int) Position {
	if offset > len(src) {
		offset = len(src)
	}
	lineNum := bytes.Count(src[:offset], []byte{'\n'})
	lineStart := bytes.LastIndexByte(src[:offset], '\n') + 1
	return Position{Line: lineNum, Character: offset - lineStart}
}

// CheckErrorsToDiagnostics converts structured checker errors to LSP Diagnostics.
func CheckErrorsToDiagnostics(errs []*checker.CheckError, srcBytes []byte) []Diagnostic {
	var diags []Diagnostic
	for _, e := range errs {
		pos := ByteOffsetToPosition(srcBytes, e.ByteOffset)
		end := Position{Line: pos.Line, Character: pos.Character + 20}
		diags = append(diags, Diagnostic{
			Range:    Range{Start: pos, End: end},
			Severity: 1,
			Source:   "goanna",
			Message:  e.Message,
		})
	}
	return diags
}

// MergeDiagnostics combines gopls and checker diagnostics.
func MergeDiagnostics(goplsDiags, checkerDiags []Diagnostic) []Diagnostic {
	out := make([]Diagnostic, 0, len(goplsDiags)+len(checkerDiags))
	out = append(out, goplsDiags...)
	out = append(out, checkerDiags...)
	return out
}

// SendDiagnostics sends a textDocument/publishDiagnostics notification to w.
func SendDiagnostics(w *Writer, uri string, diags []Diagnostic) error {
	if diags == nil {
		diags = []Diagnostic{} // send empty array, not null
	}
	params := map[string]interface{}{
		"uri":         uri,
		"diagnostics": diags,
	}
	b, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return w.Write(&Message{
		Method: "textDocument/publishDiagnostics",
		Params: json.RawMessage(b),
	})
}
