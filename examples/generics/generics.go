//	@api {
//	  @title Generics Example
//	  @version 1.0.0
//	  @description Demonstrates Go generics for type-safe response wrappers.
//	  Generic templates are NOT emitted to components/schemas.
//	  Type aliases that instantiate generics ARE emitted.
//	  @defaultContentType json
//	}
package generics

import "net/http"

// -----------------------------------------------------------------------------
// Generic Response Wrappers (Templates - NOT emitted)
// -----------------------------------------------------------------------------

// Response is a generic wrapper template.
// This is NOT emitted to components/schemas because it's a template.
// @schema
type Response[T any] struct {
	// @field { @description Whether the request was successful }
	Success bool `json:"success"`

	// @field { @description Response data }
	Data T `json:"data"`

	// @field { @description Error message if unsuccessful }
	Error *string `json:"error,omitempty"`
}

// ListResponse is a generic wrapper for list responses.
// This is NOT emitted to components/schemas because it's a template.
// @schema
type ListResponse[T any] struct {
	// @field { @description Whether the request was successful }
	Success bool `json:"success"`

	// @field { @description List of items }
	Items []T `json:"items"`

	// @field { @description Total count of items }
	Total int `json:"total"`

	// @field { @description Error message if unsuccessful }
	Error *string `json:"error,omitempty"`
}

// -----------------------------------------------------------------------------
// Data Schemas
// -----------------------------------------------------------------------------

// User represents a user in the system
// @schema
type User struct {
	// @field { @description User ID @format uuid }
	ID string `json:"id"`

	// @field { @description User email @format email }
	Email string `json:"email"`

	// @field { @description User display name }
	Name string `json:"name"`
}

// Post represents a blog post
// @schema
type Post struct {
	// @field { @description Post ID @format uuid }
	ID string `json:"id"`

	// @field { @description Post title }
	Title string `json:"title"`

	// @field { @description Post content }
	Content string `json:"content"`

	// @field { @description Post author }
	Author User `json:"author"`
}

// -----------------------------------------------------------------------------
// Type Aliases (Instantiated Generics - ARE emitted)
// -----------------------------------------------------------------------------

// UserResponse is Response[User] - emitted to components/schemas
type UserResponse = Response[User]

// UserListResponse is ListResponse[User] - emitted to components/schemas
type UserListResponse = ListResponse[User]

// PostResponse is Response[Post] - emitted to components/schemas
type PostResponse = Response[Post]

// PostListResponse is ListResponse[Post] - emitted to components/schemas
type PostListResponse = ListResponse[Post]

// -----------------------------------------------------------------------------
// Path Parameters
// -----------------------------------------------------------------------------

// UserIDPath for user endpoints
// @path
type UserIDPath struct {
	// @field { @description User ID @format uuid }
	ID string `path:"id"`
}

// PostIDPath for post endpoints
// @path
type PostIDPath struct {
	// @field { @description Post ID @format uuid }
	ID string `path:"id"`
}

// -----------------------------------------------------------------------------
// Endpoints
// -----------------------------------------------------------------------------

// GetUser returns a user wrapped in the generic response
//
//	@endpoint GET /users/{id} {
//	  @operationID getUser
//	  @summary Get a user
//	  @description Returns a user wrapped in the type-safe Response[User] wrapper.
//	  @path UserIDPath
//	  @response 200 {
//	    @body UserResponse
//	    @description User found
//	  }
//	}
func GetUser(w http.ResponseWriter, r *http.Request) {}

// ListUsers returns users wrapped in the generic list response
//
//	@endpoint GET /users {
//	  @operationID listUsers
//	  @summary List users
//	  @description Returns users wrapped in the type-safe ListResponse[User] wrapper.
//	  @response 200 {
//	    @body UserListResponse
//	    @description List of users
//	  }
//	}
func ListUsers(w http.ResponseWriter, r *http.Request) {}

// GetPost returns a post wrapped in the generic response
//
//	@endpoint GET /posts/{id} {
//	  @operationID getPost
//	  @summary Get a post
//	  @description Returns a post wrapped in the type-safe Response[Post] wrapper.
//	  @path PostIDPath
//	  @response 200 {
//	    @body PostResponse
//	    @description Post found
//	  }
//	}
func GetPost(w http.ResponseWriter, r *http.Request) {}

// ListPosts returns posts wrapped in the generic list response
//
//	@endpoint GET /posts {
//	  @operationID listPosts
//	  @summary List posts
//	  @description Returns posts wrapped in the type-safe ListResponse[Post] wrapper.
//	  @response 200 {
//	    @body PostListResponse
//	    @description List of posts
//	  }
//	}
func ListPosts(w http.ResponseWriter, r *http.Request) {}
