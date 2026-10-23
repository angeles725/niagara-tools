// Package box is the read side of mcp_n4/box.py: the stdlib-only Niagara BOX JSON client. It
// sends the same HTTP requests as the Python BoxClient, maps the same auth, HTTP and transport
// failures to the same error texts, polls with the same delays and returns the same load_tree
// result. Every JSON body goes through pyjson, so request bytes equal json.dumps output.
package box

import "fmt"

// authHints are box.AUTH_HINTS: what to check after a 401/403. A failed login is never retried.
var authHints = map[int]string{
	401: "check the user's Authentication Scheme Name = HTTPBasicScheme and the password; " +
		"do not retry (lockout: 5 failures in 30 s)",
	403: "the user is authenticated but lacks permission on this resource; " +
		"do not retry (lockout: 5 failures in 30 s)",
}

// AuthMessage is box.auth_message: the AuthError text for HTTP code (401/403).
func AuthMessage(code int) string {
	msg := fmt.Sprintf("HTTP %d from station", code)
	if hint, ok := authHints[code]; ok {
		msg += ": " + hint
	}
	return msg
}

// Kind tells the Python class an *Error stands for.
type Kind int

const (
	// KindBox is box.BoxError.
	KindBox Kind = iota
	// KindAuth is box.AuthError (HTTP 401/403), never retried.
	KindAuth
	// KindHTTPStatus is box.HttpStatusError: a 403/404 on /file/, no failed login.
	KindHTTPStatus
)

// Error is a BoxError, AuthError or HttpStatusError. Msg is str(exc); Channel and Key are empty
// where Python's are None (a refused redirect); Code is set for KindHTTPStatus only.
type Error struct {
	Kind         Kind
	Msg          string
	Channel, Key string
	Code         int
}

func (e *Error) Error() string { return e.Msg }

// PyType is the Python class name.
func (e *Error) PyType() string {
	return [...]string{"BoxError", "AuthError", "HttpStatusError"}[e.Kind]
}

// IsAuth reports whether e is an AuthError.
func (e *Error) IsAuth() bool { return e.Kind == KindAuth }

func boxError(msg, channel, key string) *Error {
	return &Error{Kind: KindBox, Msg: msg, Channel: channel, Key: key}
}

// PyError is a Python exception the BoxClient lets escape without wrapping it (a TypeError or
// KeyError on a malformed station reply, an http.client error that is no OSError). The server
// answers it as "internal error (<Type>)", so only the type is kept.
type PyError struct{ Type string }

func (e *PyError) Error() string { return e.Type }

// PyType is the Python class name.
func (e *PyError) PyType() string { return e.Type }
