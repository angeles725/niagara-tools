package session

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

type latch struct {
	reason      string
	until       float64
	credentials string // fingerprint, "" when none was in use
}

// authLatch is the authentication-failure latch of a Context (audit 2026-10-03 F4).
type authLatch struct {
	latch   *latch
	credKey []byte
	active  string
}

// CredentialsFingerprint is a keyed digest of the credentials (never stored in clear, never output).
func (c *Context) CredentialsFingerprint(user, secret string) string {
	m := hmac.New(sha256.New, c.credKey)
	m.Write([]byte(user + "\x00" + secret))
	return hex.EncodeToString(m.Sum(nil))
}

// UseCredentials records the credentials the next station call goes out with (what a failure latches).
func (c *Context) UseCredentials(fingerprint string) { c.active = fingerprint }

// NoteAuthFailure latches: station calls are refused for AuthCooldown seconds after any AuthError.
func (c *Context) NoteAuthFailure(err error) {
	c.latch = &latch{reason: c.Scrub(err.Error()), until: c.Clock() + c.AuthCooldown, credentials: c.active}
}

// ClearAuthLatch drops the latch: these credentials logged in, so an earlier failure is moot.
func (c *Context) ClearAuthLatch() { c.latch = nil }

// AuthRetryIn is the whole seconds left in the cooldown, or 0 (the latch is cleared once it passed).
func (c *Context) AuthRetryIn() int {
	if c.latch == nil {
		return 0
	}
	left := c.latch.until - c.Clock()
	if left <= 0 {
		c.latch = nil
		return 0
	}
	return int(math.Ceil(left))
}

// AuthBlock is why station calls are paused, or "".
func (c *Context) AuthBlock() string { return c.authBlock("", false) }

// AuthBlockFor is AuthBlock for n4_connect with the given credentials: other credentials than the
// ones that failed may try again during the cooldown.
func (c *Context) AuthBlockFor(fingerprint string) string { return c.authBlock(fingerprint, true) }

func (c *Context) authBlock(fingerprint string, given bool) string {
	left := c.AuthRetryIn()
	if left == 0 || given && fingerprint != c.latch.credentials {
		return ""
	}
	return fmt.Sprintf("station calls are paused after an authentication failure (%s): retry in "+
		"%d s, or fix %s_USER/%s_PASSWORD and call n4_connect again; every rejected "+
		"call counts toward the station lock-out (5 failures in 30 s)",
		c.latch.reason, left, c.CredentialEnv, c.CredentialEnv)
}

// AddAuthReport sets auth_paused {reason, retry_in_s} on out while the latch holds.
func (c *Context) AddAuthReport(out *pyjson.Object) {
	if left := c.AuthRetryIn(); left != 0 {
		paused := pyjson.NewObject()
		paused.Set("reason", c.latch.reason)
		paused.Set("retry_in_s", left)
		out.Set("auth_paused", paused)
	}
}
