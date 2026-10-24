package box

import (
	"errors"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// This file is the Go side of the op language of mcp-n4-kit/tests/go_box.py: the same ops, the
// same result shapes, encoded with pyjson so a result line equals json.dumps of Python's.

func field(o *pyjson.Object, k string) any {
	v, _ := o.Get(k)
	return v
}

func intField(o *pyjson.Object, k string) int {
	n, ok := field(o, k).(pyjson.Number)
	if !ok {
		return 0
	}
	i, _ := n.BigInt()
	return int(i.Int64())
}

func floatField(o *pyjson.Object, k string) float64 {
	n, ok := field(o, k).(pyjson.Number)
	if !ok {
		return 0
	}
	f, _ := n.Float64()
	return f
}

func orNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// describe is go_box.describe: what of an error the comparison sees.
func describe(err error) *pyjson.Object {
	var be *Error
	var ve *pyjson.ValueError
	switch {
	case errors.As(err, &be):
		var code any
		if be.Kind == KindHTTPStatus {
			code = be.Code
		}
		return obj("type", be.PyType(), "message", be.Msg, "channel", orNil(be.Channel),
			"key", orNil(be.Key), "code", code)
	case errors.As(err, &ve):
		return obj("type", "ValueError", "message", ve.Msg)
	}
	return obj("type", "Exception", "go", err.Error())
}

// clientFrom is go_box.make_client.
func clientFrom(cfg *pyjson.Object, url string) (*Client, error) {
	c := Config{BaseURL: url, Username: "admin", Password: "secret", AllowHTTP: true,
		Timeout: floatField(cfg, "timeout"), LoadWait: floatField(cfg, "load_wait"),
		MaxReplyBytes: intField(cfg, "max_reply_bytes")}
	if v, ok := field(cfg, "username").(string); ok {
		c.Username = v
	}
	if v, ok := field(cfg, "password").(string); ok {
		c.Password = v
	}
	if v, ok := field(cfg, "allow_http").(bool); ok {
		c.AllowHTTP = v
	}
	if v, ok := field(cfg, "insecure_tls").(bool); ok {
		c.InsecureTLS = v
	}
	return New(c)
}

// runOps is go_box.run_ops: one result line per op.
func runOps(t *testing.T, c *Client, ops []any) []string {
	t.Helper()
	authErrors := []any{}
	c.OnAuthError = func(e *Error) { authErrors = append(authErrors, e.Msg) }
	var out []string
	for _, raw := range ops {
		op := raw.(*pyjson.Object)
		kind, _ := field(op, "op").(string)
		sleeps := []any{}
		c.Sleep = func(s float64) { sleeps = append(sleeps, s) }
		var res any
		var err error
		str := func(k string) string { s, _ := field(op, k).(string); return s }
		switch kind {
		case "open":
			res, err = c.Open()
		case "close":
			c.Close()
		case "call":
			res, err = c.Call(str("channel"), str("key"), field(op, "body"))
		case "ssc":
			res, err = c.SSC(str("key"), field(op, "arg"))
		case "sync":
			res, err = c.Sync(field(op, "sync"))
		case "poll":
			res, err = c.Poll()
		case "load_tree":
			depth := 2
			if _, ok := op.Get("depth"); ok {
				depth = intField(op, "depth")
			}
			res, err = c.LoadTree(str("ord"), depth, LoadOptions{Attempts: intField(op, "attempts"),
				Delay: floatField(op, "delay"), Handle: field(op, "handle")})
		case "get_ord":
			var text string
			var cut bool
			text, cut, err = c.GetOrd(str("ord"), floatField(op, "timeout"), intField(op, "max_bytes"))
			res = []any{text, cut}
		case "get_file":
			var f *File
			if f, err = c.GetFile(str("path"), floatField(op, "timeout"), intField(op, "max_bytes")); err == nil {
				var ctype any
				if f.ContentType != nil {
					ctype = *f.ContentType
				}
				res = obj("body", hexOf(f.Body), "truncated", f.Truncated, "content_type", ctype,
					"content_length", f.ContentLength)
			}
		case "about":
			res, err = c.About(floatField(op, "timeout"), intField(op, "max_bytes"))
		case "drain":
			res = c.DrainPendingEvents()
		case "invalidate":
			if p, ok := field(op, "prefix").(string); ok {
				c.InvalidateHandles(p)
			} else {
				c.ClearHandles()
			}
		case "set_handle":
			c.SetHandle(str("ord"), field(op, "handle"))
		case "state":
			res = obj("sid", c.SID(), "handles", c.Handles(), "pending", c.PendingEvents(),
				"dropped", c.DroppedEvents(), "outstanding", c.LoadsOutstanding(),
				"auth_errors", append([]any{}, authErrors...))
		default:
			t.Fatalf("unknown op %q", kind)
		}
		entry := obj("ok", res)
		if err != nil {
			entry = obj("error", describe(err))
		}
		if kind == "load_tree" {
			entry.Set("sleeps", sleeps)
		}
		line, merr := pyjson.Marshal(entry, pyjson.Options{})
		if merr != nil {
			t.Fatalf("op %s: %v", kind, merr)
		}
		out = append(out, string(line))
	}
	return out
}

func hexOf(b []byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, 0, 2*len(b))
	for _, c := range b {
		out = append(out, digits[c>>4], digits[c&15])
	}
	return string(out)
}
