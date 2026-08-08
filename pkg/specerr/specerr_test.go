package specerr

import (
	"errors"
	"testing"
)

func TestErrorText(t *testing.T) {
	tests := []struct {
		name string
		err  Error
		want string
	}{
		{
			name: "with path",
			err:  Error{Path: "@schema[Widget].ID", Message: "invalid pattern"},
			want: "@schema[Widget].ID: invalid pattern",
		},
		{
			name: "without path",
			err:  Error{Message: "missing @api annotation"},
			want: "missing @api annotation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestListError pins the rendered shape, which is product surface: it is what
// specgen prints when it cannot generate, and what the error fixtures compare
// byte-for-byte.
func TestListError(t *testing.T) {
	t.Run("single error renders bare", func(t *testing.T) {
		l := List{Stage: "parse"}
		l.Add("@schema[Widget]", "unknown annotation @bogus")

		want := "@schema[Widget]: unknown annotation @bogus"
		if got := l.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("multiple errors are counted and numbered", func(t *testing.T) {
		l := List{Stage: "validation"}
		l.Add("@schema[A].X", "first")
		l.Add("@schema[B].Y", "second")

		want := "2 validation errors:\n" +
			"  1. @schema[A].X: first\n" +
			"  2. @schema[B].Y: second\n"
		if got := l.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("stage names itself in the header", func(t *testing.T) {
		l := List{Stage: "resolve"}
		l.Add("@schema[A]", "first")
		l.Add("@schema[B]", "second")

		if got := l.Error(); got[:17] != "2 resolve errors:" {
			t.Errorf("Error() header = %q, want %q", got[:17], "2 resolve errors:")
		}
	})
}

// TestListErr covers the nil-interface trap: an empty list must return a nil
// error, not a non-nil interface holding an empty *List.
func TestListErr(t *testing.T) {
	empty := List{Stage: "parse"}
	if err := empty.Err(); err != nil {
		t.Errorf("Err() on empty list = %v, want nil", err)
	}

	filled := List{Stage: "parse"}
	filled.Add("@schema[A]", "boom")
	if err := filled.Err(); err == nil {
		t.Error("Err() on non-empty list = nil, want an error")
	}
}

func TestListAddfAndWrap(t *testing.T) {
	l := List{Stage: "parse"}
	l.Addf("@schema[A]", "unknown annotation %s in %s", "@bogus", "@schema")
	l.Wrap("@schema[B]", errors.New("underlying failure"))

	if got, want := l.Errors[0].Error(), "@schema[A]: unknown annotation @bogus in @schema"; got != want {
		t.Errorf("Addf produced %q, want %q", got, want)
	}
	if got, want := l.Errors[1].Error(), "@schema[B]: underlying failure"; got != want {
		t.Errorf("Wrap produced %q, want %q", got, want)
	}
}

// TestListUnwrap covers errors.Is reaching through the list, which is why
// Unwrap exists at all.
func TestListUnwrap(t *testing.T) {
	sentinel := errors.New("sentinel")

	l := List{Stage: "parse"}
	l.Add("@schema[A]", "unrelated")
	l.Errors = append(l.Errors, sentinel)

	if !errors.Is(l.Err(), sentinel) {
		t.Error("errors.Is did not find the sentinel through the list")
	}
}
