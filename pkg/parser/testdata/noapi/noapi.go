// Package noapi has no @api annotation anywhere.
package noapi

// Thing is annotated, but without an @api block the package cannot be parsed.
// @schema
type Thing struct {
	// @field { @description Identifier }
	ID string `json:"id"`
}
