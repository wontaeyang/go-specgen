// Package inline_ref references named @schemas from places that go through
// the generator's ref-unaware field path: an in-function response struct, an
// in-function request struct, and a @bind wrapper's non-bound fields.
//
// The ref-aware path emits $ref for these; the ref-unaware one does not know
// what a schema is and falls back to the field's OpenAPI type.
//
// A schema is either request or response - never both - so the request struct
// references AuthorInput while the response struct and Envelope reference
// Author. All three reference paths are direction seeds: the inline request
// makes AuthorInput request-only, and both the inline response and the bind
// wrapper's ProducedBy field make Author response-only.
//
//	@api {
//	  @title Inline Ref Fixture
//	  @version 1.0.0
//	  @description Named schema types referenced from inline structs.
//	  @defaultContentType json
//	}
package inline_ref

import "net/http"

// Author is a named schema referenced from inline structs.
// @schema
type Author struct {
	// @field { @description Author identifier @format uuid }
	ID string `json:"id"`

	// @field { @description Author display name }
	Name string `json:"name"`
}

// AuthorInput is the request-side twin of Author.
// @schema
type AuthorInput struct {
	// @field { @description Author display name }
	Name string `json:"name"`
}

// Envelope wraps a payload alongside a schema-typed field.
// @schema
type Envelope struct {
	// @field { @description The wrapped payload }
	Data any `json:"data"`

	// @field { @description Who produced this response }
	ProducedBy Author `json:"produced_by"`
}

// CreatePost accepts and returns inline structs that reference Author.
//
//	@endpoint POST /posts {
//	  @summary Create a post
//	}
func CreatePost(w http.ResponseWriter, r *http.Request) {
	// @request
	var req struct {
		// @field { @description Post title @minLength 1 }
		Title string `json:"title"`

		// @field { @description Post author }
		Author AuthorInput `json:"author"`

		// @field { @description Contributing authors }
		Contributors []AuthorInput `json:"contributors"`
	}

	// @response 201 { @bind Envelope.Data }
	var created struct {
		// @field { @description Post identifier @format uuid }
		ID string `json:"id"`

		// @field { @description Post author }
		Author Author `json:"author"`
	}

	_, _ = req, created
}
