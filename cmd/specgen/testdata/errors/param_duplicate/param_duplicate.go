// Package param_duplicate declares the same parameter twice in one location.
//
// This is the duplicate that matters: the emitted document would list one
// parameter twice, and a reader has no way to tell which description applies.
// It is also the case the old name-keyed rule could miss, because one of the
// two declarations is in-function and that rule only read the named ones.
//
//	@api {
//	  @title Duplicate Parameter Fixture
//	  @version 1.0.0
//	  @description The same name declared twice in the same location.
//	  @defaultContentType json
//	}
package param_duplicate

import "net/http"

// ListQuery names the page size.
// @query
type ListQuery struct {
	// @field { @description Page size }
	Limit *int `query:"limit"`
}

// ListItems declares limit again, in the handler body.
//
//	@endpoint GET /items {
//	  @summary List items
//	  @query ListQuery
//	  @response 200 { @body []string }
//	}
func ListItems(w http.ResponseWriter, r *http.Request) {
	// @query
	var extra struct {
		// @field { @description Page size, again }
		Limit *int `query:"limit"`
	}

	_ = extra
}
