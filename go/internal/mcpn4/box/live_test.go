package box

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// liveCase is one HTTP-level scenario: the Python BoxClient (go_box.py --live, until S17) and the Go client
// run the same ops against the same scriptedStation, one after the other.
type liveCase struct {
	name   string
	client string // the go_box client config as JSON (url added)
	url    string // instead of the station's (a refused port)
	answer func(s seen) *reply
	delay  time.Duration // how long the station waits before each answer
	ops    string        // a JSON list of go_box ops
}

func status(code int, hdr ...string) func(seen) *reply {
	return func(seen) *reply { return &reply{Status: code, Header: pairs(hdr)} }
}

func pairs(kv []string) map[string]string {
	m := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return m
}

func body(code int, text string, hdr ...string) func(seen) *reply {
	return func(seen) *reply { return &reply{Status: code, Body: text, Header: pairs(hdr)} }
}

const openOps = `[{"op": "open"}, {"op": "state"}]`

var liveCases = []liveCase{
	{name: "a whole session: frames and headers", answer: nil,
		ops: `[{"op": "open"}, {"op": "ssc", "key": "noSuchKey", "arg": {"x": [1, 2.5, "\u00e9"]}}, {"op": "poll"}, {"op": "close"}, {"op": "state"}]`},
	{name: "401 everywhere", client: `{"password": "wrong"}`, answer: status(401), ops: openOps},
	{name: "403 on makessc", answer: func(s seen) *reply {
		if s.key() == "makessc" {
			return &reply{Status: 403}
		}
		return nil
	}, ops: openOps},
	{name: "301 redirect", answer: status(301, "Location", "http://127.0.0.2:9/box/"), ops: openOps},
	{name: "302 relative redirect", answer: status(302, "Location", "/elsewhere"), ops: openOps},
	{name: "307 redirect to an upper-case host", answer: status(307, "Location", "HTTP://EVIL.Example:8443/x"), ops: openOps},
	{name: "308 redirect through the URI header", answer: status(308, "Uri", "https://other.example/"), ops: openOps},
	{name: "302 without a location", answer: status(302), ops: openOps},
	{name: "303 to a refused scheme", answer: status(303, "Location", "mailto:a@b.example"), ops: openOps},
	{name: "304 is a plain status", answer: status(304), ops: openOps},
	{name: "502", answer: status(502), ops: openOps},
	{name: "not utf-8", answer: body(200, "{\"m\": \"\xe2\x82\"}"), ops: openOps},
	{name: "byte order mark", answer: body(200, "\ufeff{}"), ops: openOps},
	{name: "trailing comma", answer: body(200, "{\"m\": [1,\n]}"), ops: openOps},
	{name: "oversized", client: `{"max_reply_bytes": 64}`, answer: body(200, `{"pad": "`+strings.Repeat("y", 64)+`"}`), ops: openOps},
	{name: "remote end closes", answer: func(seen) *reply { return &reply{Hang: true} }, ops: openOps},
	{name: "refused port", url: "refused", ops: openOps},
	{name: "no answer within the timeout", client: `{"timeout": 0.3}`, delay: 600 * time.Millisecond,
		ops: `[{"op": "open"}]`},
	{name: "body shorter than its content length", answer: func(seen) *reply {
		return &reply{Raw: "HTTP/1.1 200 OK\r\nContent-Length: 100\r\n\r\n{\"m\": [{\"b\": 1}]}"}
	}, ops: `[{"op": "call", "channel": "ssession", "key": "make", "body": {}}, {"op": "about"}, {"op": "get_ord", "ord": "x"}]`},
	{name: "not http at all", answer: func(seen) *reply { return &reply{Raw: "SSH-2.0-OpenSSH\r\n"} },
		ops: `[{"op": "open"}, {"op": "about"}, {"op": "get_ord", "ord": "x"}, {"op": "get_file", "path": "/file/x"}]`},
	{name: "a short body on every read", answer: func(seen) *reply {
		return &reply{Raw: "HTTP/1.1 200 OK\r\nContent-Type: text/csv\r\nContent-Length: 50\r\n\r\na,b\n"}
	}, ops: `[{"op": "get_file", "path": "/file/x"}, {"op": "get_ord", "ord": "x", "max_bytes": 3}, {"op": "about", "max_bytes": 4}]`},
	{name: "a cut chunked body", answer: func(seen) *reply {
		return &reply{Raw: "HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nab"}
	}, ops: `[{"op": "call", "channel": "ssession", "key": "make", "body": {}}, {"op": "get_ord", "ord": "x"}, {"op": "get_file", "path": "/file/x"}, {"op": "about"}]`},
	{name: "a reply nested past the C stack", answer: body(200, `{"n": 1, "m": [{"b": `+strings.Repeat("[", 200000)+
		strings.Repeat("]", 200000)+`}]}`), ops: `[{"op": "call", "channel": "ssession", "key": "make", "body": {}}]`},
	{name: "http refused without allow_http", client: `{"allow_http": false}`, ops: openOps},
	{name: "csv over get_ord", answer: body(200, "a,b\r\n1,\"x\ny\"\r\n", "Content-Type", "text/csv"),
		ops: `[{"op": "get_ord", "ord": "station:|slot:/A B/\u00e9|bql:select * from control:NumericPoint"}]`},
	{name: "get_ord cut inside a character", answer: body(200, "ab\u00e9\u00e9cd", "Content-Type", "text/csv"),
		ops: `[{"op": "get_ord", "ord": "x", "max_bytes": 3}, {"op": "get_ord", "ord": "x", "max_bytes": 4}, {"op": "get_ord", "ord": "x", "max_bytes": 7}]`},
	{name: "get_ord html content type", answer: body(200, "a,b\n", "Content-Type", "Text/HTML; charset=\xe9"),
		ops: `[{"op": "get_ord", "ord": "x"}]`},
	{name: "get_ord markup body", answer: body(200, " \n\xef\xbb\xbf\t<!DocType html>", "Content-Type", ""),
		ops: `[{"op": "get_ord", "ord": "x"}]`},
	{name: "get_ord markup past 256 bytes", answer: body(200, strings.Repeat(" ", 256)+"<html>", "Content-Type", "text/csv"),
		ops: `[{"op": "get_ord", "ord": "x"}]`},
	{name: "get_ord 400", answer: status(400), ops: `[{"op": "get_ord", "ord": "x"}, {"op": "state"}]`},
	{name: "get_ord 401", answer: status(401), ops: `[{"op": "get_ord", "ord": "x"}, {"op": "state"}]`},
	{name: "get_ord 404", answer: status(404), ops: `[{"op": "get_ord", "ord": "x"}]`},
	{name: "get_ord lone surrogates", ops: `[{"op": "get_ord", "ord": "ab\udc80c"}, {"op": "get_ord", "ord": "\u00e9\ud800\udc80\udfff!"}]`},
	{name: "get_file", answer: body(200, "\x00\x01<px/>", "Content-Type", "text/xml"),
		ops: `[{"op": "get_file", "path": "/file/px/a%20b.px"}, {"op": "get_file", "path": "/file/x", "max_bytes": 2}]`},
	{name: "get_file statuses", answer: func(s seen) *reply {
		code := map[string]int{"/file/403": 403, "/file/404": 404, "/file/401": 401, "/file/500": 500}[s.Path]
		return &reply{Status: code}
	}, ops: `[{"op": "get_file", "path": "/file/403"}, {"op": "get_file", "path": "/file/404"}, {"op": "get_file", "path": "/file/401"}, {"op": "get_file", "path": "/file/500"}, {"op": "get_file", "path": "/ord/x"}, {"op": "state"}]`},
	{name: "about", answer: body(200, "<obj>\xff</obj>", "Content-Type", "text/xml"),
		ops: `[{"op": "about"}, {"op": "about", "max_bytes": 3}]`},
	{name: "about statuses", answer: func(s seen) *reply { return &reply{Status: 403} }, ops: `[{"op": "about"}, {"op": "state"}]`},
}

// TestLiveDifferential runs every liveCase through Go against a scriptedStation and compares
// both the result lines and the requests the station saw (method, path, body and every header
// except User-Agent) with what the Python BoxClient printed and sent for the same case
// (go_box.py --live, before S17 removed it). Those are recorded in liveGolden, which is frozen:
// a missing or stale golden fails.
func TestLiveDifferential(t *testing.T) {
	recorded := readLiveGolden(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	refused := ln.Addr().String()
	ln.Close()
	for i, lc := range liveCases {
		t.Run(lc.name, func(t *testing.T) {
			answers := &answerLog{}
			st := newScripted(t, answers.wrap(lc))
			url := st.URL
			if lc.url != "" {
				url = "http://" + refused
			}
			// The ports are the only run-to-run difference: both addresses become placeholders in
			// everything recorded or compared (the URL inside the case spec, any output that
			// quotes it). Host is not in seen.Header: net/http moves it to Request.Host.
			alias := strings.NewReplacer(st.Listener.Addr().String(), "127.0.0.1:<station>",
				refused, "127.0.0.1:<refused>").Replace
			cfg := lc.client
			if cfg == "" {
				cfg = "{}"
			}
			cfgObj := mustObject(t, cfg)
			cfgObj.Set("url", url)
			spec, _ := pyjson.Marshal(obj("client", cfgObj, "ops", mustParse(t, lc.ops)), pyjson.Options{})
			if i >= len(recorded) || recorded[i].Name != lc.name || recorded[i].Spec != alias(string(spec)) {
				t.Fatalf("golden %s is stale: case %d is not %q with this spec; %s", liveGolden, i+1, lc.name, liveHint)
			}
			want := recorded[i]
			var got []string
			if c, err := clientFrom(cfgObj, url); err != nil {
				line, _ := pyjson.Marshal(obj("error", describe(err)), pyjson.Options{})
				got = []string{string(line)}
			} else {
				got = runOps(t, c, mustParse(t, lc.ops).([]any))
			}
			if g := alias(strings.Join(got, "\n")); g != strings.Join(want.Stdout, "\n") {
				t.Errorf("results:\n go     %s\n python %s", strings.Join(got, "\n        "),
					strings.Join(want.Stdout, "\n        "))
			}
			compareRequests(t, liveRequests(st.requests(), alias), want.Requests)
			// Go sent Python's requests, so the station must have answered them as recorded; a
			// case's answer edited without re-recording would compare Go with Python's old run.
			if got := answers.take(lc.delay, alias); !t.Failed() && got != want.Answers {
				t.Fatalf("golden %s is stale: the station's answers for %q changed; %s", liveGolden, lc.name, liveHint)
			}
		})
	}
	if len(recorded) != len(liveCases) {
		t.Errorf("golden %s is stale: %d recorded cases, the test has %d; %s", liveGolden, len(recorded), len(liveCases), liveHint)
	}
}

// liveGolden holds one liveRecord per liveCase, in order, one JSON object per line.
const liveGolden = "testdata/live.golden.jsonl"

const liveHint = "the golden is frozen (the Python BoxClient was removed in S17): restore it"

// liveRecord is what Python did for one case: the spec it was sent (to catch a case edited
// after recording), the lines it printed and the requests the station saw.
type liveRecord struct {
	Name     string        `json:"name"`
	Spec     string        `json:"spec"`
	Stdout   []string      `json:"stdout"`
	Requests []liveRequest `json:"requests"`
	Answers  string        `json:"answers"` // answerLog.take of the station's replies to Python
}

// answerLog records every reply the station sends for a case, the default one included, so the
// golden can tell when a case's answer (status, headers, body, raw text, hang, delay) changed.
type answerLog struct {
	mu   sync.Mutex
	sent []*reply
}

// wrap is lc's answer, logged before lc.delay so a reply the client timed out on is logged too.
func (l *answerLog) wrap(lc liveCase) func(seen) *reply {
	return func(s seen) *reply {
		var r *reply
		if lc.answer != nil {
			r = lc.answer(s)
		}
		if r == nil {
			r = defaultAnswer(s)
		}
		l.mu.Lock()
		l.sent = append(l.sent, r)
		l.mu.Unlock()
		time.Sleep(lc.delay)
		return r
	}
}

// take is the SHA-256 of the delay and the replies logged since the last take, and clears the
// log. %#v quotes every byte of a body (JSON would turn invalid UTF-8 into U+FFFD) and sorts
// the header keys; the ports become placeholders.
func (l *answerLog) take(delay time.Duration, alias func(string) string) string {
	l.mu.Lock()
	sent := l.sent
	l.sent = nil
	l.mu.Unlock()
	text := delay.String()
	for _, r := range sent {
		text += fmt.Sprintf("\n%#v", *r)
	}
	sum := sha256.Sum256([]byte(alias(text)))
	return hex.EncodeToString(sum[:])
}

// liveRequest is a seen request as compared: Headers drops User-Agent (Python's names urllib
// and its version, Go's names Go) and joins a repeated header's values with NUL.
type liveRequest struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

func liveRequests(reqs []seen, alias func(string) string) []liveRequest {
	out := []liveRequest{}
	for _, s := range reqs {
		h := map[string]string{}
		for k, v := range s.Header {
			if k != "User-Agent" {
				h[k] = alias(strings.Join(v, "\x00"))
			}
		}
		out = append(out, liveRequest{Method: s.Method, Path: alias(s.Path), Headers: h, Body: alias(string(s.Body))})
	}
	return out
}

func readLiveGolden(t *testing.T) []liveRecord {
	t.Helper()
	data, err := os.ReadFile(liveGolden)
	if errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("golden %s missing: %s", liveGolden, liveHint)
	}
	if err != nil {
		t.Fatal(err)
	}
	var recs []liveRecord
	dec := json.NewDecoder(bytes.NewReader(data))
	for {
		var r liveRecord
		if err := dec.Decode(&r); err == io.EOF {
			return recs
		} else if err != nil {
			t.Fatalf("golden %s: record %d: %v", liveGolden, len(recs)+1, err)
		}
		recs = append(recs, r)
	}
}

func compareRequests(t *testing.T, got, want []liveRequest) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("Go sent %d requests, Python %d", len(got), len(want))
	}
	for i := range got {
		if !reflect.DeepEqual(got[i], want[i]) {
			g, _ := json.Marshal(got[i])
			w, _ := json.Marshal(want[i])
			t.Errorf("request %d:\n go     %s\n python %s", i, g, w)
		}
	}
}
