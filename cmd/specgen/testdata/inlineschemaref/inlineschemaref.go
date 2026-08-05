//	@api {
//	  @title InlineSchemaRef Fixture
//	  @version 1.0.0
//	}
package inlineschemaref

// Widget is a named schema referenced from an inline struct.
//
//	@schema {
//	  @description A widget
//	}
type Widget struct {
	// @field { @description Widget name }
	Name string `json:"name"`
}

// GetBox characterizes how a named-schema-typed field renders inside an
// in-function inline body (historically the ref-unaware builder had no
// schema map and could not emit a $ref).
//
//	@endpoint GET /box {
//	}
func GetBox() {
	// @response 200
	var box struct {
		// @field { @description The boxed widget }
		Widget Widget `json:"widget"`
	}
	_ = box
}
