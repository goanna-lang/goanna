package lsp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nahmanmate/gounion/ast"
	"github.com/nahmanmate/gounion/checker"
	"github.com/nahmanmate/gounion/resolver"
)

// CompletionItem is an LSP completion item.
type CompletionItem struct {
	Label  string `json:"label"`
	Kind   int    `json:"kind"` // 13 = EnumMember
	Detail string `json:"detail,omitempty"`
}

// UnionSwitchCompletionContext detects if the cursor is on a case line inside a union switch.
// Returns variant completion items and whether this is a union switch context.
func UnionSwitchCompletionContext(
	srcBytes []byte,
	astFile *ast.File,
	tbl *resolver.SymbolTable,
	line, col int,
) ([]CompletionItem, bool) {
	cursorOffset := positionToByteOffset(srcBytes, line, col)

	for _, item := range astFile.Items {
		sw, ok := item.(ast.UnionSwitch)
		if !ok {
			continue
		}
		if sw.EndOffset == 0 {
			continue
		}
		if cursorOffset < sw.Line || cursorOffset >= sw.EndOffset {
			continue
		}

		lineText := getLineText(srcBytes, line)
		if !isCaseLine(lineText) {
			continue
		}

		unionName := checker.InferUnionName(sw, tbl)
		variants, ok := tbl.Unions[unionName]
		if !ok {
			return nil, false
		}

		var items []CompletionItem
		for _, v := range variants {
			items = append(items, CompletionItem{
				Label:  v.Name,
				Kind:   13,
				Detail: fmt.Sprintf("%s variant of %s", v.Name, unionName),
			})
		}
		return items, true
	}
	return nil, false
}

func isCaseLine(line string) bool {
	return strings.HasPrefix(strings.TrimLeft(line, " \t"), "case")
}

func getLineText(src []byte, line int) string {
	lines := bytes.Split(src, []byte{'\n'})
	if line >= len(lines) {
		return ""
	}
	return string(lines[line])
}

func positionToByteOffset(src []byte, line, col int) int {
	offset := 0
	for i := 0; i < line && offset < len(src); i++ {
		idx := bytes.IndexByte(src[offset:], '\n')
		if idx == -1 {
			return len(src)
		}
		offset += idx + 1
	}
	offset += col
	if offset > len(src) {
		return len(src)
	}
	return offset
}

// completionResponse builds a JSON-RPC result for a completion list.
func completionResponse(items []CompletionItem) json.RawMessage {
	type completionList struct {
		IsIncomplete bool             `json:"isIncomplete"`
		Items        []CompletionItem `json:"items"`
	}
	if items == nil {
		items = []CompletionItem{}
	}
	result := completionList{IsIncomplete: false, Items: items}
	b, _ := json.Marshal(result)
	return b
}
