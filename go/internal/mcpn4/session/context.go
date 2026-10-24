// Package session is the per-process state of the mcp-n4 server (mcp-n4-kit/mcp_n4/tools_read.py:
// 33-195): at most one station session, the authentication-failure latch (audit 2026-10-03 F4)
// and the secret scrub. Absent values (Python's None) are empty strings.
package session

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/tiers"
)

// AuthCooldownMin is AUTH_COOLDOWN_MIN: the station locks an account after 5 failed logins in
// 30 s (B1179), so after one failure no station call goes out for at least this long.
const AuthCooldownMin = 30

// Client is what a session needs of its station client.
type Client interface{ Close() }

// Session is tools_read.Session: one open station session.
type Session struct {
	Client      Client
	StationName string
	BaseURL     string
	RootHandle  any
	// Version is the detected product version ("" when undetected) and Tier its tier.
	Version, Tier string
	// VersionError is why the version is unknown (shown to the operator), "" when detected.
	VersionError string
	// IdentityVerified is true only when the session was opened with a matching expected_station.
	IdentityVerified bool
	// SessionID is fresh per connect: a write plan, so its token, is bound to it (audit F1).
	SessionID string
}

// NewSession is Session(..., identity_verified=True): connect builds it only after the identity
// check passed.
func NewSession(client Client, stationName, baseURL string, rootHandle any, version, versionError string) *Session {
	id := make([]byte, 16)
	rand.Read(id) // uuid4().hex: 32 hex digits, never reused
	return &Session{Client: client, StationName: stationName, BaseURL: baseURL, RootHandle: rootHandle,
		Version: version, Tier: tiers.TierOf(version), VersionError: versionError,
		IdentityVerified: true, SessionID: hex.EncodeToString(id)}
}

// Config is the operator policy fixed at server start; cli.Check has already refused a bad one.
type Config struct {
	AllowWrites, AllowHTTP              bool
	Stations                            map[string]string // NAME -> URL
	CredentialEnv                       string
	InsecureTLS, AllowTierB, AllowTierC []string
	AuthCooldown                        float64 // seconds, >= AuthCooldownMin
	HTTPTimeout, LoadWait               float64 // seconds per request, total wait of one load
	// StateDir is the expanded --state-dir (result files, journal, audit); ProgressPath the
	// expanded --progress-file, "" when none.
	StateDir, ProgressPath string
}

// Context is tools_read.Context.
type Context struct {
	Config
	// TierOptIn maps a tier ("B"/"C") to the stations the operator opted in: the only way past
	// the tier gate.
	TierOptIn map[string]map[string]bool
	// Getenv reads the credential variables (os.Getenv; tests replace it).
	Getenv func(string) string
	// Clock is a monotonic clock in seconds (tests replace it).
	Clock func() float64
	// Session is the active station session, nil when none.
	Session *Session
	// Observations are the in-memory read observations for the session retro (no audit line
	// exists for reads).
	Observations []*pyjson.Object
	// StartedAt is when this server started (Now()): the default window of the session retro.
	StartedAt string

	authLatch
	secret string
}

// New returns the context of a server started with cfg.
func New(cfg Config) *Context {
	start := time.Now()
	ctx := &Context{Config: cfg, Getenv: os.Getenv, authLatch: authLatch{credKey: make([]byte, 32)},
		Clock:     func() float64 { return time.Since(start).Seconds() },
		TierOptIn: map[string]map[string]bool{"B": set(cfg.AllowTierB), "C": set(cfg.AllowTierC)},
		StartedAt: Now()}
	rand.Read(ctx.credKey)
	return ctx
}

func set(names []string) map[string]bool {
	out := map[string]bool{}
	for _, n := range names {
		out[n] = true
	}
	return out
}

// Mode is "writes-allowed" or "read-only".
func (c *Context) Mode() string {
	if c.AllowWrites {
		return "writes-allowed"
	}
	return "read-only"
}

// StationNames is sorted(ctx.stations).
func (c *Context) StationNames() []string {
	names := make([]string, 0, len(c.Stations))
	for n := range c.Stations {
		names = append(names, n)
	}
	sort.Strings(names) // UTF-8 byte order is code point order
	return names
}

// Connected reports whether a session is active.
func (c *Context) Connected() bool { return c.Session != nil }

// Close clears the session (first, so it is cleared whatever happens) and closes its client.
func (c *Context) Close() {
	sess := c.Session
	c.Session = nil
	if sess != nil {
		sess.Client.Close()
	}
}

// TierBlock is why the session's version tier refuses writes, or "" (tiers.WriteBlock).
func (c *Context) TierBlock(s *Session) string {
	return tiers.WriteBlock(s.Tier, s.Version, s.StationName, c.TierOptIn)
}

// AddTierReport sets version, version_source, tier and tier_writes of s on out (all null without
// a session), then version_error when there is one.
func (c *Context) AddTierReport(out *pyjson.Object, s *Session) {
	if s == nil {
		for _, k := range []string{"version", "version_source", "tier", "tier_writes"} {
			out.Set(k, nil)
		}
		return
	}
	writes := "refused"
	if c.TierBlock(s) == "" {
		writes = "allowed-by-opt-in"
		if s.Tier == "A" {
			writes = "allowed"
		}
	}
	out.Set("version", orNone(s.Version))
	if s.Version != "" {
		out.Set("version_source", tiers.AboutPath)
	} else {
		out.Set("version_source", nil)
	}
	out.Set("tier", s.Tier)
	out.Set("tier_writes", writes)
	if s.VersionError != "" {
		out.Set("version_error", s.VersionError)
	}
}

func orNone(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// SetSecret names the secret Scrub hides.
func (c *Context) SetSecret(secret string) { c.secret = secret }

// Scrub is defence in depth: the secret never appears in an output string.
func (c *Context) Scrub(text string) string {
	if c.secret == "" {
		return text
	}
	return strings.ReplaceAll(text, c.secret, "***")
}

// Now is retro.now(): the current UTC time as the ISO text the audit log uses.
func Now() string { return time.Now().UTC().Format("2006-01-02T15:04:05+00:00") }
