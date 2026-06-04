# Contributing

GoUnion is pre-1.0 and experimental. Contributions welcome — breaking changes still happen.

## Prerequisites

- Go 1.26+
- [golangci-lint](https://golangci-lint.run/usage/install/) (for linting)

## Development setup

```sh
git clone https://github.com/nahmanmate/gounion
cd gounion/transpiler
go mod download
```

## Project layout

```
transpiler/
  cmd/gounion/        # CLI entry point
  internal/
    ast/              # union-extended AST types
    parser/           # parses .union.go files
    resolver/         # builds variant → union symbol table
    checker/          # exhaustiveness validation
    emitter/          # rewrites AST to idiomatic Go
    pipeline/         # orchestrates the above stages
  testdata/           # golden-file test fixtures
```

## Making changes

The pipeline runs sequentially: `parser → resolver → checker → emitter`. Each stage is isolated — input is the previous stage's output. When adding a language feature, walk the stages in order.

**Parser** changes almost always require **AST** changes first, then **emitter** changes to handle the new nodes.

**Checker** rules are independent of emission — keep them that way.

## Tests

```sh
cd transpiler
go test ./...                          # all tests
go test -run TestParser ./internal/parser/
go test -race -shuffle=on ./...        # as CI runs it
```

Golden files live in `testdata/`. Update them with:

```sh
go test ./... -update
```

(Check the diff before committing — golden updates are the most common source of accidental regressions.)

## Linting

```sh
golangci-lint run ./...
```

CI enforces lint clean. Fix all warnings before opening a PR.

## Pull requests

- One logical change per PR.
- PR title follows [Conventional Commits](https://www.conventionalcommits.org/): `feat:`, `fix:`, `chore:`, `docs:`, etc.
- New syntax features need parser + emitter + checker coverage, plus golden-file tests.
- Bug fixes need a regression test.

## Commit messages

Follow [Conventional Commits](https://www.conventionalcommits.org/): `feat:`, `fix:`, `chore:`, `docs:`, `test:`, etc.

## Reporting issues

Open an issue with a minimal `.union.go` reproducer and the actual vs. expected output.

For security issues, see [SECURITY.md](SECURITY.md).
