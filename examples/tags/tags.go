// Package tags demonstrates struct tag field name resolution.
//
// Field names are resolved using this fallback chain:
//
//  1. json tag
//
//  2. xml tag
//
//  3. Go field name
//
//     @api {
//     @title Tag Resolution Example
//     @version 1.0.0
//     @description Demonstrates how field names are resolved from struct tags.
//     For different field names per content type, use separate structs.
//     }
package tags

// User demonstrates the tag fallback chain for field name resolution.
//
// @schema
type User struct {
	// Has json tag - uses "user_id"
	// @field { @description User identifier }
	ID string `json:"user_id"`

	// Has only xml tag - falls back to "UserName"
	// @field { @description User display name }
	Name string `xml:"UserName"`

	// Has both tags - json takes priority, uses "email"
	// @field { @description User email address }
	Email string `json:"email" xml:"EmailAddress"`

	// No tags - falls back to Go field name "Age"
	// @field { @description User age }
	Age int
}

// UserEdgeCases demonstrates edge cases in tag resolution.
//
// @schema
type UserEdgeCases struct {
	// json:"-" skips field entirely (not in schema)
	// This field will NOT appear in OpenAPI output
	SkippedField string `json:"-" xml:"WontBeUsed"`

	// json:"" (empty) falls back to xml tag
	// @field { @description Falls back to xml tag }
	EmptyJSON string `json:"" xml:"from_xml"`

	// json:",omitempty" (empty name) falls back to xml tag
	// @field { @description Falls back to xml tag }
	OmitEmptyJSON string `json:",omitempty" xml:"from_xml_omit"`

	// json:",omitempty" with no xml falls back to Go field name
	// @field { @description Uses Go field name }
	NoXMLFallback string `json:",omitempty"`

	// xml:"-" with no json skips field entirely
	// This field will NOT appear in OpenAPI output
	XMLSkipped string `xml:"-"`

	// Private (unexported) fields are not included in schema
	// This field will NOT appear in OpenAPI output
	privateField string
}

// CreateUserRequest shows recommended pattern: use json tags consistently.
//
// @schema
type CreateUserRequest struct {
	// @field { @description User name @minLength 1 }
	Name string `json:"name"`

	// @field { @description User email @format email }
	Email string `json:"email"`
}

// XMLUser shows recommended pattern for XML-specific structure.
// Use separate structs when you need different field names per content type.
//
// @schema
type XMLUser struct {
	// @field { @description User identifier }
	ID string `xml:"UserID"`

	// @field { @description User display name }
	Name string `xml:"UserName"`

	// @field { @description User email address }
	Email string `xml:"EmailAddress"`
}

// UserIDPath defines path parameter for user endpoints.
//
// @path
type UserIDPath struct {
	// @field { @description User ID }
	ID string `path:"id"`
}

// GetUser returns a user - uses User schema with json field names
//
//	@endpoint GET /users/{id} {
//	  @summary Get user by ID
//	  @path UserIDPath
//	  @response 200 {
//	    @contentType json
//	    @body User
//	    @description User found
//	  }
//	}
func GetUser() {}

// GetUserXML returns a user in XML format - uses XMLUser schema
//
//	@endpoint GET /users/{id}/xml {
//	  @summary Get user by ID (XML)
//	  @path UserIDPath
//	  @response 200 {
//	    @contentType xml
//	    @body XMLUser
//	    @description User found
//	  }
//	}
func GetUserXML() {}

// CreateUser creates a new user
//
//	@endpoint POST /users {
//	  @summary Create a user
//	  @request {
//	    @contentType json
//	    @body CreateUserRequest
//	  }
//	  @response 201 {
//	    @contentType json
//	    @body User
//	    @description User created
//	  }
//	}
func CreateUser() {}
