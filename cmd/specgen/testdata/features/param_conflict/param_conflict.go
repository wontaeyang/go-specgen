// Package param_conflict declares the same parameter name in two locations
// (bug #6).
//
// OpenAPI keys parameter uniqueness on (name, in), so a path "id" alongside a
// query "id" is legal and unambiguous: they address different things and the
// document says which is which. specgen used to key on the name alone and
// reject this.
//
//	@api {
//	  @title Parameter Conflict Fixture
//	  @version 1.0.0
//	  @description The same parameter name in two different locations.
//	  @defaultContentType json
//	}
package param_conflict

import "net/http"

// ItemPath names the item being addressed.
// @path
type ItemPath struct {
	// @field { @description Item identifier }
	ID string `path:"id"`
}

// ItemQuery names the tenant the lookup runs against.
// @query
type ItemQuery struct {
	// @field { @description Tenant identifier }
	ID *string `query:"id"`
}

// GetItem addresses an item by path id, scoped by query id.
//
//	@endpoint GET /items/{id} {
//	  @summary Get an item
//	  @path ItemPath
//	  @query ItemQuery
//	  @response 200 { @body []string }
//	}
func GetItem(w http.ResponseWriter, r *http.Request) {}
