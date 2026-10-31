package portparity

import (
	"reflect"
	"testing"
)

func TestParseWaivers(t *testing.T) {
	t.Parallel()
	text := "# port-parity waivers\n" +
		"ref: ../reference/viewer\n" +
		"port: src/rc\n" +
		"\n" +
		"function eqData -- folded into eqStore [decision: W5]\n" +
		"id noModel -- the empty state moved to a banner [decision: user 2026-10-13]\n" +
		"file sim.js -- simulator not shipped\n" +
		"rename: update-banner.js -> banner.js\n" +
		"widget foo -- unknown kind [decision: W1]\n" +
		"function -- no name [decision: W1]\n" +
		"id lonely\n" +
		"id tagged -- reason [decision: ]\n"
	wf := ParseWaivers(text)
	if wf.Ref != "../reference/viewer" || wf.Port != "src/rc" {
		t.Fatalf("headers ref=%q port=%q", wf.Ref, wf.Port)
	}
	want := []Waiver{
		{Line: 5, Kind: KindFunction, Name: "eqData", Reason: "folded into eqStore", Decision: "W5"},
		{Line: 6, Kind: KindID, Name: "noModel", Reason: "the empty state moved to a banner", Decision: "user 2026-10-13"},
		{Line: 7, Kind: KindFile, Name: "sim.js", Reason: "simulator not shipped"},
		{Line: 8, Kind: KindRename, Of: KindFile, Name: "update-banner.js", To: "banner.js"},
		{Line: 9, Problem: `malformed waiver (want "<file|id|function> <name> -- <reason> [decision: <ref>]" or "rename: [function|id] <ref> -> <port> -- <reason>"): widget foo -- unknown kind [decision: W1]`},
		{Line: 10, Problem: `malformed waiver (want "<file|id|function> <name> -- <reason> [decision: <ref>]" or "rename: [function|id] <ref> -> <port> -- <reason>"): function -- no name [decision: W1]`},
		{Line: 11, Problem: `malformed waiver (want "<file|id|function> <name> -- <reason> [decision: <ref>]" or "rename: [function|id] <ref> -> <port> -- <reason>"): id lonely`},
		{Line: 12, Kind: KindID, Name: "tagged", Reason: "reason"},
	}
	if !reflect.DeepEqual(wf.Waivers, want) {
		t.Errorf("waivers =\n%#v\nwant\n%#v", wf.Waivers, want)
	}
}

func TestParseWaiversCRLFAndNoHeaders(t *testing.T) {
	t.Parallel()
	wf := ParseWaivers("function f -- r [decision: D1]\r\n")
	if wf.Ref != "" || wf.Port != "" {
		t.Errorf("no headers expected, got ref=%q port=%q", wf.Ref, wf.Port)
	}
	if len(wf.Waivers) != 1 || wf.Waivers[0].Decision != "D1" || wf.Waivers[0].Reason != "r" {
		t.Errorf("waivers = %#v", wf.Waivers)
	}
}

func TestParseKindRenames(t *testing.T) {
	t.Parallel()
	wf := ParseWaivers("rename: function drawEq -> renderEq -- renamed in the port\nrename: id oldPanel -> newPanel\n")
	want := []Waiver{
		{Line: 1, Kind: KindRename, Of: KindFunction, Name: "drawEq", To: "renderEq", Reason: "renamed in the port"},
		{Line: 2, Kind: KindRename, Of: KindID, Name: "oldPanel", To: "newPanel"},
	}
	if !reflect.DeepEqual(wf.Waivers, want) {
		t.Errorf("waivers =\n%#v\nwant\n%#v", wf.Waivers, want)
	}
}
