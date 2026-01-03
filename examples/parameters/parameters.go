//	@api {
//	  @title Parameters Example
//	  @version 1.0.0
//	  @description Demonstrates all parameter types: path, query, header, and cookie.
//	  @defaultContentType json
//	}
package parameters

import "net/http"

// -----------------------------------------------------------------------------
// Schemas
// -----------------------------------------------------------------------------

// Resource represents a generic resource
// @schema
type Resource struct {
	// @field { @description Resource ID @format uuid }
	ID string `json:"id"`

	// @field { @description Resource name }
	Name string `json:"name"`

	// @field { @description Resource type }
	Type string `json:"type"`
}

// -----------------------------------------------------------------------------
// Path Parameters
// -----------------------------------------------------------------------------

// ResourcePath demonstrates path parameters
// Path parameters are always required and support simple types only
// @path
type ResourcePath struct {
	// @field { @description Resource ID @format uuid }
	ID string `path:"id"`
}

// NestedPath demonstrates multiple path parameters
// @path
type NestedPath struct {
	// @field { @description Organization ID @format uuid }
	OrgID string `path:"orgId"`

	// @field { @description Project ID @format uuid }
	ProjectID string `path:"projectId"`

	// @field { @description Resource ID as integer }
	ResourceID int64 `path:"resourceId"`
}

// -----------------------------------------------------------------------------
// Query Parameters
// -----------------------------------------------------------------------------

// SearchQuery demonstrates various query parameter patterns
// @query
type SearchQuery struct {
	// @field { @description Search query string @minLength 1 @maxLength 100 }
	Q string `query:"q"`

	// @field { @description Maximum results @minimum 1 @maximum 100 @default 20 }
	Limit *int `query:"limit"`

	// @field { @description Page offset @minimum 0 @default 0 }
	Offset *int `query:"offset"`

	// @field { @description Sort order @enum asc,desc @default asc }
	Sort *string `query:"sort"`

	// @field { @description Filter by tags (can specify multiple) }
	Tags []string `query:"tags"`

	// @field { @description Include deleted items }
	IncludeDeleted *bool `query:"include_deleted"`
}

// FilterQuery demonstrates filtering query parameters
// @query
type FilterQuery struct {
	// @field { @description Filter by status @enum active,inactive,pending }
	Status *string `query:"status"`

	// @field { @description Filter by type @enum a,b,c }
	Type *string `query:"type"`

	// @field { @description Filter by creation date (ISO 8601) @format date }
	CreatedAfter *string `query:"created_after"`

	// @field { @description Filter by IDs (can specify multiple) @format uuid }
	IDs []string `query:"ids"`
}

// -----------------------------------------------------------------------------
// Header Parameters
// -----------------------------------------------------------------------------

// RequestHeaders demonstrates header parameters
// @header
type RequestHeaders struct {
	// @field { @description Unique request ID for tracing @format uuid }
	RequestID string `header:"X-Request-ID"`

	// @field { @description API version to use @default v1 }
	APIVersion *string `header:"X-API-Version"`

	// @field { @description Correlation ID for distributed tracing }
	CorrelationID *string `header:"X-Correlation-ID"`
}

// IdempotencyHeader demonstrates idempotency key header
// @header
type IdempotencyHeader struct {
	// @field { @description Idempotency key for safe retries @format uuid }
	IdempotencyKey string `header:"X-Idempotency-Key"`
}

// -----------------------------------------------------------------------------
// Cookie Parameters
// -----------------------------------------------------------------------------

// SessionCookie demonstrates cookie parameters
// @cookie
type SessionCookie struct {
	// @field { @description Session identifier @format uuid }
	SessionID string `cookie:"session_id"`
}

// PreferencesCookie demonstrates optional cookie parameters
// @cookie
type PreferencesCookie struct {
	// @field { @description User's preferred theme @enum light,dark,system }
	Theme *string `cookie:"theme"`

	// @field {
	//   @description User's preferred language
	//   @pattern ^[a-z]{2}(-[A-Z]{2})?$
	// }
	Language *string `cookie:"lang"`
}

// -----------------------------------------------------------------------------
// Endpoints
// -----------------------------------------------------------------------------

// GetResource demonstrates path and header parameters
//
//	@endpoint GET /resources/{id} {
//	  @operationID getResource
//	  @summary Get a resource by ID
//	  @description Retrieves a single resource using path and header parameters.
//	  @path ResourcePath
//	  @header RequestHeaders
//	  @response 200 {
//	    @body Resource
//	    @description Resource found
//	  }
//	}
func GetResource(w http.ResponseWriter, r *http.Request) {}

// SearchResources demonstrates query parameters
//
//	@endpoint GET /resources {
//	  @operationID searchResources
//	  @summary Search resources
//	  @description Search and filter resources with various query parameters.
//	  @query SearchQuery
//	  @query FilterQuery
//	  @response 200 {
//	    @body []Resource
//	    @description Search results
//	  }
//	}
func SearchResources(w http.ResponseWriter, r *http.Request) {}

// GetNestedResource demonstrates multiple path parameters
//
//	@endpoint GET /orgs/{orgId}/projects/{projectId}/resources/{resourceId} {
//	  @operationID getNestedResource
//	  @summary Get a nested resource
//	  @description Retrieves a resource using multiple path parameters.
//	  @path NestedPath
//	  @response 200 {
//	    @body Resource
//	    @description Resource found
//	  }
//	}
func GetNestedResource(w http.ResponseWriter, r *http.Request) {}

// CreateResource demonstrates header and cookie parameters together
//
//	@endpoint POST /resources {
//	  @operationID createResource
//	  @summary Create a resource
//	  @description Creates a resource with idempotency and session tracking.
//	  @header IdempotencyHeader
//	  @cookie SessionCookie
//	  @request {
//	    @body Resource
//	  }
//	  @response 201 {
//	    @body Resource
//	    @description Resource created
//	  }
//	}
func CreateResource(w http.ResponseWriter, r *http.Request) {}

// GetPreferences demonstrates cookie parameters
//
//	@endpoint GET /preferences {
//	  @operationID getPreferences
//	  @summary Get user preferences
//	  @description Retrieves preferences from cookies.
//	  @cookie PreferencesCookie
//	  @response 200 {
//	    @body Resource
//	    @description Preferences retrieved
//	  }
//	}
func GetPreferences(w http.ResponseWriter, r *http.Request) {}
