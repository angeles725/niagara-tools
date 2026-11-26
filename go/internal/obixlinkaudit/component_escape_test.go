package obixlinkaudit

import (
	"path/filepath"
	"testing"
)

// TestEscapedComponentNameIsGetAsStored: a station name with an escape ($20 for a space) is sent unchanged,
// the form the station's /config/ path matches (#938).
func TestEscapedComponentNameIsGetAsStored(t *testing.T) {
	t.Parallel()
	s := newStation(t, false, 404, "")
	wiring := filepath.Join("..", "..", "..", "tests", "fixtures", "obix-link-audit", "wiring-map.md")
	env := map[string]string{"OBIX_USER": "reader", "OBIX_PASS": "secret"}
	run(env, "--map", wiring, "--obix", s.URL+"/obix", "--component", "/Plant/Cold$20Room")
	got := s.requests()
	if len(got) != 1 || got[0] != "GET /obix/config/Plant/Cold$20Room/ reader:secret" {
		t.Errorf("requests %q, want one GET of /obix/config/Plant/Cold$20Room/", got)
	}
}
