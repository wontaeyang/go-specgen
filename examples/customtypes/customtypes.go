//	@api {
//	  @title Custom Types Example
//	  @version 1.0.0
//	  @description Demonstrates field type resolution for custom types within schemas.
//	  @defaultContentType json
//	}
package customtypes

import (
	"net/http"
	"time"
)

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
// Built-in Type Resolution
// =============================================================================

// Primitives documents how each built-in Go type resolves, including the
// integer and float formats, and the same types inside slices and maps.
// Unsigned integers carry no format.
//
// @schema
type Primitives struct {
	// @field { @description Signed 32-bit }
	Int32 int32 `json:"int32"` // -> integer, format: int32

	// @field { @description Signed 64-bit }
	Int64 int64 `json:"int64"` // -> integer, format: int64

	// @field { @description Unsigned 32-bit }
	Uint32 uint32 `json:"uint32"` // -> integer, no format

	// @field { @description Unsigned 64-bit }
	Uint64 uint64 `json:"uint64"` // -> integer, no format

	// @field { @description Single precision }
	Float32 float32 `json:"float32"` // -> number, format: float

	// @field { @description Double precision }
	Float64 float64 `json:"float64"` // -> number, format: double

	// @field { @description Flag }
	Bool bool `json:"bool"` // -> boolean

	// @field { @description Timestamp }
	When time.Time `json:"when"` // -> string, format: date-time

	// @field { @description Raw bytes }
	Blob []byte `json:"blob"` // -> format: byte

	// Slices and maps carry the element resolution into items and
	// additionalProperties.

	// @field { @description Counts }
	Int64s []int64 `json:"int64s"`

	// @field { @description Ratios }
	Float32s []float32 `json:"float32s"`

	// @field { @description Timestamps }
	Times []time.Time `json:"times"`

	// @field { @description Counts by key }
	CountsByKey map[string]int64 `json:"counts_by_key"`

	// @field { @description Flags by key }
	FlagsByKey map[string]bool `json:"flags_by_key"`

	// @field { @description Timestamps by key }
	TimesByKey map[string]time.Time `json:"times_by_key"`
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

// GetPrimitives returns the built-in type matrix.
//
//	@endpoint GET /primitives {
//	  @operationID getPrimitives
//	  @summary Get primitive type resolution
//	  @response 200 {
//	    @body Primitives
//	  }
//	}
func GetPrimitives(w http.ResponseWriter, r *http.Request) {}
