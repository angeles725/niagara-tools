// Package token holds the confirmation tokens of the write tools and the argument handling
// around them (mcp-n4-kit/mcp_n4/safety.py:47-113): bound_args, redact_args and
// ConfirmationTokens, single-use HMAC-SHA256 tokens under a per-process random key.
package token

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
)

// Redacted replaces the value of a secret-looking key (REDACTED).
const Redacted = "***"

var secretMarkers = []string{"pass", "secret", "token", "credential"}

// expiryPart is _EXPIRY: ASCII digits only, at most 15 (str.isdigit() would accept others).
var expiryPart = regexp.MustCompile(`^[0-9]{1,15}$`)

// BoundArgs is bound_args: a copy of args without dry_run and confirmation_token, the args a
// token is bound to.
func BoundArgs(args *pyjson.Object) *pyjson.Object {
	out := pyjson.NewObject()
	for _, k := range args.Keys() {
		if k != "dry_run" && k != "confirmation_token" {
			v, _ := args.Get(k)
			out.Set(k, v)
		}
	}
	return out
}

// Redact is redact_args: a copy of v in which the value of every key whose str.lower() holds
// pass, secret, token or credential is Redacted, at any depth of objects and lists.
func Redact(v any) any {
	switch x := v.(type) {
	case *pyjson.Object:
		out := pyjson.NewObject()
		for _, k := range x.Keys() {
			val, _ := x.Get(k)
			out.Set(k, Redact(val))
			lower := pyjson.Lower(k)
			for _, m := range secretMarkers {
				if strings.Contains(lower, m) {
					out.Set(k, Redacted)
					break
				}
			}
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = Redact(item)
		}
		return out
	}
	return v
}

// Tokens is ConfirmationTokens. A token is "<expiry>.<nonce>.<mac>"; the mac covers the tool
// name, the canonical bound args, the plan hash, the expiry and the nonce.
type Tokens struct {
	ttl     int64
	clock   func() float64
	key     []byte
	nonce   func() string
	pending map[string]int64 // nonce -> expiry of issued, unspent tokens
}

// New returns tokens living ttl seconds by clock (time.time when nil) under a fresh random key.
func New(ttl int, clock func() float64) *Tokens {
	if clock == nil {
		clock = func() float64 { return float64(time.Now().UnixNano()) / 1e9 }
	}
	return &Tokens{ttl: int64(ttl), clock: clock, key: randomBytes(32), pending: map[string]int64{},
		nonce: func() string { return hex.EncodeToString(randomBytes(8)) }}
}

// randomBytes is secrets.token_bytes(n).
func randomBytes(n int) []byte {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err) // crypto/rand never fails on a supported platform
	}
	return b
}

func (t *Tokens) mac(tool string, args *pyjson.Object, planHash string, expiry int64, nonce string) (string, error) {
	bound, err := pyjson.Marshal(BoundArgs(args), pyjson.Canonical)
	if err != nil {
		return "", err
	}
	msg, err := pyjson.Marshal([]any{tool, string(bound), planHash, expiry, nonce}, pyjson.Canonical)
	if err != nil {
		return "", err
	}
	h := hmac.New(sha256.New, t.key)
	h.Write(msg)
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Issue returns a token for this exact tool, args and plan hash, and the time it expires.
func (t *Tokens) Issue(tool string, args *pyjson.Object, planHash string) (string, int64, error) {
	now := t.clock()
	for n, e := range t.pending {
		if !(float64(e) > now) {
			delete(t.pending, n)
		}
	}
	expiry, nonce := int64(now)+t.ttl, t.nonce()
	mac, err := t.mac(tool, args, planHash, expiry, nonce)
	if err != nil {
		return "", 0, err
	}
	t.pending[nonce] = expiry
	return strconv.FormatInt(expiry, 10) + "." + nonce + "." + mac, expiry, nil
}

// Consume spends token, any value the caller passed: a *safety.Error unless it is a string that
// is authentic, bound to tool, args and planHash, not expired and not spent before.
func (t *Tokens) Consume(tool string, args *pyjson.Object, planHash string, token any) error {
	s, _ := token.(string)
	parts := strings.Split(s, ".")
	if _, ok := token.(string); !ok || len(parts) != 3 || !expiryPart.MatchString(parts[0]) || !isASCII(s) {
		return &safety.Error{Msg: safety.ReasonTokenMissing + " or malformed: run a dry run " +
			"first and pass the token it returns"}
	}
	expiry, _ := strconv.ParseInt(parts[0], 10, 64)
	nonce := parts[1]
	want, err := t.mac(tool, args, planHash, expiry, nonce)
	if err != nil {
		return err
	}
	if !hmac.Equal([]byte(parts[2]), []byte(want)) {
		return &safety.Error{Msg: safety.ReasonTokenMismatch + ", these arguments " +
			"and this plan: run the dry run again"}
	}
	if t.clock() >= float64(expiry) {
		delete(t.pending, nonce)
		return &safety.Error{Msg: safety.ReasonTokenExpired + ": run the dry run again"}
	}
	if _, ok := t.pending[nonce]; !ok {
		return &safety.Error{Msg: safety.ReasonTokenReused + ": run the dry run again"}
	}
	delete(t.pending, nonce)
	return nil
}

// isASCII is str.isascii(): a WTF-8 surrogate is not ASCII either.
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}
