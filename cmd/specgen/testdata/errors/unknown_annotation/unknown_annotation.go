// Package unknown_annotation uses an annotation the grammar does not define.
//
// This is a parser-level failure rather than a validator one, so it also pins
// the fact that the parser still fails fast where the validator accumulates.
//
//	@api {
//	  @title Unknown Annotation Fixture
//	  @version 1.0.0
//	  @description An annotation that is not in the grammar.
//	  @defaultContentType json
//	}
package unknown_annotation

import "net/http"

// Widget carries a field annotation that does not exist.
// @schema
type Widget struct {
	// @field { @description Widget identifier @multipleOf 5 }
	ID string `json:"id"`
}

// GetWidget returns a widget.
//
//	@endpoint GET /widgets {
//	  @summary Get a widget
//	  @response 200 { @body Widget }
//	}
func GetWidget(w http.ResponseWriter, r *http.Request) {}
