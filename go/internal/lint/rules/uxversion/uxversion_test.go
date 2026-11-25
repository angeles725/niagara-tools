package uxversion

import "testing"

// TestVerdict: which references are checked, and how. A reference is the path as written in the source.
func TestVerdict(t *testing.T) {
	const want = "1.2.3"
	for _, c := range []struct{ ref, id, got string }{
		{"js/a.js?v=1.2.3", "", ""},
		{"js/a.js?x=1&v=1.2.3", "", ""},
		{"js/a.js?v=1.2.3&x=1", "", ""},
		{"js/a.js?v=1.2.3#frag", "", ""},
		{"img/s.svg?v=1.2.3#icon", "", ""},
		{"js/a.js?v=1.2.2", "stale-version", "1.2.2"},
		{"js/a.js?v=1.2", "stale-version", "1.2"},
		{"js/a.js?v=1.2.3.1", "stale-version", "1.2.3.1"},
		{"js/a.js?x=1&v=0.9#frag", "stale-version", "0.9"},
		{"/p/img/a.png", "no-version", ""},
		{"js/a.js?x=1", "no-version", ""},
		{"js/a.js?ver=1.2.3", "no-version", ""},
		{"js/a.js#v=1.0", "no-version", ""},
		{"js/a.js?v=", "", ""},       // the version is appended at run time
		{"js/a.js?v=${ver}", "", ""}, // a template
		{"{{ img }}", "", ""},        // a server template
		{"", "", ""},                 // nothing to load
		{"#frag", "", ""},            // a document fragment
		{"https://cdn.example.org/a.js", "", ""}, {"http://h/a.png", "", ""}, {"//cdn/a.js", "", ""},
		{"data:image/png;base64,AAAA", "", ""}, {"blob:abc", "", ""}, {"mailto:a@b.c", "", ""},
		{"javascript:void(0)", "", ""},
	} {
		if id, got := verdict(c.ref, want); id != c.id || got != c.got {
			t.Errorf("verdict(%q) = %q, %q; want %q, %q", c.ref, id, got, c.id, c.got)
		}
	}
}

// TestWaiver: an allow marker names its check and carries a reason; the comment closer is not the reason.
func TestWaiver(t *testing.T) {
	for _, c := range []struct{ line, id, want string }{
		{`<script src="a.js?v=0.1"></script> <!-- lint-ux-version: allow stale-version pinned build -->`, "stale-version", "pinned build"},
		{`var a = "x.png"; // lint-ux-version: allow no-version generated`, "no-version", "generated"},
		{`a { b: url(c.png) } /* lint-ux-version: allow no-version sprite */`, "no-version", "sprite"},
		{`x // lint-ux-version: allow stale-version`, "stale-version", ""},
		{`x // lint-ux-version: allow stale-version   `, "stale-version", ""},
		{`x // lint-ux-version: allow no-version reason`, "stale-version", ""},
		{`x // lint-ux-version: allow stale-versions reason`, "stale-version", ""},
		{`x // nothing`, "stale-version", ""},
	} {
		if got := waiver(c.line, c.id); got != c.want {
			t.Errorf("waiver(%q, %q) = %q; want %q", c.line, c.id, got, c.want)
		}
	}
}
