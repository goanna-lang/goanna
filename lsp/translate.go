package lsp

import "encoding/json"

// ForwardPosition maps a .union.go position to a .go position via the source map.
func ForwardPosition(sm *SourceMap, line, col int) (int, int) {
	if sm == nil {
		return line, col
	}
	return sm.ToGenerated(line, col)
}

// BackPosition maps a .go position to a .union.go position via the source map.
func BackPosition(sm *SourceMap, line, col int) (int, int) {
	if sm == nil {
		return line, col
	}
	return sm.ToSource(line, col)
}

// forwardParams translates the URI and position/range in standard LSP request params.
// Handles: {textDocument: {uri}, position: {...}} and {textDocument: {uri}, range: {...}}
func forwardParams(params json.RawMessage, genURI string, sm *SourceMap) json.RawMessage {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(params, &m); err != nil {
		return params
	}

	// Translate textDocument.uri
	if tdRaw, ok := m["textDocument"]; ok {
		var td map[string]json.RawMessage
		if err := json.Unmarshal(tdRaw, &td); err == nil {
			if _, hasURI := td["uri"]; hasURI {
				td["uri"], _ = json.Marshal(genURI)
				m["textDocument"], _ = json.Marshal(td)
			}
		}
	}

	// Translate position
	if posRaw, ok := m["position"]; ok {
		var pos Position
		if err := json.Unmarshal(posRaw, &pos); err == nil {
			pos.Line, pos.Character = ForwardPosition(sm, pos.Line, pos.Character)
			m["position"], _ = json.Marshal(pos)
		}
	}

	// Translate range
	if rangeRaw, ok := m["range"]; ok {
		var r Range
		if err := json.Unmarshal(rangeRaw, &r); err == nil {
			r.Start.Line, r.Start.Character = ForwardPosition(sm, r.Start.Line, r.Start.Character)
			r.End.Line, r.End.Character = ForwardPosition(sm, r.End.Line, r.End.Character)
			m["range"], _ = json.Marshal(r)
		}
	}

	out, _ := json.Marshal(m)
	return out
}

// backTranslateResult rewrites position fields in a gopls response result.
// Only translates ranges/positions when the URI matches the virtual .go file.
func backTranslateResult(result json.RawMessage, genURI, sourceURI string, sm *SourceMap) json.RawMessage {
	if len(result) == 0 || string(result) == "null" {
		return result
	}

	// Try as object.
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(result, &obj); err == nil {
		// Only rewrite uri and range when the location refers to the virtual
		// generated file; other URIs must pass through unchanged.
		if uriRaw, ok := obj["uri"]; ok {
			var uri string
			if err := json.Unmarshal(uriRaw, &uri); err == nil && uri == genURI {
				obj["uri"], _ = json.Marshal(sourceURI)
				if rangeRaw, ok := obj["range"]; ok {
					var r Range
					if err := json.Unmarshal(rangeRaw, &r); err == nil {
						r.Start.Line, r.Start.Character = BackPosition(sm, r.Start.Line, r.Start.Character)
						r.End.Line, r.End.Character = BackPosition(sm, r.End.Line, r.End.Character)
						obj["range"], _ = json.Marshal(r)
					}
				}
				out, _ := json.Marshal(obj)
				return out
			}
		}
		return result
	}

	// Try as array (e.g., definition returns []Location).
	var arr []json.RawMessage
	if err := json.Unmarshal(result, &arr); err == nil {
		for i, elem := range arr {
			arr[i] = backTranslateResult(elem, genURI, sourceURI, sm)
		}
		out, _ := json.Marshal(arr)
		return out
	}

	return result
}
