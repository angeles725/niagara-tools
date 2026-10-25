package box

import (
	"fmt"
	"strings"
	"testing"
)

// Go twins of test_box tests no other Go test proves (docs/go-port-plan.md section 21).

func TestIsComponentTypeIsTheSharedStationAddRule(t *testing.T) {
	for _, tt := range []struct {
		spec any
		want bool
	}{
		{"kitControl:NumericConst", true}, {"control:NumericWritable", true}, {"baja:Folder", true},
		{"baja:Double", false}, {"baja:StatusNumeric", false}, {"baja:WsAnnotation", false},
		{"baja:Link", false}, {"", false}, {nil, false}, {5, false},
		// F17: driver address structs are values, not components (the naming rule says otherwise)
		{"modbusCore:FlexAddress", false}, {"bacnet:BacnetAddress", false},
		// B1200-G3: an entry of the table wins over the naming rule both ways
		{"control:PriorityLevel", false}, {"baja:UnrestrictedFolder", true}, {"baja:Component", true},
		// unknown types fall back to the naming rule
		{"acme:UnknownWidget", true}, {"baja:RelTime", false},
	} {
		if got := IsComponentType(tt.spec); got != tt.want {
			t.Errorf("IsComponentType(%v) = %v, want %v", tt.spec, got, tt.want)
		}
	}
}

// clientReads is test_box's CLIENT_READS: every client read must turn a bad answer into a BoxError.
var clientReads = []struct {
	label string
	read  func(c *Client) error
}{
	{"call", func(c *Client) error { _, err := c.Call(Channel, "make", obj()); return err }},
	{"get_ord", func(c *Client) error { _, _, err := c.GetOrd("station:|slot:/", 0, 0); return err }},
	{"get_file", func(c *Client) error { _, err := c.GetFile("/file/a.txt", 0, 0); return err }},
	{"about", func(c *Client) error { _, err := c.About(0, 0); return err }},
}

// rawClient is a client of a station answering every request with the same raw bytes.
func rawClient(t *testing.T, raw string) *Client {
	st := newScripted(t, func(seen) *reply { return &reply{Raw: raw} })
	return newClient(t, st.URL, func(c *Config) { c.Timeout = 5 })
}

func assertEveryReadFails(t *testing.T, raw, want string) {
	t.Helper()
	c := rawClient(t, raw)
	for _, r := range clientReads {
		e, ok := r.read(c).(*Error)
		if !ok || e.Kind != KindBox || e.Msg != want {
			t.Errorf("%s: %#v, want BoxError %q", r.label, e, want)
		}
	}
}

func TestANonHttpAnswerIsATransportFailure(t *testing.T) {
	assertEveryReadFails(t, "NOT HTTP AT ALL\r\n\r\n", "transport failure: invalid HTTP answer (BadStatusLine)")
}

func TestABodyShorterThanItsContentLengthIsNeverReturned(t *testing.T) {
	assertEveryReadFails(t, "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 100\r\n\r\n"+
		`{"m": [{"b": 1}]}`, "transport failure: incomplete read (got 17 of 100 bytes)")
}

func TestACutChunkedBodyIsATransportFailure(t *testing.T) {
	assertEveryReadFails(t, "HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nab",
		"transport failure: invalid HTTP answer (IncompleteRead)")
}

// The Content-Length check compares with what was asked for: a capped read of a long, complete
// body is the documented truncation, not an incomplete read.
func TestABodyLongerThanTheCapIsStillOnlyTruncated(t *testing.T) {
	body := strings.Repeat("a,b\n", 64)
	c := rawClient(t, fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/csv\r\nContent-Length: %d\r\n\r\n%s", len(body), body))
	if text, cut, err := c.GetOrd("station:|slot:/", 0, 10); text != "a,b\na,b\na," || !cut || err != nil {
		t.Errorf("get_ord = %q, %v, %v", text, cut, err)
	}
	if text, err := c.About(0, 8); text != "a,b\na,b\n" || err != nil {
		t.Errorf("about = %q, %v", text, err)
	}
}

// Close deletes the server session best effort: whatever the del gets back, the sid is gone.
func TestCloseSwallowsAnyException(t *testing.T) {
	for _, bad := range []*reply{{Status: 500}, {Raw: "NOT HTTP\r\n\r\n"}, {Hang: true}, {Status: 401}} {
		st := newScripted(t, func(s seen) *reply {
			if s.key() == "del" {
				return bad
			}
			return nil
		})
		c := newClient(t, st.URL, nil)
		if _, err := c.Open(); err != nil {
			t.Fatal(err)
		}
		c.Close()
		if keys := st.keys(); c.SID() != nil || keys[len(keys)-1] != "del" {
			t.Errorf("%+v: sid %v after %v", bad, c.SID(), keys)
		}
	}
}

// The "a load of another handle" flag is split's result, never client state a later call reads.
func TestSplitReturnsTheOtherLoadFlagInsteadOfKeepingItOnTheClient(t *testing.T) {
	c := newClient(t, "http://127.0.0.1:1", nil)
	other := mustParse(t, `{"evs": {"ops": [{"nm": "l", "h": "9", "b": {"nm": "p", "t": "baja:Folder"}}]}}`)
	if found, sawOther := c.split([]any{other}, "3"); found != nil || !sawOther {
		t.Fatalf("split(other) = %v, %v", found, sawOther)
	}
	if found, sawOther := c.split(nil, "3"); found != nil || sawOther {
		t.Fatalf("a later split(nothing) = %v, %v", found, sawOther)
	}
}

// PendingEvents is a read-only view: changing what it returns never changes the client.
func TestPendingEventsIsReadOnlyView(t *testing.T) {
	c := newClient(t, "http://127.0.0.1:1", nil)
	c.keep(mustParse(t, `{"evs": {"ops": [{"nm": "s", "h": "9"}]}}`))
	view := c.PendingEvents()
	view[0] = "changed"
	_ = append(view[:1], "appended")
	if got := c.PendingEvents(); len(got) != 1 || got[0] == "changed" {
		t.Fatalf("pending = %v", got)
	}
}
