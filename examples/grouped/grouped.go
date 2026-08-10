// Package grouped covers grouped field declarations (bug #5).
//
// One *ast.Field with several Names expands to several *types.Var, so an
// annotation written above "X, Y float64" has to apply to both. Today only the
// first name is annotated.
//
//	@api {
//	  @title Grouped Fields Fixture
//	  @version 1.0.0
//	  @description Field declarations that name more than one field.
//	  @defaultContentType json
//	}
package grouped

import "net/http"

// Point declares its coordinates in one statement.
// @schema
type Point struct {
	// @field { @description Coordinate in degrees @minimum -180 @maximum 180 }
	X, Y float64

	// @field { @description Elevation in meters }
	Z float64

	// @field { @description Axis label }
	LabelX, LabelY string `json:"-"`
}

// GetPoint returns a point.
//
//	@endpoint GET /points {
//	  @summary Get a point
//	  @response 200 { @body Point }
//	}
func GetPoint(w http.ResponseWriter, r *http.Request) {}
