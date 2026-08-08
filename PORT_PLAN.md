# Porting v1's good parts into v2

`wy/cleanup-refactor` (v1) and `wy/cleanup-refactor-v2` (v2) are independent
refactors of `main`. v2 is the branch being kept: it changed the data model
(recursive `TypeRef`), the grammar package, the error model, and the CLI. v1
changed the presentation. This plan moves the parts of v1 that are genuinely
better into v2 without disturbing what v2 got right.

Read `REFACTOR_PLAN.md` first — it is still the design contract. Two items here
(P2, and the appendix) are things that plan asked for and that did not land.

One item — P1.1 — is **not** a v1 port. It is a bug both branches inherit from
`main`, surfaced while working out how to port the rest of P1. It is here because
the port is unsafe without it.

---

## Invariants

Every commit holds these. They are the v2 wins the port must not touch.

- `resolver.TypeRef` stays recursive. No flat type descriptor reappears.
- `pkg/annotation` stays the only grammar. Nothing else hardcodes annotation names.
- Error accumulation stays **per annotation**, not per declaration — a package
  with three bad `@field`s reports three errors.
- The generator stays 3.1/3.2 only.
- Both corpora keep running: `examples/*` must generate, `cmd/specgen/testdata/errors/*`
  must fail.
- Golden impact is named per commit in advance. `-update` runs only after the
  diff has been read.

---

## Not ported

| v1 thing | Why not |
|---|---|
| The file layout wholesale | v1's `parse.go` (701) and `extract.go` (610) are the same catch-all shape as v2's `parser.go` (765) and `ast.go` (666). The layout win is localized to the tokenizer and to `errors.go` — that is P1 and P2, and nothing else. |
| `grammar.go` living inside `pkg/parser` | v2's `pkg/annotation` is strictly better: a separate package resolves the "schema means five things" collision. |
| 4 annotation kinds instead of 6 | v2's `Marker` and `SubCommand` are documented and load-bearing (`HasBlockChildren`, `Validate`). Collapsing them loses information. |
| `findInlineMarker`'s hardcoded `inlineMarkers` list | v2's `detectInlineAnnotation` looks names up in `annotation.Declaration`. v2 is better here. |
| Standalone in-function `@response` | A v1 language feature v2 deliberately lacks. A feature, not a cleanup. |
| `SchemaBuilder`, OpenAPI 3.0, per-version goldens | Deleted on purpose in v2 (C1). |
| `Render(doc, format)` + `OutputFormat` enum | v2's `RenderYAML`/`RenderJSON` is simpler. |
| v1's `ParseBracedBlock` | Carries the same grammar-blind brace counting as v2's. See P1.1 — this is fixed, not ported. |
| Source-order iteration instead of sorted maps | Open question — see Decisions. |

---

## P1 — Block extraction

### The bug

`ParseBracedBlock` counts braces before the grammar is consulted. `@pattern` is
`RawValue` precisely because a regex is a DSL with its own brace grammar, and a
regex may legitimately carry unbalanced braces (`^a}b$`, `[{]`). The scanner
reads them as structure.

This is a `main` bug that **both branches inherit** — v1's `ParseBracedBlock`
counts per line the same way. Neither v1 nor v2 fixes it. What v1 contributes is
the layering that lets the fix land in one place instead of two.

Two scanners have it:

- `ParseBracedBlock` (`annotations.go:95`) — finds the block's extent.
- `parseChildren`'s collection loop (`annotations.go:324`) — decides how many
  lines a child block spans. For a multi-line `@field` whose body holds
  `@pattern ^a}b$`, depth reaches zero early and the closing `}` line is never
  collected.

The two entry points then disagree, because `ParseInlineAnnotation` does not
brace-count at all — it takes the last `}` on the line, which is accidentally
correct:

```go
// @field { @pattern ^a}b$ }      →  pattern: ^a}b$        (correct)

// @field {
//   @pattern ^a}b$               →  no pattern, no error  (silently dropped)
// }
```

### What the corpus does and does not cover

It does **not** cover this. Every brace-bearing value in both corpora is
balanced, so the two algorithms agree and everything stays green:

```
examples/inline/inline.go:222          @pattern ^[A-Z]{2}$
examples/nested/nested.go:33           @pattern ^[A-Z]{2}$
examples/parameters/parameters.go:138  @pattern ^[a-z]{2}(-[A-Z]{2})?$
```

That is why the divergence survived. New fixtures are required whichever way this
goes.

What the corpus *does* cover heavily is the routing change in P1.2: 354
single-line `@field`s in `examples/` and 33 in the error fixtures all take the
duplicate path today, while single-line `@schema`, `@response` and `@request`
blocks already go through `ParseAnnotationBlock`. Any behavioral difference in
the routing surfaces on the first run.

### The fix

Take the correct half from each path rather than picking one.

- **Single-line block** — opener and closer on the same line, i.e. `len(lines) == 1`,
  the signal `ParseAnnotationBlock` already routes on. Content is everything
  between the block opener and the **last unescaped `}`**. No counting and no
  grammar needed: the block ends on that line by definition, and a genuinely
  nested block written on one line is rejected separately. This is what
  `ParseInlineAnnotation` does today, made escape-aware.

- **Multi-line block** — count per line as now, but skip the value of a line
  whose leading annotation resolves to a `RawValue` child.

`ParseBracedBlock` needs the `*annotation.Def` for the second half. It is called
from `ParseAnnotationBlock` (`annotations.go:252`) where `node` is already in
scope, and `parseChildren` already holds `childNode`. Passing it is a signature
change, not a layering change: the tokenizer is already grammar-driven — it
reads `node.RawValue` in `resolveValue`, plus `SupportsMultiline`, `Repeatable`,
`HasMetadata` and `CanBeEmpty()`. `ParseBracedBlock` is the one function in the
chain that was never given the def.

### Commits

**P1.1 — grammar-aware block extraction.** The fix above, at both scanner sites.
This lands **first**. It is the only commit in P1 that changes what specgen can
parse, and without it P1.2 breaks the single-line form: unified onto today's
`ParseBracedBlock`, `@field { @pattern ^a}b$ }` counts to depth −1 and errors,
so a working annotation would be broken by a commit whose stated purpose is
cleanup.
*Goldens: none in `examples/` — every corpus brace is balanced. Two new fixtures:
an unbalanced raw value that must now generate (`examples/`), and a genuinely
unbalanced block that must still fail (`testdata/errors/`).*

**P1.2 — single entry point.** `parseFieldAnnotation` calls `ParseAnnotationBlock`
unconditionally. Delete `IsInlineFormat`, `ParseInlineAnnotation` and
`validateNoNestedBraces`. Move `parseInlineChildren` next to `parseChildren`;
`inline.go` disappears. The nested-block-inline error survives in
`parseInlineChildren`'s existing guard (`Kind == SubCommand && len(Children) > 0
&& ContainsUnescapedBrace`); widen to v1's broader rule (`len(childNode.Children) > 0`)
only if a fixture shows a gap.

This is cleanup, not a fix — it removes the mechanism that let the two paths
diverge, after P1.1 has made them agree.
*Goldens: none expected. Any movement means the paths still differ on a real
input — stop and read it before updating.*

**P1.3 — one escape engine.** v2's `UnescapeValue` is `strings.ReplaceAll` with an
`"\x00BS\x00"` placeholder; every other escape-aware operation uses
`unescapedBytes`. Port v1's `Char{Pos, Val, Escaped}` + `scan` — which yields
escaped characters rather than skipping them — and rebuild `UnescapeValue`,
`CountUnescapedBraces`, `FindUnescaped`, `SplitOnUnescapedAt` and `findBlockOpener`
on it. The last-unescaped-`}` scan from P1.1 uses it too.

This is the uniform-escape rule applied to the one function that opted out.
*Goldens: none.*

**P1.4 — unbalanced blocks report.** After P1.1 the common cause of early closure
is gone, but a block that really is unbalanced should say so rather than return
an empty annotation and drop its contents.
*Goldens: new fixture `testdata/errors/unbalanced_block/`.*

### Result

Four brace scanners across three files —
`findBlockOpener`/`countBracesFromPosition` (annotations.go),
`CountUnescapedBraces` (escape.go), `validateNoNestedBraces` (inline.go, its own
hand-rolled escape loop), and raw `strings.Index`/`LastIndex` (inline.go) —
become one engine with one escape contract, which is what `escape.go` was for and
what v1's `tokenize.go` states in its header.

---

## P2 — Positioned errors

### What is wrong now

v1 carries `Line{Text, Pos}` through the tokenizer and has a 42-line `errors.go`
(`Error{Pos, Msg}`, `errorf`, `wrapf`, innermost known position wins), so every
parse failure names its line:

```
/abs/path/x.go:9:2: failed to parse @field for Widget.ID: unknown annotation @multipleOf in @field
```

v2 reports a better *path* and no position:

```
@field[Widget.ID]: unknown annotation @multipleOf in @field
```

These are complementary — the path says which declaration, the position says
which line. `REFACTOR_PLAN.md` §Errors asked for `token.Position`; it did not land.

### Design

- `specerr.Error` gains `Pos token.Position`. Render `file:line:col: path: message`
  when valid, today's form when not — so the validator, which has no positions,
  is unaffected.
- `parser.CommentBlock.Lines` becomes `[]Line`. `ParseAnnotationBlock`,
  `parseChildren`, `parseInlineChildren`, `ParseBracedBlock` and `ExtractMetadata`
  take `[]Line`/`Line`.
- Port `Annotation.Pos` and `ChildPos(name)` so a child's error points at the
  child's line, not the enclosing block's.
- **Filename normalization.** `token.Position.Filename` is absolute (verified
  against the v1 binary). Absolute paths in `expected_error.txt` would make the
  corpus machine-dependent. Normalize to cwd-relative via `filepath.Rel`, falling
  back to absolute. `go test` runs with cwd set to the package directory, so
  `testdata/errors/x/x.go:9:2` is stable in CI, and `examples/petstore/api.go:9:2`
  is clickable from the repo root.

### Commits

**P2.1** — `Line` type and `[]Line` threading. Positions carried, not yet rendered.
*Goldens: none.*

**P2.2** — `specerr.Error.Pos`, rendering, filename normalization.
*Goldens: all 11 `expected_error.txt` gain a position prefix. One reviewed diff.*

**P2.3** (optional, separate) — resolver positions from `pkg.Fset.Position(field.Pos())`
on `*types.Var`. *Goldens: resolver fixtures only.* See Decision 3.

---

## P3 — Two validation rules v2 lost

v1's validator has three uniqueness checks. v2 has one (duplicate field name,
`validator.go:130` and `:153`). The other two are missing, and both are silent
data loss:

```go
// two handlers, same route, same operationID
paths:
  /users:
    get:
      summary: Second handler...     ← the first handler is gone, no error
```

- **`validateUniqueRoutes`** — a second `GET /users` overwrites the first in the
  paths map.
- **`validateUniqueOperationIDs`** — OpenAPI requires `operationId` to be unique
  document-wide; client generators name methods from it. Endpoints without an
  `@operationID` are unnamed, not named `""`, and are not compared to each other.

**Commit P3.1** — port both, reporting at `@endpoint[<funcName>]` to match v2's
existing path vocabulary. New fixtures `testdata/errors/duplicate_route/` and
`testdata/errors/duplicate_operationid/`.
*Goldens: verified that no example package has a duplicate route or operationID,
so `examples/` is untouched.*

---

## P4 — Bare relative package path

v1's `load()` prefixes `./` when the path is relative and does not start with a
dot, because `packages.Load` reads a bare relative path as an import path:

```
-package examples/petstore    → package examples/petstore is not in std (...)
-package ./examples/petstore  → works
```

`cmd/specgen/errors_test.go:55` already carries a comment explaining the
workaround it performs for this. Port the normalization into `ExtractComments`
so the library is correct and the harness's `pkgDir` becomes incidental rather
than load-bearing.

**Commit P4.1.** *Goldens: none.*

---

## Order

| # | Commit | Golden impact | |
|---|---|---|---|
| 1 | P1.1 grammar-aware extraction | none moved; `examples/rawvalue/` added | **done** |
| 2 | P4.1 relative path | none | **done** |
| 3 | P1.2 single entry point | `parse_multiple` — see below | **done** |
| 4 | P1.3 one escape engine | none | **done** |
| 5 | P3.1 two validators | 2 new error fixtures | **done** |
| 6 | P1.4 unbalanced blocks report | 1 new error fixture | **done** |
| — | P2 positioned errors | — | **dropped, see below** |

P1.1 first: it is the only bug here, and no commit in history should break a
working annotation form. P2.2 last because it rewrites error goldens and should
sit on top of a settled message set.

## What landed

**P1.1 — grammar-aware block extraction.** `ParseBracedBlock` takes the
`*annotation.Def` and applies two rules: a block that closes on its opening line
ends at the **last unescaped `}`** of that line, and a multi-line block counts
per line but skips a line whose leading annotation is a `RawValue` child.
`parseChildren`'s collection loop uses the same rules. `countBracesFromPosition`
became `openerDepth`, which counts only up to the first child annotation on the
opening line. `examples/rawvalue/` pins seven regexes written **twice** each,
inline and as a block; the pairs must agree, which is the property that broke.

**P1.2 — single entry point.** `IsInlineFormat`, `ParseInlineAnnotation` and
`validateNoNestedBraces` are gone; `parseInlineChildren` sits next to
`parseChildren`; `inline.go` is deleted. `annotation.Def.HasBlockChildren`
became dead with them and was removed too. Its tests were retargeted onto
`ParseAnnotationBlock` rather than dropped, and the ones that were really about
the AST moved to `ast_test.go`.

*This moved a golden, which the plan said to stop and read.* Unifying exposed
that the multi-line path wrapped child errors in `failed to parse X children:`
and the one-line path did not. The wrapper was redundant three ways over —
`@schema[AlphaGadget]: failed to parse @schema children: unknown annotation
@bogusStructRule in @schema` — and it contradicted the rule already written in
`parseFieldAnnotation`: the path carries the location, so the message is free to
be only what went wrong. It was dropped, so both forms now agree on the shorter
message. The `failed to parse X:` wrapper around a *block* error stays, because
that leaf message does not name its annotation.

**P1.3 — one escape engine.** `unescapedBytes` became `scan`, yielding a `char`
with `Pos`/`Val`/`Escaped` so escaped characters are visible rather than
skipped. `UnescapeValue` is built on it and no longer needs the `"\x00BS\x00"`
placeholder that kept `\\` from being unescaped twice. `findBlockOpener` uses it
too. Four brace scanners across three files are now one engine in one file.

**P1.4 — a block closed by a line carrying other text reports.** The remaining
silent drop: `@description has a } brace` inside a block ended it, discarding
that line and everything under it. Fixture is `testdata/errors/brace_in_value/`
rather than the planned `unbalanced_block/`, because that is what the case is.

**P3.1 — two validation rules.** `validateUniqueRoutes` and
`validateUniqueOperationIDs`, reported at `@endpoint[METHOD PATH]` to match the
existing path vocabulary, with the colliding function named in the message since
the path cannot tell two handlers on one route apart. `endpointPath` is now a
helper `validateEndpoint` shares. No example package had a collision, so
`examples/` was untouched.

**P4.1 — bare relative package path.** Normalized in `ExtractComments`. All
three forms now work: `examples/petstore`, `./examples/petstore`, and absolute.

**P2 — positioned errors: built, then dropped on purpose.**

It worked. `CommentBlock.Lines` became `[]Line{Text, Pos}` threaded through the
tokenizer, `specerr.Error` gained a `Pos`, and filenames rendered relative to the
working directory so fixtures stayed machine-independent:

```
parse: testdata/errors/parse_multiple/parse_multiple.go:55:1: @schema[AlphaGadget]: unknown annotation @bogusStructRule in @schema
```

It was reverted because it could only be done for one stage. Location data is not
evenly available:

| Stage | Error sites | What it holds |
|---|---|---|
| parser | 27 | comment lines — a position is right there |
| resolver | 5 | Go types; `types.Var.Pos()` reaches the field |
| validator | 50 | resolver IR only, with no link back to source |

Levelling *up* means carrying a position on five IR types through to the
validator and touching all 50 of its error sites — roughly 150 lines and a
rewrite of every error fixture — so that messages gain a prefix. Levelling
*down* costs nothing and keeps all three stages reporting the same shape.
v1's parser could afford positions partly because v1 shipped no must-fail
corpus at all, so it never had to make the other two stages agree with it.

The reasoning is recorded in `pkg/specerr`'s package doc, which is where someone
about to re-add a `Pos` field will be looking. **Do not re-propose this without
also proposing the validator half** — one stage printing positions and two not
is worse than none doing it.

### P1.1 as landed

`ParseBracedBlock` takes the `*annotation.Def` and applies two rules: a block
that closes on its opening line ends at the **last unescaped `}`** of that line
(no counting — only the final brace can be the block's), and a multi-line block
counts per line but skips a line whose leading annotation is a `RawValue` child.
`parseChildren`'s collection loop uses the same two rules to decide how far a
child block reaches. `countBracesFromPosition` is replaced by `openerDepth`,
which counts only up to the first child annotation on the opening line, because
everything after it is text. `LastUnescaped` joins the escape engine in
`escape.go`.

`examples/rawvalue/` pins it: seven regexes covering a quantifier, an unpaired
`}`, a literal `[{]`, and a space-before-brace, each written **twice** — once
inline and once as a block. The two forms produce identical output, which is the
property that was broken. No existing golden moved.

Still open, and still P1.4: a **non-raw** value with an unescaped brace is
dropped the same silent way. `@description has a } brace` inside a block yields
a field with no description and no error. `resolveValue` would have rejected it,
but block extraction consumes the line before the value is ever read.

---

## Decisions

**1. Which values may carry unbalanced braces.** `@pattern` is the only
`RawValue` in the grammar, so it is the only value that may carry unbalanced
braces unescaped. A non-raw value now reports instead of being dropped
(`testdata/errors/brace_in_value/`), and a JSON literal still has to be written
`@example \{"a":1\}`. Should `@example` and `@default` become `RawValue` so JSON
works unescaped? Grammar decision, not a tokenizer one — left as it was.

**2. Error ordering — declined; v2's alphabetical sort stands.** v1 iterates
declarations in source order, v2 sorts map keys by name. Both are deterministic,
which is the property that actually matters, and it is already met. Source order
would read more like the file does, but buying it means an ordering key on the
IR that survives parser → resolver → validator — the same kind of structure the
position work was rejected for, for a smaller return. `error_ordering` keeps
pinning alphabetical order on purpose, not by accident.

**3. P2.3 scope — moot.** P2 was dropped entirely; see above.

---

## Appendix — v2 debt this plan does not touch

Not from v1. Listed so it is not lost.

- `schemas map[string]*resolver.Schema` is threaded through 10 generator
  functions and read at exactly one line (`generator.go:176`).
  `generateSchema(schema, allSchemas)` never reads `allSchemas`;
  `generateWrappedSchema` uses `body.Bind.WrapperSchema` instead.
  `REFACTOR_PLAN.md` said "generator loses its `schemas` map".
- `resolver.Package.Schemas` and `.Parameters` are still maps; the plan called
  for ordered slices.
- `Resolver` holds `schemaNames` and `path` on the receiver with a comment
  explaining why, but threads `schemas`, `parameters` and `defaultContentType`
  through six endpoint functions for the same reason.
- `pkg/resolver/inline.go`'s `ParseInlineDeclaration` parses annotations inside
  the resolver; `REFACTOR_PLAN.md` gives everything comment-shaped to `pkg/parser`.
- `generator.NewGenerator` does not validate its version string; only `main.go` does.
