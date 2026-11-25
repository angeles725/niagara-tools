package writeops

import (
	"os"
	"strings"
	"testing"
)

// The tool description and the Go refusal text say the same measured facts and the same recovery
// (#1067 item 2): the Go constants are the source, the registry must quote them.
func TestTheRegistryDescriptionQuotesTheRunawayConstants(t *testing.T) {
	data, err := os.ReadFile("../registry/registry.json")
	if err != nil {
		t.Fatal(err)
	}
	desc := string(data)
	if i := strings.Index(desc, `"name": "n4_make_station_template"`); i < 0 {
		t.Fatal("n4_make_station_template is not in the registry")
	} else {
		desc = desc[i:]
		desc = desc[:strings.Index(desc, `"inputSchema"`)]
	}
	facts := msgMakeCause[strings.Index(msgMakeCause, "66.9 GB"):strings.Index(msgMakeCause, ")")]
	recovery := strings.TrimPrefix(msgMakeRecovery[:strings.Index(msgMakeRecovery, ";")], "recovery: ")
	for _, want := range []string{facts, recovery} {
		if !strings.Contains(strings.ToLower(desc), strings.ToLower(want)) {
			t.Errorf("the registry description does not quote %q", want)
		}
	}
}
