# Bash commands
- go mod tidy: cleanup/download dependency
- go test: run tests
- go fmt: format code
- go import: import dependency
- go vet: run vet on existing code to make sure it compiles

# Code style
- Follow Golang best practices and idiomatic patterns
- Keep implementation simple
- Don't over abstract
- Consider readability, extensibility, and maintainability
- Dont use LOC reduction as a metric for success during refactor/cleanup

# Making code changes
- Golang uses tabs instead of spaces for indentation
- Run Golang LSP
- After editing Go code, run `go fmt`, `go vet`

# Golden file tests
- `examples/` is the coverage surface: a new annotation or emission behavior should be demonstrated by an example, not only by a test fixture
- Each example is rendered at every version that emits differently, producing two golden files: `examples/<name>/<name>_30.yaml` and `_31.yaml`. 3.2 has no golden: the generator has no 3.2-specific emission, so its output is its 3.1 output with a different version header. Give 3.2 goldens when it grows behavior of its own (add it to `openAPIVersions` in `cmd/specgen/golden_test.go`)
- Adding an example means adding its directory name to the `examples` list in `cmd/specgen/golden_test.go`; filenames are derived
- To run golden tests: `go test ./cmd/specgen -run TestGoldenFiles`
- To update golden files after code changes: `go test ./cmd/specgen -run TestGoldenFiles -update`
- Comparison is byte-for-byte exact match — always update golden files when output changes
- Any golden change must be intentional and traceable to a named bug fix or feature; review every changed line before committing (a single behavior change moves up to two files per example)

# Test layout
- `cmd/specgen` has one spec-output harness, `TestGoldenFiles`. There is no separate fixture harness: if a behavior is worth locking, demonstrate it in an example
- Version-specific emission (nullability, exclusive bounds, `$ref` siblings) is decided by `pkg/generator/schema_builder.go` and `refSchema`. That logic is ours, and every example covers it at both versions that emit differently
- Don't write tests that assert libopenapi's own behavior (that JSON parses as JSON, that keys are ordered). Output formatting is owned by libopenapi + go.yaml.in/yaml/v4 — do not upgrade those pins or replace `doc.Render()` without expecting every golden byte to move
- `pkg/parser/testdata` and `pkg/resolver/testdata` hold negative fixtures: packages that must fail to parse or validate, which is why they cannot be examples
