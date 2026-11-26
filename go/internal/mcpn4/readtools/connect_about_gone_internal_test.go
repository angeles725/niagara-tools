package readtools

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
)

type goneAbout struct{ Client }

func (goneAbout) About(float64, int) (string, error) {
	return "", &box.Error{Kind: box.KindHTTPStatus, Msg: "HTTP 410 from station", Channel: "obix", Key: "about", Code: 410}
}

// A 410 on /obix/about/ is the station's disabled oBIX network (live, 4.14.0.162 lab station):
// the version error says so and how to restore the version, instead of a bare "unreadable".
func TestDetectVersionNamesADisabledObixNetwork(t *testing.T) {
	version, why := detectVersion(goneAbout{})
	if version != "" {
		t.Fatalf("version %q", version)
	}
	for _, want := range []string{"HTTP 410", "Drivers/ObixNetwork", "enabled"} {
		if !strings.Contains(why, want) {
			t.Errorf("missing %q in %q", want, why)
		}
	}
}
