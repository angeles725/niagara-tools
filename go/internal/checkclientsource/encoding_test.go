package checkclientsource

import (
	"path/filepath"
	"testing"
	"unicode/utf16"
)

// utf16Bytes is text encoded as UTF-16 with the byte-order mark of its byte order.
func utf16Bytes(text string, bigEndian bool) string {
	var b []byte
	for _, u := range append([]uint16{0xfeff}, utf16.Encode([]rune(text))...) {
		if bigEndian {
			b = append(b, byte(u>>8), byte(u))
		} else {
			b = append(b, byte(u), byte(u>>8))
		}
	}
	return string(b)
}

// TestBuildStateInUTF16 (retro coordination-stretch K9): a BUILD-STATE an editor saved as UTF-16 with its
// byte-order mark (either byte order) declares what it declares; read as bytes it named no module and the
// check WARNed "undeclared", exit 0. UTF-16 that cannot be decoded (no mark, so NUL bytes; an odd byte
// count) is refused, exit 3, never read as undeclared.
func TestBuildStateInUTF16(t *testing.T) {
	f := newFixture(t)
	state := filepath.Join(f.base, "BUILD-STATE.md")
	text := "<!-- build-state.v1 -->\r\nmodule: Mod\r\nsource_of_truth: r@main@" + f.second + "\r\n<!-- /build-state.v1 -->\r\n"
	pass := "PASS  source-of-truth  " + f.top + " at the declared commit " + f.second + " on main\n"
	for name, big := range map[string]bool{"little-endian": false, "big-endian": true} {
		write(t, state, utf16Bytes(text, big))
		expect(t, name, []string{"--build-state", state, "--module", "Mod", f.root}, ExitOK, pass, "")
	}
	le := utf16Bytes(text, false)
	write(t, state, le[2:])
	expect(t, "no byte-order mark", []string{"--build-state", state, "--module", "Mod", f.root}, ExitEnv, "",
		"check-client-source: "+state+" holds NUL bytes (UTF-16 without a byte-order mark?): save it as UTF-8\n")
	write(t, state, le+"x")
	expect(t, "an odd byte count", []string{"--build-state", state, "--module", "Mod", f.root}, ExitEnv, "",
		"check-client-source: "+state+" starts with a UTF-16 byte-order mark but holds an odd number of bytes: save it as UTF-8\n")
}
