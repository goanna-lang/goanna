# Compilation Speed Benchmarks

Machine: Intel Core i5-10300H @ 2.50GHz, linux/amd64  
Run: `go test ./pipeline/ ./parser/ ./emitter/ -bench=. -benchmem -benchtime=3s`

## Full pipeline

| Benchmark | Input | ns/op | MB/s | B/op | allocs/op |
|---|---|---:|---:|---:|---:|
| `Transpile` (with gofmt) | gender_basic | 65,439 | 6.27 | 25,972 | 414 |
| `Transpile` (with gofmt) | payload_only | 59,471 | 6.81 | 29,767 | 409 |
| `Transpile` (with gofmt) | full_example | 147,059 | 5.82 | 63,415 | 782 |
| `Transpile` (with gofmt) | crud_api | 2,552,836 | 7.33 | 1,469,918 | 12,496 |
| `TranspileForLSP` (no gofmt) | gender_basic | 11,320 | 36.22 | 11,989 | 123 |
| `TranspileForLSP` (no gofmt) | payload_only | 14,235 | 28.45 | 18,104 | 129 |
| `TranspileForLSP` (no gofmt) | full_example | 26,214 | 32.65 | 36,385 | 230 |
| `TranspileForLSP` (no gofmt) | crud_api | 650,275 | 28.80 | 979,914 | 3,057 |

## Per stage (parse / emit)

| Benchmark | Input | ns/op | MB/s | B/op | allocs/op |
|---|---|---:|---:|---:|---:|
| `Parse` | gender_basic | 7,691 | 53.31 | 8,272 | 80 |
| `Parse` | payload_only | 9,498 | 42.64 | 14,784 | 94 |
| `Parse` | full_example | 19,063 | 44.90 | 29,616 | 164 |
| `Parse` | crud_api | 458,801 | 40.81 | 810,774 | 2,163 |
| `Emit` | gender_basic | 2,573 | 159.34 | 2,096 | 34 |
| `Emit` | payload_only | 2,198 | 184.27 | 1,696 | 25 |
| `Emit` | full_example | 4,894 | 174.91 | 4,345 | 54 |
| `Emit` | crud_api | 94,265 | 198.64 | 129,594 | 829 |
| `EmitWithLineMap` | crud_api | 99,901 | 187.44 | 129,595 | 829 |

## Notes

**gofmt dominates.** `Transpile` (2,553 µs) is ~3.9× slower than `TranspileForLSP` (650 µs) on `crud_api`. `go/format` internally re-parses the generated Go source, accounting for nearly all of that gap. The LSP path correctly skips it.

**Parse dominates non-gofmt work.** 459 µs of the 650 µs `TranspileForLSP` total (~70%). Emit is cheap at 94 µs.

**`EmitWithLineMap` is essentially free** relative to `Emit` — same alloc count, ~6% overhead. The line map costs nothing meaningful.

## Test inputs

| File | Lines | Union types | Union switches |
|---|---:|---:|---:|
| `gender_basic.goa` | ~25 | 1 | 1 |
| `payload_only.goa` | ~35 | 1 | 1 |
| `full_example.goa` | ~60 | 2 | 2 |
| `crud_api.goa` | 764 | 13 | ~20 |
