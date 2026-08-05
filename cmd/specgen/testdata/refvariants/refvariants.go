//	@api {
//	  @title RefVariants Fixture
//	  @version 1.0.0
//	}
package refvariants

// Owner is a referenced schema.
//
//	@schema {
//	  @description Resource owner
//	}
type Owner struct {
	// @field { @description Owner name }
	Name string `json:"name"`
}

// Item exercises every $ref emission variant: bare ref (no siblings),
// ref with sibling description (3.0 allOf wrapper vs 3.1 siblings),
// nullable ref (3.0 allOf+nullable vs 3.1 oneOf with null), and
// nullable scalar (3.0 nullable:true vs 3.1 type list).
//
//	@schema {
//	  @description An item with reference variants
//	}
type Item struct {
	Plain Owner `json:"plain"`

	// @field { @description Documented owner reference }
	Documented Owner `json:"documented"`

	NullableRef *Owner `json:"nullable_ref"`

	// @field { @description Optional note }
	Note *string `json:"note"`
}

// GetItem returns an item.
//
//	@endpoint GET /item {
//	  @response 200 { @body Item @description The item }
//	}
func GetItem() {}
