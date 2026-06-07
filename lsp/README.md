# goanna-lsp

An LSP proxy that adds Goanna-aware intelligence to any editor already using `gopls`.

It sits between your editor and `gopls`, transparently transpiling `.goa` files before gopls sees them and translating positions back so diagnostics, completions, and go-to-definition all point at your source — not the generated code.

---

## How it works

```
editor  ──►  goanna-lsp  ──►  gopls
              │   ▲
              │   │  transpile .goa → .go
              │   │  translate positions  back
              ▼   │
            store (VirtualFile per open .goa)
```

For every `.goa` file the editor opens or edits:

1. The proxy transpiles the source in-memory via the same pipeline as `goanna build`
2. The generated `.go` is forwarded to gopls as a virtual file
3. Gopls responses (diagnostics, completions, definition locations) have their positions mapped back to the `.goa` source coordinates
4. Goanna-specific diagnostics (exhaustiveness errors) are merged with gopls diagnostics and sent as a single `publishDiagnostics` notification

Non-union files are forwarded to gopls unchanged.

### Union switch completions

Inside a `switch x.(union) { case ... }` block, the proxy intercepts `textDocument/completion` and returns the union's variants directly, without hitting gopls. Outside a union switch, completions fall through to gopls as normal.

---

## Installation

```sh
go install github.com/nahmanmate/goanna/lsp/cmd/goanna-lsp@latest
```

Requires `gopls` to be installed and on `PATH` (or passed via `--gopls=`).

---

## Usage

```sh
goanna-lsp                      # uses gopls from PATH
goanna-lsp --gopls=/path/gopls  # explicit gopls binary
```

The server reads JSON-RPC from stdin and writes to stdout — standard LSP stdio transport. Configure your editor to launch it as an LSP server for `*.goa` files.

### Neovim (via nvim-lspconfig)

Add filetype detection for `.goa` files:

```lua
vim.filetype.add({ extension = { goa = 'goa' } })
```

Then enable the server:

```lua
vim.lsp.enable('goanna_ls')
```

If `goanna_ls` is not yet in nvim-lspconfig, register it manually:

```lua
vim.lsp.config('goanna_ls', {
  cmd = { 'goanna-lsp' },
  filetypes = { 'goa' },
  root_markers = { 'go.work', 'go.mod' },
})
vim.lsp.enable('goanna_ls')
```

### VS Code

There is no dedicated goanna VS Code extension. Use the [generic LSP client extension](https://marketplace.visualstudio.com/items?itemName=llvm-vs-code-extensions.vscode-clangd) configured for Go files.

---

## Features

| Capability | Status |
|---|---|
| Diagnostics (exhaustiveness, unknown variants) | ✅ |
| Diagnostics from gopls (type errors, etc.) | ✅ merged |
| Completions — union variant names inside `case` | ✅ |
| Completions — everything else | ✅ via gopls |
| Go-to-definition | ✅ position translated |
| Hover | ✅ via gopls |
| References | ✅ via gopls |
| Formatting | ✅ via gopls (on generated .go) |

---

## Architecture

| File | Responsibility |
|---|---|
| `lsp.go` | Entry point — `Run()` wires the two I/O pumps |
| `proxy.go` | Message routing: editor → gopls and gopls → editor |
| `virtualfiles.go` | `Store` — tracks open `.goa` files and their gopls diagnostics |
| `sourcemap.go` | Bidirectional line map between `.goa` and generated `.go` |
| `translate.go` | JSON-level position/URI translation for arbitrary LSP params |
| `completions.go` | Union switch completion context detection |
| `diagnostics.go` | LSP `Diagnostic` types and `publishDiagnostics` helpers |
| `jsonrpc.go` | Content-Length framed JSON-RPC reader/writer |
| `gopls.go` | `gopls serve` subprocess management |

---

## Limitations

- `.goa` files are not formatted by gopls (gopls operates on the generated `.go`). Run `goanna build --keep` and format the output if needed.
- Rename / workspace symbol search operate on the generated file's symbol names (e.g. `_genderMale`), not the union source names.
- The proxy does not persist virtual files to disk; a gopls restart will lose state until the editor re-opens files.
