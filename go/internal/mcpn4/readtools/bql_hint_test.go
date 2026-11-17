package readtools_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
)

// Field retro 2026-11-16 row U6: `where name = 'Numeric$20Writable40005'` returned no rows, because
// BqlQuery decodes $20 to a space inside the query while slot names are stored escaped. An empty
// result whose quoted literal holds a $xx escape names the form that matches.
func TestEmptyResultWithAnEscapedLiteralHintsTheDollar24Form(t *testing.T) {
	s := alarmServer(t, 3)
	s.Station.OrdRoutes = append(s.Station.OrdRoutes, readtoolstest.OrdRoute{Needle: "bql:", Body: "Slot Path\n"})
	out := callOK(t, s, "n4_bql_query", "base", "station:|slot:/Drivers",
		"query", "select slotPath from control:NumericWritable where name = 'Numeric$20Writable40005'")
	hint, _ := out.Get("hint")
	text, _ := hint.(string)
	if !strings.Contains(text, "'Numeric$20Writable40005' is read by BQL as 'Numeric Writable40005'") ||
		!strings.Contains(text, "'Numeric$2420Writable40005'") {
		t.Errorf("hint = %q", text)
	}
}

func TestNoHintWhenRowsCameBackOrNoEscape(t *testing.T) {
	for _, c := range []struct{ query, body string }{
		{"select slotPath from control:NumericWritable where name = 'Numeric$20Writable40005'", "Slot Path\nslot:/A\n"},
		{"select slotPath from control:NumericWritable where name = 'Power'", "Slot Path\n"},
	} {
		s := alarmServer(t, 3)
		s.Station.OrdRoutes = append(s.Station.OrdRoutes, readtoolstest.OrdRoute{Needle: "bql:", Body: c.body})
		out := callOK(t, s, "n4_bql_query", "base", "station:|slot:/Drivers", "query", c.query)
		if hint, ok := out.Get("hint"); ok {
			t.Errorf("%q: unexpected hint %v", c.query, hint)
		}
	}
}
