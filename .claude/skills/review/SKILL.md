---
name: review
description: Review branch changes against main for correctness and spec compliance
disable-model-invocation: true
allowed-tools: Bash, Read, Grep, Glob
---

Review the current branch against main. Run the following steps:

## 1. Understand the changes
- Run `git diff main...HEAD --stat` to see changed files
- Run `git log main..HEAD --oneline` to see commits
- Run `git diff main...HEAD` to read the full diff

## 2. Code review
- Check Go style: tabs for indentation, idiomatic patterns, simplicity
- Check error handling
- Check for over-abstraction or unnecessary complexity
- Verify test coverage for new/changed behavior

## 3. Generated output review
This is critical — do NOT skip this step.

For any changed YAML/JSON output files (especially in `examples/`):
- Check the declared OpenAPI version (`openapi: 3.x.x`) at the top of each file
- Verify every changed output is **correct for that version**, not just "looks reasonable"
- Specifically check version-sensitive constructs:
  - **3.0**: `nullable: true`, `allOf` wrapping for `$ref` with nullable, `exclusiveMinimum` as boolean
  - **3.1+**: `type: ["string", "null"]` or `oneOf` with null type, no `nullable` keyword, `exclusiveMinimum` as numeric value
- Flag any output that uses a 3.0 pattern in a 3.1 spec or vice versa

## 4. Golden file check
- Run `go test ./cmd/specgen -run TestGoldenFiles` to verify golden files are up to date
- Run `go test ./...` to verify all tests pass

## 5. Summary
Provide a summary with:
- What the changes do
- Any issues found
- Any suggestions for improvement
