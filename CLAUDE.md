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

# Test corpora
Two harnesses, one directory each. Both auto-discover, so adding a case means
adding a directory — there is no list to update.

- `examples/*/` — packages that must generate. Compared against `<name>.yaml`,
  and against `<name>.json` when that file exists (JSON is opt-in per package:
  `touch examples/x/x.json` then `-update`).
- `cmd/specgen/testdata/errors/*/` — packages that must fail. Compared against
  `expected_error.txt`. They live under `testdata/` because they are wrong on
  purpose and `go build`/`go vet` skip that directory.

One directory is one case: `@api` is package-level, so a directory holds exactly
one document.

- All expected files are OpenAPI 3.1
- Run: `go test ./cmd/specgen`
- Update: `go test ./cmd/specgen -update`
- Comparison is byte-for-byte exact — `-update` rewrites unconditionally, so
  always read the diff before committing it
