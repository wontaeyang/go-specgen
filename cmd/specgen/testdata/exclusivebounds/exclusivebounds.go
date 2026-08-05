//	@api {
//	  @title ExclusiveBounds Fixture
//	  @version 1.0.0
//	}
package exclusivebounds

// Metric demonstrates exclusive numeric bounds, which render differently
// in OpenAPI 3.0 (boolean flags) and 3.1 (numeric values).
//
//	@schema {
//	  @description A measured metric
//	}
type Metric struct {
	// @field { @description Ratio strictly between 0 and 1 @exclusiveMinimum 0 @exclusiveMaximum 1 }
	Ratio float64 `json:"ratio"`

	// @field { @description Inclusive count @minimum 0 @maximum 100 }
	Count int `json:"count"`
}

// GetMetric returns a metric.
//
//	@endpoint GET /metric {
//	  @response 200 { @body Metric @description Metric values }
//	}
func GetMetric() {}
