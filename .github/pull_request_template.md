## Summary

<!-- What does this PR do and why? One paragraph max. -->

## Type of change

<!-- Check all that apply. PR title must use the matching Conventional Commits prefix. -->

- [ ] `feat:` — new language feature or CLI capability
- [ ] `fix:` — bug fix
- [ ] `docs:` — documentation only
- [ ] `test:` — tests only, no production code change
- [ ] `chore:` — build, deps, CI, tooling
- [ ] `refactor:` — internal restructure, no behaviour change

## Related issue

<!-- Closes #, Relates to #, or N/A -->

## Checklist

**General**

- [ ] PR title follows [Conventional Commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, etc.)
- [ ] One logical change per PR

**CI**

- [ ] `go test -race -shuffle=on ./...` passes
- [ ] `golangci-lint run ./...` passes
- [ ] `go mod tidy && git diff --exit-code` is clean

**Feature work** *(skip if not adding syntax)*

- [ ] Parser, AST, emitter, and checker all updated in order
- [ ] Golden files updated (`go test ./... -update`) and diff reviewed

**Bug fixes** *(skip if not a fix)*

- [ ] Regression test added

**Breaking change**

- [ ] This PR introduces a breaking change

<!-- If checked above: what breaks and how should callers migrate? -->
