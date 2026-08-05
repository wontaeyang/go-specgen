//	@api {
//	  @title Inline Only Example
//	  @version 1.0.0
//	  @description Demonstrates an API defined entirely from in-function structs.
//	  No package-level schema is declared, so nothing is registered under
//	  components and every body is emitted inline at its use site.
//	  @defaultContentType json
//	}
package inlineonly

import "net/http"

// Health reports service status.
//
//	@endpoint GET /health {
//	  @operationID health
//	  @summary Health check
//	}
func Health(w http.ResponseWriter, r *http.Request) {
	// @response 200
	var ok struct {
		// @field { @description Service status @enum ok,degraded }
		Status string `json:"status"`

		// @field { @description Seconds since start @minimum 0 }
		UptimeSeconds int `json:"uptime_seconds"`
	}

	_ = ok
}

// Echo returns its request body.
//
//	@endpoint POST /echo {
//	  @operationID echo
//	  @summary Echo a message
//	}
func Echo(w http.ResponseWriter, r *http.Request) {
	// @request
	var req struct {
		// @field { @description Message to echo @minLength 1 }
		Message string `json:"message"`
	}

	// @response 200
	var resp struct {
		// @field { @description The echoed message }
		Message string `json:"message"`
	}

	_ = req
	_ = resp
}
