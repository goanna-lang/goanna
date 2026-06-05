package lsp

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/nahmanmate/gounion/pipeline"
)

// pendingRequest tracks a forwarded request awaiting a gopls response.
type pendingRequest struct {
	sourceURI string // empty if not a .union.go file
	method    string
}

type pendingMap struct {
	mu sync.Mutex
	m  map[string]pendingRequest
}

func newPendingMap() *pendingMap { return &pendingMap{m: make(map[string]pendingRequest)} }

func (pm *pendingMap) set(id string, req pendingRequest) {
	pm.mu.Lock()
	pm.m[id] = req
	pm.mu.Unlock()
}

func (pm *pendingMap) get(id string) (pendingRequest, bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	r, ok := pm.m[id]
	return r, ok
}

func (pm *pendingMap) del(id string) {
	pm.mu.Lock()
	delete(pm.m, id)
	pm.mu.Unlock()
}

func idStr(id json.RawMessage) string {
	if id == nil {
		return ""
	}
	return string(id)
}

// Proxy routes LSP messages between the editor and a gopls subprocess.
type Proxy struct {
	store     *Store
	editorOut *Writer
	gopls     *GoplsProcess
	pending   *pendingMap
	logger    *log.Logger
	nextID    atomic.Int64
}

func NewProxy(store *Store, editorOut *Writer, gopls *GoplsProcess, logger *log.Logger) *Proxy {
	p := &Proxy{
		store:     store,
		editorOut: editorOut,
		gopls:     gopls,
		pending:   newPendingMap(),
		logger:    logger,
	}
	p.nextID.Store(1_000_000)
	return p
}

// HandleEditorMessage processes an incoming message from the editor.
func (p *Proxy) HandleEditorMessage(msg *Message) {
	switch msg.Method {
	case "textDocument/didOpen":
		p.handleDidOpen(msg)
	case "textDocument/didChange":
		p.handleDidChange(msg)
	case "textDocument/didClose":
		p.handleDidClose(msg)
	case "textDocument/didSave":
		p.handleDidSave(msg)
	case "textDocument/completion":
		p.handleCompletion(msg)
	case "initialize":
		p.handleInitialize(msg)
	default:
		p.forwardEditorMessage(msg)
	}
}

// HandleGoplsMessage processes an incoming message from gopls.
func (p *Proxy) HandleGoplsMessage(msg *Message) {
	if msg.IsNotification() && msg.Method == "textDocument/publishDiagnostics" {
		p.handlePublishDiagnostics(msg)
		return
	}
	if msg.IsResponse() {
		p.handleGoplsResponse(msg)
		return
	}
	// Forward everything else (window/progress, workspace, etc.)
	if err := p.editorOut.Write(msg); err != nil {
		p.logger.Printf("write to editor: %v", err)
	}
}

func (p *Proxy) handleInitialize(msg *Message) {
	// Register pending so we can intercept the response and force full sync.
	if msg.ID != nil {
		p.pending.set(idStr(msg.ID), pendingRequest{method: "initialize"})
	}
	_ = p.gopls.In.Write(msg)
}

func (p *Proxy) handleDidOpen(msg *Message) {
	var params struct {
		TextDocument struct {
			URI     string `json:"uri"`
			Text    string `json:"text"`
			Version int    `json:"version"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		p.logger.Printf("didOpen parse: %v", err)
		return
	}
	if !p.store.IsUnionFile(params.TextDocument.URI) {
		_ = p.gopls.In.Write(msg)
		return
	}

	vf := p.transpileAndStore(params.TextDocument.URI, []byte(params.TextDocument.Text))
	if vf.ParseError != nil {
		p.sendCheckerDiags(vf)
		return
	}
	p.sendDidOpenToGopls(vf, params.TextDocument.Version)
	p.sendCheckerDiags(vf)
}

func (p *Proxy) handleDidChange(msg *Message) {
	var params struct {
		TextDocument struct {
			URI     string `json:"uri"`
			Version int    `json:"version"`
		} `json:"textDocument"`
		ContentChanges []struct {
			Text string `json:"text"`
		} `json:"contentChanges"`
	}
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		p.logger.Printf("didChange parse: %v", err)
		return
	}
	if !p.store.IsUnionFile(params.TextDocument.URI) {
		_ = p.gopls.In.Write(msg)
		return
	}
	if len(params.ContentChanges) == 0 {
		return
	}

	src := []byte(params.ContentChanges[len(params.ContentChanges)-1].Text)
	vf := p.transpileAndStore(params.TextDocument.URI, src)
	if vf.ParseError != nil {
		p.sendCheckerDiags(vf)
		return
	}

	changeParams := map[string]any{
		"textDocument": map[string]any{
			"uri":     vf.GenURI,
			"version": params.TextDocument.Version,
		},
		"contentChanges": []map[string]any{
			{"text": string(vf.GenBytes)},
		},
	}
	b, _ := json.Marshal(changeParams)
	_ = p.gopls.In.Write(&Message{Method: "textDocument/didChange", Params: b})
	p.sendCheckerDiags(vf)
}

func (p *Proxy) handleDidClose(msg *Message) {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(msg.Params, &params); err != nil || !p.store.IsUnionFile(params.TextDocument.URI) {
		_ = p.gopls.In.Write(msg)
		return
	}

	if vf, ok := p.store.Get(params.TextDocument.URI); ok {
		closeParams := map[string]any{
			"textDocument": map[string]any{"uri": vf.GenURI},
		}
		b, _ := json.Marshal(closeParams)
		_ = p.gopls.In.Write(&Message{Method: "textDocument/didClose", Params: b})
	}
	p.store.Delete(params.TextDocument.URI)
	_ = SendDiagnostics(p.editorOut, params.TextDocument.URI, nil)
}

func (p *Proxy) handleDidSave(msg *Message) {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(msg.Params, &params); err != nil || !p.store.IsUnionFile(params.TextDocument.URI) {
		_ = p.gopls.In.Write(msg)
		return
	}
	if vf, ok := p.store.Get(params.TextDocument.URI); ok {
		saveParams := map[string]any{
			"textDocument": map[string]any{"uri": vf.GenURI},
		}
		b, _ := json.Marshal(saveParams)
		_ = p.gopls.In.Write(&Message{Method: "textDocument/didSave", Params: b})
	}
}

func (p *Proxy) handleCompletion(msg *Message) {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Position Position `json:"position"`
	}
	if err := json.Unmarshal(msg.Params, &params); err != nil || !p.store.IsUnionFile(params.TextDocument.URI) {
		p.forwardEditorMessage(msg)
		return
	}

	vf, ok := p.store.Get(params.TextDocument.URI)
	if !ok || vf.ASTFile == nil {
		p.forwardEditorMessage(msg)
		return
	}

	items, isUnion := UnionSwitchCompletionContext(
		vf.SourceBytes, vf.ASTFile, vf.SymTable,
		params.Position.Line, params.Position.Character,
	)
	if isUnion {
		_ = p.editorOut.Write(&Message{ID: msg.ID, Result: completionResponse(items)})
		return
	}
	p.forwardEditorMessage(msg)
}

func (p *Proxy) handlePublishDiagnostics(msg *Message) {
	var params struct {
		URI         string       `json:"uri"`
		Diagnostics []Diagnostic `json:"diagnostics"`
	}
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		_ = p.editorOut.Write(msg)
		return
	}

	vf, ok := p.store.GetByGenURI(params.URI)
	if !ok {
		_ = p.editorOut.Write(msg)
		return
	}

	// Translate positions back to source file coordinates.
	var translated []Diagnostic
	for _, d := range params.Diagnostics {
		if vf.SourceMap != nil {
			d.Range.Start.Line, d.Range.Start.Character = BackPosition(vf.SourceMap, d.Range.Start.Line, d.Range.Start.Character)
			d.Range.End.Line, d.Range.End.Character = BackPosition(vf.SourceMap, d.Range.End.Line, d.Range.End.Character)
		}
		d.Source = "gopls"
		translated = append(translated, d)
	}

	p.store.StoreGoplsDiags(vf.SourceURI, translated)

	merged := MergeDiagnostics(translated, CheckErrorsToDiagnostics(vf.CheckErrors, vf.SourceBytes))
	_ = SendDiagnostics(p.editorOut, vf.SourceURI, merged)
}

func (p *Proxy) handleGoplsResponse(msg *Message) {
	key := idStr(msg.ID)
	req, ok := p.pending.get(key)
	if !ok {
		_ = p.editorOut.Write(msg)
		return
	}
	p.pending.del(key)

	// Special: intercept initialize response to force full text sync.
	if req.method == "initialize" {
		msg.Result = forceFullSync(msg.Result)
		_ = p.editorOut.Write(msg)
		return
	}

	if req.sourceURI == "" || msg.Result == nil {
		_ = p.editorOut.Write(msg)
		return
	}

	vf, ok := p.store.Get(req.sourceURI)
	if !ok || vf.SourceMap == nil {
		_ = p.editorOut.Write(msg)
		return
	}

	msg.Result = backTranslateResult(msg.Result, vf.GenURI, vf.SourceURI, vf.SourceMap)
	_ = p.editorOut.Write(msg)
}

// forwardEditorMessage translates URI/position for any textDocument message and forwards to gopls.
func (p *Proxy) forwardEditorMessage(msg *Message) {
	if msg.Params == nil {
		_ = p.gopls.In.Write(msg)
		return
	}

	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(msg.Params, &params); err != nil || !p.store.IsUnionFile(params.TextDocument.URI) {
		_ = p.gopls.In.Write(msg)
		return
	}

	vf, ok := p.store.Get(params.TextDocument.URI)
	if !ok {
		_ = p.gopls.In.Write(msg)
		return
	}

	translated := &Message{
		JSONRPC: msg.JSONRPC,
		ID:      msg.ID,
		Method:  msg.Method,
		Params:  forwardParams(msg.Params, vf.GenURI, vf.SourceMap),
	}
	if msg.ID != nil {
		p.pending.set(idStr(msg.ID), pendingRequest{sourceURI: params.TextDocument.URI, method: msg.Method})
	}
	_ = p.gopls.In.Write(translated)
}

// transpileAndStore transpiles src for uri and updates the store.
func (p *Proxy) transpileAndStore(uri string, src []byte) *VirtualFile {
	vf := &VirtualFile{
		SourceURI:   uri,
		GenURI:      sourceToGenURI(uri),
		SourceBytes: src,
	}

	result, err := pipeline.TranspileForLSP(src, uri)
	if err != nil {
		vf.ParseError = err
		p.store.Set(vf)
		return vf
	}

	sm := Build(src, result.ASTFile, result.ItemRanges)
	vf.GenBytes = result.Generated
	vf.SourceMap = sm
	vf.ASTFile = result.ASTFile
	vf.SymTable = result.SymTable
	vf.CheckErrors = result.CheckErrors
	p.store.Set(vf)
	return vf
}

func (p *Proxy) sendDidOpenToGopls(vf *VirtualFile, version int) {
	params := map[string]any{
		"textDocument": map[string]any{
			"uri":        vf.GenURI,
			"languageId": "go",
			"version":    version,
			"text":       string(vf.GenBytes),
		},
	}
	b, _ := json.Marshal(params)
	_ = p.gopls.In.Write(&Message{Method: "textDocument/didOpen", Params: b})
}

func (p *Proxy) sendCheckerDiags(vf *VirtualFile) {
	var checkerDiags []Diagnostic
	if vf.ParseError != nil {
		checkerDiags = append(checkerDiags, Diagnostic{
			Severity: 1,
			Source:   "gounion",
			Message:  fmt.Sprintf("parse error: %v", vf.ParseError),
		})
	} else {
		checkerDiags = CheckErrorsToDiagnostics(vf.CheckErrors, vf.SourceBytes)
	}
	merged := MergeDiagnostics(p.store.GetGoplsDiags(vf.SourceURI), checkerDiags)
	_ = SendDiagnostics(p.editorOut, vf.SourceURI, merged)
}

// forceFullSync patches the initialize result to use full document sync.
func forceFullSync(result json.RawMessage) json.RawMessage {
	if len(result) == 0 {
		return result
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(result, &obj); err != nil {
		return result
	}
	capsRaw, ok := obj["capabilities"]
	if !ok {
		return result
	}
	var caps map[string]json.RawMessage
	if err := json.Unmarshal(capsRaw, &caps); err != nil {
		return result
	}
	// textDocumentSync = 1 (Full)
	caps["textDocumentSync"], _ = json.Marshal(map[string]any{
		"openClose": true,
		"change":    1,
		"save":      true,
	})
	obj["capabilities"], _ = json.Marshal(caps)
	out, _ := json.Marshal(obj)
	return out
}
