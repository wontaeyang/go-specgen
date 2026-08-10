//	@api {
//	  @title Duplicate Operation ID
//	  @version 1.0.0
//	}
package duplicate_operationid

// operationId must be unique across the document: client generators name the
// method they emit after it, so two operations sharing one produce a client
// that cannot compile.
//
// The routes differ here. Only the @operationID collides.

//	@endpoint GET /widgets {
//	  @operationID listWidgets
//	  @response 200 { @description ok }
//	}
func ListWidgets() {}

//	@endpoint GET /gadgets {
//	  @operationID listWidgets
//	  @response 200 { @description ok }
//	}
func ListGadgets() {}

// An endpoint with no @operationID is unnamed rather than named "", so these two
// are not a collision with each other.

//	@endpoint GET /unnamed-one {
//	  @response 200 { @description ok }
//	}
func UnnamedOne() {}

//	@endpoint GET /unnamed-two {
//	  @response 200 { @description ok }
//	}
func UnnamedTwo() {}
