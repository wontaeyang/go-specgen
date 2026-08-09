// Package misplaced_annotation writes real annotations on declarations that
// cannot carry them. Each name exists in the grammar somewhere, so each is a
// mistake with one correct reading rather than a comment specgen has no opinion
// about — which is why these fail rather than being reported and skipped.
//
//	@api {
//	  @title Misplaced Annotation Fixture
//	  @version 1.0.0
//	  @description Real annotations on the wrong declarations.
//	  @defaultContentType json
//	}
package misplaced_annotation

import "net/http"

// @summary belongs inside @endpoint, not on a type.
//
// @summary A widget
type Widget struct {
	// @description belongs inside @field, not on a field directly.
	//
	// @description Widget ID
	ID string `json:"id"`
}

// @schema belongs on a type, not on a func.
//
// @schema
func ListWidgets(w http.ResponseWriter, r *http.Request) {}

// GetWidget is here so the document has an operation.
//
//	@endpoint GET /widgets {
//	  @summary Get a widget
//	  @response 200 { @description The widget @body Widget }
//	}
func GetWidget(w http.ResponseWriter, r *http.Request) {}
