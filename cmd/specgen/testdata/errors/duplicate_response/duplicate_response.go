//	@api {
//	  @title Duplicate Response
//	  @version 1.0.0
//	}
package duplicate_response

// @response repeats to describe several status codes, not to describe one twice.
// A second block for the same code has nothing to merge with the first, so it
// replaced it and the operation lost a response. The in-function form has always
// rejected this; the doc-comment form used to take it in silence.

// @schema
type Widget struct {
	//	@field { @description Widget ID }
	ID string `json:"id"`
}

// @schema
type Gadget struct {
	//	@field { @description Gadget ID }
	ID string `json:"id"`
}

//	@endpoint GET /widgets {
//	  @response 200 { @description first @body Widget }
//	  @response 200 { @description second @body Gadget }
//	}
func ListWidgets() {}
