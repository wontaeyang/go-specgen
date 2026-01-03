//	@api {
//	  @title Inline Structs Example
//	  @version 1.0.0
//	  @description Demonstrates inline struct declarations within handler functions.
//	  Inline structs are NOT added to components/schemas - they are inlined in the spec.
//	  @defaultContentType json
//	}
package inline

import "net/http"

// -----------------------------------------------------------------------------
// Package-Level Schema (for comparison)
// -----------------------------------------------------------------------------

// Error is a package-level schema (goes to components/schemas)
// @schema
type Error struct {
	// @field { @description Error code }
	Code string `json:"code"`

	// @field { @description Error message }
	Message string `json:"message"`
}

// -----------------------------------------------------------------------------
// Endpoints with Inline Structs
// -----------------------------------------------------------------------------

// GetUser demonstrates inline path and response (used by tests)
// @endpoint GET /users/{id} {
// }
func GetUser(w http.ResponseWriter, r *http.Request) {
	// @path
	var path struct {
		// @field { @description User ID @format uuid }
		ID string `path:"id"`
	}

	// @response 200
	var success struct {
		// @field { @description User ID @format uuid }
		ID string `json:"id"`

		// @field { @description User email @format email }
		Email string `json:"email"`

		// @field { @description User name }
		Name string `json:"name"`
	}

	_ = path
	_ = success
}

// CreateUser demonstrates inline request, header, and response (used by tests)
// @endpoint POST /users {
// }
func CreateUser(w http.ResponseWriter, r *http.Request) {
	// @header
	var headers struct {
		// @field { @description Request tracking ID @format uuid }
		RequestID string `header:"X-Request-ID"`
	}

	// @request
	var req struct {
		// @field { @description User email @format email }
		Email string `json:"email"`

		// @field { @description User name }
		Name string `json:"name"`
	}

	// @response 201
	var created struct {
		// @field { @description User ID @format uuid }
		ID string `json:"id"`
	}

	// @response 400
	var badRequest struct {
		// @field { @description Error code }
		Code string `json:"code"`

		// @field { @description Error message }
		Message string `json:"message"`
	}

	_ = headers
	_ = req
	_ = created
	_ = badRequest
}

// ListUsers demonstrates inline query and response (used by tests)
// @endpoint GET /users {
// }
func ListUsers(w http.ResponseWriter, r *http.Request) {
	// @query
	var query struct {
		// @field { @description Maximum results @minimum 1 @maximum 100 }
		Limit *int `query:"limit"`

		// @field { @description Page offset @minimum 0 }
		Offset *int `query:"offset"`

		// @field { @description Filter by status @enum active,inactive }
		Status *string `query:"status"`
	}

	// @response 200
	var list struct {
		// @field { @description List of users }
		Users []struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"users"`

		// @field { @description Total count }
		Total int `json:"total"`
	}

	_ = query
	_ = list
}

// GetOrder demonstrates inline path, query, and response structs
// @endpoint GET /orders/{id} {
// }
func GetOrder(w http.ResponseWriter, r *http.Request) {
	// @path
	var path struct {
		// @field { @description Order ID @format uuid }
		ID string `path:"id"`
	}

	// @query
	var query struct {
		// @field { @description Fields to include in response @enum items,shipping,billing }
		Include []string `query:"include"`

		// @field { @description Whether to include metadata }
		WithMeta *bool `query:"with_meta"`
	}

	// @response 200
	var success struct {
		// @field { @description Order ID @format uuid }
		ID string `json:"id"`

		// @field { @description Order status @enum pending,processing,shipped,delivered }
		Status string `json:"status"`

		// @field { @description Order total in cents @minimum 0 }
		TotalCents int `json:"total_cents"`

		// @field { @description Order items }
		Items []struct {
			// @field { @description Item name }
			Name string `json:"name"`

			// @field { @description Item quantity @minimum 1 }
			Quantity int `json:"quantity"`

			// @field { @description Item price in cents @minimum 0 }
			PriceCents int `json:"price_cents"`
		} `json:"items"`
	}

	// @response 404
	var notFound struct {
		// @field { @description Error code }
		Code string `json:"code"`

		// @field { @description Error message }
		Message string `json:"message"`
	}

	// Use variables to avoid "unused" errors
	_ = path
	_ = query
	_ = success
	_ = notFound
}

// CreateOrder demonstrates inline request and response structs
// @endpoint POST /orders {
// }
func CreateOrder(w http.ResponseWriter, r *http.Request) {
	// @header
	var headers struct {
		// @field { @description Idempotency key @format uuid }
		IdempotencyKey string `header:"X-Idempotency-Key"`
	}

	// @request
	var req struct {
		// @field { @description Customer ID @format uuid }
		CustomerID string `json:"customer_id"`

		// @field { @description Order items }
		Items []struct {
			// @field { @description Product ID @format uuid }
			ProductID string `json:"product_id"`

			// @field { @description Quantity to order @minimum 1 }
			Quantity int `json:"quantity"`
		} `json:"items"`

		// @field { @description Shipping address }
		ShippingAddress struct {
			// @field { @description Street address }
			Street string `json:"street"`

			// @field { @description City }
			City string `json:"city"`

			// @field { @description Postal code }
			PostalCode string `json:"postal_code"`

			// @field { @description Country code @pattern ^[A-Z]{2}$ }
			Country string `json:"country"`
		} `json:"shipping_address"`
	}

	// @response 201
	var created struct {
		// @field { @description Order ID @format uuid }
		ID string `json:"id"`

		// @field { @description Order status }
		Status string `json:"status"`

		// @field { @description Estimated delivery date @format date }
		EstimatedDelivery string `json:"estimated_delivery"`
	}

	// @response 400 { @body Error }
	// Uses package-level Error schema via $ref

	_ = headers
	_ = req
	_ = created
}

// ListOrders demonstrates inline query with multiple response codes
// @endpoint GET /orders {
// }
func ListOrders(w http.ResponseWriter, r *http.Request) {
	// @query
	var query struct {
		// @field { @description Filter by status @enum pending,processing,shipped,delivered }
		Status *string `query:"status"`

		// @field { @description Maximum results @minimum 1 @maximum 100 @default 20 }
		Limit *int `query:"limit"`

		// @field { @description Pagination cursor }
		Cursor *string `query:"cursor"`
	}

	// @response 200
	var list struct {
		// @field { @description List of orders }
		Orders []struct {
			// @field { @description Order ID @format uuid }
			ID string `json:"id"`

			// @field { @description Order status }
			Status string `json:"status"`

			// @field { @description Order total in cents }
			TotalCents int `json:"total_cents"`
		} `json:"orders"`

		// @field { @description Next page cursor }
		NextCursor *string `json:"next_cursor,omitempty"`

		// @field { @description Whether there are more results }
		HasMore bool `json:"has_more"`
	}

	_ = query
	_ = list
}

// DeleteOrder demonstrates inline path with explicit responses in endpoint block
//
//	@endpoint DELETE /orders/{id} {
//	  @response 204 {
//	    @contentType empty
//	    @description Order deleted
//	  }
//	  @response 404 {
//	    @body Error
//	    @description Order not found
//	  }
//	}
func DeleteOrder(w http.ResponseWriter, r *http.Request) {
	// @path
	var path struct {
		// @field { @description Order ID @format uuid }
		ID string `path:"id"`
	}

	_ = path
}

// -----------------------------------------------------------------------------
// Response Headers for Rate Limiting
// -----------------------------------------------------------------------------

// RateLimitHeaders defines response headers for rate limiting
// @header
type RateLimitHeaders struct {
	// @field { @description Request limit per hour }
	Limit int `header:"X-RateLimit-Limit"`

	// @field { @description Remaining requests in current window }
	Remaining int `header:"X-RateLimit-Remaining"`

	// @field { @description Unix timestamp when limit resets }
	Reset int `header:"X-RateLimit-Reset"`
}

// ListOrdersWithHeaders demonstrates inline response with headers and description
// @endpoint GET /orders/with-headers
func ListOrdersWithHeaders(w http.ResponseWriter, r *http.Request) {
	// @response 200 {
	//   @header RateLimitHeaders
	//   @description List of orders with rate limit info
	// }
	var resp struct {
		// @field { @description List of orders }
		Orders []struct {
			// @field { @description Order ID @format uuid }
			ID string `json:"id"`

			// @field { @description Order status }
			Status string `json:"status"`
		} `json:"orders"`
	}

	_ = resp
}
