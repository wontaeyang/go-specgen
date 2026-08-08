// Package specerr holds the error types the pipeline stages share.
//
// It exists because all three stages that report problems in an annotated
// package want the same two things — a path saying which declaration is at
// fault, and a list that accumulates rather than stopping at the first — and
// the dependency runs parser <- resolver <- validator, so none of them can
// import the type from another. A package below all three is the only place it
// can live once more than one stage needs it.
//
// Nothing here knows about annotations, Go types, or OpenAPI. It is the shape
// of an error report and nothing else.
package specerr

import (
	"fmt"
	"strings"
)

// Error is one problem found in an annotated package.
//
// Path names the declaration in the vocabulary the user wrote, not Go's:
// "@schema[Widget].ID", "@endpoint[GET /widgets]", "@query[ListQuery]". It is
// what turns a message into something you can act on without reading the
// source to work out where it came from.
type Error struct {
	Path    string
	Message string
}

func (e *Error) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("%s: %s", e.Path, e.Message)
	}
	return e.Message
}

// List accumulates the errors from one pipeline stage.
//
// Stage is the noun that appears in the header — "parse", "resolve",
// "validation" — so a caller can tell which stage rejected the package without
// the messages having to repeat it.
type List struct {
	Stage  string
	Errors []error
}

// Add records a problem at path.
func (l *List) Add(path, message string) {
	l.Errors = append(l.Errors, &Error{Path: path, Message: message})
}

// Addf records a problem at path, formatting the message.
func (l *List) Addf(path, format string, args ...any) {
	l.Add(path, fmt.Sprintf(format, args...))
}

// Append records an error that already knows its own path, such as one built
// deeper in a stage and handed up unchanged.
func (l *List) Append(err error) {
	l.Errors = append(l.Errors, err)
}

// Wrap records an error produced elsewhere, attributing it to path.
//
// The wrapped error's text becomes the message, so a failure raised deep in the
// annotation grammar arrives labeled with the declaration it came from rather
// than with the chain of calls that found it.
func (l *List) Wrap(path string, err error) {
	l.Add(path, err.Error())
}

// Len reports how many errors have accumulated. Callers use it to decide
// whether a later pass can safely run on what this one produced.
func (l *List) Len() int { return len(l.Errors) }

// Err returns the list as an error, or nil when nothing accumulated.
//
// Returning the *List directly would hand back a non-nil error interface
// holding a nil-behaving value, which is the classic way an "if err != nil"
// starts firing on success.
func (l *List) Err() error {
	if len(l.Errors) == 0 {
		return nil
	}
	return l
}

// Error renders the accumulated errors.
//
// A single error renders bare: a numbered list of one reads as ceremony, and
// most failures are single. Two or more get a count and numbers, because
// "which of these am I looking at" is the first question a list raises.
func (l *List) Error() string {
	if len(l.Errors) == 1 {
		return l.Errors[0].Error()
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "%d %s errors:\n", len(l.Errors), l.Stage)
	for i, err := range l.Errors {
		fmt.Fprintf(&sb, "  %d. %s\n", i+1, err.Error())
	}
	return sb.String()
}

// Unwrap exposes the accumulated errors to errors.Is and errors.As, so a
// caller can ask whether any one of them is a particular error without
// knowing the list is there.
func (l *List) Unwrap() []error { return l.Errors }
