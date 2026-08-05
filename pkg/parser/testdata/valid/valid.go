//	@api {
//	  @title Parser Fixture API
//	  @version 2.1.0
//	  @description Fixture package covering every annotation the parser reads.
//	  This second line continues the description.
//	  @termsOfService https://example.com/terms
//	  @contact {
//	    @name API Team
//	    @email support\@example.com
//	    @url https://example.com/support
//	  }
//	  @license {
//	    @name MIT
//	    @url https://opensource.org/licenses/MIT
//	  }
//	  @server https://api.example.com/v1 {
//	    @description Production
//	  }
//	  @server https://staging.example.com/v1 {
//	    @description Staging
//	  }
//	  @securityScheme bearerAuth {
//	    @type http
//	    @scheme bearer
//	    @bearerFormat JWT
//	    @description JWT bearer token
//	  }
//	  @securityScheme apiKeyAuth {
//	    @type apiKey
//	    @in header
//	    @name X-API-Key
//	  }
//	  @security {
//	    @with bearerAuth {
//	      @scope read:users
//	      @scope write:users
//	    }
//	  }
//	  @security {
//	    @with apiKeyAuth
//	  }
//	  @tag users {
//	    @description User operations
//	  }
//	  @tag admin
//	  @defaultContentType json
//	}
package valid

// -----------------------------------------------------------------------------
// Schemas
// -----------------------------------------------------------------------------

// User exercises every @field child.
//
//	@schema {
//	  @description A user of the system.
//	  Second line of the schema description.
//	}
type User struct {
	// @field {
	//   @description User ID
	//   @format uuid
	//   @readOnly
	// }
	ID string `json:"id"`

	// @field { @description Email address @format email @example alice\@example.com }
	Email string `json:"email"`

	// @field { @description Login name @minLength 3 @maxLength 32 @pattern ^[a-z]{3,32}$ }
	Username string `json:"username"`

	// @field { @description Role @enum admin, user , guest @default user }
	Role string `json:"role"`

	// @field { @description Age @minimum 0 @maximum 130 @exclusiveMinimum 0.5 @exclusiveMaximum 129.5 }
	Age int `json:"age"`

	// @field { @description Labels @minItems 1 @maxItems 10 @uniqueItems }
	Labels []string `json:"labels"`

	// @field { @description Password @writeOnly }
	Password string `json:"password"`

	// @field { @description Legacy handle @deprecated }
	Legacy string `json:"legacy"`

	// @field { @required false @nullable true }
	Nickname *string `json:"nickname"`

	// No annotation: the parser produces no Field entry for this one.
	Internal string `json:"internal"`
}

// Escapes covers escape sequences in values and raw pattern passthrough.
// @schema
type Escapes struct {
	// @field { @description Braces \{like this\} and an at \@sign }
	Text string `json:"text"`

	// @field { @pattern ^\d{3}-\d{4}$ }
	Phone string `json:"phone"`
}

// LegacyUser is retired.
//
//	@schema {
//	  @description Superseded by User.
//	  @deprecated
//	}
type LegacyUser struct {
	// @field { @description User ID }
	ID string `json:"id"`
}

// Error is the shared error body.
// @schema
type Error struct {
	// @field { @description Machine readable code }
	Code string `json:"code"`

	// @field { @description Human readable message }
	Message string `json:"message"`
}

// Envelope wraps bound bodies.
// @schema
type Envelope struct {
	// @field { @description Wrapped payload }
	Data any `json:"data"`
}

// Anonymous carries anonymous structs, whose fields have no declaration of
// their own to hang @field annotations on.
// @schema
type Anonymous struct {
	// @field { @description The shipping address }
	Address struct {
		// @field { @description Street and number }
		Street string `json:"street"`

		// @field { @description Two-letter country code }
		Country string `json:"country"`
	} `json:"address"`

	// Untagged carries nested annotations without one of its own.
	Untagged []struct {
		// @field { @description Line quantity @minimum 1 }
		Quantity int `json:"quantity"`
	} `json:"untagged"`

	// @field { @description A plain field, with nothing nested }
	Note string `json:"note"`
}

// Status is not a struct, so its @schema annotation is ignored — only struct
// declarations and type aliases can become schemas.
// @schema
type Status string

// -----------------------------------------------------------------------------
// Generics and aliases
// -----------------------------------------------------------------------------

// Response is a generic wrapper template.
// @schema
type Response[T any] struct {
	// @field { @description Whether the call succeeded }
	Success bool `json:"success"`

	// @field { @description Payload }
	Data T `json:"data"`
}

// Pair exercises multiple type arguments.
// @schema
type Pair[K comparable, V any] struct {
	// @field { @description Key }
	Key K `json:"key"`

	// @field { @description Value }
	Value V `json:"value"`
}

// UserResponse instantiates the generic wrapper: it becomes a schema of its own.
type UserResponse = Response[User]

// StringUserPair instantiates the two-argument generic.
type StringUserPair = Pair[string, User]

// UserAlias aliases a non-generic schema: no derived schema entry.
type UserAlias = User

// -----------------------------------------------------------------------------
// Parameter structs
// -----------------------------------------------------------------------------

// UserPath carries the path parameter.
// @path
type UserPath struct {
	// @field { @description User ID @format uuid }
	ID string `path:"id"`
}

// ListQuery carries list query parameters.
// @query
type ListQuery struct {
	// @field { @description Page size @minimum 1 @maximum 100 @default 20 }
	Limit *int `query:"limit"`

	// Unannotated: the resolver picks it up from the Go struct.
	Cursor *string `query:"cursor"`
}

// RateLimitHeaders carries rate limit headers.
// @header
type RateLimitHeaders struct {
	// @field { @description Requests allowed per hour }
	Limit int `header:"X-RateLimit-Limit"`
}

// SessionCookie carries the session cookie.
// @cookie
type SessionCookie struct {
	// @field { @description Session token }
	Token string `cookie:"session"`
}

// -----------------------------------------------------------------------------
// Go-type fixtures
//
// These carry few annotations on purpose: they exist so the resolver has a
// package with the Go shapes it has to reason about — tag options, embedding,
// pointers — without needing a fixture package of its own.
// -----------------------------------------------------------------------------

// FieldRequiredTest covers the tag/type combinations that decide whether a
// field is required and whether it is nullable.
// @schema
type FieldRequiredTest struct {
	Value          string  `json:"value"`
	ValuePtr       *string `json:"value_ptr"`
	ValueOmit      string  `json:"value_omit,omitempty"`
	ValuePtrOmit   *string `json:"value_ptr_omit,omitempty"`
	ValueZero      string  `json:"value_zero,omitzero"`
	ValuePtrZero   *string `json:"value_ptr_zero,omitzero"`
	ValueStruct    User    `json:"value_struct"`
	ValueStructPtr *User   `json:"value_struct_ptr"`

	ValueStructOmit User `json:"value_struct_omit,omitempty"`
	ValueStructZero User `json:"value_struct_zero,omitzero"`

	ValueSlice        []string          `json:"value_slice"`
	ValueSliceOmit    []string          `json:"value_slice_omit,omitempty"`
	ValueSlicePtr     *[]string         `json:"value_slice_ptr"`
	ValueSlicePtrOmit *[]string         `json:"value_slice_ptr_omit,omitempty"`
	ValueMap          map[string]string `json:"value_map"`
	ValueMapOmit      map[string]string `json:"value_map_omit,omitempty"`
}

// BaseModel is embedded by other schemas.
type BaseModel struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// Auditable is embedded alongside BaseModel.
type Auditable struct {
	DeletedAt *string `json:"deleted_at,omitempty"`
	DeletedBy *string `json:"deleted_by,omitempty"`
}

// EmbeddedTest embeds a struct by value.
// @schema
type EmbeddedTest struct {
	BaseModel
	Name string `json:"name"`
}

// EmbeddedPtrTest embeds a struct by pointer.
// @schema
type EmbeddedPtrTest struct {
	*BaseModel
	Label string `json:"label"`
}

// NestedEmbedTest embeds two structs.
// @schema
type NestedEmbedTest struct {
	BaseModel
	Auditable
	Status string `json:"status"`
}

// CommonQueryParams is embedded by a parameter struct.
type CommonQueryParams struct {
	Limit  *int `query:"limit,omitempty"`
	Offset *int `query:"offset,omitempty"`
}

// EmbeddedQueryParams embeds shared query parameters.
// @query
type EmbeddedQueryParams struct {
	CommonQueryParams

	// @field { @description Search term }
	Search string `query:"search"`
}

// -----------------------------------------------------------------------------
// Endpoints
// -----------------------------------------------------------------------------

// GetUser returns one user.
//
//	@endpoint GET /users/{id} {
//	  @operationID getUser
//	  @summary Get a user
//	  @description Returns a single user.
//	  This second line continues the description.
//	  @tag users
//	  @tag admin
//	  @auth bearerAuth
//	  @path UserPath
//	  @query ListQuery
//	  @header RateLimitHeaders
//	  @cookie SessionCookie
//	  @response 200 {
//	    @contentType json
//	    @body User
//	    @description Found
//	    @header RateLimitHeaders
//	  }
//	  @response 404 { @body Error @description Not found }
//	  @response default { @body Error }
//	}
func GetUser() {}

// CreateUser creates a user.
//
//	@endpoint POST /users {
//	  @deprecated
//	  @request {
//	    @contentType xml
//	    @body User
//	    @bind Envelope.Data
//	  }
//	  @response 201 {
//	    @body []User
//	    @bind Envelope.Data
//	  }
//	  @response 204 { @contentType empty }
//	}
func CreateUser() {}

// RepeatStatus declares the same status twice: the last one wins.
//
//	@endpoint GET /repeat {
//	  @response 200 { @description first }
//	  @response 200 { @description second }
//	}
func RepeatStatus() {}

// InlineHandler declares its parameters and bodies in the function body.
//
//	@endpoint POST /orders {
//	  @summary Create an order
//	}
func InlineHandler() {
	// @path
	var path struct {
		// @field { @description Order ID @format uuid }
		ID string `path:"id"`
	}

	// @query
	var filters struct {
		// @field { @description Status filter @enum open,closed }
		Status *string `query:"status"`
	}

	// @query
	var paging struct {
		// @field { @description Page size @minimum 1 }
		Limit *int `query:"limit"`
	}

	// @header
	var headers struct {
		// @field { @description Idempotency key }
		Key string `header:"X-Idempotency-Key"`
	}

	// @cookie
	var cookies struct {
		// @field { @description Session token }
		Session string `cookie:"session"`
	}

	// @request { @contentType xml @description Order payload @bind Envelope.Data }
	var req struct {
		// @field { @description Customer ID @format uuid }
		CustomerID string `json:"customer_id"`
	}

	// @response 201
	var created struct {
		// @field { @description Order ID }
		ID string `json:"id"`
	}

	// @response 4XX {
	//   @contentType json
	//   @description Client error
	//   @header RateLimitHeaders
	// }
	var clientError struct {
		// @field { @description Error message }
		Message string `json:"message"`
	}

	// @response default
	var fallback struct {
		// @field { @description Error message }
		Message string `json:"message"`
	}

	// @response
	var unspecified struct {
		// @field { @description Whether the call succeeded }
		OK bool `json:"ok"`
	}

	// A standalone @response: no declaration follows it, so it names its body.
	// @response 404 {
	//   @body Error
	//   @description Order not found
	//   @header RateLimitHeaders
	// }

	_, _, _, _, _ = path, filters, paging, headers, cookies
	_, _, _, _ = req, created, clientError, fallback
	_ = unspecified
}

// ClosureHandler declares its request in the factory and its response in the
// returned closure.
//
//	@endpoint POST /greet {
//	  @summary Greet someone
//	}
func ClosureHandler() func() {
	// @request
	type request struct {
		// @field { @description Name to greet }
		Name string `json:"name"`
	}

	return func() {
		// @response 200
		type response struct {
			// @field { @description Greeting }
			Greeting string `json:"greeting"`
		}

		_ = response{}
	}
}

// helper is not an endpoint, so its inline declarations are ignored.
func helper() {
	// @query
	var ignored struct {
		// @field { @description Ignored }
		A string `query:"a"`
	}

	_ = ignored
}
