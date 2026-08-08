// Package param_struct_field covers struct-typed parameter fields (bug #2).
//
// net/http exposes parameters as map[string][]string, so there is no
// standard-library model for a structured parameter and deepObject is out of
// scope. Today these silently render as "type: string" whether or not the
// struct is a @schema; they are meant to be an error.
//
// This fixture moves to testdata/errors when that lands.
//
//	@api {
//	  @title Struct Parameter Field Fixture
//	  @version 1.0.0
//	  @description Parameter fields whose Go type is a struct.
//	  @defaultContentType json
//	}
package param_struct_field

import "net/http"

// Coords is a @schema, and is still not representable as a parameter.
// @schema
type Coords struct {
	// @field { @description Latitude }
	Lat float64 `json:"lat"`

	// @field { @description Longitude }
	Lon float64 `json:"lon"`
}

// DateRange is not a @schema, and is equally unrepresentable.
type DateRange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// AreaQuery holds two struct-typed fields.
// @query
type AreaQuery struct {
	// @field { @description Search origin }
	Origin Coords `query:"origin"`

	// @field { @description Time window }
	Window DateRange `query:"window"`

	// @field { @description Search radius in meters @minimum 0 }
	Radius *int `query:"radius"`
}

// SearchArea reads the area query.
//
//	@endpoint GET /areas {
//	  @summary Search an area
//	  @query AreaQuery
//	  @response 200 { @body []Coords }
//	}
func SearchArea(w http.ResponseWriter, r *http.Request) {}
