package blackbox

// n4_spy_page reads a registry type page, named by its type spec (lab retro 2026-11-15 row 3).

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

func TestSpyPageReadsARegistryTypePageByItsTypeSpec(t *testing.T) {
	r, sv := shxSpyRig(t)
	shxServeSpy(sv, "sysManagers/registryManager/types/kitBench:Dash", boxstation.SpyPage{
		Body: []byte("<html><body><h1>Type kitBench:Dash</h1><table><tr><td>superType</td><td>baja:Component</td></tr></table></body></html>")})
	out := r.Srv.OK("n4_spy_page", Args{"path": "sysManagers/registryManager/types/kitBench:Dash"})
	if out["status"] != "ok" || out["path"] != "sysManagers/registryManager/types/kitBench:Dash" || out["row_count"] != 1.0 {
		t.Errorf("out = %v", out)
	}
	wantStrings(t, "spy requests", shxSpyRequests(sv), []string{"/ord?spy:/sysManagers/registryManager/types/kitBench%3ADash"})
	for _, bad := range []string{"sysManagers/registryManager/types/kitBench:Dash/x", "sysManagers/kitBench:Dash",
		"sysManagers/registryManager/types/x:y/../..", "station:|slot:/"} {
		if text := r.Srv.Err("n4_spy_page", Args{"path": bad}); !strings.Contains(text, "refusing spy path") {
			t.Errorf("%q: %s", bad, text)
		}
	}
	if got := shxSpyRequests(sv); len(got) != 1 {
		t.Errorf("a refused path was fetched: %v", got)
	}
}
