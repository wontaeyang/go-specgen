//	@api {
//	  @title ReadWrite Fixture
//	  @version 1.0.0
//	}
package readwrite

// User demonstrates readOnly and writeOnly fields.
//
//	@schema {
//	  @description User account
//	}
type User struct {
	// @field { @description User ID @readOnly }
	ID string `json:"id"`

	// @field { @description Password @writeOnly }
	Password string `json:"password"`

	// @field { @description Display name }
	Name string `json:"name"`
}

// GetMe returns the current user.
//
//	@endpoint GET /me {
//	  @response 200 { @body User @description Current user }
//	}
func GetMe() {}
