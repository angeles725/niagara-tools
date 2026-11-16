package box

import "errors"

// RegistryRetryDelay is the pause, in seconds, before the one retry of a registry call that got
// no answer.
const RegistryRetryDelay = 0.5

// RegistryNoAnswer opens the error of a registry call that got no answer twice.
const RegistryNoAnswer = "the station's type registry did not answer (it may have just started): retry"

// callRegistry is the registry channel's loadTypes call, asked again once, after
// RegistryRetryDelay, when the first request got no answer: an HTTP error status, a transport
// failure or a malformed reply, as right after a station starts or a module first loads (client
// A retro 2026-11-16 row M13). An answer is never asked again: the station's own error reply, an
// entry saying a type is unknown ({"isErr": true}) or an AuthError (a failed login is never
// retried); nor is an interrupted request. Two requests without an answer are a BoxError opening
// with RegistryNoAnswer and naming the last failure. loadTypes only reads type definitions, so
// asking twice changes nothing on the station.
func (c *Client) callRegistry(body any) (any, error) {
	reply, err := c.Call(RegistryChannel, "loadTypes", body)
	if !noAnswer(err) || c.interrupt.Err() != nil {
		return reply, err
	}
	c.Sleep(RegistryRetryDelay)
	reply, err = c.Call(RegistryChannel, "loadTypes", body)
	if noAnswer(err) {
		return nil, boxError(RegistryNoAnswer+" (last failure: "+err.Error()+")", RegistryChannel, "loadTypes")
	}
	return reply, err
}

// noAnswer reports whether err is a request the station did not answer: a BoxError that is not
// its own error reply. Anything else (no error, an AuthError, another error) is final.
func noAnswer(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Kind == KindBox && !e.Station
}
