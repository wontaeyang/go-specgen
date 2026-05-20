// Package standalone_api demonstrates that the @api annotation may appear as
// a free-floating comment block in the file, not just as the package doc comment
// attached directly to the `package` clause.
package standalone_api

// @api {
//   @title Standalone API
//   @version 1.0.0
//   @description Demonstrates \@api declared as a standalone comment block.
//   @defaultContentType json
// }

// Item is a minimal schema exercised by the example.
//
// @schema
type Item struct {
	// @field { @description Item ID @format uuid }
	ID string `json:"id"`

	// @field { @description Human-readable name }
	Name string `json:"name"`
}

// ItemPath carries the path parameter for /items/{id}.
//
// @path
type ItemPath struct {
	// @field { @description Item ID @format uuid }
	ID string `path:"id"`
}

// GetItem returns an item by id.
//
//	@endpoint GET /items/{id} {
//	  @summary Get item by ID
//	  @path ItemPath
//	  @response 200 {
//	    @body Item
//	    @description The item
//	  }
//	}
func GetItem() {}
