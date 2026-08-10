// Package unknown_param_ref references parameter structs that were never
// declared: once from an endpoint's @query, and once from each of the two
// forms a response can be written in.
//
//	@api {
//	  @title Unknown Parameter Reference Fixture
//	  @version 1.0.0
//	  @description References to parameter structs that do not exist.
//	  @defaultContentType json
//	}
package unknown_param_ref

import "net/http"

// Widget is the body every response here carries.
//
//	@schema {
//	  @description A widget.
//	}
type Widget struct {
	// @field { @description Widget ID }
	ID string `json:"id"`
}

// WidgetPath carries the widget id.
// @path
type WidgetPath struct {
	// @field { @description Widget ID }
	ID string `path:"id"`
}

// GetWidget declares its response in the handler, naming a header struct that
// does not exist.
//
//	@endpoint GET /widgets/{id} {
//	  @summary Get a widget
//	  @path WidgetPath
//	}
func GetWidget(w http.ResponseWriter, r *http.Request) {
	// @response 200 {
	//   @header RateLimitHeaders
	//   @description The widget
	// }
	var resp struct {
		// @field { @description Widget ID }
		ID string `json:"id"`
	}

	_ = resp
}

// ListWidgets misspells its query struct and names a response header struct
// that does not exist.
//
//	@endpoint GET /widgets {
//	  @summary List widgets
//	  @query ListWidgetsQuery
//	  @response 200 {
//	    @description The widgets
//	    @body []Widget
//	    @header RateLimitHeaders
//	  }
//	}
func ListWidgets(w http.ResponseWriter, r *http.Request) {}
