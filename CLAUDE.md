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
- Each example is rendered at every supported version, producing three golden files: `examples/<name>/<name>_30.yaml`, `_31.yaml`, `_32.yaml`
- Adding an example means adding its directory name to the `examples` list in `cmd/specgen/golden_test.go`; filenames are derived
- To run golden tests: `go test ./cmd/specgen -run TestGoldenFiles`
- To update golden files after code changes: `go test ./cmd/specgen -run TestGoldenFiles -update`
- Comparison is byte-for-byte exact match — always update golden files when output changes
- Any golden change must be intentional and traceable to a named bug fix or feature; review every changed line before committing (a single behavior change now moves up to three files per example)

# Feature fixture tests
- `cmd/specgen/testdata/*/` covers only what examples cannot reach: JSON rendering, and edge cases that would read as noise in documentation (empty components, block/inline response merge order, the primitive type matrix, inline-body builder behavior)
- Version-specific output belongs in examples, since every example renders at 3.0, 3.1, and 3.2
- To run: `go test ./cmd/specgen -run TestFeatureFixtures`; update with `-update` (same review rule as goldens)
- Version-specific emission (nullability, exclusive bounds, `$ref` siblings) is decided by `pkg/generator/schema_builder.go` and `refSchema` — that logic is ours; libopenapi only serializes it
- Output formatting is owned by libopenapi + go.yaml.in/yaml/v4 — do not upgrade those pins or replace `doc.Render()` without expecting every golden byte to move
