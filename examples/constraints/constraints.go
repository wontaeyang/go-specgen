//	@api {
//	  @title Constraints Example
//	  @version 1.0.0
//	  @description Demonstrates field constraints: read/write visibility, array
//	  bounds and uniqueness, numeric bounds (inclusive and exclusive), string
//	  lengths and patterns, and deprecation at every level.
//	  @defaultContentType json
//	}
package constraints

import "net/http"

// -----------------------------------------------------------------------------
// Schemas
// -----------------------------------------------------------------------------

// Account demonstrates read/write visibility and string constraints.
//
//	@schema {
//	  @description A user account
//	}
type Account struct {
	// Server-assigned: clients receive it but never send it.
	// @field { @description Account ID @format uuid @readOnly }
	ID string `json:"id"`

	// Accepted on write but never echoed back.
	// @field { @description Account password @writeOnly @minLength 12 @maxLength 128 }
	Password string `json:"password"`

	// @field { @description Account handle @minLength 3 @maxLength 30 @pattern ^[a-z0-9_]+$ }
	Handle string `json:"handle"`

	// @field { @description Creation timestamp @format date-time @readOnly }
	CreatedAt string `json:"created_at"`
}

// Inventory demonstrates array constraints and numeric bounds.
//
//	@schema {
//	  @description Stock levels for a product
//	}
type Inventory struct {
	// @field { @description Distinct warehouse codes @uniqueItems @minItems 1 @maxItems 25 }
	Warehouses []string `json:"warehouses"`

	// @field { @description Recent daily counts, newest first @minItems 7 @maxItems 365 }
	History []int `json:"history"`

	// Inclusive bounds: 0 and 100000 are both allowed.
	// @field { @description Units on hand @minimum 0 @maximum 100000 }
	OnHand int `json:"on_hand"`

	// Exclusive bounds: the ratio must be strictly between 0 and 1. OpenAPI 3.1
	// emits these as numbers; 3.0 emits a boolean flag alongside minimum/maximum.
	// @field { @description Fill rate, strictly between 0 and 1 @exclusiveMinimum 0 @exclusiveMaximum 1 }
	FillRate float64 `json:"fill_rate"`

	// @field { @description Reorder threshold @minimum 1 }
	ReorderAt int `json:"reorder_at,omitempty"`
}

// LegacyAccount demonstrates deprecation at the schema and field level.
//
//	@schema {
//	  @description Superseded by Account
//	  @deprecated
//	}
type LegacyAccount struct {
	// @field { @description Numeric account ID, replaced by Account.ID @deprecated }
	AccountNo int `json:"account_no"`

	// @field { @description Account handle }
	Handle string `json:"handle"`
}

// -----------------------------------------------------------------------------
// Parameters
// -----------------------------------------------------------------------------

// AccountPath identifies an account.
// @path
type AccountPath struct {
	// @field { @description Account ID @format uuid }
	ID string `path:"id"`
}

// InventoryQuery demonstrates constraints on query parameters.
// @query
type InventoryQuery struct {
	// @field { @description Maximum results @minimum 1 @maximum 200 @default 50 }
	Limit int `query:"limit"`

	// @field { @description Warehouse codes to include @uniqueItems @maxItems 25 }
	Warehouses []string `query:"warehouses"`

	// @field { @description Minimum fill rate to report @exclusiveMinimum 0 }
	MinFillRate float64 `query:"min_fill_rate"`
}

// -----------------------------------------------------------------------------
// Endpoints
// -----------------------------------------------------------------------------

// GetAccount returns one account.
//
//	@endpoint GET /accounts/{id} {
//	  @operationID getAccount
//	  @summary Get an account
//	  @path AccountPath
//	  @response 200 {
//	    @body Account
//	    @description The account
//	  }
//	}
func GetAccount(w http.ResponseWriter, r *http.Request) {}

// CreateAccount creates an account.
//
//	@endpoint POST /accounts {
//	  @operationID createAccount
//	  @summary Create an account
//	  @description Write-only fields are accepted here and omitted from responses.
//	  @request {
//	    @body Account
//	  }
//	  @response 201 {
//	    @body Account
//	    @description The created account
//	  }
//	}
func CreateAccount(w http.ResponseWriter, r *http.Request) {}

// ListInventory returns stock levels.
//
//	@endpoint GET /inventory {
//	  @operationID listInventory
//	  @summary List inventory
//	  @query InventoryQuery
//	  @response 200 {
//	    @body []Inventory
//	    @description Matching inventory records
//	  }
//	}
func ListInventory(w http.ResponseWriter, r *http.Request) {}

// ListLegacyAccounts is retained for older clients.
//
//	@endpoint GET /legacy/accounts {
//	  @operationID listLegacyAccounts
//	  @summary List legacy accounts
//	  @description Superseded by GET /accounts.
//	  @deprecated
//	  @response 200 {
//	    @body []LegacyAccount
//	    @description Legacy accounts
//	  }
//	}
func ListLegacyAccounts(w http.ResponseWriter, r *http.Request) {}
