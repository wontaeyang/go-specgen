// Package param_wrongkind covers a field tagged for a different parameter kind
// than the struct it lives in.
//
// A @query struct describes query parameters and nothing else. A field carrying
// only a path: tag is either a mistake or a struct doing double duty, and
// either way the emitted parameter would not match what the binder reads.
//
//	@api {
//	  @title Wrong Parameter Kind Fixture
//	  @version 1.0.0
//	  @description A path-tagged field inside a query parameter struct.
//	  @defaultContentType json
//	}
package param_wrongkind

import "net/http"

// ListQuery mixes a query field with a path-tagged one.
// @query
type ListQuery struct {
	// @field { @description Page size }
	Limit *int `query:"limit"`

	// @field { @description Tagged for the path, not the query }
	TenantID string `path:"tenant_id"`
}

// List reads the query struct.
//
//	@endpoint GET /items {
//	  @summary List items
//	  @query ListQuery
//	  @response 200 { @body []string }
//	}
func List(w http.ResponseWriter, r *http.Request) {}
