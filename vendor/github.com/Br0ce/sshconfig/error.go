package sshconfig

import "fmt"

// ParseError reports an invalid line in a config file or one of its includes.
type ParseError struct {
	File string // file name, empty for a config read by Decode
	Line int
	Err  error
}

func (e *ParseError) Error() string {
	if e.File == "" {
		return fmt.Sprintf("line %d: %v", e.Line, e.Err)
	}
	return fmt.Sprintf("%s:%d: %v", e.File, e.Line, e.Err)
}

func (e *ParseError) Unwrap() error {
	return e.Err
}
