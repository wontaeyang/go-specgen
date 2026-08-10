// Package unknown_schema references type names that were never declared as
// schemas, on the request body, the response body, and the @bind wrapper.
//
//	@api {
//	  @title Unknown Schema Fixture
//	  @version 1.0.0
//	  @description Bodies referencing types that are not schemas.
//	  @defaultContentType json
//	}
package unknown_schema

import "net/http"

// CreateWidget names a request body and a response body that do not exist.
//
//	@endpoint POST /widgets {
//	  @summary Create a widget
//	  @request { @body WidgetInput }
//	  @response 201 { @body Widget }
//	}
func CreateWidget(w http.ResponseWriter, r *http.Request) {}
