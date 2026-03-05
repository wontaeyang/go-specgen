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
