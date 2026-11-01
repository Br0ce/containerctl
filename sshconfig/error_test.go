package sshconfig

import (
	"errors"
	"testing"
)

func TestParseError(t *testing.T) {
	cause := errors.New("unterminated quote")
	tests := []struct {
		err  *ParseError
		want string
	}{
		{&ParseError{Line: 3, Err: cause}, "line 3: unterminated quote"},
		{&ParseError{File: "config", Line: 3, Err: cause}, "config:3: unterminated quote"},
	}
	for _, tt := range tests {
		if got := tt.err.Error(); got != tt.want {
			t.Errorf("Error() = %q, want %q", got, tt.want)
		}
		if !errors.Is(tt.err, cause) {
			t.Errorf("errors.Is(%v, cause) = false, want true", tt.err)
		}
	}
}
