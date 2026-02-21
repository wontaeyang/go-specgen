//	@api {
//	  @title Closure Handler Example
//	  @version 1.0.0
//	  @description Demonstrates handler factory pattern with inline structs.
//	  @defaultContentType json
//	}
package closure

import "net/http"

// @schema
type User struct {
	// @field { @description User ID @format uuid }
	ID string `json:"id"`

	// @field { @description User name }
	Name string `json:"name"`

	// @field { @description User email @format email }
	Email string `json:"email"`
}

// HandleGreet demonstrates the closure handler pattern.
// Request struct is in the outer function, response struct is in the returned handler.
//
//	@endpoint POST /greet {
//	  @summary Greet a user
//	}
func HandleGreet() http.HandlerFunc {
	// @request
	type request struct {
		// @field { @description Name of the person to greet }
		Name string `json:"name"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// @response 200
		type response struct {
			// @field { @description Greeting message }
			Greeting string `json:"greeting"`
		}

		_ = response{}
	}
}
