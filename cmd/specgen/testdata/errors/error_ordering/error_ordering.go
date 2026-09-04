// Package error_ordering pins the order the validator reports errors in when
// they come from more than one schema or parameter struct.
//
// Every other fixture keeps its errors inside a single declaration, so none of
// them can see this: resolver.Package.Schemas and .Parameters are maps, and
// ranging over a map directly made the accumulated list come out in a different
// order on nearly every run. Nothing failed, because no fixture spanned two.
// The first one written would have flaked, and the diff — the same messages,
// permuted — would have read as a regression in the message text rather than
// as an ordering problem.
//
// The maps stay maps; five call sites look types up by name. The validator
// sorts at the two loops that accumulate errors.
//
// Declarations here run backwards through the alphabet on purpose. Sorted by
// name, the errors come out Alpha, Mid, Zeta and then AreaQuery, ZoneQuery —
// the reverse of the order they are written in, and unrelated to either the
// declaration order or whatever order the map happens to hand out. Renaming a
// type here reorders expected_error.txt.
//
//	@api {
//	  @title Error Ordering Fixture
//	  @version 1.0.0
//	  @description One validation failure in each of several declarations.
//	  @defaultContentType json
//	}
package error_ordering

import "net/http"

// Zeta reports readOnly and writeOnly at once.
// @schema
type Zeta struct {
	// @field { @description Readable and writable at once @readOnly @writeOnly }
	Token string `json:"token"`
}

// Mid reports uniqueness on a non-array.
// @schema
type Mid struct {
	// @field { @description Uniqueness on a non-array @uniqueItems }
	Label string `json:"label"`
}

// Alpha reports inverted numeric bounds.
// @schema
type Alpha struct {
	// @field { @description Inverted numeric bounds @minimum 100 @maximum 10 }
	Count int `json:"count"`
}

// ZoneQuery reports a struct-typed parameter field.
// @query
type ZoneQuery struct {
	// @field { @description Zone corner }
	Corner Alpha `query:"corner"`
}

// AreaQuery reports a struct-typed parameter field.
// @query
type AreaQuery struct {
	// @field { @description Search origin }
	Origin Alpha `query:"origin"`
}

// SearchZone reads the zone query.
//
//	@endpoint GET /zones {
//	  @summary Search a zone
//	  @query ZoneQuery
//	  @response 200 { @body Zeta }
//	}
func SearchZone(w http.ResponseWriter, r *http.Request) {}

// SearchArea reads the area query.
//
//	@endpoint GET /areas {
//	  @summary Search an area
//	  @query AreaQuery
//	  @response 200 { @body Mid }
//	}
func SearchArea(w http.ResponseWriter, r *http.Request) {}
