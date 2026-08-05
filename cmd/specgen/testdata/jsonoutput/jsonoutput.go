//	@api {
//	  @title JSONOutput Fixture
//	  @version 1.0.0
//	}
package jsonoutput

// Ping is a minimal schema for exercising JSON rendering.
//
//	@schema {
//	  @description Ping response
//	}
type Ping struct {
	// @field { @description Response message @example pong }
	Message string `json:"message"`
}

// GetPing returns a ping.
//
//	@endpoint GET /ping {
//	  @response 200 { @body Ping @description Pong }
//	}
func GetPing() {}
