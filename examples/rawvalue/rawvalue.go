//	@api {
//	  @title Raw Value Example
//	  @version 1.0.0
//	  @description Demonstrates that a raw value's braces are text, not block structure.
//	  @defaultContentType json
//	}
package rawvalue

// Pattern holds fields whose @pattern carries braces of its own.
//
// A regex is a grammar in its own right: {2} is a quantifier, } may stand alone,
// and [{] is a literal brace. Counting those as block structure closed the block
// early and dropped everything after it, reporting nothing.
//
// Every case appears twice, once written on one line and once as a block. The
// two forms were parsed by different code and gave different answers to the same
// input, so pairing them is the point of this package rather than a decoration.
//
// @schema
type Pattern struct {
	// @field { @description Country code @pattern ^[A-Z]{2}$ }
	CountryCode string `json:"countryCode"`

	// @field {
	//   @description Locale
	//   @pattern ^[a-z]{2}(-[A-Z]{2})?$
	// }
	Locale string `json:"locale"`

	// @field { @description Unpaired closing brace @pattern ^a}b$ }
	Closer string `json:"closer"`

	// @field {
	//   @description Unpaired closing brace, block form
	//   @pattern ^a}b$
	// }
	CloserBlock string `json:"closerBlock"`

	// @field { @description Literal brace in a character class @pattern ^[{]+$ }
	Opener string `json:"opener"`

	// @field {
	//   @description Literal brace in a character class, block form
	//   @pattern ^[{]+$
	// }
	OpenerBlock string `json:"openerBlock"`

	// A space before a brace is what marks a block opener everywhere else, so a
	// raw value that contains one is the case most likely to be misread.
	// @field { @description Quantified space @pattern ^a {2}$ }
	SpacedQuantifier string `json:"spacedQuantifier"`

	// @field {
	//   @description Quantified space, block form
	//   @pattern ^a {2}$
	// }
	SpacedQuantifierBlock string `json:"spacedQuantifierBlock"`
}

// GetPatterns exists so Pattern is reachable: a schema's direction is inferred
// from the endpoints that use it, and an unreferenced schema is an error.
//
//	@endpoint GET /patterns {
//	  @summary Get the pattern samples
//	  @response 200 {
//	    @body Pattern
//	    @description Pattern samples
//	  }
//	}
func GetPatterns() {}
