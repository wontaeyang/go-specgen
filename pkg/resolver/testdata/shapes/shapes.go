//	@api {
//	  @title Resolver Shapes
//	  @version 1.0.0
//	}
package shapes

import (
	"net/url"
	"time"
)

// Address is a referenced schema.
//
// @schema
type Address struct {
	// @field { @description Street }
	Street string `json:"street"`
}

// Untagged carries no @schema annotation, so a field referencing it has
// nothing to reference.
type Untagged struct {
	Name string `json:"name"`
}

// Shapes covers the Go type shapes the resolver maps.
//
// @schema
type Shapes struct {
	Text      string               `json:"text"`
	Count     int32                `json:"count"`
	Big       int64                `json:"big"`
	Ratio     float32              `json:"ratio"`
	Precise   float64              `json:"precise"`
	Unsigned  uint16               `json:"unsigned"`
	Flag      bool                 `json:"flag"`
	Created   time.Time            `json:"created"`
	Link      url.URL              `json:"link"`
	Blob      []byte               `json:"blob"`
	Names     []string             `json:"names"`
	Fixed     [3]int               `json:"fixed"`
	Addresses []Address            `json:"addresses"`
	PtrAddrs  []*Address           `json:"ptr_addrs"`
	Book      map[string]Address   `json:"book"`
	Counts    map[string]int       `json:"counts"`
	Times     map[string]time.Time `json:"times"`
	Anything  any                  `json:"anything"`
	Home      Address              `json:"home"`
	Optional  *Address             `json:"optional"`
	Stranger  Untagged             `json:"stranger"`
	Strangers []Untagged           `json:"strangers"`
	Skipped   string               `json:"-"`
	FromXML   string               `xml:"from_xml"`
	NoTag     string
	hidden    string
}

// Nested covers anonymous structs, which are inlined rather than referenced.
//
// @schema
type Nested struct {
	// @field { @description An inline object }
	Inline struct {
		// @field { @description Dropped: nested annotations are not applied }
		Name string `json:"name"`
	} `json:"inline"`

	// @field { @description A list of inline objects }
	Items []struct {
		Name string `json:"name"`
	} `json:"items"`

	// @field { @description A map of inline objects }
	Values map[string]struct {
		Name string `json:"name"`
	} `json:"values"`

	Empty struct{} `json:"empty"`
}

// Loop embeds Ring back, so flattening Ring has to terminate.
type Loop struct {
	*Ring
	Label string `json:"label"`
}

// Ring embeds a struct that embeds Ring again.
//
// @schema
type Ring struct {
	Loop
	Name string `json:"name"`
}

// Paging is embedded by Filters.
type Paging struct {
	Offset *int `query:"offset"`
}

// Filters covers parameter tag handling.
//
// @query
type Filters struct {
	Paging

	// @field { @description Search term }
	Q string `query:"q,required"`

	Limit    *int     `query:"limit"`
	Ignored  string   `query:"-"`
	Untagged string   //nolint:unused // resolves under its Go name
	List     []string `query:"list"`
}

// Key carries the path parameter.
//
// @path
type Key struct {
	ID string `path:"id"`
}

// List returns shapes.
//
//	@endpoint GET /shapes/{id} {
//	  @path Key
//	  @query Filters
//	  @response 200 { @body Shapes }
//	}
func List() {}

// Create stores a shape. Its request binds into a wrapper that does not
// exist; its response binds into one that does.
//
//	@endpoint POST /shapes {
//	  @request { @body Address @bind Missing.Data }
//	  @response 201 { @body Address @bind Address.Street }
//	}
func Create() {}

// Mixed declares responses in both places, including a status both declare.
//
//	@endpoint GET /mixed {
//	  @response 200 { @body Address @description From the block }
//	  @response 500 { @contentType empty @description Bodyless }
//	}
func Mixed() {
	// @response 404
	var notFound struct {
		Code string `json:"code"`
	}

	// @response 200
	var conflicting struct {
		X string `json:"x"`
	}

	_, _ = notFound, conflicting
}

func (s Shapes) unused() string { return s.hidden }
