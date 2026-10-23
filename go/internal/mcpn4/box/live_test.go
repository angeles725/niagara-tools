package box

import (
	"bufio"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// liveCase is one HTTP-level scenario: the Python BoxClient (go_box.py --live) and the Go client
// run the same ops against the same scriptedStation, one after the other.
type liveCase struct {
	name   string
	client string // the go_box client config as JSON (url added)
	url    string // instead of the station's (a refused port)
	answer func(s seen) *reply
	ops    string // a JSON list of go_box ops
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
	{name: "no answer within the timeout", client: `{"timeout": 0.3}`, answer: func(seen) *reply {
		time.Sleep(600 * time.Millisecond)
		return nil
	}, ops: `[{"op": "open"}]`},
	{name: "body shorter than its content length", answer: func(seen) *reply {
		return &reply{Raw: "HTTP/1.1 200 OK\r\nContent-Length: 100\r\n\r\n{\"m\": [{\"b\": 1}]}"}
	}, ops: `[{"op": "call", "channel": "ssession", "key": "make", "body": {}}, {"op": "about"}, {"op": "get_ord", "ord": "x"}]`},
	{name: "not http at all", answer: func(seen) *reply { return &reply{Raw: "SSH-2.0-OpenSSH\r\n"} },
		ops: `[{"op": "open"}, {"op": "about"}]`},
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

func TestLiveDifferential(t *testing.T) {
	py, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not on PATH")
	}
	driver, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "mcp-n4-kit", "tests", "go_box.py"))
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	refused := "http://" + ln.Addr().String()
	ln.Close()
	for _, lc := range liveCases {
		t.Run(lc.name, func(t *testing.T) {
			st := newScripted(t, lc.answer)
			url := st.URL
			if lc.url != "" {
				url = refused
			}
			cfg := lc.client
			if cfg == "" {
				cfg = "{}"
			}
			cfgObj := mustObject(t, cfg)
			cfgObj.Set("url", url)
			spec, _ := pyjson.Marshal(obj("client", cfgObj, "ops", mustParse(t, lc.ops)), pyjson.Options{})
			cmd := exec.Command(py, "-I", driver, "--live")
			cmd.Stdin = strings.NewReader(string(spec))
			out, err := cmd.Output()
			if err != nil {
				t.Fatalf("python3 driver: %v", err)
			}
			pyReqs := st.requests()
			st.mu.Lock()
			st.seen = nil
			st.mu.Unlock()
			var want []string
			sc := bufio.NewScanner(strings.NewReader(string(out)))
			sc.Buffer(nil, 1<<24)
			for sc.Scan() {
				want = append(want, sc.Text())
			}
			var got []string
			if c, err := clientFrom(cfgObj, url); err != nil {
				line, _ := pyjson.Marshal(obj("error", describe(err)), pyjson.Options{})
				got = []string{string(line)}
			} else {
				got = runOps(t, c, mustParse(t, lc.ops).([]any))
			}
			if strings.Join(got, "\n") != strings.Join(want, "\n") {
				t.Errorf("results:\n go     %s\n python %s", strings.Join(got, "\n        "),
					strings.Join(want, "\n        "))
			}
			compareRequests(t, st.requests(), pyReqs)
		})
	}
}

func compareRequests(t *testing.T, got, want []seen) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("Go sent %d requests, Python %d", len(got), len(want))
	}
	for i := range got {
		g, w := got[i], want[i]
		g.Header.Del("User-Agent")
		w.Header.Del("User-Agent")
		if g.Method != w.Method || g.Path != w.Path || string(g.Body) != string(w.Body) ||
			pyjson.Repr(headerObj(g.Header)) != pyjson.Repr(headerObj(w.Header)) {
			t.Errorf("request %d:\n go     %s %s %v %s\n python %s %s %v %s", i, g.Method, g.Path,
				g.Header, g.Body, w.Method, w.Path, w.Header, w.Body)
		}
	}
}

func headerObj(h http.Header) *pyjson.Object {
	o := pyjson.NewObject()
	for _, k := range sortedKeys(h) {
		o.Set(k, strings.Join(h[k], "\x00"))
	}
	return o
}

func sortedKeys(h http.Header) []string {
	var keys []string
	for k := range h {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}
