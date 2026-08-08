// Package resolve_multiple has two parameter structs that cannot resolve.
//
// The resolver accumulates the same way the parser does, down to the field.
// param_wrongkind covers what the message says; this covers that the second
// struct is reported at all, and that a second bad field within one struct is
// too -- ZebraQuery mis-tags both of its fields and both are named.
//
// The structs are declared in reverse alphabetical order, so the expected file
// shows the resolver walking its map by name rather than in whatever order the
// map offers. Fields inside a struct keep Go declaration order, since that is
// the order the resolver walks the struct type. Both are the reason this
// fixture is stable enough to compare byte-for-byte.
//
//	@api {
//	  @title Resolve Multiple Fixture
//	  @version 1.0.0
//	  @description Two parameter structs tagged for the wrong kind.
//	  @defaultContentType json
//	}
package resolve_multiple

import "net/http"

// @schema
type Thing struct {
	// @field { @description Thing ID }
	ID string `json:"id"`
}

// ZebraQuery tags both of its fields for locations they do not sit in.
// @query
type ZebraQuery struct {
	// @field { @description Stripe count }
	Stripes int `header:"stripes"`

	// @field { @description Zone name }
	Zone string `cookie:"zone"`
}

// AlphaQuery does the same with a different location.
// @query
type AlphaQuery struct {
	// @field { @description Tenant ID }
	TenantID string `path:"tenant_id"`
}

// ListThings reads one of the queries.
//
//	@endpoint GET /things {
//	  @summary List things
//	  @query AlphaQuery
//	  @response 200 { @body []Thing }
//	}
func ListThings(w http.ResponseWriter, r *http.Request) {}
