package blackbox

// n4_bql_query and n4_inventory on station answers that are not a CSV table (port of
// mcp-n4-kit/tests/test_bql.py, TestNonCsvAnswers; retro 2026-10-02 D1, D7, D8): an HTTP 200
// that is not a CSV table is an error, never a zero-row result. The CSV fixtures under
// mcp-n4-kit/tests/fixtures are recorded station output, anonymized, read in place.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

const bqlQuery = "select slotPath, name from driver:Device"

// bqlErr answers the device query with body as ctype and returns the tool error text.
func bqlErr(r *Rig, sv *boxstation.Servlets, body []byte, ctype string) string {
	r.T.Helper()
	routeOrds(sv, boxstation.OrdRoute{Needle: "driver:Device", Body: body, ContentType: ctype})
	return r.Srv.Err("n4_bql_query", Args{"base": "/Drivers", "query": bqlQuery})
}

func TestAnHTMLPageIsABqlErrorWhateverItsContentType(t *testing.T) {
	r, sv := servletRig(t, nil)
	page := []byte("<!DOCTYPE html>\n<html><body>Login</body></html>\n")
	for _, c := range []struct {
		body  []byte
		ctype string
	}{{page, "text/html; charset=UTF-8"}, {page, "text/csv"}, {[]byte("Slot Path,Name\n"), "text/html"}} {
		if text := bqlErr(r, sv, c.body, c.ctype); !strings.Contains(text, "HTML") {
			t.Errorf("%q as %s: %q", c.body, c.ctype, text)
		}
	}
}

func TestAnEmptyBodyIsABqlErrorNotZeroRows(t *testing.T) {
	r, sv := servletRig(t, nil)
	if text := bqlErr(r, sv, []byte{}, ""); !strings.Contains(text, "no header row") {
		t.Errorf("refusal = %q", text)
	}
}

func TestAnEmptyBodyFailsTheInventoryInsteadOfZeroTotals(t *testing.T) {
	r, sv := servletRig(t, nil)
	routeOrds(sv,
		boxstation.OrdRoute{Needle: "driver:DeviceNetwork", Body: kitFixture(t, "bql_networks.csv")},
		boxstation.OrdRoute{Needle: "driver:Device", Body: []byte{}},
		boxstation.OrdRoute{Needle: "control:ControlPoint", Body: kitFixture(t, "bql_points.csv")})
	if text := r.Srv.Err("n4_inventory", nil); !strings.Contains(text, "no header row") {
		t.Errorf("refusal = %q", text)
	}
}

func TestARaggedRowIsABqlError(t *testing.T) {
	r, sv := servletRig(t, nil)
	if text := bqlErr(r, sv, []byte("Slot Path,Name\nslot:/A,a\nslot:/B\n"), ""); !strings.Contains(text, "record 2 has 1 cell") {
		t.Errorf("refusal = %q", text)
	}
}
