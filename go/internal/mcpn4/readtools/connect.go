// Package readtools holds the read-only station tools of the mcp-n4 server
// (mcp-n4-kit/mcp_n4/tools_read.py:198-500, 789-969). This slice serves n4_connect and
// n4_describe_session; every other read tool still answers jsonrpc.NotImplemented.
package readtools

import (
	"errors"
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/tiers"
)

// RootOrd is ROOT_ORD.
const RootOrd = "station:|slot:/"

// Client is what the tools need of a station client (*box.Client has it all).
type Client interface {
	Open() (any, error)
	LoadTree(ord string, depth int, opt box.LoadOptions) (*pyjson.Object, error)
	BaseURL() string
	Close()
}

// aboutReader is a client that can read the oBIX About document; one that cannot is tier C.
type aboutReader interface {
	About(timeout float64, maxBytes int) (string, error)
}

// Factory is Context.client_factory: a client for cfg that calls onAuthError with each AuthError
// it raises. A client may ignore the hook: the dispatcher latches on the returned error too.
type Factory func(cfg box.Config, onAuthError func(*box.Error)) (Client, error)

// BoxFactory makes the real BOX client.
func BoxFactory(cfg box.Config, onAuthError func(*box.Error)) (Client, error) {
	c, err := box.New(cfg)
	if err != nil {
		return nil, err
	}
	c.OnAuthError = onAuthError
	return c, nil
}

// Tools is the state the read tools share.
type Tools struct {
	Ctx           *session.Context
	NewClient     Factory
	ServerVersion string
}

// Register serves the ported read tools on s, with t.Ctx as its session state.
func Register(s *jsonrpc.Server, t *Tools) {
	s.Ctx = t.Ctx
	s.Handlers["n4_connect"] = t.Connect
	s.Handlers["n4_describe_session"] = t.DescribeSession
}

func toolError(format string, a ...any) error {
	return &jsonrpc.ToolError{Msg: fmt.Sprintf(format, a...)}
}

// Connect is n4_connect: it replaces any session with one on a configured station whose real
// stationName equals expected_station (default: the NAME), and reports its version tier.
func (t *Tools) Connect(args *pyjson.Object) (*pyjson.Object, error) {
	ctx := t.Ctx
	ctx.Close() // first, so any failure below never leaves the old session active
	v, _ := args.Get("station")
	name := v.(string) // validated against the schema: a required string
	if len(ctx.Stations) == 0 {
		return nil, toolError("no station is configured: start the server with --station NAME=URL")
	}
	url, ok := ctx.Stations[name]
	if !ok {
		return nil, toolError("unknown station %s: configured stations are %s (set with --station NAME=URL)",
			pyjson.StrRepr(name), strings.Join(ctx.StationNames(), ", "))
	}
	prefix := ctx.CredentialEnv
	user, secret := ctx.Getenv(prefix+"_USER"), ctx.Getenv(prefix+"_PASSWORD")
	if user == "" || secret == "" {
		return nil, toolError("credentials missing: set env vars %s_USER and %s_PASSWORD", prefix, prefix)
	}
	ctx.SetSecret(secret)
	credentials := ctx.CredentialsFingerprint(user, secret)
	if block := ctx.AuthBlockFor(credentials); block != "" {
		// the same credentials just failed: one more try would only near the lock-out
		return nil, toolError("%s", block)
	}
	ctx.UseCredentials(credentials)
	insecure := false
	for _, n := range ctx.InsecureTLS {
		insecure = insecure || n == name
	}
	client, err := t.NewClient(box.Config{BaseURL: url, Username: user, Password: secret,
		Timeout: ctx.HTTPTimeout, InsecureTLS: insecure, AllowHTTP: ctx.AllowHTTP, LoadWait: ctx.LoadWait},
		func(e *box.Error) { ctx.NoteAuthFailure(e) })
	if err != nil {
		return nil, err
	}
	root, nodes, err := open(client)
	if err != nil {
		client.Close()
		return nil, err
	}
	ctx.ClearAuthLatch() // these credentials logged in: an earlier failure is moot
	var stationName any
	if node, ok := lookup(nodes, "stationName").(*pyjson.Object); ok {
		stationName = lookup(node, "v")
	}
	expected := name
	if e, ok := args.Get("expected_station"); ok {
		expected = e.(string)
	}
	if s, ok := stationName.(string); !ok || s != expected {
		client.Close()
		return nil, toolError("station identity mismatch: expected %s but connected to %s",
			pyjson.StrRepr(expected), pyjson.Repr(stationName))
	}
	version, versionError := detectVersion(client)
	ctx.Session = session.NewSession(client, expected, client.BaseURL(), root, version, ctx.Scrub(versionError))
	out := obj("station_name", expected, "base_url", client.BaseURL(), "root_handle", root, "mode", ctx.Mode())
	ctx.AddTierReport(out, ctx.Session)
	ctx.AddAuthReport(out)
	return out, nil
}

// open logs in and loads the root one level deep: the root handle and the flat nodes.
func open(client Client) (any, *pyjson.Object, error) {
	root, err := client.Open()
	if err != nil {
		return nil, nil, err
	}
	var handle any
	if r, ok := root.(*pyjson.Object); ok {
		handle = lookup(r, "h")
	}
	if !pyjson.Truthy(handle) {
		return nil, nil, &box.Error{Kind: box.KindBox, Msg: "station returned no root handle"}
	}
	nodes, err := client.LoadTree(RootOrd, 1, box.LoadOptions{Handle: handle})
	return handle, nodes, err
}

// detectVersion is _detect_version: the oBIX productVersion, or "" and why. It never fails a
// connect: one GET, never retried, and an undetected version is tier C, so a failure here only
// makes the server stricter. A 401/403 may count as a failed login, so it is reported.
func detectVersion(client Client) (version, why string) {
	about, ok := client.(aboutReader)
	if !ok {
		return "", "this client cannot read " + tiers.AboutPath
	}
	text, err := about.About(0, 0)
	var be *box.Error
	var typed jsonrpc.PyTyped
	switch {
	case errors.As(err, &be) && be.IsAuth():
		return "", fmt.Sprintf("%s refused: %s; the station may count it as a failed login, so check "+
			"that this user may read oBIX before reconnecting", tiers.AboutPath, be.Msg)
	case errors.As(err, &be):
		return "", fmt.Sprintf("%s unreadable: %s", tiers.AboutPath, be.Msg)
	case errors.As(err, &typed):
		return "", fmt.Sprintf("%s unreadable (%s)", tiers.AboutPath, typed.PyType())
	case err != nil:
		return "", tiers.AboutPath + " unreadable (Exception)"
	}
	if version = tiers.ProductVersion(text); version == "" {
		return "", tiers.AboutPath + " has no productVersion"
	}
	return version, ""
}

// DescribeSession is n4_describe_session: the session, the server mode and the configured
// station names (never their URLs). It works without a session.
func (t *Tools) DescribeSession(*pyjson.Object) (*pyjson.Object, error) {
	ctx, sess := t.Ctx, t.Ctx.Session
	names := []any{}
	for _, n := range ctx.StationNames() {
		names = append(names, n)
	}
	out := obj("connected", sess != nil, "station_name", nil, "base_url", nil)
	if sess != nil {
		out.Set("station_name", sess.StationName)
		out.Set("base_url", sess.BaseURL)
	}
	out.Set("mode", ctx.Mode())
	out.Set("server_version", t.ServerVersion)
	out.Set("configured_stations", names)
	ctx.AddTierReport(out, sess)
	ctx.AddAuthReport(out)
	return out, nil
}

func obj(kv ...any) *pyjson.Object {
	o := pyjson.NewObject()
	for i := 0; i < len(kv); i += 2 {
		o.Set(kv[i].(string), kv[i+1])
	}
	return o
}

func lookup(o *pyjson.Object, k string) any {
	v, _ := o.Get(k)
	return v
}
