package spapoll

import "testing"

// TestExitsCatch pins the handler-level reading of a catch (issue #966 item 3). The handler text is what
// swallowsThenStamps passes: a block catch's "{ ... " body, or a promise catch's "(handler" argument, each
// without its closing byte.
func TestExitsCatch(t *testing.T) {
	tests := []struct {
		name, handler string
		want          bool
	}{
		{"block: throw", "{ paint(); throw e; ", true},
		{"block: return", "{ return; ", true},
		{"block: return of a call", "{ return Promise.reject(e); ", true},
		{"block: neither", "{ paint(); ", false},
		{"block: a conditional exit is an exit", "{ if (x) { return; } ", true},
		{"block: exit after a nested callback", "{ list.map(i => i); return; ", true},
		{"block: return in a nested arrow body", "{ list.forEach(i => { return; }); ", false},
		{"block: return in a nested function", "{ list.forEach(function (i) { return i; }); ", false},
		{"block: throw in a nested callback", "{ later(() => { throw e; }); ", false},
		{"block: Promise.reject not returned", "{ Promise.reject(e); ", false},
		{"promise: concise Promise.reject", "(e => Promise.reject(e)", true},
		{"promise: concise Promise.reject, async", "(async e => Promise.reject(e)", true},
		{"promise: concise Promise.reject after a nested arrow", "(e => ok(x => x) || Promise.reject(e)", true},
		{"promise: block returning Promise.reject", "(e => { return Promise.reject(e); }", true},
		{"promise: block throw", "(e => { throw e; }", true},
		{"promise: function throw", "(function (e) { throw e; }", true},
		{"promise: function return", "(function named(e) { return null; }", true},
		{"promise: concise call", "(e => paint()", false},
		{"promise: concise value", "(e => 0", false},
		{"promise: block without exit", "(e => { paint(); }", false},
		{"promise: Promise.reject only in a nested callback", "(e => later(() => Promise.reject(e))", false},
		{"promise: return only in a nested callback", "(e => { later(() => { return 1; }); }", false},
		{"promise: function with a nested throw", "(function (e) { later(function () { throw e; }); }", false},
		{"promise: Promise.reject not returned", "(e => { Promise.reject(e); }", false},
		{"promise: a named handler", "(onError", false},
	}
	for _, tt := range tests {
		if got := exitsCatch(tt.handler); got != tt.want {
			t.Errorf("%s: exitsCatch(%q) = %v, want %v", tt.name, tt.handler, got, tt.want)
		}
	}
}
