// Package schema_both_directions uses one schema as both a request body and a
// response body. A schema's direction is inferred from the endpoints that
// reference it, and each direction carries its own required/nullable rule, so
// a schema cannot serve both sides.
//
// User is reached directly by both bodies; Author only through User's field,
// so the two errors pin both chain renderings: the direct seed and the
// transitive field hop.
//
//	@api {
//	  @title Both Directions Fixture
//	  @version 1.0.0
//	  @description One schema used as request and response body.
//	  @defaultContentType json
//	}
package schema_both_directions

import "net/http"

// User is both the create payload and the response shape.
// @schema
type User struct {
	// @field { @description User name }
	Name string `json:"name"`

	// @field { @description Who created this user }
	Author Author `json:"author"`
}

// Author is reachable only through User, in both directions.
// @schema
type Author struct {
	// @field { @description Author name }
	Name string `json:"name"`
}

// CreateUser sends User in.
//
//	@endpoint POST /users {
//	  @summary Create a user
//	  @request { @body User }
//	  @response 204 { @description Created }
//	}
func CreateUser(w http.ResponseWriter, r *http.Request) {}

// GetUser sends User back out.
//
//	@endpoint GET /users/{id} {
//	  @summary Get a user
//	  @response 200 { @body User }
//	}
func GetUser(w http.ResponseWriter, r *http.Request) {
	// @path
	var path struct {
		// @field { @description User ID }
		ID string `path:"id"`
	}
	_ = path
}
