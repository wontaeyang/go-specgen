// Package schema_unreferenced declares a schema no endpoint body reaches. A
// schema's direction is inferred from the endpoints that reference it, so a
// schema nothing references has no direction — and would otherwise sit in the
// document silently, which is how a typo in a @body name hides.
//
//	@api {
//	  @title Unreferenced Schema Fixture
//	  @version 1.0.0
//	  @description A schema no request or response body references.
//	  @defaultContentType json
//	}
package schema_unreferenced

import "net/http"

// Widget is what the endpoint actually returns.
// @schema
type Widget struct {
	// @field { @description Widget name }
	Name string `json:"name"`
}

// Orphan is declared and never referenced.
// @schema
type Orphan struct {
	// @field { @description Orphan name }
	Name string `json:"name"`
}

// GetWidget references Widget, leaving Orphan unreachable.
//
//	@endpoint GET /widgets/{id} {
//	  @summary Get a widget
//	  @response 200 { @body Widget }
//	}
func GetWidget(w http.ResponseWriter, r *http.Request) {
	// @path
	var path struct {
		// @field { @description Widget ID }
		ID string `path:"id"`
	}
	_ = path
}
