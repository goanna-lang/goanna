# Goanna

Goanna brings sum types (discriminated unions) to Go via a transpiler. Write `.goa` files using an extended syntax, run `goanna build`, and get valid, idiomatic Go out.

Designed around [this proposal](https://github.com/golang/go/issues/76920). If sum types land in the language spec, the syntax is intended to be compatible.

---

## Syntax

### Declaring a union type

```go
// Atom variants — no payload, enum-like
type gender union {
    Male, Female atom
}

// Payload variants — each variant carries a struct
type deskConfig union {
    config1 normalConfig
    config2 fixedConfig
    config3 strangeConfig
}
```

`atom` is a built-in marker type meaning "no payload".

### Switching over a union

```go
// Exhaustive switch — transpiler error if any variant is missing
switch greg.gender.(union) {
case Male:
case Female:
}

// Binding form — v takes the type of the matched variant's payload
switch v := greg.deskConfig.(union) {
case config1:
    _ = v.randNum   // v is normalConfig
case config2:
    _ = v.numb      // v is fixedConfig
case config3:
    _ = v.randStr   // v is strangeConfig
}

// default opts out of exhaustiveness checking
switch v := greg.deskConfig.(union) {
case config1:
    _ = v.people
default:
}
```

---

## What gets generated

| Input                                    | Output                                                                                   |
| ---------------------------------------- | ---------------------------------------------------------------------------------------- |
| Atom variant (`Male atom`)               | Private wrapper struct `_genderMale{}` + package-level `var Male gender = _genderMale{}` |
| Payload variant (`config1 normalConfig`) | `func (normalConfig) isDeskConfig() {}` on the existing struct                           |
| Union type                               | Sealed interface `type gender interface{ isGender() }`                                   |
| `switch x.(union)`                       | `switch x.(type)` with concrete private/struct cases                                     |

---

## Installation

```sh
go install github.com/goanna-lang/goanna/transpiler/cmd/goanna@latest
```

---

## Usage

### Build a project

```sh
goanna build ./...
```

Transpiles all `.goa` files in the module to a temp directory and passes them to `go build` via `-overlay`. Source tree stays clean — no generated files written to disk.

```sh
goanna build --keep ./...      # write generated .go files alongside source
goanna build ./pkg/...         # specific subtree only
```

### Transpile only (no build)

```sh
goanna foo.goa              # writes foo.go alongside
goanna foo.goa -o out.go    # explicit output path
goanna foo.goa -o -         # stdout
cat foo.goa | goanna        # stdin → stdout
```

### Validate without emitting output

```sh
goanna --check foo.goa         # single file
goanna build --check ./...     # validate all .goa files in module
```

### Formatting

Output is not formatted by default. Pass `--fmt` or `--gofumpt` to opt in. Both flags work with the direct transpile command and `goanna build`.

```sh
# gofmt (go/format)
goanna --fmt foo.goa
goanna build --fmt ./...

# gofumpt — stricter formatting, superset of gofmt
goanna --gofumpt foo.goa
goanna build --gofumpt ./...
```

If `gofumpt` is not installed when `--gofumpt` is passed, goanna prompts to install it via `go install`. On refusal or install failure, output falls back to `go/format`.

### CI / GitHub Actions

Add the reusable action to any workflow to validate or generate union types in CI.

**Validate only (recommended for most projects):**

```yaml
- uses: nahmanmate/goanna/action@main
```

**Generate and commit transpiled files:**

```yaml
- uses: nahmanmate/goanna/action@main
  with:
    mode: generate
```

**Full example workflow:**

```yaml
name: goanna

on:
  pull_request:
    paths: ['**/*.goa']

jobs:
  check:
    runs-on: ubuntu-latest
    permissions:
      contents: read
    steps:
      - uses: actions/checkout@v4
      - uses: nahmanmate/goanna/action@main
        with:
          mode: check      # validate exhaustiveness — no files written
          pattern: ./...
```

**Inputs:**

| Input | Default | Description |
|---|---|---|
| `mode` | `check` | `check` — validate only; `generate` — write transpiled `.go` files to disk |
| `pattern` | `./...` | Package pattern passed to `goanna build` |
| `version` | `latest` | goanna version to install (e.g. `v0.1.0`) |
| `go-version` | _(from go.mod)_ | Go version; defaults to the version declared in your `go.mod` |

---

## Exhaustiveness

Switches over a union type are exhaustive by default — the transpiler rejects the file if any variant is missing:

```go
// error: switch on gender is non-exhaustive: missing cases [Female]
switch g.(union) {
case Male:
}
```

Add `default:` to opt out:

```go
switch g.(union) {
case Male:
default: // ok — not all cases required
}
```

---

## Editor support (LSP)

`goanna-lsp` is a language server proxy that adds Goanna intelligence to any editor already running `gopls`. It handles `.goa` files transparently — diagnostics, completions, go-to-definition, and hover all work against your source, not the generated code.

```sh
go install github.com/goanna-lang/goanna/lsp/cmd/goanna-lsp@latest
goanna-lsp          # reads stdin, writes stdout — standard LSP stdio transport
```

See [`lsp/README.md`](lsp/README.md) for editor configuration, features, and architecture.

---

## How it works

```
.goa  →  parser  →  resolver  →  checker  →  emitter  →  [formatter]  →  .go
```

1. **Parser** — custom parser handles the `union` keyword and `.(union)` syntax; everything else is passed through verbatim
2. **Resolver** — builds a symbol table mapping variant names to their union types
3. **Checker** — validates exhaustiveness of every `.(union)` switch
4. **Emitter** — rewrites union declarations and switches to idiomatic Go
5. **Formatter** — optional; off by default. `--fmt` runs `go/format`, `--gofumpt` runs gofumpt

The generated code uses only standard Go — sealed interfaces, unexported wrapper structs, and type switches. No runtime dependency on Goanna.

The LSP proxy (`goanna-lsp`) runs the same pipeline in-memory on every edit, without formatting, and maps positions back to the source file so the editor never sees the generated code.
