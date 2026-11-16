package spy

import "testing"

// Verifier probes for the registry type pages: only one module:Type segment directly under
// sysManagers/registryManager/types/ holds a ':', in CleanPath, IsURLPath and page links.

func TestCleanPathAcceptsOnlyTheTypePageShape(t *testing.T) {
	const types = "sysManagers/registryManager/types/"
	for in, want := range map[string]string{
		types + "a:b":           types + "a:b",
		"/" + types + "a_1:B2/": types + "a_1:B2",
		types + "util:gc":       types + "util:gc",
	} {
		if got, err := CleanPath(in); err != nil || got != want {
			t.Errorf("CleanPath(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{types + "a:b/..", types + "a:b/.", types + "a:b ", types + " a:b", "//" + types + "a:b",
		types + "a:b//", types + "a:b#x", types + "a:b;x", types + "a:b\\c", types + "a\\:b", types + "a:b\n", types + "á:b",
		types + "a-b:c", types + "%2E%2E", types + "a%2Fb", "x/" + types + "a:b", "metrics:x", "http://x"} {
		if got, err := CleanPath(in); err == nil {
			t.Errorf("CleanPath(%q) = %q, want a refusal", in, got)
		}
	}
}

func TestIsURLPathAcceptsOnlyTheCanonicalTypePageURL(t *testing.T) {
	const types = Root + "sysManagers/registryManager/types/"
	if !IsURLPath(types + "a%3Ab") {
		t.Errorf("IsURLPath(%q) = false", types+"a%3Ab")
	}
	for _, u := range []string{types + "a%3ab", types + "a:b", Root + "sysManagers%2FregistryManager%2Ftypes/a%3Ab",
		types + "a%3Ab%2F..", types + "%2E%2E/a%3Ab", types + "a%3Ab%3Ac", types + "a%253Ab", types + "a%3Ab%7Cslot",
		types + "a%5C%3Ab", "/ord?station:|slot:/"} {
		if IsURLPath(u) {
			t.Errorf("IsURLPath(%q) = true", u)
		}
	}
}

func TestATypesListPageLinksToItsTypePagesOnly(t *testing.T) {
	body := `<html><body><a href="/ord?spy:/sysManagers/registryManager/types/control:NumericWritable">nw</a>` +
		`<a href="/ord?spy:/sysManagers/registryManager/types/x:y/z">deeper</a><a href="spy:/metrics">m</a></body></html>`
	p, err := Parse(body, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	want := []Link{{Name: "nw", Path: "sysManagers/registryManager/types/control:NumericWritable"}, {Name: "m", Path: "metrics"}}
	if len(p.Links) != len(want) {
		t.Fatalf("links = %+v, want %+v", p.Links, want)
	}
	for i := range want {
		if p.Links[i] != want[i] {
			t.Errorf("link %d = %+v, want %+v", i, p.Links[i], want[i])
		}
	}
}
