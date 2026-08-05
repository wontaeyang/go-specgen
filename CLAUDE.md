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
- Golden files live in `examples/*/` as `.yaml` files
- All golden files are generated with OpenAPI 3.1 and YAML format
- To run golden tests: `go test ./cmd/specgen -run TestGoldenFiles`
- To update golden files after code changes: `go test ./cmd/specgen -run TestGoldenFiles -update`
- Comparison is byte-for-byte exact match — always update golden files when output changes
- Any golden change must be intentional and traceable to a named bug fix or feature; review every changed line before committing

# Feature fixture tests
- Mini-golden fixtures live in `cmd/specgen/testdata/*/` and lock features the examples don't cover (readOnly/writeOnly, exclusive bounds, OpenAPI 3.0 output, JSON output, $ref variants, mixed block+inline responses, primitive type matrix)
- To run: `go test ./cmd/specgen -run TestFeatureFixtures`; update with `-update` (same review rule as goldens)
- Output formatting is owned by libopenapi + go.yaml.in/yaml/v4 — do not upgrade those pins or replace `doc.Render()` without expecting every golden byte to move
