// Package paramtags covers the parameter tag matrix (bugs #7 and #8).
//
// Parameters are meant to mirror encoding/json naming: a tag names the
// parameter, "-" skips it, and an absent tag falls back to the Go field name.
// Today "-" emits a phantom parameter named by the Go field, and ",required" is
// matched against the whole raw struct tag rather than the kind-specific one.
//
//	@api {
//	  @title Parameter Tags Fixture
//	  @version 1.0.0
//	  @description The parameter naming and required-detection matrix.
//	  @defaultContentType json
//	}
package paramtags

import "net/http"

// Paging is embedded into a parameter struct.
type Paging struct {
	// @field { @description Results per page @minimum 1 @maximum 100 }
	PerPage *int `query:"per_page"`
}

// TagMatrix covers one row per tag state.
// @query
type TagMatrix struct {
	// @field { @description Named by its tag }
	Limit *int `query:"limit"`

	// @field { @description Opted in as required }
	Q string `query:"q,required"`

	// @field { @description Skipped, mirroring encoding/json }
	Internal *string `query:"-"`

	// @field { @description Literally named "-", using json's escape }
	Dash *string `query:"-,"`

	// @field { @description No query tag, so the Go field name is used }
	Cursor *string

	// @field { @description Only the json tag carries ",required" }
	Note *string `query:"note" json:"note,required"`

	// Embedded parameter fields belong to the same parameter list.
	Paging
}

// Search reads the tag matrix.
//
//	@endpoint GET /search {
//	  @summary Search
//	  @query TagMatrix
//	  @response 200 { @body []string }
//	}
func Search(w http.ResponseWriter, r *http.Request) {}
