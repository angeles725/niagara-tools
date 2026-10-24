// Package readtoolstest is an in-memory station and a wired server for the Go twins of the
// connect tests (go_server.go_twin): what the Python tests reach in-process (srv.ctx, the clock,
// client_factory) is a field here. The black-box suite keeps the Python FakeStation.
package readtoolstest

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/registry"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// Password is the station password the default environment holds.
const Password = "s3cret-pw-value"

// Station is the fake station: its name, password and oBIX productVersion ("" answers HTTP 404),
// and what reached it. Requests counts logins and loads, AboutRequests the About reads.
type Station struct {
	Name, Password, Version string
	AboutErr                error // returned by About instead of the document
	NoAbout                 bool  // clients cannot read /obix/about/
	NoHook                  bool  // clients ignore the auth hook
	Requests, AboutRequests int
	Sessions                map[int]bool // open station sessions
	Configs                 []box.Config // every client made, in order
	// Trees maps an ORD to the nodes a load of it answers (any depth); an ORD missing here
	// answers the root with its stationName.
	Trees map[string]*pyjson.Object
	// OrdRoutes answer GET /ord/ (test_bql's FakeOpener): the first route whose Needle the ORD
	// holds answers Body, or the error of HTTP status Code. OrdCap, when set, is the byte cap
	// in place of the caller's; OrdRequests is every ORD read, in order.
	OrdRoutes   []OrdRoute
	OrdCap      int
	OrdRequests []OrdRequest
	// Files answer GET /file/<path> (FakeStation.files): a path's body and content type, or
	// the HTTP status Code to answer instead; a missing path or a .bog is a 404 / 403.
	// FileRequests is every decoded path read, in order.
	Files        map[string]File
	FileRequests []string
	next         int
}

// NewStation is a tier A station named FakeStation.
func NewStation() *Station {
	return &Station{Name: "FakeStation", Password: Password, Version: "4.14.0.162", Sessions: map[int]bool{}}
}

// Factory is the readtools.Factory of clients on st.
func (st *Station) Factory(cfg box.Config, hook func(*box.Error)) (readtools.Client, error) {
	st.Configs = append(st.Configs, cfg)
	if st.NoHook {
		hook = nil
	}
	c := &client{st: st, cfg: cfg, hook: hook}
	if st.NoAbout {
		return c, nil
	}
	return aboutClient{c}, nil
}

type client struct {
	st   *Station
	cfg  box.Config
	hook func(*box.Error)
	sid  int
}

func (c *client) authError(channel, key string) error {
	e := &box.Error{Kind: box.KindAuth, Msg: box.AuthMessage(401), Channel: channel, Key: key}
	if c.hook != nil {
		c.hook(e)
	}
	return e
}

func (c *client) Open() (any, error) {
	c.st.Requests++
	if c.cfg.Password != c.st.Password {
		return nil, c.authError(box.Channel, "make")
	}
	c.st.next++
	c.sid = c.st.next
	c.st.Sessions[c.sid] = true
	root := pyjson.NewObject()
	root.Set("h", box.DefaultRootHandle)
	return root, nil
}

func (c *client) LoadTree(ord string, depth int, opt box.LoadOptions) (*pyjson.Object, error) {
	c.st.Requests++
	if !c.st.Sessions[c.sid] {
		return nil, &box.Error{Kind: box.KindBox, Msg: "no such session", Channel: box.Channel, Key: "loadSlots"}
	}
	if nodes, ok := c.st.Trees[ord]; ok {
		return nodes, nil
	}
	root, name, nodes := pyjson.NewObject(), pyjson.NewObject(), pyjson.NewObject()
	root.Set("h", opt.Handle)
	name.Set("v", c.st.Name)
	nodes.Set("", root)
	nodes.Set("stationName", name)
	return nodes, nil
}

// OrdRoute is one answer of GET /ord/.
type OrdRoute struct {
	Needle, Body string
	Code         int
}

// OrdRequest is one GET /ord/ that reached the station.
type OrdRequest struct {
	Ord     string
	Timeout float64
}

// GetOrd answers from the station's OrdRoutes like box.Client.GetOrd: a 401/403 is an
// AuthError (latched through the hook), a 400 names the query, a body longer than the cap is
// cut and reported truncated.
func (c *client) GetOrd(ord string, timeout float64, maxBytes int) (string, bool, error) {
	c.st.OrdRequests = append(c.st.OrdRequests, OrdRequest{ord, timeout})
	for _, r := range c.st.OrdRoutes {
		if !strings.Contains(ord, r.Needle) {
			continue
		}
		switch {
		case r.Code == 401 || r.Code == 403:
			return "", false, c.authError("ord", "get")
		case r.Code == 400:
			return "", false, &box.Error{Kind: box.KindBox, Msg: "HTTP 400 from station: the ORD or BQL query was " +
				"rejected (check the type spec, column names and where clause)", Channel: "ord", Key: "get"}
		case r.Code != 0:
			return "", false, &box.Error{Kind: box.KindBox, Msg: fmt.Sprintf("HTTP %d from station", r.Code), Channel: "ord", Key: "get"}
		}
		if c.st.OrdCap > 0 {
			maxBytes = c.st.OrdCap
		}
		if len(r.Body) > maxBytes {
			return r.Body[:maxBytes], true, nil
		}
		return r.Body, false, nil
	}
	return "", false, &box.Error{Kind: box.KindBox, Msg: "unexpected ORD " + ord, Channel: "ord", Key: "get"}
}

// File is one answer of GET /file/; NoType sends no Content-Type header.
type File struct {
	Body, ContentType string
	NoType            bool
	Code              int
}

// GetFile answers from the station's Files like box.Client.GetFile: a wrong password is an
// AuthError, a 403/404 an HttpStatusError, a body longer than maxBytes cut and truncated.
func (c *client) GetFile(urlPath string, timeout float64, maxBytes int) (*box.File, error) {
	path, _ := url.PathUnescape(strings.TrimPrefix(urlPath, "/file/"))
	c.st.FileRequests = append(c.st.FileRequests, path)
	if c.cfg.Password != c.st.Password {
		return nil, c.authError("file", "get")
	}
	f, ok := c.st.Files[path]
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".bog") || strings.HasSuffix(lower, ".bog.gz"):
		f = File{Code: 403}
	case !ok:
		f = File{Code: 404}
	}
	if f.Code != 0 {
		return nil, &box.Error{Kind: box.KindHTTPStatus, Msg: fmt.Sprintf("HTTP %d from station", f.Code),
			Channel: "file", Key: "get", Code: f.Code}
	}
	body := []byte(f.Body)
	out := &box.File{Body: body, Truncated: len(body) > maxBytes, ContentLength: pyjson.IntLiteral(fmt.Sprint(len(body)))}
	if !f.NoType {
		out.ContentType = &f.ContentType
	}
	if out.Truncated {
		out.Body = body[:maxBytes]
	}
	return out, nil
}

func (c *client) BaseURL() string { return strings.TrimRight(c.cfg.BaseURL, "/") }

func (c *client) Close() {
	delete(c.st.Sessions, c.sid)
	c.sid = 0
}

type aboutClient struct{ *client }

func (c aboutClient) About(float64, int) (string, error) {
	c.st.AboutRequests++
	var be *box.Error
	switch {
	case c.st.AboutErr != nil:
		if e, ok := c.st.AboutErr.(*box.Error); ok && e.IsAuth() && c.hook != nil {
			c.hook(e)
		}
		return "", c.st.AboutErr
	case c.st.Version == "":
		be = &box.Error{Kind: box.KindBox, Msg: "HTTP 404 from station", Channel: "obix", Key: "about"}
		return "", be
	}
	return `<obj is="obix:About"><str name="productVersion" val="` + c.st.Version + `"/></obj>`, nil
}

// Server is a jsonrpc server with the read tools wired to a Station, a fixed clock (Now) and an
// environment map (Env) in place of the process ones.
type Server struct {
	*jsonrpc.Server
	Ctx     *session.Context
	Station *Station
	Env     map[string]string
	Now     float64
}

// NewServer serves st under cfg; empty fields take server.py's defaults and one station,
// FakeStation, is configured unless cfg names its own.
func NewServer(st *Station, cfg session.Config) *Server {
	if cfg.Stations == nil {
		cfg.Stations = map[string]string{"FakeStation": "http://fake.invalid/"}
	}
	if cfg.CredentialEnv == "" {
		cfg.CredentialEnv = "MCP_N4"
	}
	if cfg.AuthCooldown == 0 {
		cfg.AuthCooldown = session.AuthCooldownMin
	}
	cfg.AllowHTTP = true
	s := &Server{Station: st, Now: 1000,
		Env: map[string]string{"MCP_N4_USER": "admin", "MCP_N4_PASSWORD": Password}}
	s.Ctx = session.New(cfg)
	s.Ctx.Getenv = func(k string) string { return s.Env[k] }
	s.Ctx.Clock = func() float64 { return s.Now }
	s.Server = jsonrpc.New(registry.Default())
	readtools.Register(s.Server, &readtools.Tools{Ctx: s.Ctx, NewClient: st.Factory, ServerVersion: "test"})
	return s
}

// Call is one tools/call with arguments as key, value pairs: the text content, whether the
// result is an error, and its structuredContent (nil for an error).
func (s *Server) Call(t testing.TB, name string, kv ...any) (string, bool, *pyjson.Object) {
	t.Helper()
	args := pyjson.NewObject()
	for i := 0; i < len(kv); i += 2 {
		args.Set(kv[i].(string), kv[i+1])
	}
	params, msg := pyjson.NewObject(), pyjson.NewObject()
	params.Set("name", name)
	params.Set("arguments", args)
	msg.Set("id", 1)
	msg.Set("method", "tools/call")
	msg.Set("params", params)
	reply := s.Dispatch(msg)
	res, ok := reply.Get("result")
	if !ok {
		t.Fatalf("%s: no result: %s", name, pyjson.Repr(reply))
	}
	r := res.(*pyjson.Object)
	content, _ := r.Get("content")
	text, _ := content.([]any)[0].(*pyjson.Object).Get("text")
	isErr, _ := r.Get("isError")
	sc, _ := r.Get("structuredContent")
	out, _ := sc.(*pyjson.Object)
	return text.(string), isErr == true, out
}

// Connect calls n4_connect on FakeStation and fails t unless it succeeds.
func (s *Server) Connect(t testing.TB, kv ...any) *pyjson.Object {
	t.Helper()
	text, isErr, out := s.Call(t, "n4_connect", append([]any{"station", "FakeStation"}, kv...)...)
	if isErr {
		t.Fatalf("n4_connect: %s", text)
	}
	return out
}

// Escaped is a Python exception the client lets escape unwrapped: only its type name shows.
type Escaped string

func (e Escaped) Error() string { return string(e) }

// PyType is the Python class name.
func (e Escaped) PyType() string { return string(e) }
