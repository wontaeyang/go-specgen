// @api {
//   @title Test API
//   @version 1.0.0
// }
package testdata

// @schema
type User struct {
	// @field {
	//   @description User ID
	//   @format uuid
	// }
	ID string `json:"id"`

	// @field {
	//   @description User email
	//   @format email
	// }
	Email string `json:"email"`

	Name string `json:"name"` // No annotation
}

// @path
type UserIDPath struct {
	// @field
	ID string `path:"id"`
}

// @query
type SearchQuery struct {
	// @field
	Query string `query:"q"`

	// @field
	Limit *int `query:"limit,omitempty"`
}

// @endpoint GET /users/{id} {
//   @summary Get user by ID
//   @path UserIDPath
//   @response 200 {
//     @contentType json
//     @body User
//   }
// }
func GetUser() {}

// @endpoint POST /users {
//   @summary Create user
//   @request {
//     @contentType json
//     @body User
//   }
//   @response 201 {
//     @contentType json
//     @body User
//   }
// }
func CreateUser() {}

// @schema
type FieldRequiredTest struct {
	Value         string  `json:"value"`
	ValuePtr      *string `json:"value_ptr"`
	ValueOmit     string  `json:"value_omit,omitempty"`
	ValuePtrOmit  *string `json:"value_ptr_omit,omitempty"`
}

// No annotation - should be ignored
func HelperFunction() {}

// Regular struct without annotation
type InternalStruct struct {
	Field string
}
