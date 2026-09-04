# go-specgen

Generate OpenAPI 3.x specifications from Go code annotations.

> Built with [Claude Code](https://claude.ai/code) | Powered by [libopenapi](https://github.com/pb33f/libopenapi)

---

## Overview

**go-specgen** generates OpenAPI 3.x specifications from Go code annotations. It is:

- **Framework agnostic** - Works with net/http, Chi, Gin, Echo, or any Go HTTP framework
- **Spec-only** - Generates OpenAPI specs; does not handle runtime request/response binding
- **Type-safe** - Uses Go's type system to ensure spec accuracy
- **Single source of truth** - API definitions live in your code, not separate YAML files

### How It Works

1. Annotate your Go handlers and types with `@` comments
2. Run `specgen` to parse annotations and generate OpenAPI YAML/JSON
3. Use the generated spec with Swagger UI, client generators, or API gateways

### Example

**Using package-level structs:**

```go
// @api {
//   @title User API
//   @version 1.0.0
// }
package api

// @path
type UserPath struct {
    ID string `path:"id"`
}

// @schema
type UpdateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

// @schema
type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// @endpoint PUT /users/{id} {
//   @summary Update a user
//   @path UserPath
//   @request { @body UpdateUserRequest }
//   @response 200 { @body User }
// }
func UpdateUser(w http.ResponseWriter, r *http.Request) {}
```

**Or using inline structs:**

```go
// @api {
//   @title User API
//   @version 1.0.0
// }
package api

// @endpoint PUT /users/{id} {
//   @summary Update a user
// }
func UpdateUser(w http.ResponseWriter, r *http.Request) {
    // @path
    var path struct {
        ID string `path:"id"`
    }

    // using type declaration
    // @request
    type request struct {
        Name  string `json:"name"`
        Email string `json:"email"`
    }

    // using var declaration
    // @response 200
    var resp struct {
        ID    string `json:"id"`
        Name  string `json:"name"`
        Email string `json:"email"`
    }
}
```

---

## Installation

### Using go tool (Go 1.24+, recommended)

Add to your project's `go.mod`:

```
tool github.com/wontaeyang/go-specgen/cmd/specgen@latest
```

Or pin to a specific version:

```
tool github.com/wontaeyang/go-specgen/cmd/specgen@v1.0.0
```

Then run:

```bash
go tool specgen -package ./handlers -yaml openapi.yaml
```

### Using go install

```bash
go install github.com/wontaeyang/go-specgen/cmd/specgen@latest
specgen -package ./handlers -yaml openapi.yaml
```

### Build from source

```bash
git clone https://github.com/wontaeyang/go-specgen
cd go-specgen
go build -o bin/specgen ./cmd/specgen
```

---

## CLI Usage

```bash
specgen [options]

Options:
  -package string    Path to Go package (default ".")
  -yaml string       Write the spec as YAML to this path
  -json string       Write the spec as JSON to this path
  -openapi string    OpenAPI version: 3.1 or 3.2 (default "3.1")
  -version           Show version
  -help              Show help
```

At least one of `-yaml` or `-json` is required. The spec is written only to the
paths you name — there is no default output file. Passing both renders both from
a single pass, so the two files can never disagree.

**Examples:**

```bash
# Both formats from one run
specgen -package ./api -yaml openapi.yaml -json openapi.json

# YAML only, into a directory that may not exist yet
specgen -package ./api -yaml docs/openapi.yaml

# OpenAPI 3.2 (identical output apart from the version string)
specgen -package ./api -openapi 3.2 -yaml openapi.yaml
```

---

## Core Concepts

### Parameter Sources

go-specgen uses struct tags to define where parameters come from:

| Tag | Location | Example |
|-----|----------|---------|
| `path:"id"` | URL path | `/users/{id}` |
| `query:"limit"` | Query string | `?limit=10` |
| `header:"Authorization"` | HTTP header | `Authorization: Bearer ...` |
| `cookie:"session"` | HTTP cookie | `session=abc123` |

**Note:** These tags define parameter locations in the OpenAPI spec. go-specgen does **not** bind request data at runtime.

### Type Resolution

Go types map to OpenAPI types automatically:

| Go Type | OpenAPI Type | Format |
|---------|--------------|--------|
| `string` | `string` | - |
| `int`, `int32` | `integer` | `int32` |
| `int64` | `integer` | `int64` |
| `float32` | `number` | `float` |
| `float64` | `number` | `double` |
| `bool` | `boolean` | - |
| `time.Time` | `string` | `date-time` |
| `url.URL` | `string` | `uri` |
| `[]T` | `array` | items: T |
| `[N]T` | `array` | items: T |
| `[]byte` | `string` | `byte` |
| `map[string]T` | `object` | additionalProperties: T |
| `*T` | nullable T | - |
| `any` | `{}` | any JSON value |

Containers nest to any depth, and every level keeps its own type and format:
`map[string][]time.Time` is an object whose `additionalProperties` is an array
whose `items` are `date-time` strings.

A Go type with no OpenAPI representation is an error, not a guess. Channels,
functions, complex numbers and `uintptr` are rejected — `encoding/json` refuses
to marshal them too, so there is nothing truthful to emit. A field whose type is
a struct without `@schema` is rejected the same way.

Custom types resolve to their underlying type:

```go
type UserID string      // -> type: string
type StatusCode int     // -> type: integer
type Tags []string      // -> type: array, items: string
```

### Required vs Optional

**Schema fields** — determined by Go types:

| Type | Required | Nullable |
|------|----------|----------|
| `string` | Yes | No |
| `*string` | Yes | Yes |
| `string` with `omitempty` | No | No |
| `*string` with `omitempty` | No | No |
| struct (e.g. `time.Time`) with `omitempty` | Yes | No |
| struct with `omitzero` | No | No |

These defaults are outcome-oriented: they describe what `encoding/json`
actually puts on the wire, not what the tag text says.

A pointer alone makes a field nullable because `encoding/json` marshals a nil
pointer as `null`. Adding `omitempty` changes that: a nil pointer is omitted
entirely, so the field can never appear as `null` on the wire — it is optional,
not nullable. Use `@nullable true` to opt back in (e.g., a PATCH API that
accepts explicit `null` to clear a value).

`omitempty` can only drop values `encoding/json` considers empty — `false`,
`0`, `""`, a nil pointer or interface, and an empty string, slice, map, or
array. A non-pointer struct is never empty, so `time.Time` with `omitempty`
is still emitted on every response and stays required.

`omitzero` (Go 1.24+) omits the zero value of any type, so it always makes a
field optional — including structs, where it omits the zero value `omitempty`
cannot. On pointers it behaves like `omitempty`: the nil pointer is omitted
rather than encoded as `null`, so the field is optional and non-nullable.

**In-function `@request` structs** are the one exception. A named `@schema` is
directionless and always uses the marshal rule above, but an inline `@request`
struct exists only as a request body, so its own fields describe what
`json.Unmarshal` tolerates instead:

| Type | Required | Nullable |
|------|----------|----------|
| `string` | Yes | No |
| `*string` | No | No |
| `string` with `omitempty`/`omitzero` | Yes | No |

A pointer means the field may be absent, but decoding `null` into a non-pointer
fails, so nothing is nullable by default. `omitempty`/`omitzero` only affect
encoding and are ignored. `@required`/`@nullable` overrides still win. The rule
applies only to the inline struct's own fields — a named schema referenced from
one keeps the marshal rule.

**Parameter fields** — determined by parameter type:

| Parameter type | Default | Required when |
|----------------|---------|---------------|
| `path` | Always required | Always |
| `query` | Optional | The `query` tag has the `required` option (e.g., `query:"q,required"`) |
| `header` | Optional | The `header` tag has the `required` option |
| `cookie` | Optional | The `cookie` tag has the `required` option |

The option is read from the tag matching the parameter's own kind. A `,required`
sitting in some other tag on the same field has no effect.

**Overrides** — `@required` and `@nullable` on `@field` let you decouple the OpenAPI contract from Go's type/tag defaults when they don't match what you want to expose:

```go
// @schema
type LoginRequest struct {
    // Non-pointer, but optional in the request — avoids *string just to mark optional
    // @field { @description 2FA code @required false }
    TFACode string `json:"tfa_code"`
}

// @schema
type CreatePromo struct {
    // Pointer so the handler can distinguish "not sent" from "sent 0",
    // but the field must still be present in the request
    // @field { @description Discount percent @required true }
    Percent *int `json:"percent"`
}

// @schema
type UpdateUserPatch struct {
    // PATCH semantics: omit to leave unchanged, send null to clear.
    // Pointer+omitempty defaults to non-nullable, so opt back in explicitly
    // @field { @description Replace email; omit to leave unchanged, null to clear @nullable true }
    Email *string `json:"email,omitempty"`
}
```

Both take `true` or `false`. Written bare — `@required`, `@nullable` — they mean
`true`, the same as every other modifier inside `@field`. The explicit value is
there for the `false` case, which is the one Go's defaults cannot express.

When omitted, behavior falls back to the Go-type rules in the tables above.

### Embedded Structs

Embedded (anonymous) fields follow `encoding/json` exactly:

```go
type BaseModel struct {
    // @field { @description Identifier }
    ID string `json:"id"`
}

type Meta struct{ Revision int `json:"revision"` }
type Hidden struct{ Secret string `json:"secret"` }
type Label string

// @schema
type Document struct {
    BaseModel                 // untagged struct    -> fields flattened into Document
    Meta      `json:"meta"`   // tagged struct      -> nested under "meta"
    Hidden    `json:"-"`      // tagged "-"         -> omitted entirely
    Label                     // untagged non-struct -> field named "Label"

    Title string `json:"title"`
}
```

An untagged embedded struct does not need `@schema` — its fields are flattened
in, and they bring their own `@field` annotations with them. A *tagged* embedded
struct becomes a nested field, so it does need `@schema` in order to be
referenced.

Embedded `time.Time` and the other standard-library types that serialize as
scalars are not flattened; they stay single fields.

### Schema References

Fields referencing other `@schema` types automatically generate `$ref`:

```go
// @schema
type Address struct {
    Street string `json:"street"`
    City   string `json:"city"`
}

// @schema
type User struct {
    Home      Address            `json:"home"`       // $ref: Address
    Work      []Address          `json:"work"`       // array of $ref
    Locations map[string]Address `json:"locations"`  // additionalProperties: $ref
}
```

---

## Features

### Inline Structs

Define handler-specific structs directly inside endpoint functions:

```go
// @endpoint GET /users
func ListUsers(w http.ResponseWriter, r *http.Request) {
    // @query
    var query struct {
        // @field { @description Maximum results @minimum 1 @maximum 100 }
        Limit int `query:"limit"`
    }

    // @response 200
    var resp struct {
        Users []User `json:"users"`
        Total int    `json:"total"`
    }
}
```

- Supports `@query`, `@path`, `@header`, `@cookie`, `@request`, and `@response`
- Inlined in spec (not added to `components/schemas`)
- Explicit references in `@endpoint` block override auto-discovery

**Handler factory (closure) pattern:**

Inline structs also work inside handler factories that return a closure. Annotations in the outer function body and inside the returned `func` literal are both discovered:

```go
// @endpoint POST /greet {
//   @summary Greet a user
// }
func HandleGreet() http.HandlerFunc {
    // @request
    type request struct {
        Name string `json:"name"`
    }

    return func(w http.ResponseWriter, r *http.Request) {
        // @response 200
        type response struct {
            Greeting string `json:"greeting"`
        }
    }
}
```

### Response Wrappers

**Using @bind:**

```go
// @schema
type APIResponse struct {
    Status string `json:"status"`
    Data   any    `json:"data"`
}

// @endpoint GET /users/{id} {
//   @response 200 {
//     @body User
//     @bind APIResponse.Data
//   }
// }
```

**Using generics:**

```go
// @schema
type APIResponse[T any] struct {
    Status string `json:"status"`
    Data   T      `json:"data"`
}

type UserResponse = APIResponse[User]

// @endpoint GET /users/{id} {
//   @response 200 { @body UserResponse }
// }
```

### Default Content Type

Set a default at the `@api` level:

```go
// @api {
//   @title My API
//   @version 1.0.0
//   @defaultContentType json
// }
```

---

## Annotation Reference

### Top-Level Annotations

| Annotation | Target | Description |
|------------|--------|-------------|
| `@api { }` | Package | API metadata, servers, security |
| `@schema` | Struct | Mark as OpenAPI schema |
| `@path` | Struct | Path parameters (fields use `path:` tag) |
| `@query` | Struct | Query parameters (fields use `query:` tag) |
| `@header` | Struct | Header parameters (fields use `header:` tag) |
| `@cookie` | Struct | Cookie parameters (fields use `cookie:` tag) |
| `@endpoint METHOD /path { }` | Function | Define an endpoint |
| `@field { }` | Field | Field metadata |

### @api

The `@api` block can appear directly above the `package` keyword or as a standalone comment anywhere in the file. This lets you keep your package documentation separate from API metadata.

```
@api {
  @title           (required) API title
  @version         (required) API version
  @description     API description (multi-line supported)
  @termsOfService  URL to terms
  @defaultContentType  Default content type (json, xml, etc.)
  @contact { }     Contact info
  @license { }     License info
  @server URL { }  Server definition (repeatable)
  @tag name { }    Tag definition (repeatable)
  @securityScheme name { }  Security scheme (repeatable)
  @security { }    Default security requirement (repeatable)
}
```

### @schema

```
@schema
@schema {
  @description   Schema description (multi-line supported)
  @deprecated    Mark as deprecated
}
```

### @endpoint

```
@endpoint METHOD /path {
  @operationID     Operation identifier
  @summary         Short summary
  @description     Detailed description (multi-line supported)
  @tag             Tag reference (repeatable)
  @deprecated      Mark as deprecated
  @auth            Security scheme to use
  @path            Path parameter struct (repeatable)
  @query           Query parameter struct (repeatable)
  @header          Header parameter struct (repeatable)
  @cookie          Cookie parameter struct (repeatable)
  @request { }     Request body
  @response CODE { }  Response definition (repeatable)
}
```

### @request / @response

```
@request {
  @contentType   json|form|multipart|text|binary
  @body Schema   Schema reference
}

@response CODE {
  @contentType   json|text|binary|empty
  @body Schema   Schema reference
  @bind Wrapper.Field   Wrap body in response envelope
  @header Name   Response header struct reference (repeatable)
  @description   Response description (defaults to the reason phrase)
}
```

CODE can be a specific status (`200`, `404`), a range (`2XX`, `4XX`, `5XX`), or `default`:

```
@response 200 { @body User }
@response 4XX { @body Error @description Client error }
@response 5XX { @body Error @description Server error }
@response default { @body Error }
```

The Response Object requires a description, so one is always emitted. Without
`@description` it is the status code's reason phrase — `200` becomes `OK`, `201`
becomes `Created`, `204` becomes `No Content`. A range or `default` names no
single status and so has no phrase; those describe what they cover instead
(`Response for status 4XX`, `Default response`). Both response forms — the named
one above and the in-function one — use the same fallback.

**Content type support:**

| Keyword | MIME | Schema Support |
|---------|------|----------------|
| `json` | `application/json` | Full |
| `form` | `application/x-www-form-urlencoded` | Full |
| `multipart` | `multipart/form-data` | Full |
| `xml` | `application/xml` | Keyword only |
| `text` | `text/plain` | None |
| `binary` | `application/octet-stream` | None |
| `html` | `text/html` | None |
| `empty` | (none) | None |

### @field

```
@field {
  @description        Field description (multi-line supported)
  @format             Format: email, uuid, date-time, uri, etc.
  @example            Example value
  @enum               Comma-separated values
  @default            Default value
  @minimum            Minimum value (numbers)
  @maximum            Maximum value (numbers)
  @exclusiveMinimum   Exclusive minimum value (numbers)
  @exclusiveMaximum   Exclusive maximum value (numbers)
  @minLength          Minimum length (strings)
  @maxLength          Maximum length (strings)
  @minItems           Minimum items (arrays)
  @maxItems           Maximum items (arrays)
  @uniqueItems        Require unique items (arrays)
  @pattern            Regex pattern
  @deprecated         Mark as deprecated
  @readOnly           Mark as read-only
  @writeOnly          Mark as write-only
  @required           Override required (true|false, bare means true)
  @nullable           Override nullable (true|false, bare means true)
}
```

### @securityScheme

```
@securityScheme name {
  @type          (required) http|apiKey|oauth2|openIdConnect
  @scheme        bearer|basic (for http type)
  @bearerFormat  JWT, etc.
  @in            header|query|cookie (for apiKey)
  @name          Parameter name (for apiKey)
  @description   Description
}
```

### @security

```
@security {
  @with schemeName           Simple reference
  @with schemeName {         With OAuth2 scopes
    @scope read:users
    @scope write:users
  }
}
```

**AND/OR logic:**

| Pattern | Syntax | Meaning |
|---------|--------|---------|
| OR | Multiple `@security` blocks | Any one scheme works |
| AND | Multiple `@with` in one `@security` | All schemes required |

---

## Examples

See the [examples/](examples/) directory for complete working examples:

- Basic API setup
- Path, query, header, and cookie parameters
- Request/response bodies
- Security schemes
- Inline structs
- Closure handler factories
- Response wrappers
- Generics

---

## Reference Details

### Block Syntax

All block annotations use curly braces `{ }` for grouping. Blocks can be written in multiple formats:

**Multi-line:**
```go
// @endpoint GET /users/{id} {
//   @summary Get user by ID
//   @response 200 {
//     @body User
//   }
// }
```

**Inline:**
```go
// @field { @description User email @format email }
// @response 200 { @body User @description Found }
```

**Empty block:**
```go
// @schema { }
```

**Nested blocks rule:** Inline blocks cannot contain other blocks. Use multi-line format for nesting.

### Escaping Special Characters

| Escape | Result | Use case |
|--------|--------|----------|
| `\{`   | `{`    | A literal brace in a value — JSON examples and defaults |
| `\}`   | `}`    | A literal brace in a value — JSON examples and defaults |
| `\@`   | `@`    | Email addresses; a description line that starts with `@` |
| `\\`   | `\`    | Literal backslash |

```go
// @field { @description Contact admin\@example.com }
// @field { @example \{"name": "John"\} }    // JSON example
```

**`@pattern` is the exception.** A regex is a language with its own brace rules,
so `@pattern` is the one annotation whose value is passed through verbatim —
write the regex exactly as you mean it, escaping nothing:

```go
// @field { @pattern ^[A-Z]{2}$ }             // quantifier, as written
// @field { @pattern ^a}b$ }                  // unpaired brace is fine
// @field { @pattern [{] }                    // so is a literal one
```

Escaping a `@pattern` changes it: `^[A-Z]\{2\}$` reaches the spec with the
backslashes intact, where `\{2\}` matches a literal `{2}` rather than repeating
the previous character twice. `examples/rawvalue/` pins this — every regex there
is written twice, inline and as a block, and the two forms must agree.

### Multi-line Descriptions

Only `@description` supports multi-line values:

```go
// @api {
//   @title My API
//   @version 1.0.0
//   @description This is a multi-line description.
//   It continues on this line.
//   And this line too.
// }
```

The value ends at the next line beginning with an unescaped `@`, whether or not
specgen recognizes the name. A line of prose that starts with `@` escapes it:

```go
// @field {
//   @description Questions go to
//   \@support, not the on-call rota.
// }
```

Without the escape, that line is an annotation named `@support`, and specgen
reports it as unknown. That is deliberate: the alternative is reading a
misspelled annotation as prose and publishing it in the description.

### Parameter Rules

**Path (`@path`):** Always required, scalars only, no arrays.

**Query (`@query`):** Optional by default; `query:"q,required"` marks it required. Arrays allowed, for repeated parameters.

**Header (`@header`):** Optional by default; `header:"X-Key,required"` marks it required. No arrays.

**Cookie (`@cookie`):** Optional by default; `cookie:"session,required"` marks it required. No arrays.

**Naming** mirrors `encoding/json`, so one rule covers parameters and bodies alike:

| Tag | Parameter |
|-----|-----------|
| `query:"limit"` | named `limit` |
| `query:"-"` | skipped |
| `query:"-,"` | named `-` |
| no `query` tag | named by the Go field |

Field names are never transformed: what you write is what the spec says.

A parameter is a scalar or a list of scalars, and nothing else. `net/http` hands
parameters over as `map[string][]string`, so there is no structured value to
decode into — a struct-typed parameter field is an error, and `deepObject` is not
supported. A field tagged for a different kind than the struct it sits in
(`path:"x"` inside a `@query` struct) is an error too, rather than being renamed.

### Where the spec and `encoding/json` differ

specgen describes what `encoding/json` puts on the wire, so most of the time the
two agree by construction. These are the places they deliberately do not:

| | `encoding/json` | specgen |
|---|---|---|
| nil slice / nil map | writes `null` | not nullable — a handler returning one nearly always means "empty", and `@nullable true` opts in |
| pointer parameter | n/a | never nullable: a parameter is text in a URL or header, which cannot carry `null`. The pointer only lets the handler tell absent from zero |
| generic template | marshals normally | not emitted to `components/schemas` — a template is not a type. Aliases that instantiate it are emitted |
| unmarshalable type | fails at runtime | rejected at generation time |

And two things specgen does *not* do, deliberately:

- **No name transformation.** An untagged field is named exactly as it is
  declared in Go. There is no snake_case conversion anywhere in the pipeline.
- **No `deepObject`.** Structured parameters have no `net/http` model, so they
  are an error rather than a guess at an encoding.

### Errors

specgen fails rather than emitting a spec it cannot stand behind. The cases:

- a type with no OpenAPI representation (channel, function, complex, `uintptr`)
- a field whose type is a struct without `@schema`
- a parameter that is not a scalar or a list of scalars
- a parameter field tagged for a different kind than its struct
- a body naming a schema that does not exist
- an endpoint tag with no API-level `@tag`
- the same parameter name twice in the same location
- constraints that contradict each other, or apply to the wrong type
- an annotation on a declaration that cannot carry it (see below)

Every stage accumulates, so one run reports every mistake it can reach rather
than stopping at the first:

```
$ specgen -package ./api -yaml openapi.yaml
Error: parse: 4 parse errors:
  1. @endpoint[ListOrders]: failed to parse @endpoint children: unknown annotation @produces in @endpoint
  2. @field[Customer.Tier]: unknown annotation @oneOf in @field
  3. @field[Order.ItemCount]: unknown annotation @desc in @field
  4. @field[Order.TotalCents]: unknown annotation @multipleOf in @field
```

Each error names the declaration it came from — `@schema[Order].ItemCount`,
`@query[OrderFilter].TenantID`, `@endpoint[GET /orders]` — in the annotation
vocabulary rather than Go's.

Two things still end a run early. Inside a single annotation block, parsing stops
at the first name it does not recognize — `{ @desc Line items @min 1 }` reports
`@desc`, and `@min` turns up on the next run. And stage boundaries hold: parse,
then resolve, then validate. The resolver never sees a package that failed to
parse, so a run reports everything wrong at one stage rather than across all
three.

### Annotations in the wrong place

The annotation that opens a doc comment decides what the declaration is, and each
kind of declaration accepts its own:

```
package doc        @api
type               @schema @path @query @header @cookie
func               @endpoint
struct field       @field
in a func body     @path @query @header @cookie @request @response
```

Writing a real annotation somewhere it cannot mean anything is an error, because
there is exactly one thing you meant:

```
Error: parse: 2 parse errors:
  1. type Widget: @summary cannot annotate a type; valid here: @cookie, @header, @path, @query, @schema
  2. func ListWidgets: @schema cannot annotate a func; valid here: @endpoint
```

A name that appears in no annotation at all is reported instead, and the run
continues:

```
$ specgen -package ./api -yaml openapi.yaml
specgen: func ListWidgets: @endpoin is not an annotation; it was skipped
Wrote openapi.yaml
```

It cannot be an error, because a doc comment is also ordinary documentation and
nothing distinguishes `@endpoin` from an `@author` line in a package that never
asked specgen for an opinion. But a misspelled `@endpoint` costs a whole
operation, so it is not silent either. Write `\@` for a doc comment that really
does begin a line with `@`.

Only the *opening* annotation is subject to this. Names written inside a block
are checked against the grammar and always error, and a comment that discusses an
annotation in prose before declaring the real one is fine — what matters is
whether any line claims the declaration, not what the first line says.

### Limitations

- **JSON only** - Field names parsed from `json` struct tags
- **OpenAPI 3.1+** - Supports 3.1 and 3.2 (not 3.0, not OpenAPI 2.0/Swagger)

### Requirements

- Go 1.21+ (for module support and field ordering)
- Go 1.24+ (for `go tool` directive)

---

## Future Features

- XML/YAML struct tag support
- OAuth2 flows configuration
- External documentation support
- Custom type support for structs implementing `json.Marshaler` / `json.Unmarshaler` (e.g. `sql.NullString`)

---

## License

MIT License - see [LICENSE](LICENSE) file.

---

## Acknowledgments

This project is made possible by [libopenapi](https://github.com/pb33f/libopenapi) from [pb33f](https://pb33f.io).

Created and maintained with [Claude Code](https://claude.ai/code).
