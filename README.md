# GoUnion

GoUnion brings sum types (discriminated unions) to Go via a transpiler. Write `.union.go` files using an extended syntax, run `gounion build`, and get valid, idiomatic Go out.

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
go install github.com/nahmanmate/gounion/cmd/gounion@latest
```

---

## Usage

### Build a project

```sh
gounion build ./...
```

Transpiles all `.union.go` files in the module to a temp directory and passes them to `go build` via `-overlay`. Source tree stays clean — no generated files written to disk.

```sh
gounion build --keep ./...   # write generated .go files alongside source
gounion build ./pkg/...      # specific subtree only
```

> **Note on `--keep`:** if both `foo.union.go` and the generated `foo.go` are present, plain `go build ./...` will try to compile both. Add `//go:build ignore` to your `.union.go` files to prevent this.

### Transpile only (no build)

```sh
gounion foo.union.go              # writes foo.go alongside
gounion foo.union.go -o out.go    # explicit output path
gounion foo.union.go -o -         # stdout
cat foo.union.go | gounion        # stdin → stdout
```

### Validate without emitting output

```sh
gounion --check foo.union.go
gounion build --check ./...       # not yet supported; use --check per file
```

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

## How it works

```
.union.go  →  parser  →  resolver  →  checker  →  emitter  →  go/format  →  .go
```

1. **Parser** — custom parser handles the `union` keyword and `.(union)` syntax; everything else is passed through verbatim
2. **Resolver** — builds a symbol table mapping variant names to their union types
3. **Checker** — validates exhaustiveness of every `.(union)` switch
4. **Emitter** — rewrites union declarations and switches to idiomatic Go
5. **Formatter** — runs `go/format` on the output

The generated code uses only standard Go — sealed interfaces, unexported wrapper structs, and type switches. No runtime dependency on GoUnion.
