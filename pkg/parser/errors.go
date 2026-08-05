package parser

import (
	"errors"
	"fmt"
	"go/token"
)

// Error is a parse error with the source position it came from. It renders as
// "file.go:12:3: message", or as the bare message when no position is known.
type Error struct {
	Pos token.Position
	Msg string
}

func (e *Error) Error() string {
	if !e.Pos.IsValid() {
		return e.Msg
	}
	return e.Pos.String() + ": " + e.Msg
}

// errorf builds a positioned error.
func errorf(pos token.Position, format string, args ...any) *Error {
	return &Error{Pos: pos, Msg: fmt.Sprintf(format, args...)}
}

// wrapf adds context to err. The innermost known position wins, so the result
// still renders as "file.go:12:3: context: detail" rather than burying the
// position in the middle of the message.
func wrapf(pos token.Position, err error, format string, args ...any) *Error {
	msg := fmt.Sprintf(format, args...)

	var inner *Error
	if errors.As(err, &inner) {
		if inner.Pos.IsValid() {
			pos = inner.Pos
		}
		return &Error{Pos: pos, Msg: msg + ": " + inner.Msg}
	}
	return &Error{Pos: pos, Msg: msg + ": " + err.Error()}
}
