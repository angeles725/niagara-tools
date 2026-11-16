package box

import (
	"errors"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// A spy read only ever GETs a URL of the spy root: anything else fails before a request (the
// client's base URL answers nothing here, so a request would be a transport failure instead).
func TestGetSpyRefusesEveryURLOutsideTheSpyRootBeforeARequest(t *testing.T) {
	c, err := New(Config{BaseURL: "https://127.0.0.1:1", Username: "u", Password: "p", Timeout: 1})
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"/file/px/x.px", "/ord?slot:/", "/ord?spy:/util/gc", "/ord?spy:/a%7Cb", "/ord?spy:/..",
		"/ord?spy:/metrics?x=1", "https://other.invalid/ord?spy:/", "/box/"} {
		_, err := c.GetSpy(bad, 1, 0)
		var ve *pyjson.ValueError
		if !errors.As(err, &ve) || !strings.Contains(ve.Msg, "get_spy reads") {
			t.Errorf("GetSpy(%q) = %v, want the refusal", bad, err)
		}
	}
	if _, err := c.GetSpy("/ord?spy:/metrics", 1, 0); err == nil || strings.Contains(err.Error(), "get_spy reads") {
		t.Errorf("a spy URL was not sent: %v", err)
	}
}

// A registry type page is a spy URL (its module:Type segment escaped); a colon anywhere else is
// still refused before a request.
func TestGetSpySendsARegistryTypePageOnly(t *testing.T) {
	c, err := New(Config{BaseURL: "https://127.0.0.1:1", Username: "u", Password: "p", Timeout: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetSpy("/ord?spy:/sysManagers/registryManager/types/kitBench%3ADash", 1, 0); err == nil ||
		strings.Contains(err.Error(), "get_spy reads") {
		t.Errorf("the type page was not sent: %v", err)
	}
	for _, bad := range []string{"/ord?spy:/sysManagers/registryManager/types/kitBench%3ADash/x",
		"/ord?spy:/sysManagers/registryManager/kitBench%3ADash", "/ord?spy:/metrics%3Ax"} {
		if _, err := c.GetSpy(bad, 1, 0); err == nil || !strings.Contains(err.Error(), "get_spy reads") {
			t.Errorf("GetSpy(%q) = %v, want the refusal", bad, err)
		}
	}
}
