package mudrex

import "fmt"

// MudrexError is the base error type for the Mudrex SDK.
type MudrexError struct {
	Message string
}

func (e *MudrexError) Error() string {
	return e.Message
}

// MudrexAPIError is returned when the Mudrex API responds with an error.
type MudrexAPIError struct {
	MudrexError
	Code       int
	StatusCode int
	Body       string
}

func (e *MudrexAPIError) Error() string {
	if e.Code != 0 {
		return fmt.Sprintf("[%d] %s", e.Code, e.Message)
	}
	return e.Message
}

// MudrexRequestError is returned on network or connection failures.
type MudrexRequestError struct {
	MudrexError
	OriginalError error
}

func (e *MudrexRequestError) Error() string {
	return e.Message
}

func (e *MudrexRequestError) Unwrap() error {
	return e.OriginalError
}
