package mudrex

import (
	"testing"
)

func TestMudrexAPIErrorStringWithCode(t *testing.T) {
	err := &MudrexAPIError{MudrexError: MudrexError{Message: "Bad request"}, Code: 400}
	if err.Error() != "[400] Bad request" {
		t.Fatalf("got %q", err.Error())
	}
}

func TestMudrexAPIErrorStringWithoutCode(t *testing.T) {
	err := &MudrexAPIError{MudrexError: MudrexError{Message: "Something failed"}}
	if err.Error() != "Something failed" {
		t.Fatalf("got %q", err.Error())
	}
}

func TestMudrexAPIErrorAttributes(t *testing.T) {
	err := &MudrexAPIError{
		MudrexError: MudrexError{Message: "msg"},
		Code:        500,
		Body:        "resp_obj",
	}
	if err.Message != "msg" || err.Code != 500 || err.Body != "resp_obj" {
		t.Fatalf("unexpected attributes: %+v", err)
	}
}

func TestMudrexAPIErrorIsMudrexError(t *testing.T) {
	var base *MudrexError
	err := &MudrexAPIError{MudrexError: MudrexError{Message: "x"}}
	if _, ok := any(err).(*MudrexAPIError); !ok {
		t.Fatal("expected MudrexAPIError")
	}
	_ = base
}

func TestMudrexRequestErrorAttributes(t *testing.T) {
	orig := &MudrexRequestError{MudrexError: MudrexError{Message: "conn"}}
	err := &MudrexRequestError{
		MudrexError:   MudrexError{Message: "Network fail"},
		OriginalError: orig,
	}
	if err.Message != "Network fail" || err.OriginalError != orig {
		t.Fatalf("unexpected attributes: %+v", err)
	}
}

func TestMudrexRequestErrorIsMudrexError(t *testing.T) {
	err := &MudrexRequestError{MudrexError: MudrexError{Message: "x"}}
	if err.Message != "x" {
		t.Fatalf("got %q", err.Message)
	}
}
