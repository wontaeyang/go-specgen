//	@api {
//	  @title Deprecated Fixture
//	  @version 1.0.0
//	}
package deprecated

// Legacy demonstrates deprecation at schema and field level.
//
//	@schema {
//	  @description A legacy record
//	  @deprecated
//	}
type Legacy struct {
	// @field { @description Old identifier @deprecated }
	Old string `json:"old"`

	// @field { @description Replacement identifier }
	New string `json:"new"`
}

// GetLegacy demonstrates endpoint-level deprecation.
//
//	@endpoint GET /legacy {
//	  @deprecated
//	  @response 200 { @body Legacy @description Legacy record }
//	}
func GetLegacy() {}
