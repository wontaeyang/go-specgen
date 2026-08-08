// Package unsupported_types covers Go types with no OpenAPI equivalent (bug #3).
//
// resolveType ends in "else { OpenAPIType = "string" }", so a channel, a
// function, or a complex number renders as "type: string" — a spec that claims
// something the encoder can never produce. These are meant to be an error.
//
// This fixture moves to testdata/errors when that lands.
//
//	@api {
//	  @title Unsupported Types Fixture
//	  @version 1.0.0
//	  @description Go types that have no OpenAPI representation.
//	  @defaultContentType json
//	}
package unsupported_types

import "net/http"

// Job holds fields that encoding/json itself refuses to marshal.
// @schema
type Job struct {
	// @field { @description Job identifier }
	ID string `json:"id"`

	// @field { @description Completion signal }
	Done chan struct{} `json:"done"`

	// @field { @description Cancellation hook }
	Cancel func() `json:"cancel"`

	// @field { @description Phase angle }
	Phase complex128 `json:"phase"`

	// @field { @description Raw memory address }
	Handle uintptr `json:"handle"`
}

// GetJob returns a job.
//
//	@endpoint GET /jobs {
//	  @summary Get a job
//	  @response 200 { @body Job }
//	}
func GetJob(w http.ResponseWriter, r *http.Request) {}
