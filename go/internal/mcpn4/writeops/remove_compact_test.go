package writeops_test

// n4_remove_component compact=true with acknowledge_links (issue #971 item 4): the compact dry run
// replaces only the inverse by its summary; the outgoing links the operator consents to stay listed
// in the plan (outgoing_links_broken), and the plan the token hashes is the full run's.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

func TestCompactWithAcknowledgeLinksKeepsTheOutgoingLinksListed(t *testing.T) {
	full, err := esPlan(t, "x", "acknowledge_links", true)
	if err != nil {
		t.Fatal(err)
	}
	compact, err := esPlan(t, "x", "acknowledge_links", true, "compact", true)
	if err != nil {
		t.Fatal(err)
	}
	if full.Summary != nil {
		t.Errorf("a plan without compact has a summary: %s", esJSON(t, full.Summary))
	}
	if got := get(compact.Summary, "outgoing_links"); got != 1 {
		t.Errorf("summary outgoing_links = %v, want the count 1 beside the list: %s", got, esJSON(t, compact.Summary))
	}
	sess := &session.Session{StationName: "FakeStation"}
	shown, hash, err := plan.Hashed("n4_remove_component", compact, sess)
	if err != nil {
		t.Fatal(err)
	}
	links := esJSON(t, get(shown, "outgoing_links_broken"))
	for _, want := range []string{`"target_slot": "in10"`, `"source_slot": "x"`, `"target": "station:|slot:/Folder/Q"`} {
		if !strings.Contains(links, want) {
			t.Errorf("outgoing_links_broken lacks %s: %s", want, links)
		}
	}
	if _, fullHash, _ := plan.Hashed("n4_remove_component", full, sess); fullHash != hash {
		t.Errorf("compact plan_hash %s differs from the full dry run's %s", hash, fullHash)
	}
}

// Without compact nothing changes: no summary, the inverse listed, the links listed as before.
func TestRemoveWithoutCompactKeepsItsInverseAndLinks(t *testing.T) {
	p, err := esPlan(t, "x", "acknowledge_links", true)
	if err != nil {
		t.Fatal(err)
	}
	shown, _, err := plan.Hashed("n4_remove_component", p, &session.Session{StationName: "FakeStation"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := shown.Get("inverse"); !ok || p.Summary != nil || get(shown, "outgoing_links_broken") == nil {
		t.Errorf("plan = %s", esJSON(t, shown))
	}
}
