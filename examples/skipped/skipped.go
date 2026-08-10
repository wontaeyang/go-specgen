// @api { @title Skipped @version 1.0.0 }
package skipped

// @schemaa
type Widget struct {
	// @fild { @description Widget ID }
	ID string `json:"id"`
}

// Gadget is documented in prose that mentions @schema before declaring it, the
// way testdata/errors/brace_in_value does. It must stay silent.
//
// @schema
type Gadget struct {
	// @field { @description Gadget ID }
	ID string `json:"id"`
}

// @endpoin GET /widgets
func ListWidgets() {}

// Helper is ordinary documentation and says nothing about annotations.
func Helper() {}

// \@author is escaped, so this is prose rather than an annotation.
func Escaped() {}

//	@endpoint GET /gadgets {
//	  @response 200 { @description ok @body Gadget }
//	}
func ListGadgets() {
	// @quer
	var filters struct {
		Limit int `query:"limit"`
	}

	_ = filters
}
