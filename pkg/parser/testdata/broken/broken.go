//	@api {
//	  @title Broken Fixture
//	  @version 1.0.0
//	}
package broken

// Good parses cleanly, so it proves the broken items below do not stop it.
// @schema
type Good struct {
	// @field { @description Fine }
	Name string `json:"name"`
}

// BadNumber has a non-numeric @minimum.
// @schema
type BadNumber struct {
	// @field { @minimum abc }
	Count int `json:"count"`
}

// BadBool has a non-boolean @required.
// @schema
type BadBool struct {
	// @field { @required maybe }
	Name string `json:"name"`
}

// BadEndpoint has a misspelled child annotation.
//
//	@endpoint GET /things {
//	  @summry Typo
//	}
func BadEndpoint() {}

// DuplicateRequest declares two inline request bodies.
//
//	@endpoint POST /things {
//	}
func DuplicateRequest() {
	// @request
	var first struct {
		// @field { @description First }
		A string `json:"a"`
	}

	// @request
	var second struct {
		// @field { @description Second }
		B string `json:"b"`
	}

	_, _ = first, second
}

// BodylessStandalone writes an @response attached to nothing and names no
// body, so it describes nothing at all.
//
//	@endpoint GET /bodyless {
//	}
func BodylessStandalone() {
	// @response 400

	_ = 1
}

// TwoBodies declares a struct that is the response body and names a second
// one with @body.
//
//	@endpoint GET /twobodies {
//	}
func TwoBodies() {
	// @response 200 { @body Good }
	var resp struct {
		// @field { @description Fine }
		Name string `json:"name"`
	}

	_ = resp
}
