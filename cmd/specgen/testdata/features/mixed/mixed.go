// Package mixed combines named and inline declarations on the same endpoint.
//
// No example package does this — all 14 use one style or the other — so the
// merge and ordering rules between named and inline parameters and responses
// are otherwise untested.
//
//	@api {
//	  @title Mixed Fixture
//	  @version 1.0.0
//	  @description Named and inline declarations on one endpoint.
//	  @defaultContentType json
//	}
package mixed

import "net/http"

// Report is the named response body.
// @schema
type Report struct {
	// @field { @description Report identifier }
	ID string `json:"id"`
}

// ReportPath carries the named path parameter.
// @path
type ReportPath struct {
	// @field { @description Report identifier }
	ID string `path:"id"`
}

// ReportQuery carries the named query parameters.
// @query
type ReportQuery struct {
	// @field { @description Include archived reports }
	IncludeArchived *bool `query:"include_archived"`
}

// GetReport mixes a named path struct and a named query struct with inline
// header parameters, a named 200 response, and an inline 404 response.
//
//	@endpoint GET /reports/{id} {
//	  @summary Get a report
//	  @path ReportPath
//	  @query ReportQuery
//	  @response 200 { @body Report }
//	}
func GetReport(w http.ResponseWriter, r *http.Request) {
	// @query
	var extra struct {
		// @field { @description Response format }
		Format *string `query:"format"`
	}

	// @header
	var headers struct {
		// @field { @description Correlation identifier @format uuid }
		RequestID string `header:"X-Request-ID"`
	}

	// @response 404
	var notFound struct {
		// @field { @description Error message }
		Message string `json:"message"`
	}

	_, _, _ = extra, headers, notFound
}
