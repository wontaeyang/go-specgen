//	@api {
//	  @title AnyConstraints Fixture
//	  @version 1.0.0
//	}
package anyconstraints

// GetData characterizes how an any-typed field with annotations renders
// inside an in-function inline body (historically the ref-unaware builder
// emitted the description only and dropped other annotations).
//
//	@endpoint GET /data {
//	}
func GetData() {
	// @response 200
	var resp struct {
		// @field { @description Arbitrary payload @example 42 }
		Data any `json:"data"`

		// @field { @description Typed sibling for contrast @minLength 1 }
		Label string `json:"label"`
	}
	_ = resp
}
