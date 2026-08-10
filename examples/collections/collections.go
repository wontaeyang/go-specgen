// Package collections covers nested container types.
//
// Groups is bug #1: the resolver reduces map value types by string surgery, so
// map[string][]Item loses its array-ness and renders as a bare $ref.
//
//	@api {
//	  @title Collections Fixture
//	  @version 1.0.0
//	  @description Exercises maps and slices, including nested combinations.
//	  @defaultContentType json
//	}
package collections

import (
	"net/http"
	"time"
)

// Item is referenced from inside containers.
// @schema
type Item struct {
	// @field { @description Item identifier }
	ID string `json:"id"`
}

// Containers holds one field per container shape.
// @schema
type Containers struct {
	// @field { @description Items keyed by ID }
	ByID map[string]Item `json:"by_id"`

	// @field { @description Item lists keyed by group name }
	Groups map[string][]Item `json:"groups"`

	// @field { @description String lists keyed by name }
	Tags map[string][]string `json:"tags"`

	// @field { @description List of string maps }
	Buckets []map[string]string `json:"buckets"`

	// @field { @description Nested maps }
	Nested map[string]map[string]string `json:"nested"`

	// @field { @description List of lists }
	Matrix [][]int `json:"matrix"`

	// @field { @description List of items }
	Items []Item `json:"items"`

	// Element formats: nothing anywhere else uses a formatted scalar inside a
	// container, so this is the only coverage of whether the format survives.

	// @field { @description Identifiers }
	IDs []int64 `json:"ids"`

	// @field { @description Event times }
	Times []time.Time `json:"times"`

	// @field { @description Sizes keyed by name }
	Sizes map[string]int64 `json:"sizes"`

	// @field { @description Ratios }
	Ratios []float32 `json:"ratios"`
}

// GetContainers returns the container sampler.
//
//	@endpoint GET /containers {
//	  @summary Get containers
//	  @response 200 { @body Containers }
//	}
func GetContainers(w http.ResponseWriter, r *http.Request) {}
