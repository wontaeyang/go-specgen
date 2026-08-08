// Package constraints covers the @field constraint annotations that no example
// package uses, so a change to constraint emission cannot pass unnoticed.
//
//	@api {
//	  @title Constraints Fixture
//	  @version 1.0.0
//	  @description Exercises every field constraint annotation.
//	  @defaultContentType json
//	}
package constraints

import "net/http"

// Measurement carries every numeric and array constraint.
// @schema
type Measurement struct {
	// @field { @description Server-assigned identifier @readOnly }
	ID string `json:"id"`

	// @field { @description Strictly positive reading @exclusiveMinimum 0 }
	Reading float64 `json:"reading"`

	// @field { @description Ratio strictly below one @exclusiveMaximum 1 }
	Ratio float64 `json:"ratio"`

	// @field { @description Open on both sides @exclusiveMinimum 0 @exclusiveMaximum 100 }
	Percent float64 `json:"percent"`

	// @field { @description Inclusive bounds, for contrast @minimum 0 @maximum 100 }
	Score int `json:"score"`

	// @field { @description Distinct labels @uniqueItems @minItems 1 @maxItems 10 }
	Labels []string `json:"labels"`

	// @field { @description Bounded list without the uniqueness flag @minItems 2 @maxItems 4 }
	Samples []int `json:"samples"`

	// @field { @description Allowed sample counts @enum 1,5,10 }
	AllowedCounts []int `json:"allowed_counts"`

	// @field { @description Allowed units @enum mm,cm,m }
	AllowedUnits []string `json:"allowed_units"`

	// @field { @description Calibration secret, accepted but never returned @writeOnly }
	CalibrationKey string `json:"calibration_key"`
}

// MeasurementQuery filters measurements, with the same enum shapes as a parameter.
// @query
type MeasurementQuery struct {
	// @field { @description Filter by sample count @enum 1,5,10 }
	Counts []int `query:"counts"`

	// @field { @description Filter by unit @enum mm,cm,m }
	Units []string `query:"units"`
}

// ListMeasurements returns measurements.
//
//	@endpoint GET /measurements {
//	  @summary List measurements
//	  @query MeasurementQuery
//	  @response 200 { @body []Measurement }
//	}
func ListMeasurements(w http.ResponseWriter, r *http.Request) {}
