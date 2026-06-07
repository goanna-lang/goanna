# Changelog

## [1.1.0](https://github.com/nahmanmate/goanna/compare/v1.0.0...v1.1.0) (2026-06-07)


### Features

* **transpiler:** auto-generate goanna_types.go per package ([a34a259](https://github.com/nahmanmate/goanna/commit/a34a25958ad0fcb90e933d86231d22de95c28dc7)), closes [#17](https://github.com/nahmanmate/goanna/issues/17)

## 1.0.0 (2026-06-07)


### ⚠ BREAKING CHANGES

* source file extension changes from  to . Rename all source files and update editor/CI configuration accordingly.

### Features

* **ast:** define node types for union decls and switches ([41c36d0](https://github.com/nahmanmate/goanna/commit/41c36d0dec3b14541f4bad0494c63456a9e65898))
* **checker:** add CheckError struct, export InferUnionName ([eb7fe0d](https://github.com/nahmanmate/goanna/commit/eb7fe0d794a58bc5144479b83e468992a49cb162))
* **checker:** enforce exhaustive union switches at transpile time ([f96541f](https://github.com/nahmanmate/goanna/commit/f96541fd73dcc4b338b7fcb6ddf8286881c8268f))
* **cli:** add `build` subcommand with overlay-based transpilation ([b1fe619](https://github.com/nahmanmate/goanna/commit/b1fe619a782a8b1821f8de077930ed765551740f))
* **cli:** added `--check` flag to build command ([fffa522](https://github.com/nahmanmate/goanna/commit/fffa522759b8913e32a8216ad016e2c7fd3ee89b))
* **cli:** added version flag ([7ab40c1](https://github.com/nahmanmate/goanna/commit/7ab40c1db571d81f42e76a79a466b954c869b68e))
* **emitter:** add EmitWithLineMap for per-item line ranges ([4925b96](https://github.com/nahmanmate/goanna/commit/4925b9636ca0febc48e6f06704deae191a5f7f62))
* **emitter:** generate sealed interfaces and type switches ([112ac4e](https://github.com/nahmanmate/goanna/commit/112ac4e26b6b9cb9a58abcbd9d81c3cb24ad6548))
* **parser:** implement two-level chunked scanner ([a89fe9b](https://github.com/nahmanmate/goanna/commit/a89fe9b939eff9386db71b5f0bbfd244a4f0a72d))
* **pipeline:** add TranspileForLSP, promote pipeline to public ([fc3f4c4](https://github.com/nahmanmate/goanna/commit/fc3f4c4f53f63b863d75337cd14590cab7acb824))
* **resolver:** implement variant symbol table ([608ed2d](https://github.com/nahmanmate/goanna/commit/608ed2d09b03407ca86a29cdccce131b372da2d5))
* wire pipeline and add gounion CLI ([1a59074](https://github.com/nahmanmate/goanna/commit/1a59074d27a27fad0567715f65c606bd10ab7454))


### Bug Fixes

* **pipeline:** fold emitter errors into CheckErrors in TranspileForLSP ([8054863](https://github.com/nahmanmate/goanna/commit/805486336e421f6d1c3638c598bc2f966bf76190))
* resolve golangci-lint errcheck and unused findings ([a512220](https://github.com/nahmanmate/goanna/commit/a5122201b2cff0a7d53cb98b2cac138cafcbf914))
* **testdata:** sync golden files with clarifying comments ([4637f85](https://github.com/nahmanmate/goanna/commit/4637f8543faf623b15991d68b5b3181f0091a50f))


### Miscellaneous Chores

* rename project from gounion to Goanna ([628c5fa](https://github.com/nahmanmate/goanna/commit/628c5faa6c1d89fe53165d0017a449bfe1d679c2))
