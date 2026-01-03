//	@api {
//	  @title Response Wrappers Example
//	  @version 1.0.0
//	  @description Demonstrates response wrapper patterns using @bind.
//	  The @bind directive allows wrapping response data in a consistent envelope.
//	  @defaultContentType json
//	}
package responses

import "net/http"

// -----------------------------------------------------------------------------
// Response Wrapper Schemas
// -----------------------------------------------------------------------------

// APIResponse is a standard wrapper for all API responses
//
//	@schema {
//	  @description Standard API response envelope
//	}
type APIResponse struct {
	// @field { @description Response status }
	Status string `json:"status"`

	// @field { @description Response data (varies by endpoint) }
	Data any `json:"data"`

	// @field { @description Optional error message }
	Error *string `json:"error,omitempty"`
}

// PaginatedResponse wraps paginated data
//
//	@schema {
//	  @description Paginated response envelope
//	}
type PaginatedResponse struct {
	// @field { @description Response status }
	Status string `json:"status"`

	// @field { @description Response data (varies by endpoint) }
	Data any `json:"data"`

	// @field { @description Total number of items }
	Total int `json:"total"`

	// @field { @description Current page number @minimum 1 }
	Page int `json:"page"`

	// @field { @description Items per page @minimum 1 @maximum 100 }
	PageSize int `json:"page_size"`

	// @field { @description Whether there are more pages }
	HasMore bool `json:"has_more"`
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

// Product represents a product
// @schema
type Product struct {
	// @field { @description Product ID @format uuid }
	ID string `json:"id"`

	// @field { @description Product name }
	Name string `json:"name"`

	// @field { @description Product price in cents @minimum 0 }
	PriceCents int `json:"price_cents"`

	// @field { @description Product category }
	Category string `json:"category"`
}

// Setting represents a configuration setting
// @schema
type Setting struct {
	// @field { @description Setting key }
	Key string `json:"key"`

	// @field { @description Setting value }
	Value string `json:"value"`

	// @field { @description Whether the setting is editable }
	Editable bool `json:"editable"`
}

// Error represents an error response
// @schema
type Error struct {
	// @field { @description Error code }
	Code string `json:"code"`

	// @field { @description Error message }
	Message string `json:"message"`
}

// WebhookEvent represents an incoming webhook event
//
//	@schema {
//	  @description Webhook event with dynamic payload
//	}
type WebhookEvent struct {
	// @field { @description Event ID @format uuid }
	ID string `json:"id"`

	// @field { @description Event type }
	Type string `json:"type"`

	// @field { @description Event payload (varies by type) }
	Payload any `json:"payload"`

	// @field { @description Optional metadata }
	Metadata any `json:"metadata,omitempty"`
}

// -----------------------------------------------------------------------------
// Endpoints with @bind
// -----------------------------------------------------------------------------

// UserIDPath for user endpoints
// @path
type UserIDPath struct {
	// @field { @description User ID @format uuid }
	ID string `path:"id"`
}

// ProductIDPath for product endpoints
// @path
type ProductIDPath struct {
	// @field { @description Product ID @format uuid }
	ID string `path:"id"`
}

// RateLimitHeaders for rate limiting response headers
// @header
type RateLimitHeaders struct {
	// @field { @description Request limit per hour }
	Limit int `header:"X-RateLimit-Limit"`

	// @field { @description Remaining requests in current window }
	Remaining int `header:"X-RateLimit-Remaining"`

	// @field { @description Unix timestamp when limit resets }
	Reset int `header:"X-RateLimit-Reset"`
}

// GetUser demonstrates @bind with a single object
//
//	@endpoint GET /users/{id} {
//	  @operationID getUser
//	  @summary Get a user
//	  @description Returns a user wrapped in the standard API response.
//	  @path UserIDPath
//	  @response 200 {
//	    @body User
//	    @bind APIResponse.Data
//	    @description User found
//	  }
//	}
func GetUser(w http.ResponseWriter, r *http.Request) {}

// ListUsers demonstrates @bind with an array
//
//	@endpoint GET /users {
//	  @operationID listUsers
//	  @summary List users
//	  @description Returns a paginated list of users.
//	  @response 200 {
//	    @body []User
//	    @bind PaginatedResponse.Data
//	    @description List of users
//	  }
//	}
func ListUsers(w http.ResponseWriter, r *http.Request) {}

// GetSettings demonstrates @bind with a map
//
//	@endpoint GET /settings {
//	  @operationID getSettings
//	  @summary Get all settings
//	  @description Returns a map of settings wrapped in the API response.
//	  @response 200 {
//	    @body map[string]Setting
//	    @bind APIResponse.Data
//	    @description Map of settings
//	  }
//	}
func GetSettings(w http.ResponseWriter, r *http.Request) {}

// ListProducts demonstrates @bind with array and pagination
//
//	@endpoint GET /products {
//	  @operationID listProducts
//	  @summary List products
//	  @description Returns a paginated list of products.
//	  @response 200 {
//	    @body []Product
//	    @bind PaginatedResponse.Data
//	    @description List of products
//	  }
//	}
func ListProducts(w http.ResponseWriter, r *http.Request) {}

// GetProduct demonstrates plain @body without @bind for comparison
//
//	@endpoint GET /products/{id} {
//	  @operationID getProduct
//	  @summary Get a product (no wrapper)
//	  @description Returns a product directly without wrapper for comparison.
//	  @path ProductIDPath
//	  @response 200 {
//	    @body Product
//	    @description Product found
//	  }
//	}
func GetProduct(w http.ResponseWriter, r *http.Request) {}

// CreateUser demonstrates @bind with 201 response
//
//	@endpoint POST /users {
//	  @operationID createUser
//	  @summary Create a user
//	  @description Creates a user and returns it wrapped in the API response.
//	  @request {
//	    @body User
//	  }
//	  @response 201 {
//	    @body User
//	    @bind APIResponse.Data
//	    @description User created
//	  }
//	}
func CreateUser(w http.ResponseWriter, r *http.Request) {}

// DeleteUser demonstrates compact inline syntax - entire block on one line
//
//	@endpoint DELETE /users/{id} {
//	  @operationID deleteUser
//	  @summary Delete a user
//	  @path UserIDPath
//	  @response 200 { @body User @bind APIResponse.Data @description User deleted }
//	  @response 404 { @body Error @description User not found }
//	}
func DeleteUser(w http.ResponseWriter, r *http.Request) {}

// HandleWebhook demonstrates schema with any fields without @bind
//
//	@endpoint POST /webhooks {
//	  @operationID handleWebhook
//	  @summary Handle incoming webhook
//	  @description Receives webhook events with dynamic payloads.
//	  @response 200 {
//	    @body WebhookEvent
//	    @description Webhook processed
//	  }
//	}
func HandleWebhook(w http.ResponseWriter, r *http.Request) {}

// ListUsersWithHeaders demonstrates response headers
//
//	@endpoint GET /users/with-headers {
//	  @operationID listUsersWithHeaders
//	  @summary List users with rate limit headers
//	  @description Returns a list of users with rate limiting information in response headers.
//	  @response 200 {
//	    @header RateLimitHeaders
//	    @body []User
//	    @bind PaginatedResponse.Data
//	    @description List of users with rate limit info
//	  }
//	}
func ListUsersWithHeaders(w http.ResponseWriter, r *http.Request) {}
