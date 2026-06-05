# gounion-lsp

An LSP proxy that adds GoUnion-aware intelligence to any editor already using `gopls`.

It sits between your editor and `gopls`, transparently transpiling `.union.go` files before gopls sees them and translating positions back so diagnostics, completions, and go-to-definition all point at your source — not the generated code.

---

## How it works

```
editor  ──►  gounion-lsp  ──►  gopls
              │   ▲
              │   │  transpile .union.go → .go
              │   │  translate positions  back
              ▼   │
            store (VirtualFile per open .union.go)
```

For every `.union.go` file the editor opens or edits:

1. The proxy transpiles the source in-memory via the same pipeline as `gounion build`
2. The generated `.go` is forwarded to gopls as a virtual file
3. Gopls responses (diagnostics, completions, definition locations) have their positions mapped back to the `.union.go` source coordinates
4. GoUnion-specific diagnostics (exhaustiveness errors) are merged with gopls diagnostics and sent as a single `publishDiagnostics` notification

Non-union files are forwarded to gopls unchanged.

### Union switch completions

Inside a `switch x.(union) { case ... }` block, the proxy intercepts `textDocument/completion` and returns the union's variants directly, without hitting gopls. Outside a union switch, completions fall through to gopls as normal.

---

## Installation

```sh
go install github.com/nahmanmate/gounion/lsp/cmd/gounion-lsp@latest
```

Requires `gopls` to be installed and on `PATH` (or passed via `--gopls=`).

---

## Usage

```sh
gounion-lsp                      # uses gopls from PATH
gounion-lsp --gopls=/path/gopls  # explicit gopls binary
```

The server reads JSON-RPC from stdin and writes to stdout — standard LSP stdio transport. Configure your editor to launch it as an LSP server for `*.union.go` files.

### Neovim (via nvim-lspconfig)

```lua
local configs = require('lspconfig.configs')

configs.gounion = {
  default_config = {
    cmd = { 'gounion-lsp' },
    filetypes = { 'go' },
    root_dir = require('lspconfig.util').root_pattern('go.work', 'go.mod'),
  },
}

require('lspconfig').gounion.setup({})
```

### VS Code

Add a language server entry in your workspace `.vscode/settings.json`:

```json
{
  "languageServerExample.serverPath": "gounion-lsp"
}
```

Or use the [generic LSP client extension](https://marketplace.visualstudio.com/items?itemName=llvm-vs-code-extensions.vscode-clangd) configured for Go files.

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
| `virtualfiles.go` | `Store` — tracks open `.union.go` files and their gopls diagnostics |
| `sourcemap.go` | Bidirectional line map between `.union.go` and generated `.go` |
| `translate.go` | JSON-level position/URI translation for arbitrary LSP params |
| `completions.go` | Union switch completion context detection |
| `diagnostics.go` | LSP `Diagnostic` types and `publishDiagnostics` helpers |
| `jsonrpc.go` | Content-Length framed JSON-RPC reader/writer |
| `gopls.go` | `gopls serve` subprocess management |

---

## Limitations

- `.union.go` files are not formatted by gopls (gopls operates on the generated `.go`). Run `gounion build --keep` and format the output if needed.
- Rename / workspace symbol search operate on the generated file's symbol names (e.g. `_genderMale`), not the union source names.
- The proxy does not persist virtual files to disk; a gopls restart will lose state until the editor re-opens files.
