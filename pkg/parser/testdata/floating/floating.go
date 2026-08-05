//	@api {
//	  @title Floating Annotation Fixture
//	  @version 1.0.0
//	}
package floating

// Handler writes a @field annotation attached to no declaration, where it can
// describe nothing.
//
//	@endpoint GET /things {
//	  @response 200 { @body Thing }
//	}
func Handler() {
	// @field { @description Describes nothing }

	_ = 1
}

// Thing is the response body.
// @schema
type Thing struct {
	// @field { @description Identifier }
	ID string `json:"id"`
}
