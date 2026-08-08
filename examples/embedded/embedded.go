// Package embedded covers the embedded-field family (bug #9). No example uses
// embedded fields at all, despite the README documenting them.
//
// encoding/json handles four cases; specgen currently gets three wrong:
//
//	Base           untagged struct     flatten                       (correct today)
//	Meta           tagged struct       nest under the tag            (flattens today)
//	Hidden         json:"-"            skip                          (flattens today)
//	Label          untagged non-struct field keyed by the type name  (dropped today)
//	Slug           tagged non-struct   field keyed by the tag        (dropped today)
//
//	@api {
//	  @title Embedded Fixture
//	  @version 1.0.0
//	  @description Embedded struct and non-struct fields.
//	  @defaultContentType json
//	}
package embedded

import "net/http"

// Base is embedded without a tag, so its fields flatten into the parent.
type Base struct {
	// @field { @description Identifier }
	ID string `json:"id"`

	// @field { @description Creation timestamp @format date-time }
	CreatedAt string `json:"created_at"`
}

// Meta is embedded under a tag, so encoding/json nests it.
// @schema
type Meta struct {
	// @field { @description Revision counter }
	Revision int `json:"revision"`
}

// Hidden is embedded with json:"-", so encoding/json drops it entirely.
type Hidden struct {
	// @field { @description Never serialized }
	Secret string `json:"secret"`
}

// Label is a non-struct embedded without a tag; encoding/json keys it by the
// type name.
type Label string

// Slug is a non-struct embedded under a tag.
type Slug string

// Document exercises all five embedding cases at once.
// @schema
type Document struct {
	Base
	Meta   `json:"meta"`
	Hidden `json:"-"`
	Label
	Slug `json:"slug"`

	// @field { @description Document title }
	Title string `json:"title"`
}

// GetDocument returns a document.
//
//	@endpoint GET /documents {
//	  @summary Get a document
//	  @response 200 { @body Document }
//	}
func GetDocument(w http.ResponseWriter, r *http.Request) {}
