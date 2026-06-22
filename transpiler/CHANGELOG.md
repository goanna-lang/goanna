# Changelog

## [0.2.0](https://github.com/goanna-lang/goanna/compare/v0.1.0...v0.2.0) (2026-06-22)


### ⚠ BREAKING CHANGES

* source file extension changes from  to . Rename all source files and update editor/CI configuration accordingly.

### Features

* **ast:** define node types for union decls and switches ([41c36d0](https://github.com/goanna-lang/goanna/commit/41c36d0dec3b14541f4bad0494c63456a9e65898))
* **checker:** add CheckError struct, export InferUnionName ([eb7fe0d](https://github.com/goanna-lang/goanna/commit/eb7fe0d794a58bc5144479b83e468992a49cb162))
* **checker:** enforce exhaustive union switches at transpile time ([f96541f](https://github.com/goanna-lang/goanna/commit/f96541fd73dcc4b338b7fcb6ddf8286881c8268f))
* **cli:** add `build` subcommand with overlay-based transpilation ([b1fe619](https://github.com/goanna-lang/goanna/commit/b1fe619a782a8b1821f8de077930ed765551740f))
* **cli:** added `--check` flag to build command ([fffa522](https://github.com/goanna-lang/goanna/commit/fffa522759b8913e32a8216ad016e2c7fd3ee89b))
* **cli:** added version flag ([7ab40c1](https://github.com/goanna-lang/goanna/commit/7ab40c1db571d81f42e76a79a466b954c869b68e))
* default behaviour is transpiled code is not formatted ([409b5cd](https://github.com/goanna-lang/goanna/commit/409b5cdf893cbbcfb98d38a65c5ebb23795256f9))
* **emitter:** add EmitWithLineMap for per-item line ranges ([4925b96](https://github.com/goanna-lang/goanna/commit/4925b9636ca0febc48e6f06704deae191a5f7f62))
* **emitter:** generate sealed interfaces and type switches ([112ac4e](https://github.com/goanna-lang/goanna/commit/112ac4e26b6b9cb9a58abcbd9d81c3cb24ad6548))
* improved formatter options ([0291059](https://github.com/goanna-lang/goanna/commit/0291059558b0fe50affc16067ab700da400177d8))
* **parser:** implement two-level chunked scanner ([a89fe9b](https://github.com/goanna-lang/goanna/commit/a89fe9b939eff9386db71b5f0bbfd244a4f0a72d))
* **pipeline:** add TranspileForLSP, promote pipeline to public ([fc3f4c4](https://github.com/goanna-lang/goanna/commit/fc3f4c4f53f63b863d75337cd14590cab7acb824))
* **resolver:** implement variant symbol table ([608ed2d](https://github.com/goanna-lang/goanna/commit/608ed2d09b03407ca86a29cdccce131b372da2d5))
* **transpiler:** auto-generate goanna_types.go per package ([a34a259](https://github.com/goanna-lang/goanna/commit/a34a25958ad0fcb90e933d86231d22de95c28dc7)), closes [#17](https://github.com/goanna-lang/goanna/issues/17)
* wire pipeline and add gounion CLI ([1a59074](https://github.com/goanna-lang/goanna/commit/1a59074d27a27fad0567715f65c606bd10ab7454))


### Bug Fixes

* **formatter:** skip install prompt when stdin is a pipe ([f8f5f12](https://github.com/goanna-lang/goanna/commit/f8f5f12dabf1cc91e8b3ed21e188d7e318b44ccf))
* **pipeline:** fold emitter errors into CheckErrors in TranspileForLSP ([8054863](https://github.com/goanna-lang/goanna/commit/805486336e421f6d1c3638c598bc2f966bf76190))
* resolve golangci-lint errcheck and unused findings ([a512220](https://github.com/goanna-lang/goanna/commit/a5122201b2cff0a7d53cb98b2cac138cafcbf914))
* **testdata:** sync golden files with clarifying comments ([4637f85](https://github.com/goanna-lang/goanna/commit/4637f8543faf623b15991d68b5b3181f0091a50f))
* unchecked errors now handled ([dc072b9](https://github.com/goanna-lang/goanna/commit/dc072b9176cc69d27b8644d62c425cf55a135e17))


### Performance Improvements

* **build:** parallelize multi-file transpilation ([83b2ff8](https://github.com/goanna-lang/goanna/commit/83b2ff812e10c1296b003645d413d2ff5d2995e5))


### Miscellaneous Chores

* rename project from gounion to Goanna ([628c5fa](https://github.com/goanna-lang/goanna/commit/628c5faa6c1d89fe53165d0017a449bfe1d679c2))

## [0.1.0](https://github.com/goanna-lang/goanna/commits/transpiler/v0.1.0) (2026-06-22)

Initial release under goanna-lang org.
