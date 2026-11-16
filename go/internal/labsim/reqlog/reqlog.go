// Package reqlog is the request log of lab-sim: one JSON line per request a
// simulated device received, with what was asked and what was answered, so a
// test can prove what a driver put on the wire without trusting the driver.
package reqlog

import (
	"encoding/json"
	"io"
	"sync"
)

// The values of Entry.Result.
const (
	ResultOK         = "ok"
	ResultException  = "exception"
	ResultNoResponse = "no_response"
)

// The values of Entry.Reason: why a request got an exception or no response.
const (
	ReasonUnknownUnit        = "unknown_unit"
	ReasonOffline            = "offline"
	ReasonIllegalFunction    = "illegal_function"
	ReasonIllegalDataAddress = "illegal_data_address"
	ReasonIllegalDataValue   = "illegal_data_value"
	ReasonDeviceFailure      = "device_failure"
)

// Entry is one request and its outcome.
type Entry struct {
	Time      string `json:"time"`       // when it was handled, RFC 3339 UTC
	ElapsedMS int64  `json:"elapsed_ms"` // since the simulator started
	Remote    string `json:"remote"`     // the client's address
	Unit      int    `json:"unit"`
	Device    string `json:"device,omitempty"` // the scenario name of the unit, when it has one
	Function  int    `json:"function"`
	Address   int    `json:"address"` // 0-based
	Quantity  int    `json:"quantity"`
	Written   []int  `json:"written,omitempty"` // the registers (or coils as 0/1) of a write
	Result    string `json:"result"`
	Exception int    `json:"exception,omitempty"`
	Reason    string `json:"reason,omitempty"`
	Read      []int  `json:"read,omitempty"`     // the registers (or bits as 0/1) a read returned
	DelayMS   int64  `json:"delay_ms,omitempty"` // how long the response was held
}

// Writer writes entries as JSON lines; a nil Writer drops them.
type Writer struct {
	mu  sync.Mutex
	enc *json.Encoder
}

// NewWriter logs to w, one whole line per entry even under concurrent use.
func NewWriter(w io.Writer) *Writer {
	return &Writer{enc: json.NewEncoder(w)}
}

// Log writes one entry. A write error is dropped: the log never stops the
// simulated device.
func (w *Writer) Log(e Entry) {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	_ = w.enc.Encode(e)
}
