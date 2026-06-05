# Security Policy

## Supported versions

Goanna is pre-1.0 and experimental. Only the latest commit on `main` receives security fixes.

| Version | Supported |
|---------|-----------|
| latest (`main`) | yes |
| older tags | no |

## Scope

Goanna is a transpiler — it reads source files and writes Go code. The primary security concerns are:

- **Malicious input files** — a crafted `.goa` that causes the transpiler to write unexpected output or access files outside the intended output path.
- **Generated code safety** — the emitter should never produce Go code that introduces vulnerabilities (e.g. unexported types leaking across packages in unintended ways).

Out of scope: issues that require the attacker to already control the machine running `goanna`.

## Reporting a vulnerability

Use [GitHub private security advisories](https://github.com/nahmanmate/goanna/security/advisories/new) to report vulnerabilities confidentially.

Include:

1. A minimal `.goa` reproducer.
2. The `goanna` version (`goanna --version` or commit hash).
3. What you expected vs. what happened.
4. Assessed impact.

Expect an initial response within 7 days. There is no bug bounty program.

## Disclosure policy

Fixes will be released before public disclosure. Once a fix is available, a GitHub security advisory will be published.
