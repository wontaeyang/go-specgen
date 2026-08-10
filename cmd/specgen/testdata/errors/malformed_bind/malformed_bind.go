//	@api {
//	  @title Malformed Bind
//	  @version 1.0.0
//	}
package malformed_bind

// @bind names an envelope schema and the field of it the body binds into. A
// value with no field names no target, and returning nothing for it was
// indistinguishable from writing no @bind at all: the body rendered bare, the
// envelope was dropped, and the run exited 0.
//
// Every other way to get @bind wrong is already reported -- an unknown wrapper,
// a field the wrapper does not have -- so this was the one that stayed quiet,
// and it is the one a typo produces.

// @schema
type Envelope struct {
	//	@field { @description The wrapped payload }
	Data string `json:"data"`
}

// @schema
type Widget struct {
	//	@field { @description Widget ID }
	ID string `json:"id"`
}

//	@endpoint GET /widgets {
//	  @response 200 { @description ok @body Widget @bind Envelope }
//	}
func ListWidgets() {}
