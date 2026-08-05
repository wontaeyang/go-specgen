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
go tool specgen -package ./handlers -output openapi.yaml
```

### Using go install

```bash
go install github.com/wontaeyang/go-specgen/cmd/specgen@latest
specgen -package ./handlers -output openapi.yaml
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
  -output string     Output file path (default "openapi.yaml")
  -format string     Output format: json or yaml (default "yaml")
  -openapi string    OpenAPI version: 3.0, 3.1, or 3.2 (default "3.0")
  -version           Show version
  -help              Show help
```

**Examples:**

```bash
# Generate from current directory
specgen

# Generate JSON from specific package
specgen -package ./api/handlers -format json -output openapi.json

# Generate OpenAPI 3.1
specgen -openapi 3.1
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
| `int`, `int8`, `int16` | `integer` | - |
| `int32` | `integer` | `int32` |
| `int64` | `integer` | `int64` |
| `uint`, `uint8` … `uint64` | `integer` | - (OpenAPI has no unsigned formats) |
| `float32` | `number` | `float` |
| `float64` | `number` | `double` |
| `bool` | `boolean` | - |
| `time.Time` | `string` | `date-time` |
| `url.URL` | `string` | `uri` |
| `[]byte` | `string` | `byte` (base64, not an array) |
| `[]T` | `array` | items: T, keeping T's format |
| `map[string]T` | `object` | additionalProperties: T, keeping T's format |
| `*T` | nullable T | - |
| `any` | `{}` | any JSON value |

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

**Parameter fields** — determined by parameter type:

| Parameter type | Default | Required when |
|----------------|---------|---------------|
| `path` | Always required | Always |
| `query` | Optional | Tag contains `,required` (e.g., `query:"q,required"`) |
| `header` | Optional | Tag contains `,required` |
| `cookie` | Optional | Tag contains `,required` |

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

When omitted, behavior falls back to the Go-type rules in the tables above.

### Embedded Structs

Embedded (anonymous) struct fields are flattened into the parent schema or parameter:

```go
type BaseModel struct {
    ID        string `json:"id"`
    CreatedAt string `json:"created_at"`
}

// @schema
type User struct {
    BaseModel                    // Fields flattened into User
    Name  string `json:"name"`
    Email string `json:"email"`
}
```

The embedded struct does not need a `@schema` annotation — its fields are inlined directly.

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
  @description   Response description
}
```

CODE can be a specific status (`200`, `404`), a range (`2XX`, `4XX`, `5XX`), or `default`:

```
@response 200 { @body User }
@response 4XX { @body Error @description Client error }
@response 5XX { @body Error @description Server error }
@response default { @body Error }
```

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
  @required           Override required (true|false)
  @nullable           Override nullable (true|false)
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
- Field constraints: read/write visibility, array bounds and uniqueness, numeric bounds, deprecation
- Enums and nullability overrides
- Schema references, including nullable and annotated refs
- APIs defined entirely from in-function structs, with no package-level schemas
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
| `\{`   | `{`    | Regex quantifiers, JSON examples |
| `\}`   | `}`    | Regex quantifiers, JSON examples |
| `\@`   | `@`    | Email addresses |
| `\\`   | `\`    | Literal backslash |

```go
// @field { @pattern ^[A-Z]\{2\}$ }           // Regex: ^[A-Z]{2}$
// @field { @description Contact admin\@example.com }
// @field { @example \{"name": "John"\} }    // JSON example
```

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

### Parameter Rules

**Path (`@path`):** Always required, simple types only, no arrays/objects.

**Query (`@query`):** Optional by default, use `,required` tag to mark required (e.g., `query:"q,required"`). Arrays allowed for repeated params.

**Header (`@header`):** Optional by default, use `,required` tag to mark required. No arrays/objects.

**Cookie (`@cookie`):** Optional by default, use `,required` tag to mark required. No arrays/objects.

### Limitations

- **JSON only** - Field names parsed from `json` struct tags
- **OpenAPI 3.x** - Supports 3.0, 3.1, 3.2 (not OpenAPI 2.0/Swagger)

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
