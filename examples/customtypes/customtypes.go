//	@api {
//	  @title Custom Types Example
//	  @version 1.0.0
//	  @description Demonstrates field type resolution for custom types within schemas.
//	  @defaultContentType json
//	}
package customtypes

import "net/http"

// =============================================================================
// SUPPORTED: Custom Primitive Types
// These resolve to their underlying OpenAPI types:
//   - type UserID string    -> type: string
//   - type StatusCode int   -> type: integer
//   - type Price float64    -> type: number, format: double
// =============================================================================

// UserID is a custom string type -> resolves to type: string
type UserID string

// StatusCode is a custom int type -> resolves to type: integer
type StatusCode int

// Price is a custom float type -> resolves to type: number, format: double
type Price float64

// =============================================================================
// SUPPORTED: Custom Slice Types
// These resolve to arrays with the underlying element type:
//   - type Tags []string -> type: array, items: { type: string }
// =============================================================================

// Tags is a custom slice of strings -> resolves to type: array, items: string
type Tags []string

// =============================================================================
// SUPPORTED: @schema Struct Types
// Named structs with @schema annotation generate $ref references
// =============================================================================

// Address is a struct WITH @schema annotation -> generates $ref
// @schema
type Address struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	Country string `json:"country"`
}

// =============================================================================
// Schema With Custom Type Fields
// =============================================================================

// User demonstrates field type resolution for various custom types
// @schema
type User struct {
	// Custom primitive types resolve to their underlying types
	// @field { @description User ID }
	ID UserID `json:"id"` // -> type: string

	// @field { @description Status code }
	Status StatusCode `json:"status"` // -> type: integer

	// @field { @description Account balance }
	Balance Price `json:"balance"` // -> type: number, format: double

	// Custom slice of primitives resolves to array
	// @field { @description User tags }
	Labels Tags `json:"labels"` // -> type: array, items: { type: string }

	// ===================
	// @schema struct references -> $ref
	// ===================

	// Direct @schema struct -> $ref
	WorkAddress Address `json:"work_address"`

	// Slice of @schema struct -> type: array, items: { $ref }
	// @field { @description All addresses }
	AllAddresses []Address `json:"all_addresses"`

	// Map of @schema struct -> type: object, additionalProperties: { $ref }
	// @field { @description Address book }
	AddressBook map[string]Address `json:"address_book"`

	// ===================
	// Anonymous structs -> inlined in spec
	// ===================

	// Anonymous inline struct -> inlined as object with properties
	// @field { @description Inline address }
	InlineAddress struct {
		Street string `json:"street"`
		City   string `json:"city"`
	} `json:"inline_address"`

	// Nested anonymous struct -> recursively inlined
	// @field { @description Nested location }
	Location struct {
		Name    string `json:"name"`
		Address struct {
			Street  string `json:"street"`
			City    string `json:"city"`
			Country string `json:"country"`
		} `json:"address"`
		Coordinates struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"coordinates"`
	} `json:"location"`
}

// =============================================================================
// Endpoint
// =============================================================================

// GetUser returns a user
//
//	@endpoint GET /user {
//	  @operationID getUser
//	  @summary Get a user
//	  @response 200 {
//	    @body User
//	  }
//	}
func GetUser(w http.ResponseWriter, r *http.Request) {}
