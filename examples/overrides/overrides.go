//	@api {
//	  @title Field Overrides Example
//	  @version 1.0.0
//	  @description Demonstrates \@required and \@nullable as explicit overrides on \@field.
//	  These let you decouple OpenAPI required/nullable from Go pointer-ness and json tags.
//	  @defaultContentType json
//	}
package overrides

import (
	"net/http"
	"time"
)

// -----------------------------------------------------------------------------
// Default behavior (no overrides) — for comparison
// -----------------------------------------------------------------------------
//
//   Field declaration                | Required | Nullable
//   ---------------------------------|----------|----------
//   string                           | true     | false
//   string,omitempty                 | false    | false
//   *string                          | true     | true
//   *string,omitempty                | false    | false
//   *string,omitzero                 | false    | false
//   []string                         | true     | false
//   []string,omitempty               | false    | false
//   map[string]string,omitempty      | false    | false
//   time.Time,omitempty              | true     | false
//   time.Time,omitzero               | false    | false
//
// Defaults are outcome-oriented: they describe what encoding/json actually
// puts on the wire, not what the tag text says.
//
// A pointer with omitempty is optional but NOT nullable: encoding/json omits
// a nil pointer instead of encoding null, so null never appears on the wire.
//
// omitempty only drops values encoding/json considers empty, and a non-pointer
// struct is never empty — so time.Time,omitempty is always emitted and stays
// required. omitzero (Go 1.24+) omits the zero value of any type, including
// structs, so it always makes the field optional.
//
// `omitempty` only affects JSON encoding (response side), so using it to mark
// a request field as optional conflates two concerns. The @required override
// lets you mark a request field optional without touching the json tag.

// TagDefaults exercises every row of the table above with no overrides, so the
// generated output is golden-verified documentation of the defaults.
// @schema
type TagDefaults struct {
	// @field { @description string: required, not nullable }
	Plain string `json:"plain"`

	// @field { @description string,omitempty: optional, not nullable }
	PlainOmit string `json:"plain_omit,omitempty"`

	// @field { @description *string: required, nullable — nil encodes as null }
	Ptr *string `json:"ptr"`

	// @field { @description *string,omitempty: optional, not nullable — nil is omitted, never null }
	PtrOmit *string `json:"ptr_omit,omitempty"`

	// @field { @description *string,omitzero: same as omitempty — optional, not nullable }
	PtrZero *string `json:"ptr_zero,omitzero"`

	// @field { @description []string: required, not nullable by default; add \@nullable true if the handler can return a nil slice }
	Tags []string `json:"tags"`

	// @field { @description []string,omitempty: optional; nil and empty are both omitted }
	OptTags []string `json:"opt_tags,omitempty"`

	// @field { @description map,omitempty: optional; nil and empty are both omitted }
	Labels map[string]string `json:"labels,omitempty"`

	// @field { @description time.Time,omitempty: still required — a non-pointer struct is never empty, so omitempty has no effect }
	CreatedAt time.Time `json:"created_at,omitempty"`

	// @field { @description time.Time,omitzero: optional — omitzero omits the zero time }
	UpdatedAt time.Time `json:"updated_at,omitzero"`
}

// LoginRequest demonstrates @required false on a non-pointer optional input.
//
// TFACode is optional but uses `string` (not `*string`) so the zero value is
// the empty string — avoids pointer noise in handler code.
// @schema
type LoginRequest struct {
	// @field { @description User email @format email }
	Email string `json:"email"`

	// @field { @description User password }
	Password string `json:"password"`

	// @field { @description Two-factor auth code, optional @required false }
	TFACode string `json:"tfa_code"`
}

// CreatePromo demonstrates @required true on a pointer field.
//
// Percent uses *int so the handler can distinguish "not sent" from "sent 0"
// (0 is a valid discount). The API contract still requires the field to be
// present in the request body.
// @schema
type CreatePromo struct {
	// @field { @description Promo code }
	Code string `json:"code"`

	// @field { @description Discount percent; 0 is a valid value @minimum 0 @maximum 100 @required true }
	Percent *int `json:"percent"`
}

// UpdateUserPatch demonstrates @nullable true on a pointer+omitempty field.
//
// PATCH semantics: omit the field to leave it unchanged, send explicit `null`
// to clear the value. Pointer+omitempty defaults to non-nullable (nil is
// omitted, never encoded as null), so accepting null in requests requires
// opting back in with @nullable true.
// @schema
type UpdateUserPatch struct {
	// @field { @description Replace email; omit to leave unchanged, null to clear @format email @nullable true }
	Email *string `json:"email,omitempty"`

	// @field { @description Replace display name; omit to leave unchanged, null to clear @nullable true }
	DisplayName *string `json:"display_name,omitempty"`
}

// NullableProfile demonstrates @nullable true on a non-pointer field.
//
// Useful when wrapping a custom null-aware type (e.g., sql.NullString,
// guregu/null.String) that is structurally non-pointer but semantically
// permits null at the JSON layer.
// @schema
type NullableProfile struct {
	// @field { @description User ID @format uuid }
	ID string `json:"id"`

	// @field { @description Date of birth; null if not provided @format date @nullable true }
	DateOfBirth string `json:"date_of_birth"`

	// @field { @description Biography text; null if cleared @nullable true }
	Bio string `json:"bio"`
}

// -----------------------------------------------------------------------------
// Endpoints
// -----------------------------------------------------------------------------

// Login authenticates a user (demonstrates @required false on inline request)
//
//	@endpoint POST /login {
//	  @operationID login
//	  @summary Log in
//	  @request { @body LoginRequest }
//	  @response 200 {
//	    @description Login succeeded
//	  }
//	}
func Login(w http.ResponseWriter, r *http.Request) {}

// CreatePromoCode creates a promo (demonstrates @required true on pointer)
//
//	@endpoint POST /promos {
//	  @operationID createPromo
//	  @summary Create a promo code
//	  @request { @body CreatePromo }
//	  @response 201 {
//	    @description Promo created
//	  }
//	}
func CreatePromoCode(w http.ResponseWriter, r *http.Request) {}

// PatchUser updates user fields (demonstrates @nullable true on pointer+omitempty)
//
//	@endpoint PATCH /users/{id} {
//	  @operationID patchUser
//	  @summary Patch user fields
//	  @request { @body UpdateUserPatch }
//	  @response 200 {
//	    @description User updated
//	  }
//	}
func PatchUser(w http.ResponseWriter, r *http.Request) {
	// @path
	var path struct {
		// @field { @description User ID @format uuid }
		ID string `path:"id"`
	}
	_ = path
}

// GetProfile returns a profile (demonstrates @nullable true on non-pointer)
//
//	@endpoint GET /profiles/{id} {
//	  @operationID getProfile
//	  @summary Get a profile
//	  @response 200 {
//	    @body NullableProfile
//	  }
//	}
func GetProfile(w http.ResponseWriter, r *http.Request) {
	// @path
	var path struct {
		// @field { @description Profile ID @format uuid }
		ID string `path:"id"`
	}
	_ = path
}
