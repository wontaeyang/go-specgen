//	@api {
//	  @title EmptyComponents Fixture
//	  @version 1.0.0
//	}
package emptycomponents

// Health has no named schemas or security schemes, locking the
// `components: {}` emission for a package with inline-only bodies.
//
//	@endpoint GET /health {
//	}
func Health() {
	// @response 200
	var ok struct {
		// @field { @description Service status }
		Status string `json:"status"`
	}
	_ = ok
}
