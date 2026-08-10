// Package inline_not_a_struct puts an in-function @query on a declaration that
// names a struct defined elsewhere instead of defining one.
//
//	@api {
//	  @title Inline Not A Struct Fixture
//	  @version 1.0.0
//	  @description An in-function annotation on a declaration that defines nothing.
//	  @defaultContentType json
//	}
package inline_not_a_struct

import "net/http"

// Filters is where the @query belongs.
// @query
type Filters struct {
	// @field { @description Result limit }
	Limit int `query:"limit"`
}

// ListWidgets annotates a variable of an already-declared type, so there is no
// struct definition for the @query to describe.
//
//	@endpoint GET /widgets {
//	  @summary List widgets
//	  @response 200 { @body Filters }
//	}
func ListWidgets(w http.ResponseWriter, r *http.Request) {
	// @query
	var f Filters

	_ = f
}
