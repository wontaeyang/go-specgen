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
| `int`, `int32` | `integer` | `int32` |
| `int64` | `integer` | `int64` |
| `float32` | `number` | `float` |
| `float64` | `number` | `double` |
| `bool` | `boolean` | - |
| `time.Time` | `string` | `date-time` |
| `url.URL` | `string` | `uri` |
| `[]T` | `array` | items: T |
| `*T` | nullable T | - |
| `any` | `{}` | any JSON value |

Custom types resolve to their underlying type:

```go
type UserID string      // -> type: string
type StatusCode int     // -> type: integer
type Tags []string      // -> type: array, items: string
```

### Required vs Optional

Determined by Go types, not annotations:

| Type | Required | Nullable |
|------|----------|----------|
| `string` | Yes | No |
| `*string` | No | Yes |
| `string` with `omitempty` | No | No |

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
  @description   Field description (multi-line supported)
  @format        Format: email, uuid, date-time, uri, etc.
  @example       Example value
  @enum          Comma-separated values
  @default       Default value
  @minimum       Minimum value (numbers)
  @maximum       Maximum value (numbers)
  @minLength     Minimum length (strings)
  @maxLength     Maximum length (strings)
  @minItems      Minimum items (arrays)
  @maxItems      Maximum items (arrays)
  @uniqueItems   Require unique items (arrays)
  @pattern       Regex pattern
  @deprecated    Mark as deprecated
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

**Path (`@path`):** All fields required, simple types only, no arrays/objects.

**Query (`@query`):** Pointers for optional, arrays allowed for repeated params.

**Header (`@header`):** No arrays/objects. Can be request or response headers.

**Cookie (`@cookie`):** Pointers for optional, no arrays/objects.

### Limitations

- **JSON only** - Field names parsed from `json` struct tags
- **OpenAPI 3.x** - Supports 3.0, 3.1, 3.2 (not OpenAPI 2.0/Swagger)

### Requirements

- Go 1.21+ (for module support and field ordering)
- Go 1.24+ (for `go tool` directive)

---

## Future Features

- `@exclusiveMinimum`/`@exclusiveMaximum` for numeric bounds
- `$ref` with siblings (description/nullable on references)
- XML/YAML struct tag support
- OAuth2 flows configuration
- External documentation support

---

## License

MIT License - see [LICENSE](LICENSE) file.

---

## Acknowledgments

This project is made possible by [libopenapi](https://github.com/pb33f/libopenapi) from [pb33f](https://pb33f.io).

Created and maintained with [Claude Code](https://claude.ai/code).
