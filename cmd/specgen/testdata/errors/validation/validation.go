// Package validation trips several validator rules at once.
//
// The validator is the only stage that accumulates rather than failing fast, so
// this fixture pins both the message text and the order they come out in. A
// map iteration anywhere on that path would make it flaky.
//
//	@api {
//	  @title Validation Fixture
//	  @version 1.0.0
//	  @description Several independent validation failures.
//	  @defaultContentType json
//	}
package validation

import "net/http"

// Bounds violates the constraint rules in several ways at once.
// @schema
type Bounds struct {
	// @field { @description Inverted numeric bounds @minimum 100 @maximum 10 }
	Count int `json:"count"`

	// @field { @description Inverted string bounds @minLength 50 @maxLength 5 }
	Name string `json:"name"`

	// @field { @description Length bounds on a number @minLength 1 @maxLength 5 }
	Weight float64 `json:"weight"`

	// @field { @description Uniqueness on a non-array @uniqueItems }
	Label string `json:"label"`

	// @field { @description Readable and writable at once @readOnly @writeOnly }
	Token string `json:"token"`

	// @field { @description Unparseable pattern @pattern ^( }
	Code string `json:"code"`
}

// GetBounds returns bounds.
//
//	@endpoint GET /bounds {
//	  @summary Get bounds
//	  @response 200 { @body Bounds }
//	}
func GetBounds(w http.ResponseWriter, r *http.Request) {}
