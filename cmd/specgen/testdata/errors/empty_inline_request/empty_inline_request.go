//	@api {
//	  @title Empty Inline Request
//	  @version 1.0.0
//	}
package empty_inline_request

// An in-function @request is its own body, so it cannot be missing one the way a
// named @request can. What it can be is empty, and a request body carrying no
// fields describes nothing.
//
// Unlike a response, where carrying nothing is how 204 is spelled, so an empty
// in-function @response stays legal. This case used to reach the generator and
// render as "requestBody: {}" -- a Request Body Object with no content, which
// the spec does not allow -- and exit 0.

// @schema
type Widget struct {
	//	@field { @description Widget ID }
	ID string `json:"id"`
}

//	@endpoint POST /widgets {
//	  @summary Create a widget
//	  @response 200 { @description ok @body Widget }
//	}
func CreateWidget() {
	//	@request { @contentType json }
	var body struct{}
	_ = body
}
