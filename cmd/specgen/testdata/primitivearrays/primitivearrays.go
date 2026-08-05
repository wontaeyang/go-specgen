//	@api {
//	  @title PrimitiveArrays Fixture
//	  @version 1.0.0
//	}
package primitivearrays

import "time"

// Primitives locks the Go-type → OpenAPI-type mapping for arrays and
// maps over every primitive family, plus integer/float format handling.
//
//	@schema {
//	  @description Primitive type matrix
//	}
type Primitives struct {
	Int64s   []int64              `json:"int64s"`
	Float32s []float32            `json:"float32s"`
	Bools    []bool               `json:"bools"`
	Times    []time.Time          `json:"times"`
	Blob     []byte               `json:"blob"`
	IntMap   map[string]int64     `json:"int_map"`
	BoolMap  map[string]bool      `json:"bool_map"`
	TimeMap  map[string]time.Time `json:"time_map"`
	U32      uint32               `json:"u32"`
	I32      int32                `json:"i32"`
	F64      float64              `json:"f64"`
}

// GetPrimitives returns the matrix.
//
//	@endpoint GET /primitives {
//	  @response 200 { @body Primitives @description Primitive matrix }
//	}
func GetPrimitives() {}
