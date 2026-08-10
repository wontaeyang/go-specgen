// Package parse_multiple has a bad annotation in three different declarations.
//
// The parser accumulates: it records what is wrong with each top-level
// declaration and moves to the next, so one run names all three. Before that it
// returned at the first failure, and since the passes walked maps, which of the
// three you were told about was down to Go's map iteration -- fix it, run
// again, get another one, in no particular order.
//
// The errors are grouped by the pass that found them, and sorted by name within
// each: schemas, then endpoints, then fields. Declarations here are named so
// that order is visible rather than coincidental -- ZebraWidget's bad @field is
// reported last despite being declared first, because @field parsing is a later
// pass than @schema parsing.
//
// Accumulation goes down to the individual annotation, not just the
// declaration: ZebraWidget has two bad @field blocks and both are reported.
// That is the case worth having, because misunderstanding the @field syntax
// produces the same mistake on every field of a struct, and reporting one of
// them means one run per field to work through.
//
//	@api {
//	  @title Parse Multiple Fixture
//	  @version 1.0.0
//	  @description Bad annotations in three separate declarations.
//	  @defaultContentType json
//	}
package parse_multiple

import "net/http"

// ZebraWidget parses as a @schema, then trips on its field annotation.
//
//	@schema {
//	  @description A widget
//	}
type ZebraWidget struct {
	// @field { @description Widget code @multipleOf 3 }
	Code string `json:"code"`

	// The same misunderstanding again, one field down. Both are reported.
	// @field { @description Widget size @multipleOf 5 }
	Size int `json:"size"`
}

// AlphaGadget trips on the @schema block itself.
//
// The bogus annotation sits after @description on purpose. @description takes a
// multiline value, and a line starting with an unescaped @ ends it whatever the
// name turns out to be -- so an annotation nobody recognizes is reported as
// exactly that. It used to be swallowed as a second line of prose and blamed on
// the bare @, whose suggested fix, \@, published the typo in the description.
//
//	@schema {
//	  @description A gadget
//	  @bogusStructRule yes
//	}
type AlphaGadget struct {
	// @field { @description Gadget name }
	Name string `json:"name"`
}

// ListThings trips on the @endpoint block.
//
//	@endpoint GET /things {
//	  @summary List things
//	  @bogusEndpointRule yes
//	  @response 200 { @body ZebraWidget }
//	}
func ListThings(w http.ResponseWriter, r *http.Request) {}
