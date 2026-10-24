package cli

import (
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Config is the parsed command line (server.parse_args). Repeatable flags keep their order;
// integers above int64 saturate (Python keeps the exact int; no bound needs more).
type Config struct {
	AllowWrites, AllowHTTP                bool
	WriteScopes, Stations, StationHomes   []string
	InsecureTLS, AllowTierB, AllowTierC   []string
	StateDir, CredentialEnv, ProgressFile string
	TokenTTL, MaxWrites, AuthCooldown     int64
	HTTPTimeout                           int64
	LoadWait                              float64
}

// Operator bounds (server.py: AUTH_COOLDOWN_MIN, LOAD_WAIT_RANGE, HTTP_TIMEOUT_RANGE).
const (
	AuthCooldownMin          = 30
	loadWaitLow, loadWaitHi  = 1.0, 60.0
	httpTimeoutLow, httpHigh = 5, 300
)

// Defaults is parse_args([]).
func Defaults() Config {
	return Config{CredentialEnv: "MCP_N4", TokenTTL: 300, MaxWrites: 200,
		AuthCooldown: AuthCooldownMin, HTTPTimeout: 20, LoadWait: 3.0}
}

// option is one argparse action, in add_argument order (the order ambiguity errors list).
type option struct {
	names []string // option strings; names[len-1] is the long one
	flag  bool     // store_true (no value)
	set   func(c *Config, v string) error
}

func (o option) display() string { return strings.Join(o.names, "/") }

var options = []option{
	{names: []string{"-h", "--help"}, flag: true},
	{names: []string{"--allow-writes"}, flag: true, set: func(c *Config, _ string) error { c.AllowWrites = true; return nil }},
	{names: []string{"--write-scope"}, set: appendTo(func(c *Config) *[]string { return &c.WriteScopes })},
	{names: []string{"--state-dir"}, set: func(c *Config, v string) error { c.StateDir = v; return nil }},
	{names: []string{"--token-ttl"}, set: intFlag(func(c *Config) *int64 { return &c.TokenTTL }, 1, "")},
	{names: []string{"--max-writes"}, set: intFlag(func(c *Config) *int64 { return &c.MaxWrites }, 1, "")},
	{names: []string{"--auth-cooldown"}, set: intFlag(func(c *Config) *int64 { return &c.AuthCooldown },
		AuthCooldownMin, " (the station lock-out window)")},
	{names: []string{"--load-wait"}, set: loadWait},
	{names: []string{"--http-timeout"}, set: httpTimeout},
	{names: []string{"--station"}, set: appendTo(func(c *Config) *[]string { return &c.Stations })},
	{names: []string{"--station-home"}, set: appendTo(func(c *Config) *[]string { return &c.StationHomes })},
	{names: []string{"--credential-env"}, set: func(c *Config, v string) error { c.CredentialEnv = v; return nil }},
	{names: []string{"--insecure-tls"}, set: appendTo(func(c *Config) *[]string { return &c.InsecureTLS })},
	{names: []string{"--allow-tier-b"}, set: appendTo(func(c *Config) *[]string { return &c.AllowTierB })},
	{names: []string{"--allow-tier-c"}, set: appendTo(func(c *Config) *[]string { return &c.AllowTierC })},
	{names: []string{"--progress-file"}, set: func(c *Config, v string) error { c.ProgressFile = v; return nil }},
	{names: []string{"--allow-http-for-tests"}, flag: true, set: func(c *Config, _ string) error { c.AllowHTTP = true; return nil }},
}

func appendTo(field func(*Config) *[]string) func(*Config, string) error {
	return func(c *Config, v string) error { p := field(c); *p = append(*p, v); return nil }
}

// intFlag is _positive_int (low 1) or _cooldown (low 30, with its note in the message).
func intFlag(field func(*Config) *int64, low int64, note string) func(*Config, string) error {
	return func(c *Config, v string) error {
		n, ok := pyInt(v)
		if !ok {
			return fmt.Errorf("%s is not an integer", pyjson.StrRepr(v))
		}
		if n.Cmp(big.NewInt(1)) < 0 {
			return fmt.Errorf("must be >= 1, got %s", n)
		}
		if n.Cmp(big.NewInt(low)) < 0 {
			return fmt.Errorf("must be >= %d%s, got %s", low, note, n)
		}
		*field(c) = saturate(n)
		return nil
	}
}

func loadWait(c *Config, v string) error {
	f, ok := pyjson.ParseFloat(v)
	if !ok {
		return fmt.Errorf("%s is not a number", pyjson.StrRepr(v))
	}
	if !(loadWaitLow <= f && f <= loadWaitHi) {
		return fmt.Errorf("must be between %s and %s, got %s",
			pyjson.FloatRepr(loadWaitLow), pyjson.FloatRepr(loadWaitHi), v)
	}
	c.LoadWait = f
	return nil
}

func httpTimeout(c *Config, v string) error {
	n, ok := pyInt(v)
	if !ok {
		return fmt.Errorf("%s is not a number", pyjson.StrRepr(v))
	}
	if n.Cmp(big.NewInt(httpTimeoutLow)) < 0 || n.Cmp(big.NewInt(httpHigh)) > 0 {
		return fmt.Errorf("must be between %d and %d, got %s", httpTimeoutLow, httpHigh, v)
	}
	c.HTTPTimeout = n.Int64()
	return nil
}

func saturate(n *big.Int) int64 {
	if n.IsInt64() {
		return n.Int64()
	}
	return math.MaxInt64
}

// pyMaxIntDigits is CPython's int_max_str_digits: int() refuses a longer literal.
const pyMaxIntDigits = 4300

var intRE = regexp.MustCompile(`^[+-]?[0-9](_?[0-9])*$`)

// pyInt is int(text) for ASCII digits: surrounding whitespace, a sign and single underscores
// between digits are accepted. (Python also takes non-ASCII decimal digits; this does not.)
func pyInt(text string) (*big.Int, bool) {
	s := strings.TrimFunc(text, pyjson.IsSpace)
	if !intRE.MatchString(s) {
		return nil, false
	}
	s = strings.ReplaceAll(s, "_", "")
	if len(strings.TrimLeft(s, "+-")) > pyMaxIntDigits {
		return nil, false
	}
	n, ok := new(big.Int).SetString(strings.TrimPrefix(s, "+"), 10)
	return n, ok
}
