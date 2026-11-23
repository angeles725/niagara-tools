package box

import (
	"errors"
	"strings"
)

// ReadRetryDelay is the pause, in seconds, before the one retry of a read that timed out.
const ReadRetryDelay = 0.5

// A read is asked again once when its request timed out (client A history rollout retro
// 2026-11-10 row 6: a dry run's read hit "timed out" over a metered tunnel and the identical retry
// succeeded). Only reads are asked again: the GET of an ORD (a BQL query) and the loadSlots
// request of a load, which ask the station for data and change nothing on it. A write (syncTo,
// invokeAction, save) is never sent twice by the client; it goes through Call and SSC, whose
// failure stays final. Only a timeout is retried: an HTTP status, a refused connection or a
// malformed reply is final, and so is an interrupted request.

// timedOut reports whether err is a request that got no answer in time: the transport failure
// text of a read timeout, bare or inside a BoxError.
func timedOut(err error) bool {
	var tf *transportFailure
	if errors.As(err, &tf) {
		return strings.Contains(tf.text, "timed out")
	}
	var e *Error
	return errors.As(err, &e) && e.Kind == KindBox && !e.Station &&
		strings.Contains(e.Msg, "transport failure") && strings.Contains(e.Msg, "timed out")
}

// retryRead is true, after ReadRetryDelay, when err is a timeout and the client was not
// interrupted: the caller then sends its read once more.
func (c *Client) retryRead(err error) bool {
	if !timedOut(err) || c.interrupt.Err() != nil {
		return false
	}
	c.Sleep(ReadRetryDelay)
	return true
}

// loadSlotsRequest sends loadSlots for ord; loadSlots only asks for data, so it is asked again once
// after a timeout (retryRead).
func (c *Client) loadSlotsRequest(ord string, depth int) error {
	_, err := c.SSC("loadSlots", obj("o", ord, "d", depth))
	if c.retryRead(err) {
		_, err = c.SSC("loadSlots", obj("o", ord, "d", depth))
	}
	return err
}
