//	@api {
//	  @title Brace In Value
//	  @version 1.0.0
//	}
package brace_in_value

// A block ends on a line of its own. A value that carries an unescaped brace
// therefore looks like the end of the block, and everything under it would be
// dropped — so it is reported rather than obeyed.
//
// @pattern escapes this because its value is raw: see examples/rawvalue. Every
// other value must write \} for a literal brace.
//
// @schema
type Widget struct {
	// @field {
	//   @description a closing } brace, unescaped
	//   @format uuid
	// }
	ID string `json:"id"`
}
