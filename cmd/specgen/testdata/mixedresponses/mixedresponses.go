//	@api {
//	  @title MixedResponses Fixture
//	  @version 1.0.0
//	}
package mixedresponses

// Thing is a named response body.
//
//	@schema {
//	  @description A thing
//	}
type Thing struct {
	// @field { @description Thing name }
	Name string `json:"name"`
}

// Err is a named error body.
//
//	@schema {
//	  @description An error
//	}
type Err struct {
	// @field { @description Error code }
	Code string `json:"code"`
}

// ListThings mixes block @response declarations with in-function inline
// ones on a single endpoint. This locks the golden-uncovered merge:
// sorted named statuses first (200, 500), then sorted inline (404), and
// a named/inline conflict on 200 where the named declaration wins.
//
//	@endpoint GET /things {
//	  @response 200 { @body Thing @description OK from block }
//	  @response 500 { @body Err @description Server error }
//	}
func ListThings() {
	// @response 404
	var notFound struct {
		// @field { @description Not-found code }
		Code string `json:"code"`
	}

	// @response 200
	var conflicting struct {
		// @field { @description Inline duplicate that must lose to the named 200 }
		X string `json:"x"`
	}

	_ = notFound
	_ = conflicting
}
