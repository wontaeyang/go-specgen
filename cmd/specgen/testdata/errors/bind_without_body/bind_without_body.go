//	@api {
//	  @title Bind Without Body
//	  @version 1.0.0
//	}
package bind_without_body

// @bind names the field of an envelope the body binds into, so it has nothing to
// say without a body. parseBody used to return early on the missing @body and
// drop the @bind with it.
//
// A @request in this state is caught downstream by "missing @body". A @response
// is not: a response with no body is how 204 is spelled, so nothing else was
// ever going to ask, and this one rendered as an ordinary bodyless response with
// the annotation discarded.

// @schema
type Envelope struct {
	//	@field { @description The wrapped payload }
	Data string `json:"data"`
}

//	@endpoint GET /widgets {
//	  @response 204 { @description gone @bind Envelope.Data }
//	}
func ListWidgets() {}
