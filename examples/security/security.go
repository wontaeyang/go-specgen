//	@api {
//	  @title Security Example
//	  @version 1.0.0
//	  @description Demonstrates various security scheme configurations.
//	  This example shows Bearer JWT, Basic auth, and API key authentication.
//	  @securityScheme bearerAuth {
//	    @type http
//	    @scheme bearer
//	    @bearerFormat JWT
//	    @description JWT Bearer token authentication
//	  }
//	  @securityScheme basicAuth {
//	    @type http
//	    @scheme basic
//	    @description HTTP Basic authentication
//	  }
//	  @securityScheme apiKeyHeader {
//	    @type apiKey
//	    @in header
//	    @name X-API-Key
//	    @description API key passed in header
//	  }
//	  @securityScheme apiKeyQuery {
//	    @type apiKey
//	    @in query
//	    @name api_key
//	    @description API key passed in query string
//	  }
//	  @securityScheme apiKeyCookie {
//	    @type apiKey
//	    @in cookie
//	    @name token
//	    @description API key passed in cookie
//	  }
//	  @security {
//	    @with bearerAuth
//	  }
//	  @defaultContentType json
//	}
package security

import "net/http"

// -----------------------------------------------------------------------------
// Schemas
// -----------------------------------------------------------------------------

// User represents a user in the system
// @schema
type User struct {
	// @field { @description User ID @format uuid }
	ID string `json:"id"`

	// @field { @description User email @format email }
	Email string `json:"email"`

	// @field { @description User role @enum admin,user,guest }
	Role string `json:"role"`
}

// -----------------------------------------------------------------------------
// Endpoints
// -----------------------------------------------------------------------------

// GetCurrentUser uses the default security (bearerAuth)
//
//	@endpoint GET /users/me {
//	  @operationID getCurrentUser
//	  @summary Get current user
//	  @description Returns the currently authenticated user. Uses default Bearer auth.
//	  @response 200 {
//	    @body User
//	    @description Current user info
//	  }
//	}
func GetCurrentUser(w http.ResponseWriter, r *http.Request) {}

// LoginWithBasic overrides default security with basic auth
//
//	@endpoint POST /auth/login {
//	  @operationID loginWithBasic
//	  @summary Login with basic auth
//	  @description Authenticates using HTTP Basic authentication.
//	  @auth basicAuth
//	  @response 200 {
//	    @body User
//	    @description Login successful
//	  }
//	}
func LoginWithBasic(w http.ResponseWriter, r *http.Request) {}

// GetWithApiKeyHeader uses API key in header
//
//	@endpoint GET /api/data {
//	  @operationID getWithApiKeyHeader
//	  @summary Get data with API key (header)
//	  @description Retrieves data using API key passed in X-API-Key header.
//	  @auth apiKeyHeader
//	  @response 200 {
//	    @body User
//	    @description Data retrieved
//	  }
//	}
func GetWithApiKeyHeader(w http.ResponseWriter, r *http.Request) {}

// GetWithApiKeyQuery uses API key in query string
//
//	@endpoint GET /api/query-auth {
//	  @operationID getWithApiKeyQuery
//	  @summary Get data with API key (query)
//	  @description Retrieves data using API key passed in query string.
//	  @auth apiKeyQuery
//	  @response 200 {
//	    @body User
//	    @description Data retrieved
//	  }
//	}
func GetWithApiKeyQuery(w http.ResponseWriter, r *http.Request) {}

// GetWithApiKeyCookie uses API key in cookie
//
//	@endpoint GET /api/cookie-auth {
//	  @operationID getWithApiKeyCookie
//	  @summary Get data with API key (cookie)
//	  @description Retrieves data using API key passed in cookie.
//	  @auth apiKeyCookie
//	  @response 200 {
//	    @body User
//	    @description Data retrieved
//	  }
//	}
func GetWithApiKeyCookie(w http.ResponseWriter, r *http.Request) {}

// ListUsers requires Bearer auth (uses default)
//
//	@endpoint GET /users {
//	  @operationID listUsers
//	  @summary List all users
//	  @description Returns a list of users. Requires Bearer authentication.
//	  @response 200 {
//	    @body []User
//	    @description List of users
//	  }
//	}
func ListUsers(w http.ResponseWriter, r *http.Request) {}
