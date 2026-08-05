//	@api {
//	  @title Schema References Example
//	  @version 1.0.0
//	  @description Demonstrates how a field that references another schema is
//	  emitted, and how that emission differs between OpenAPI versions. A plain
//	  reference is a bare $ref; annotating or nullifying it requires a wrapper in
//	  3.0, where $ref may not carry sibling keywords, but not in 3.1+.
//	  @defaultContentType json
//	}
package refs

import "net/http"

// -----------------------------------------------------------------------------
// Schemas
// -----------------------------------------------------------------------------

// Owner is referenced by Item below.
//
//	@schema {
//	  @description Resource owner
//	}
type Owner struct {
	// @field { @description Owner name }
	Name string `json:"name"`
}

// Item shows every form a schema reference takes.
//
//	@schema {
//	  @description A resource with references to other schemas
//	}
type Item struct {
	// An unannotated reference is emitted as a bare $ref in every version.
	Plain Owner `json:"plain"`

	// Annotating a reference adds sibling keywords. OpenAPI 3.1+ places them
	// next to the $ref; 3.0 forbids that, so the reference moves into allOf.
	// @field { @description Documented owner reference }
	Documented Owner `json:"documented"`

	// A nullable reference always needs a wrapper: 3.1+ unions it with the null
	// type via oneOf, while 3.0 uses allOf plus nullable: true.
	NullableRef *Owner `json:"nullable_ref"`

	// A nullable scalar needs no wrapper: 3.1+ widens the type to a list,
	// while 3.0 sets nullable: true.
	// @field { @description Optional note }
	Note *string `json:"note"`

	// Collections of references put the $ref inside items or
	// additionalProperties.
	// @field { @description Every owner involved }
	Owners []Owner `json:"owners"`

	// @field { @description Owners keyed by role }
	OwnersByRole map[string]Owner `json:"owners_by_role"`
}

// -----------------------------------------------------------------------------
// Endpoints
// -----------------------------------------------------------------------------

// GetItem returns one item.
//
//	@endpoint GET /items/{id} {
//	  @operationID getItem
//	  @summary Get an item
//	  @path ItemPath
//	  @response 200 {
//	    @body Item
//	    @description The item
//	  }
//	}
func GetItem(w http.ResponseWriter, r *http.Request) {}

// ItemPath identifies an item.
// @path
type ItemPath struct {
	// @field { @description Item ID @format uuid }
	ID string `path:"id"`
}
