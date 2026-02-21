// Package standalone_api tests that @api can appear as a standalone comment.
package standalone_api

// @api {
//   @title Standalone API
//   @version 2.0.0
// }

// @schema
type Item struct {
	// @field {
	//   @description Item ID
	// }
	ID string `json:"id"`
}

// @endpoint GET /items/{id} {
//   @summary Get item by ID
//   @response 200 {
//     @contentType json
//     @body Item
//   }
// }
func GetItem() {}
