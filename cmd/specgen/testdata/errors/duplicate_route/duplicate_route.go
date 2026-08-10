//	@api {
//	  @title Duplicate Route
//	  @version 1.0.0
//	}
package duplicate_route

// A document holds one operation per method per path. Declaring the route twice
// does not merge the two handlers, it drops one of them, so it has to be an
// error rather than a silent choice between them.

//	@endpoint GET /widgets {
//	  @summary First handler
//	  @response 200 { @description ok }
//	}
func ListWidgets() {}

//	@endpoint GET /widgets {
//	  @summary Second handler on the same route
//	  @response 200 { @description ok }
//	}
func ListWidgetsAgain() {}
