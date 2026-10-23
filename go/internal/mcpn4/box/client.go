package box

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The box.py constants the read side uses.
const (
	ProtocolVersion    = "2.3"
	Channel            = "ssession"
	SessionComponentID = "cs1"    // the session component makessc creates and callssc addresses
	DefaultRootHandle  = "2"      // what loadRoot answers; pass root["h"] rather than rely on it
	MaxPendingEvents   = 500      // events kept in PendingEvents; the oldest go first
	FirstPollDelay     = 0.1      // seconds: the first sleep of a load polling window
	MaxPollDelay       = 0.8      // seconds: the cap of one sleep
	MaxPollWait        = 3.0      // seconds: the default total sleep of one polling window
	DefaultTimeout     = 20.0     // seconds: the socket timeout of every request
	MaxReplyBytes      = 32 << 20 // byte cap of one BOX reply (the same 32 MB as GetOrd)
)

// Config is the BoxClient constructor's arguments. Zero Timeout and LoadWait take the Python
// defaults (20 s, MaxPollWait); a zero MaxReplyBytes takes MaxReplyBytes.
type Config struct {
	BaseURL, Username, Password string
	Timeout                     float64
	InsecureTLS, AllowHTTP      bool
	LoadWait                    float64
	MaxReplyBytes               int
}

// Client is box.BoxClient. It is not safe for concurrent use, like the Python client.
type Client struct {
	baseURL, username, auth string
	timeout                 float64
	insecureTLS             bool
	// LoadWait is the total sleep of one load polling window (--load-wait).
	LoadWait float64
	// MaxReplyBytes caps one BOX reply: a longer one is refused, never parsed in part.
	MaxReplyBytes int
	// OnAuthError is called with each AuthError before it is returned (the server latches on it
	// even where a caller swallows the error). It never changes what the caller sees.
	OnAuthError func(*Error)
	// Sleep pauses between load polls (seconds); tests replace it to record the schedule.
	Sleep func(seconds float64)

	seq              int
	sid              any
	pending          []any
	dropped          int
	loadsOutstanding int
	handles          handleCache
}

// New is BoxClient(...): it refuses an http:// URL unless AllowHTTP, and any other scheme,
// with Python's ValueError text.
func New(cfg Config) (*Client, error) {
	if strings.HasPrefix(cfg.BaseURL, "http://") && !cfg.AllowHTTP {
		return nil, &pyjson.ValueError{Msg: "refusing http:// (password would travel in clear); use https"}
	}
	if !strings.HasPrefix(cfg.BaseURL, "http://") && !strings.HasPrefix(cfg.BaseURL, "https://") {
		return nil, &pyjson.ValueError{Msg: "base_url must start with https://"}
	}
	c := &Client{baseURL: strings.TrimRight(cfg.BaseURL, "/"), username: cfg.Username,
		timeout: cfg.Timeout, insecureTLS: cfg.InsecureTLS, LoadWait: cfg.LoadWait,
		MaxReplyBytes: cfg.MaxReplyBytes,
		auth:          "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.Username+":"+cfg.Password)),
		Sleep:         func(s float64) { time.Sleep(time.Duration(s * float64(time.Second))) }}
	if c.timeout <= 0 {
		c.timeout = DefaultTimeout
	}
	if c.LoadWait <= 0 {
		c.LoadWait = MaxPollWait
	}
	if c.MaxReplyBytes <= 0 {
		c.MaxReplyBytes = MaxReplyBytes
	}
	return c, nil
}

// String is repr(client): never the password.
func (c *Client) String() string {
	return fmt.Sprintf("BoxClient(base_url=%s, username=%s)", pyjson.StrRepr(c.baseURL),
		pyjson.StrRepr(c.username))
}

// BaseURL is the station URL without trailing slashes.
func (c *Client) BaseURL() string { return c.baseURL }

// SID is the session id ("make" result), nil when no session is open.
func (c *Client) SID() any { return c.sid }

func (c *Client) authError(code int, channel, key string) *Error {
	e := &Error{Kind: KindAuth, Msg: AuthMessage(code), Channel: channel, Key: key}
	if c.OnAuthError != nil {
		c.OnAuthError(e)
	}
	return e
}

// Call is BoxClient.call: one BOX frame, the reply's first message body.
func (c *Client) Call(channel, key string, body any) (any, error) {
	c.seq++
	frame := obj("p", "box", "v", ProtocolVersion,
		"m", []any{obj("r", 0, "t", "rt", "c", channel, "k", key, "b", body)}, "n", c.seq)
	if truthy(c.sid) {
		frame.Set("id", c.sid)
	}
	data, err := pyjson.Marshal(frame, pyjson.Options{})
	if err != nil {
		return nil, err
	}
	resp, err := c.do("POST", "/box/", data, map[string]string{"Content-Type": "application/json"},
		0, c.MaxReplyBytes+1)
	if err != nil {
		return nil, c.mapError(err, channel, key, false)
	}
	if len(resp.body) > c.MaxReplyBytes {
		return nil, boxError(fmt.Sprintf("BOX reply larger than %d bytes: refused, not parsed in part",
			c.MaxReplyBytes), channel, key)
	}
	reply, err := pyjson.Loads(resp.body)
	if err != nil {
		return nil, boxError("transport failure: "+err.Error(), channel, key)
	}
	if nestsDeeper(reply, MaxReplyNesting) {
		return nil, malformed(fmt.Sprintf("nested deeper than %d levels", MaxReplyNesting), channel, key)
	}
	return c.unwrap(reply, channel, key)
}

// mapError is the except chain shared by every request: a refused redirect as it is, 401/403
// as AuthError, other statuses as BoxError (badRequest: get_ord's 400 wording), transport
// failures as "transport failure: ...".
func (c *Client) mapError(err error, channel, key string, badRequest bool) error {
	var st *httpStatus
	var tf *transportFailure
	switch {
	case errors.As(err, &st) && (st.code == 401 || st.code == 403):
		return c.authError(st.code, channel, key)
	case errors.As(err, &st) && st.code == 400 && badRequest:
		return boxError("HTTP 400 from station: the ORD or BQL query was rejected "+
			"(check the type spec, column names and where clause)", channel, key)
	case errors.As(err, &st):
		return boxError(fmt.Sprintf("HTTP %d from station", st.code), channel, key)
	case errors.As(err, &tf):
		return boxError("transport failure: "+tf.text, channel, key)
	}
	return err
}

// unwrap is BoxClient._unwrap: validate a reply frame, return its first message body.
func (c *Client) unwrap(reply any, channel, key string) (any, error) {
	r, ok := reply.(*pyjson.Object)
	if !ok {
		return nil, boxError("malformed reply: not an object", channel, key)
	}
	if n, ok := r.Get("n"); ok && !pyEqual(n, c.seq) {
		return nil, boxError(fmt.Sprintf("reply seq %s does not match request seq %d", pyjson.Repr(n), c.seq),
			channel, key)
	}
	m, _ := r.Get("m")
	msgs, ok := m.([]any)
	if !ok || len(msgs) == 0 {
		return nil, boxError("malformed reply: missing or empty message list", channel, key)
	}
	msg, ok := msgs[0].(*pyjson.Object)
	if !ok {
		return nil, boxError("malformed reply: missing or empty message list", channel, key)
	}
	if t, _ := msg.Get("t"); t == "e" {
		b, _ := msg.Get("b")
		text, ok := get(b, "m")
		if s, isStr := text.(string); ok && isStr {
			return nil, boxError(s, channel, key)
		}
		return nil, boxError("station error", channel, key)
	}
	b, _ := msg.Get("b")
	return b, nil
}
