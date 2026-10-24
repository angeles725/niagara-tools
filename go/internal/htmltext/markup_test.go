package htmltext

import "testing"

// Values from CPython 3.15: the three re.sub passes of lint-manual-labels' HTML manual view, each match
// replaced by the newlines it held.
func TestDropMarkup(t *testing.T) {
	for in, want := range map[string]string{
		"<p>a<b>b</b>\n<i\nx>c</i>":    "ab\n\nc",
		"<script>x\n</script>y":        "\ny",
		"<SCRIPT a>x</sCrIpT\n >y":     "\ny",
		"<scripts>x</scripts>y":        "xy",
		"<script>never\nclosed":        "never\nclosed",
		"<style>a</script>b</style >c": "c",
		"<!-- a\n-->b<!-->c-->d":       "\nbd",
		"<!-- open":                    "<!-- open",
		"a < b and c > d":              "a  d",
		"x<y":                          "x<y",
		"<script>a</script\t　>b":       "b",
		"<scriptx>a</scriptx>b":        "ab",
		"<scripté>a</scripté>b":        "ab",
	} {
		if got := DropMarkup(in); got != want {
			t.Errorf("DropMarkup(%q) = %q, want %q", in, got, want)
		}
	}
	in := "<p>a<pre x>\n\"v\"</PRE>b<code>c</code>d<!-- <pre> -->e<precise>f</precise></p>"
	if got, want := DropMarkup(in, "pre", "code"), "a\nbdef"; got != want {
		t.Errorf("DropMarkup(%q, pre, code) = %q, want %q", in, got, want)
	}
}
