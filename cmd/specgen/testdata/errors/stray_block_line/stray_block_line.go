// Package stray_block_line writes prose inside an annotation block, where
// every line has to be an annotation.
//
//	@api {
//	  @title Stray Block Line Fixture
//	  @version 1.0.0
//	  this line is prose, not an annotation
//	  @defaultContentType json
//	}
package stray_block_line

import "net/http"

// Widget is the response body.
//
//	@schema {
//	  @description A widget.
//	}
type Widget struct {
	// @field { @description Widget ID }
	ID string `json:"id"`
}

// GetWidget is here so the document has an operation.
//
//	@endpoint GET /widgets {
//	  @summary Get a widget
//	  @response 200 { @description The widget @body Widget }
//	}
func GetWidget(w http.ResponseWriter, r *http.Request) {}
