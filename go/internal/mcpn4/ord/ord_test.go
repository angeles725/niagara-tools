package ord

// Go twins of test_safety.TestWriteScope and test_box.TestChildOrd
// (go_server.go_twin("ord", "unit")).

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
)

func scope(t *testing.T, prefixes ...string) *Scope {
	t.Helper()
	s, err := NewScope(prefixes)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// refusal is the text of a *safety.Error, "" for nil; any other error fails t.
func refusal(t *testing.T, err error) string {
	t.Helper()
	if err == nil {
		return ""
	}
	se, ok := err.(*safety.Error)
	if !ok {
		t.Fatalf("%#v is not a *safety.Error", err)
	}
	return se.Msg
}

func accepts(t *testing.T, s *Scope, ords ...string) {
	t.Helper()
	for _, o := range ords {
		if msg := refusal(t, s.Check(o)); msg != "" {
			t.Errorf("%q refused: %s", o, msg)
		}
	}
}

const none = safety.ReasonScopeNone + ": every write is refused"

func TestNoPrefixRefusesEverything(t *testing.T) {
	s := scope(t)
	for _, o := range []any{"station:|slot:/A", "station:|h:1", nil} {
		if got := refusal(t, s.Check(o)); got != none {
			t.Fatalf("%v: %q", o, got)
		}
	}
	if got := refusal(t, s.RequireAny()); got != none {
		t.Fatalf("RequireAny: %q", got)
	}
	if refusal(t, scope(t, "station:|slot:/A").RequireAny()) != "" {
		t.Fatal("RequireAny refused a configured scope")
	}
}

func TestPrefixCoversItselfAndDescendants(t *testing.T) {
	accepts(t, scope(t, "station:|slot:/A"), "station:|slot:/A", "station:|slot:/A/", "station:|slot:/A/B",
		"station:|slot:/A/B/C", "station:|slot:/A//", "station:|slot:/A/B/..x")
}

func TestPrefixMatchRespectsSlotBoundaries(t *testing.T) {
	s := scope(t, "station:|slot:/A")
	for _, o := range []string{"station:|slot:/AB", "station:|slot:/", "station:|slot:/B/A", "station:|slot:/a"} {
		want := "ord " + pyjson.StrRepr(o) + " " + safety.ReasonScopeOutside
		if got := refusal(t, s.Check(o)); got != want {
			t.Errorf("%q: got %q want %q", o, got, want)
		}
	}
}

func TestTrailingSlashInPrefixIsEquivalent(t *testing.T) {
	s := scope(t, "station:|slot:/A/", "station:|slot:/C//")
	accepts(t, s, "station:|slot:/A/B", "station:|slot:/A", "station:|slot:/C/x")
	if refusal(t, s.Check("station:|slot:/AB")) == "" {
		t.Fatal("station:|slot:/AB accepted")
	}
}

func TestRootPrefixCoversTheWholeStation(t *testing.T) {
	accepts(t, scope(t, "station:|slot:/"), "station:|slot:/", "station:|slot:/Anything/Deep")
}

func TestAnyOfSeveralPrefixesMatches(t *testing.T) {
	s := scope(t, "station:|slot:/A", "station:|slot:/B")
	accepts(t, s, "station:|slot:/B/x", "station:|slot:/A")
	if refusal(t, s.Check("station:|slot:/C")) == "" {
		t.Fatal("station:|slot:/C accepted")
	}
}

func plainRefusal(o any) string {
	return "ord " + pyjson.Repr(o) + " " + safety.ReasonScopePlain +
		" (station:|slot:/<path>, nothing chained after the slot path)"
}

func TestParentTraversalAndForeignOrdsAreRefused(t *testing.T) {
	s := scope(t, "station:|slot:/A")
	for _, o := range []string{"station:|slot:/A/../B", "/A/B", "local:|foo", "", "station:|slot:/A/..",
		"Station:|slot:/A", "station:|slot:A"} {
		if got := refusal(t, s.Check(o)); got != plainRefusal(o) {
			t.Errorf("%q: got %q", o, got)
		}
	}
}

func TestNonStringOrdIsRefused(t *testing.T) {
	s := scope(t, "station:|slot:/")
	for _, o := range []any{nil, 5, []any{"station:|slot:/"}} {
		if got := refusal(t, s.Check(o)); got != plainRefusal(o) {
			t.Errorf("%v: got %q", o, got)
		}
	}
}

func TestAChainedOrdAfterTheSlotPathIsRefused(t *testing.T) {
	s := scope(t, "station:|slot:/A")
	for _, o := range []string{"station:|slot:/A|h:1f", "station:|slot:/A/B|slot:../X", "station:|slot:/A|slot:/Other",
		"station:|h:1f", "station:|slot:/A/B|bql:select *", "station:|slot:/A|"} {
		if got := refusal(t, s.Check(o)); got != plainRefusal(o) {
			t.Errorf("%q: got %q", o, got)
		}
	}
}

func TestAPrefixThatIsNotAPlainSlotOrdIsRefusedAtStartup(t *testing.T) {
	for _, bad := range []string{"station:|slot:/A|h:1", "station:|h:1", "slot:/A", "", "station:", "station:|slot:/A/../B", "station:|slot:/.."} {
		_, err := NewScope([]string{"station:|slot:/A", bad})
		want := "--write-scope " + pyjson.StrRepr(bad) + " must be a plain slot ORD such as " +
			"'station:|slot:/Drivers' (no '|' after the slot path)"
		if got := refusal(t, err); got != want {
			t.Errorf("%q: got %q", bad, got)
		}
	}
}

func TestEveryParentFormJoinsWithExactlyOneSlash(t *testing.T) {
	for _, c := range [][2]string{{"station:", "station:|slot:/X"}, {"station:|", "station:|slot:/X"},
		{"station:|slot:/", "station:|slot:/X"}, {"station:|slot:/A", "station:|slot:/A/X"},
		{"station:|slot:/A/", "station:|slot:/A/X"}, {"station:|slot:/A/B", "station:|slot:/A/B/X"}} {
		got, err := ChildOrd(c[0], "X")
		if err != nil || got != c[1] || strings.Contains(got, "//") {
			t.Errorf("%q: %q %v", c[0], got, err)
		}
	}
}

func TestBadNamesAndBadParentsAreRefused(t *testing.T) {
	for _, bad := range []string{"", "a/b", "a|b", ".."} {
		_, err := ChildOrd("station:|slot:/", bad)
		want := "ORD child name must be one plain slot name, got " + pyjson.StrRepr(bad)
		if ve, ok := err.(*pyjson.ValueError); !ok || ve.Msg != want {
			t.Errorf("%q: %v", bad, err)
		}
	}
	if got, err := ChildOrd("station:|slot:/", "."); err != nil || got != "station:|slot:/." {
		t.Errorf("'.' is a name Python accepts: %q %v", got, err)
	}
	_, err := ChildOrd("station:|slot://A", "X")
	if ve, ok := err.(*pyjson.ValueError); !ok || ve.Msg != "parent ORD is malformed (double slashes?): 'station:|slot://A'" {
		t.Errorf("broken parent: %v", err)
	}
}

func TestJoinOrdWalksARelativePathAndAllowsTheEmptyPath(t *testing.T) {
	for _, c := range [][3]string{{"station:|slot:/", "A/B", "station:|slot:/A/B"}, {"station:|slot:/A/", "B", "station:|slot:/A/B"},
		{"station:|slot:/A/", "", "station:|slot:/A"}, {"station:|slot:/", "", "station:|slot:/"},
		{"station:", "", "station:"}, {"station:", "A", "station:|slot:/A"}} {
		if got, err := JoinOrd(c[0], c[1]); err != nil || got != c[2] {
			t.Errorf("%q + %q: %q %v", c[0], c[1], got, err)
		}
	}
	for _, rel := range []string{"A//B", "/A", "A/"} {
		if _, err := JoinOrd("station:|slot:/", rel); err == nil {
			t.Errorf("%q joined", rel)
		}
	}
}
