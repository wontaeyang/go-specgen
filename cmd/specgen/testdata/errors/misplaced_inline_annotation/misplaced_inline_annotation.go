// Package misplaced_inline_annotation puts @schema on a declaration inside a
// handler. @schema is a real annotation, so this fails rather than being
// reported and skipped — but it fails fast, the way every other in-function
// error does, rather than accumulating with the doc-comment ones.
//
//	@api {
//	  @title Misplaced Inline Annotation Fixture
//	  @version 1.0.0
//	  @description A doc-comment annotation on an in-function declaration.
//	  @defaultContentType json
//	}
package misplaced_inline_annotation

import "net/http"

// GetWidget declares its response body in the handler, under the annotation
// that would name a package-level schema.
//
//	@endpoint GET /widgets {
//	  @summary Get a widget
//	}
func GetWidget(w http.ResponseWriter, r *http.Request) {
	// @schema
	var resp struct {
		// @field { @description Widget ID }
		ID string `json:"id"`
	}

	_ = resp
}
