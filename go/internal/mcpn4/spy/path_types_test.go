package spy

import (
	"strings"
	"testing"
)

// The registry's type pages are named by a type spec (lab retro 2026-11-15 row 3): one
// module:Type segment directly under sysManagers/registryManager/types/ is a plain name.
func TestCleanPathKeepsATypeSpecDirectlyUnderTheRegistryTypes(t *testing.T) {
	for in, want := range map[string]string{
		"sysManagers/registryManager/types/KitBenchDash:KitBenchDash":   "sysManagers/registryManager/types/KitBenchDash:KitBenchDash",
		"/sysManagers/registryManager/types/control:NumericWritable/":   "sysManagers/registryManager/types/control:NumericWritable",
		"sysManagers/registryManager/types/baja:Component":              "sysManagers/registryManager/types/baja:Component",
		"sysManagers/registryManager/types/kitControl_x:Numeric_Const2": "sysManagers/registryManager/types/kitControl_x:Numeric_Const2"} {
		got, err := CleanPath(in)
		if err != nil || got != want {
			t.Errorf("CleanPath(%q) = %q, %v; want %q", in, got, err, want)
		}
		if u := URLPath(got); !IsURLPath(u) || !strings.Contains(u, "%3A") {
			t.Errorf("URLPath(%q) = %q, IsURLPath %v", got, u, IsURLPath(u))
		}
	}
}

func TestCleanPathStillRefusesAColonAnywhereElse(t *testing.T) {
	// a colon outside the one exempt segment is refused first, whatever else the path holds
	for in, why := range map[string]string{
		"sysManagers/registryManager/types/a:b/c":         "scheme",
		"sysManagers/registryManager/types/x:y/../..":     "scheme",
		"sysManagers/registryManager/types/../types/a:b":  "scheme",
		"sysManagers/registryManager/types/a:b:c":         "scheme",
		"sysManagers/registryManager/types/:b":            "scheme",
		"sysManagers/registryManager/types/a:":            "scheme",
		"sysManagers/registryManager/types/a b:c":         "scheme",
		"sysManagers/registryManager/types/a.b:c":         "scheme",
		"sysManagers/registryManager/types/http://x":      "scheme",
		"sysManagers/registryManager/types/a:b|slot:/":    "scheme",
		"sysManagers/registryManager/types/a:b?x=1":       "scheme",
		"sysManagers/registryManager/types/a%3Ab":         "percent",
		"sysManagers/registryManager/a:b":                 "scheme",
		"sysManagers/registryManager/types/x/a:b":         "scheme",
		"other/sysManagers/registryManager/types/a:b":     "scheme",
		"SysManagers/RegistryManager/Types/a:b":           "scheme",
		"types/a:b":                                       "scheme",
		"spy:/sysManagers/registryManager/types/a:b":      "scheme",
		"station:|slot:/":                                 "scheme",
		"sysManagers/registryManager/types/station:|slot": "scheme"} {
		got, err := CleanPath(in)
		if err == nil || got != "" || !strings.Contains(err.Error(), why) {
			t.Errorf("CleanPath(%q) = %q, %v; want a refusal naming %q", in, got, err, why)
		}
	}
	for _, bad := range []string{"/ord?spy:/sysManagers/registryManager/types/a%3Ab/c",
		"/ord?spy:/sysManagers/registryManager/a%3Ab", "/ord?spy:/sysManagers/registryManager/types/a:b"} {
		if IsURLPath(bad) {
			t.Errorf("IsURLPath(%q) = true", bad)
		}
	}
}
