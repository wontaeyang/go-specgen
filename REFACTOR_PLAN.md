# go-specgen refactor plan

Architecture-level cleanup of accumulated tech debt. The four-stage pipeline
(parse → resolve → validate → generate) stays; what changes is what each stage
owns and the shape of the IR between them.

## Constraints

- The 14 example goldens stay byte-identical **except `examples/inline/inline.yaml`**,
  which gains ~30 additive lines when nested `@field` annotations start applying.
- Examples and goldens remain OpenAPI 3.1.
- `libopenapi v0.38.3` and `go.yaml.in/yaml/v4` pins are never touched. They own all
  output formatting; key order is their struct field declaration order.
- Every behavior change is verified by a fixture diff, never by editing an example.

---

## 1. Bugs being fixed

All confirmed empirically. None except #4 touches a golden.

| # | Bug | Today | After |
|---|-----|-------|-------|
| 1 | Nested containers collapse to one level | see below | full shape preserved |
| 2 | Struct-typed parameter fields | silent `type: string`, `@schema` or not | error |
| 3 | `else { "string" }` catch-all in `resolveType` | `chan`/`func`/`complex` silently become `type: string` | error |
| 4 | Nested `@field` in anonymous structs | silently dropped (16 in `examples/inline`) | applied |
| 5 | Grouped `X, Y` declarations | only `X` annotated | all names annotated |
| 6 | Parameter conflict rule | keys on name alone — rejects legal cross-location duplicates | keys on `(name, in)` per the OpenAPI spec |
| 7 | `query:"-"` | emits a phantom parameter named by the Go field | skipped |
| 8 | `,required` detection | `strings.Contains(tag, ",required")` on the whole raw tag | reads the kind-specific tag via `reflect.StructTag` |
| 9 | Embedded fields | see below | mirrors `encoding/json` |
| 10 | Undefined endpoint tag | check is guarded by `len(api.Tags) > 0`, so an API with no tags never runs it | always checked |

### Bug 1 in detail

`cmd/specgen/testdata/features/collections` shows this is wider than "maps of
arrays". Every container nested inside another loses its inner level, because the
element type is recovered by string surgery on the Go type name rather than
carried as structure:

| Go type | Today | After |
|---|---|---|
| `map[string][]Item` | `additionalProperties: $ref Item` | `additionalProperties: {type: array, items: $ref Item}` |
| `map[string][]string` | `additionalProperties: {type: string}` | `additionalProperties: {type: array, items: {type: string}}` |
| `[]map[string]string` | `items: {type: string}` | `items: {type: object, additionalProperties: {type: string}}` |
| `map[string]map[string]string` | `additionalProperties: {type: string}` | nested `additionalProperties` |
| `[][]int` | `items: {type: array}` — no inner `items` | `items: {type: array, items: {type: integer}}` |

### Parameter tag rules

Parameters mirror `encoding/json` semantics, giving one naming rule across the
whole codebase:

| Tag state | Behavior |
|---|---|
| `query:"limit"` | parameter named `limit` |
| `query:"-"` | skipped |
| no parameter-kind tag | parameter named by the Go field name |
| `path:"x"` inside a `@query` struct, no `query:` tag | **error** — tagged for a different kind |

Parameters are scalar or array-of-scalar only. Struct-typed fields error, because
`net/http` exposes parameters as `map[string][]string` — there is no standard-library
model for structured parameters. `deepObject` is explicitly out of scope.

### Embedded field family

Four cases, three currently wrong. All from `resolveSchemaFields` dispatching on
`field.Anonymous()` with no tag inspection.

| Case | `encoding/json` | Today |
|---|---|---|
| embedded struct, untagged | flatten | flatten ✓ |
| embedded struct, `json:"meta"` | nest under `meta` | flattens ✗ |
| embedded non-struct, untagged | field keyed by **type name** | dropped ✗ |
| embedded non-struct, `json:"alias"` | field keyed by `alias` | dropped ✗ |
| embedded anything, `json:"-"` | skipped | flattened ✗ |

Unified rule, parameterized by which tag is read (`json` for schemas, the kind tag
for parameters):

```
if anonymous:
    if tag == "-"            → skip
    if tag names a field     → regular field, named by the tag
    if underlying is struct  → flatten
    else                     → regular field, named by the TYPE name
else:
    regular field
```

---

## 2. Target architecture

### Packages

| Package | Owns |
|---|---|
| `pkg/annotation` | The grammar. Data plus small query methods, no parsing logic. |
| `pkg/parser` | Everything comment-shaped: escape/tokenize, block parsing, AST harvest, IR construction. |
| `pkg/resolver` | Go type resolution. Consumes `*parser.Package`, emits an emission-ready IR. |
| `pkg/validator` | Business rules over resolver IR. Accumulating, deterministic order. |
| `pkg/generator` | libopenapi high-level structs and rendering. Never parses Go type strings. |
| `cmd/specgen` | CLI, the single `buildSpec()` pipeline, all golden and fixture tests. |

`pkg/schema` is renamed to `pkg/annotation`, resolving a collision where "schema"
meant five different things.

```go
annotation.Schema       // doc-comment grammar root (was schema.AnnotationSchema)
annotation.Declaration  // in-function grammar root (was resolver.InlineAnnotationSchema)
annotation.Def          // per-tag capability record (was SchemaNode)
annotation.Block, annotation.Value, annotation.Flag,
annotation.Marker, annotation.Reference, annotation.SubCommand
```

Two roots because the grammars genuinely differ — in-function `@response` has no
`@body`, since the struct *is* the body. `detectInlineAnnotation`'s hardcoded
annotation names are replaced by lookups against the tree, so adding an annotation
means editing one file.

Dropped as dead: `IsSibling`, `GetSiblings`, `IsTopLevel`, the never-set
`Validator func(string) error` field.

### Data flow

`parser.Parse(dir) → *parser.Package` carries everything, including the loaded
`*packages.Package` and parsed in-function declarations. The `Comments()` side
channel and the resolver's duplicate `packages.Load` both disappear; the package
is loaded exactly once.

`parser.Field` gains `Fields []*Field` so annotations mirror the shape of the types
they describe, which is what lets one recursive resolver replace the current four
field-resolution paths.

### Type shape

Ten mutually-constrained fields on `ResolvedField` — `OpenAPIType`, `IsArray`,
`ItemsType`, `ItemsInlineFields`, `IsMap`, `MapValueInlineFields`, `IsAnyValue`,
`IsUnresolvedStruct`, `UnresolvedTypeName`, `InlineFields` — collapse into one
recursive descriptor:

```go
type TypeRef struct {
    Shape  Shape     // Scalar, Array, Map, Object, Ref, Any, TypeParam, Unsupported
    Format string
    Ref    string    // Shape == Ref
    Elem   *TypeRef  // Array element / Map value
    Fields []*Field  // Shape == Object (anonymous struct)
    Reason string    // Shape == Unsupported — why, for the error message
}
```

`map[string][]User` becomes `Map → Array → Ref("User")`. Four if/else-if ladders
become one switch, bug #1 stops being representable, and `Unsupported` is what makes
"error on unknown" possible at all — today "unknown" and "schema reference" are the
same value, which is why the catch-all is load-bearing.

`TypeParam` exists because generic templates must still resolve (`validateSchema`
errors on a schema with zero fields) even though the generator never emits them.

### Endpoint IR

```go
Parameters []Parameter   // flat, ordered, each carries its In
Responses  []Response    // merged, sorted, conflicts resolved
```

Ten fields become two. Ordering and conflict resolution become pure functions in
the resolver, unit-testable without rendering YAML — today the entire conflict rule
is one `continue` inside the emitter, and nothing tests it.

`Parameter` is endpoint-level and emission-ready; `ParameterStruct` is the named
`@query Filters` declaration, still needed for ref resolution and `@query[Filters]`
error paths.

The `Resolved` prefix is dropped throughout: `resolver.Schema`, `resolver.Field`,
`resolver.Endpoint`, `resolver.Package`.

`Package.Schemas` and `Package.Parameters` become ordered slices rather than maps,
removing the determinism hazard structurally instead of sorting at each use.

### Ordering contracts

Both are documented in code and locked by fixtures. Neither is verified today.

**Parameters** — grouped by location (`path → query → header → cookie`), named before
inline within each kind, declaration order within each group. Today's two-pass
emission can split same-location parameters apart.

**Responses** — merged and sorted by status string, named winning conflicts. The
existing comparator is already proven by `examples/responses` (`200, 4XX, 5XX, default`).

### Errors

Parser and resolver accumulate per top-level item with `token.Position`, joined via
`errors.Join`. Validator keeps `ValidationError{Path, Message}` and `MultiError` with
`Unwrap`. Every map iteration feeding output or errors is eliminated or sorted.

---

## 3. Versions and CLI

OpenAPI 3.0 is dropped. 3.1 is the baseline; 3.2 is emitted too, differing only in
the version string (verified — `Is31Plus()` collapses them, so no 3.2-specific
behavior exists).

This deletes `SchemaBuilder` entirely, over half of `generateRefSchema` (both 3.0
branches plus the `reflect.DeepEqual` dance), and the `reflect` import from the
generator.

```
specgen -package ./api -yaml openapi.yaml -json openapi.json
specgen -package ./api -yaml openapi.yaml
specgen -package ./api                        # error: specify -yaml and/or -json
```

| Flag | |
|---|---|
| `-package` | package to parse (default `.`) |
| `-yaml` | write YAML here — path used literally |
| `-json` | write JSON here — path used literally |
| `-openapi` | `3.1` or `3.2` (default `3.1`) |

`-output` and `-format` are removed, subsumed by the two path flags. Passing neither
destination is an error: output lands only where it was named. Both files render
from one pipeline pass.

---

## 4. Commit sequence

**Phase 0 — safety net**

- **C0** Extract `buildSpec()` shared by `main.go`, the golden test, and fixtures.
  Golden test auto-discovers `examples/*/`. Add `cmd/specgen/testdata/` and snapshot
  **current** behavior, bugs included. Everything after this produces a reviewable
  fixture diff or none at all.

**Phase 1 — structure, no behavior change**

- **C1** Drop OpenAPI 3.0; delete `SchemaBuilder`; new CLI flag surface. Done first
  so later phases restructure less code.
- **C2** `pkg/schema` → `pkg/annotation`; consolidate both grammars; drop dead surface.
- **C3** Parser owns all annotation parsing. `parser.Package` becomes the sole output;
  `Comments()` and the second `packages.Load` deleted. Parameter-struct field parsing
  joins the shared path it currently bypasses.
- **C4** Generator local dedup: delete `generateInlineParameters`, extract the duplicated
  header block, parameterize the wrapped-schema and request-body pairs, consolidate
  constraint emission, remove the ref-aware/ref-unaware fork.

**Phase 2 — IR reshape**

- **C5** `TypeRef` replaces the boolean spread; one recursive field resolver replaces
  all four paths; generator consumes structure instead of parsing type strings.
  Fixes bug #1. Generator loses its `schemas` map, `isSchemaReference`,
  `extractTypeName`, `isPrimitive`, `goTypeToPrimitive`.
- **C6** Merged, ordered `Parameters` and `Responses`; validator drops its inline
  special-case. Both ordering contracts land here.

**Phase 3 — behavior fixes**

- **C7** `(name, in)` conflict rule, covering inline parameters explicitly (bug #6),
  plus the ungated undefined-tag check (bug #10). Both are validator rules and both
  move a fixture between `features/` and `errors/`.
- **C8** Parameter tag rules and `,required` reading (bugs #2, #7, #8).
- **C9** Unsupported-type errors (bug #3).
- **C10** Recursive `@field` harvest applied (bug #4) — **the `inline.yaml` diff**.
- **C11** Grouped `X, Y` field annotations (bug #5).
- **C12** Embedded field family mirrors `encoding/json` (bug #9).

**Phase 4**

- **C13** README: 3.0 removal, new CLI, parameter tag rules, embedded-field cases,
  new error messages. Plus the divergence catalog.

---

## 5. Test strategy

`cmd/specgen/testdata/` in four groups, all snapshotted from current code first:

- **features** — the 7 annotations with zero example coverage (`@exclusiveMinimum`,
  `@exclusiveMaximum`, `@uniqueItems`, `@readOnly`, `@writeOnly`, `@minItems`,
  `@maxItems`), mixed named+inline parameters and responses, `map[string][]T`,
  struct-typed parameter fields, grouped `X, Y` fields, embedded structs in parameter
  structs, the embedded-field family, `chan`/`func` fields, the parameter tag matrix
- **errors** — packages that must fail, with expected message text. Zero coverage today
- **json** — `nested`, `responses`, `petstore`. Both formats serialize the same
  `*v3.Document`, so YAML goldens already prove specgen's correctness; these catch
  serializer breakage or a libopenapi upgrade changing output
- **guard** — `TestVersionGuard` asserts 3.2 output differs from 3.1 *solely* in the
  version line, across every example, so a future 3.2-specific behavior fails loudly
  instead of drifting in

A fixture whose behavior becomes an error moves from `features/` to `errors/` in the
commit that changes it — `param_struct_field` and `unsupported_types` at C8 and C9,
`undefined_tag` at C7 — and `param_conflict` moves the other way at C7. The move is
the diff: one directory rename plus one expected file swapping a rendered spec for a
message.

Unit tests move with the code they cover. The fixtures are the behavioral contract.

Coverage gaps this closes, all currently golden-uncovered: mixed named+inline (0 of 14
examples), embedded fields (0, despite being documented in the README), the 7 orphaned
annotations, all error messages, JSON rendering.

---

## 6. Out of scope

- `deepObject` or any structured-parameter support — not now, and not designed for
- snake_case or any derived field naming; no name transformation exists today and none
  is added
- Binder changes — separate project, see appendix

---

## Appendix: binder to-dos (separate project)

For the generated spec to be true, `bindStructWithParams` needs to match:

1. `if tag == "" || tag == "-" { continue }` → `-` skips, `""` falls back to `sf.Name`
2. Recurse into `sf.Anonymous` struct fields so embedded parameters bind
3. Add Float and Uint cases to `setFromString` — specgen emits `type: number` for
   `float64` today and the binder errors at runtime
4. Add `BindCookie` using `r.Cookie(name)` — specgen supports `@cookie` and
   `examples/parameters` uses it
5. `BindPath` only enforces `,required`, so a missing path parameter silently
   zero-values. OpenAPI requires path parameters to be required
