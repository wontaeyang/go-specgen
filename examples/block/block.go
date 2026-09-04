//	@api {
//	  @title Block Syntax Example
//	  @version 1.0.0
//	  @description Demonstrates multi-line vs inline block syntax.
//	  All block annotations support both formats. The key rule is that
//	  nested blocks cannot appear within inline format - use multi-line instead.
//	  @defaultContentType json
//	}
package block

// -----------------------------------------------------------------------------
// Multi-line Block Syntax
// -----------------------------------------------------------------------------

// Error shows multi-line block format - use when you have many attributes
// or want better readability.
//
//	@schema {
//	  @description Standard error response for all endpoints.
//	  This format allows multi-line descriptions and is easier to read
//	  when you have many child annotations.
//	}
type Error struct {
	// Multi-line field block
	// @field {
	//   @description Error code identifying the type of error.
	//   See error codes documentation for possible values.
	//   @example INVALID_INPUT
	// }
	Code string `json:"code"`

	// @field {
	//   @description Human-readable error message
	//   @example The provided email address is invalid
	// }
	Message string `json:"message"`
}

// -----------------------------------------------------------------------------
// Inline Block Syntax
// -----------------------------------------------------------------------------

// User shows inline block format - use for simple, few attributes.
// Inline is compact but doesn't support nested blocks.
//
// @schema { @description User account information }
type User struct {
	// Inline field blocks - compact for simple annotations
	// @field { @description User's unique identifier @format uuid }
	ID string `json:"id"`

	// @field { @description User's email address @format email }
	Email string `json:"email"`

	// @field { @description User's display name @minLength 1 @maxLength 100 }
	Name string `json:"name"`

	// @field { @description Account creation timestamp @format date-time }
	CreatedAt string `json:"created_at"`

	// Empty block - field with no constraints
	// @field {}
	Bio string `json:"bio,omitempty"`
}

// -----------------------------------------------------------------------------
// Parameter Types
// -----------------------------------------------------------------------------

// UserPath shows inline path parameters
// @path
type UserPath struct {
	// @field { @description User ID @format uuid }
	ID string `path:"id"`
}

// UserQuery shows inline query parameters
// @query
type UserQuery struct {
	// @field { @description Fields to include (comma-separated: id,email,name,bio) }
	Fields []string `query:"fields"`

	// @field { @description Include deleted users @default false }
	IncludeDeleted *bool `query:"include_deleted"`
}

// -----------------------------------------------------------------------------
// Endpoint Examples
// -----------------------------------------------------------------------------

// GetUser demonstrates inline response blocks within multi-line endpoint.
//
//	@endpoint GET /users/{id} {
//	  @summary Get user by ID
//	  @description Retrieves a single user by their unique identifier.
//	  @path UserPath
//	  @response 200 { @body User @description User found successfully }
//	  @response 404 { @body Error @description User not found }
//	}
func GetUser() {}

// ListUsers demonstrates a multi-line response block with longer description.
//
//	@endpoint GET /users {
//	  @summary List all users
//	  @description Retrieves a paginated list of users.
//	  @query UserQuery
//	  @response 200 {
//	    @body []User
//	    @description Successfully retrieved list of users.
//	    Pagination info is included in response headers.
//	  }
//	}
func ListUsers() {}

// CreateUser demonstrates request and multiple response blocks.
//
//	@endpoint POST /users {
//	  @summary Create a new user
//	  @description Creates a new user account.
//	  @request {
//	    @body User
//	  }
//	  @response 201 {
//	    @body User
//	    @description User created successfully
//	  }
//	  @response 400 {
//	    @body Error
//	    @description Invalid input data
//	  }
//	  @response 409 {
//	    @body Error
//	    @description User with this email already exists
//	  }
//	}
func CreateUser() {}

// UpdateUser shows multi-line blocks throughout.
//
//	@endpoint PUT /users/{id} {
//	  @summary Update user
//	  @path UserPath
//	  @request {
//	    @body User
//	  }
//	  @response 200 {
//	    @body User
//	    @description User updated
//	  }
//	  @response 400 {
//	    @body Error
//	    @description Invalid input - check the error code for details.
//	    Common errors: INVALID_EMAIL, NAME_TOO_LONG
//	  }
//	  @response 404 {
//	    @body Error
//	    @description User not found
//	  }
//	}
func UpdateUser() {}

// DeleteUser shows empty response (204 No Content).
//
//	@endpoint DELETE /users/{id} {
//	  @summary Delete user
//	  @description Permanently deletes a user account.
//	  @path UserPath
//	  @response 204 {
//	    @contentType empty
//	    @description User deleted
//	  }
//	  @response 404 {
//	    @body Error
//	    @description User not found
//	  }
//	}
func DeleteUser() {}
