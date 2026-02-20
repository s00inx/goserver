package protocol

import "errors"

// errors for parsing
var (
	errInvalid    = errors.New("invalid request")
	errIncomplete = errors.New("incomplete request")
)
