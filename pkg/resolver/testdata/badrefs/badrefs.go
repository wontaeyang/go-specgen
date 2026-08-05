//	@api {
//	  @title Bad References
//	  @version 1.0.0
//	}
package badrefs

// Thing is the response body.
//
// @schema
type Thing struct {
	ID string `json:"id"`
}

// ListThings references a parameter struct that was never declared.
//
//	@endpoint GET /things {
//	  @query Ghost
//	  @response 200 { @body Thing }
//	}
func ListThings() {}

// GetThing declares a response whose headers reference a struct that was
// never declared.
//
//	@endpoint GET /things/latest {
//	  @response 200 {
//	    @body Thing
//	    @header GhostHeaders
//	  }
//	}
func GetThing() {}
