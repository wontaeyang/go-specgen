//	@api {
//	  @title Bad Schema
//	  @version 1.0.0
//	}
package badschema

// Handle aliases a non-struct type, so it cannot become a schema.
//
// @schema
type Handle = string
