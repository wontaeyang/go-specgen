// Package anonstruct covers @field annotations nested inside anonymous structs
// (bug #4). The harvest is not recursive, so annotations below the first level
// are parsed and then dropped — 16 of them in examples/inline alone.
//
// Both entry points are covered: an anonymous struct inside a named @schema,
// and one inside an in-function declaration.
//
//	@api {
//	  @title Anonymous Struct Fixture
//	  @version 1.0.0
//	  @description Nested field annotations inside anonymous structs.
//	  @defaultContentType json
//	}
package anonstruct

import "net/http"

// Envelope nests anonymous structs two levels deep.
// @schema
type Envelope struct {
	// @field { @description Envelope identifier @format uuid }
	ID string `json:"id"`

	// @field { @description Nested payload }
	Payload struct {
		// @field { @description Payload kind @enum a,b,c }
		Kind string `json:"kind"`

		// @field { @description Nested entries @minItems 1 }
		Entries []struct {
			// @field { @description Entry name @minLength 1 }
			Name string `json:"name"`

			// @field { @description Entry weight @minimum 0 @maximum 1 }
			Weight float64 `json:"weight"`
		} `json:"entries"`
	} `json:"payload"`
}

// PutEnvelope accepts an envelope and returns an inline acknowledgement.
//
//	@endpoint PUT /envelopes {
//	  @summary Replace an envelope
//	  @request { @body Envelope }
//	}
func PutEnvelope(w http.ResponseWriter, r *http.Request) {
	// @response 200
	var ack struct {
		// @field { @description Whether the write was applied }
		Applied bool `json:"applied"`

		// @field { @description Per-entry results }
		Results []struct {
			// @field { @description Entry name @minLength 1 }
			Name string `json:"name"`

			// @field { @description Result code @enum ok,skipped,failed }
			Code string `json:"code"`
		} `json:"results"`
	}

	_ = ack
}
