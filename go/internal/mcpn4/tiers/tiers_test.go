package tiers

import (
	"strings"
	"testing"
)

func TestTierTable(t *testing.T) {
	t.Parallel()
	for version, want := range map[string]string{
		"4.13.2.18": "A", "4.14.0.162": "A", "4.14": "A", "4.13.0.0.1": "A", "04.014": "A", " 4.14\v": "A",
		"4.15.3.28": "B", "4.3.58.18": "B",
		"4.12.1.16": "C", "4.4.73.24": "C", "4.30.1": "C", "4.1.27": "C", "5.0.0": "C", "3.8.401": "C",
		"": "C", "garbage": "C", "4": "C", "v4.14.0": "C", "4.14.": "C", "4.14\n.1": "C", "４.14": "C",
		"99999999999999999999999.14": "C",
	} {
		if got := TierOf(version); got != want {
			t.Errorf("TierOf(%q) = %s, want %s", version, got, want)
		}
	}
}

func TestProductVersionIsReadFromTheAboutDocument(t *testing.T) {
	t.Parallel()
	for xml, want := range map[string]string{
		`<obj is="obix:About"><str name="serverName" val="h"/><str name="productVersion" val="4.15.3.28"/></obj>`: "4.15.3.28",
		`<obj><str val="4.13.2.18" name="productVersion" /></obj>`:                                                "4.13.2.18",
		`<str name = "productVersion"` + "　" + `val=` + " " + `"4.1" val="4.2">`:                                  "4.2",
		`<str name="productVersion" val="">`:                                                                      "",
		`<strong name="productVersion" val="1">`:                                                                  "",
		`<stré name="productVersion" val="1">`:                                                                    "",
		`<str xname="productVersion" val="1">`:                                                                    "",
		`<str éname="productVersion" val="1">`:                                                                    "",
		`<str name="productVersion" val="1"`:                                                                      "",
		"":                                                                                                        "", "<obj/>": "", `<str name="productName" val="Niagara"/>`: "",
	} {
		if got := ProductVersion(xml); got != want {
			t.Errorf("ProductVersion(%q) = %q, want %q", xml, got, want)
		}
	}
}

func TestWriteBlockPolicy(t *testing.T) {
	t.Parallel()
	in := func(tier, name string) map[string]map[string]bool {
		return map[string]map[string]bool{tier: {name: true}}
	}
	for _, tc := range []struct {
		tier, version string
		optIn         map[string]map[string]bool
		blocked       bool
	}{
		{"A", "4.14.0.162", nil, false},
		{"B", "4.15.3.28", in("B", "S"), false},
		{"C", "4.10.0", in("C", "S"), false},
		{"B", "4.15.3.28", in("B", "other"), true},
		{"B", "4.15.3.28", in("C", "S"), true},
		{"C", "", in("B", "S"), true},
	} {
		if got := WriteBlock(tc.tier, tc.version, "S", tc.optIn); (got != "") != tc.blocked {
			t.Errorf("WriteBlock(%s, %q, %v) = %q, blocked want %v", tc.tier, tc.version, tc.optIn, got, tc.blocked)
		}
	}
}

func TestRefusalsNameTheRoutes(t *testing.T) {
	t.Parallel()
	b, c := WriteBlock("B", "4.15.3.28", "S", nil), WriteBlock("C", "", "S", nil)
	for _, want := range []struct{ text, needle string }{
		{b, ReasonTier}, {b, "Workbench"}, {b, "dry run"}, {b, "--allow-tier-b S"}, {b, "4.15.3.28"},
		{c, ReasonTier}, {c, "Workbench"}, {c, "dry run"}, {c, "--allow-tier-c S"}, {c, "unknown"},
		{c, "reg.loadContract"}, {c, "loadRoot"}, {c, "scratch"},
	} {
		if !strings.Contains(want.text, want.needle) {
			t.Errorf("%q lacks %q", want.text, want.needle)
		}
	}
}
