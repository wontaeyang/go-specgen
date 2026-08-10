//	@api {
//	  @title Security Scheme Errors
//	  @version 1.0.0
//	  @securityScheme zebra { @type http }
//	  @securityScheme alpha { @type apiKey }
//	  @securityScheme middle { @type bogus }
//	}
package security_scheme_errors

// Security schemes are held in a map, so the order their errors come out in is
// the order this file's expected_error.txt pins. The three are declared zebra,
// alpha, middle and reported alpha, middle, zebra: insertion order would not
// produce that, and an unsorted range produced a different answer run to run.
//
// alpha contributes two errors on its own, so the fixture pins the order within
// one scheme as well as across several.

// @schema
type Widget struct {
	//	@field { @description Widget ID }
	ID string `json:"id"`
}

//	@endpoint GET /widgets {
//	  @response 200 { @description ok @body Widget }
//	}
func ListWidgets() {}
