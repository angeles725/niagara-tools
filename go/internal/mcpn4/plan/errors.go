package plan

import (
	"errors"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
)

// PyError is a Python exception the write tools describe by class and message, such as the
// TypeError of float(None) or the KeyError of a slot gone from a read-back. The server answers
// it as "internal error (<Type>)"; the audit log and a read-back error show "<Type>: <Msg>".
type PyError struct{ Type, Msg string }

func (e *PyError) Error() string { return e.Msg }

// PyType is the Python class name.
func (e *PyError) PyType() string { return e.Type }

// InDoubt is the in-doubt error of a journaled batch (_in_doubt): its ToolError text and the
// batch id the audit line records.
type InDoubt struct{ Msg, BatchID string }

func (e *InDoubt) Error() string { return e.Msg }

// inDoubt is _in_doubt: why is the failure, or a reason text the kit wrote (shown as is).
func inDoubt(batchID string, why error) *InDoubt {
	return &InDoubt{BatchID: batchID, Msg: "station call failed after the intent was journaled: batch " + batchID +
		" is in-doubt (the station may or may not have applied it; inspect it before retrying): " + ownWords(why)}
}

// ownWords is str(exc) for the station's or the kit's own errors and the class name otherwise.
func ownWords(err error) string {
	var te *jsonrpc.ToolError
	var se *safety.Error
	var be *box.Error
	if errors.As(err, &te) || errors.As(err, &se) || errors.As(err, &be) {
		return PyStr(err)
	}
	return PyType(err)
}

// PyType is type(exc).__name__ of an error a write can meet.
func PyType(err error) string {
	var te *jsonrpc.ToolError
	var se *safety.Error
	var ve *pyjson.ValueError
	var oe *pyjson.OverflowError
	var typed jsonrpc.PyTyped
	switch {
	case errors.As(err, &te):
		return "ToolError"
	case errors.As(err, &se):
		return "SafetyError"
	case errors.As(err, &ve):
		return "ValueError"
	case errors.As(err, &oe):
		return "OverflowError"
	case errors.As(err, &typed): // box errors, PyError, journal.PermissionError
		return typed.PyType()
	}
	return "Exception"
}

// PyStr is str(exc).
func PyStr(err error) string {
	var te *jsonrpc.ToolError
	var se *safety.Error
	var be *box.Error
	var ve *pyjson.ValueError
	switch {
	case errors.As(err, &te):
		return te.Msg
	case errors.As(err, &se):
		return se.Msg
	case errors.As(err, &be):
		return be.Msg
	case errors.As(err, &ve):
		return ve.Msg
	}
	return err.Error()
}

// TypeName is type(v).__name__ of a decoded value.
func TypeName(v any) string {
	switch x := v.(type) {
	case nil:
		return "NoneType"
	case bool:
		return "bool"
	case string:
		return "str"
	case pyjson.Number:
		if x.IsFloat() {
			return "float"
		}
		return "int"
	case float64:
		return "float"
	case []any:
		return "list"
	}
	return "dict"
}
