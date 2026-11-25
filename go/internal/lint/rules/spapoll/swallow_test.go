package spapoll

import "testing"

// TestSwallowsThenStampsReadsOnlyAnUnconditionalStampAfterTheCatch pins the `conditional` regex and the brace-depth
// filter of swallow.go (issue #966 item 1): a lastOk* stamp behind if, ?:, &&, || or at another brace depth than the
// catch is not the swallowed-failure shape; the plain stamp after the catch is.
func TestSwallowsThenStampsReadsOnlyAnUnconditionalStampAfterTheCatch(t *testing.T) {
	const catchPaint = "try { await load(); } catch (e) { paint(); }\n"
	const stamp = "lastOkAt = Date.now();"
	const catchNested = "try { await load(); } catch (e) { later(() => { throw e; }); }\n"
	tests := []struct {
		name string
		body string
		want bool
	}{
		{"unconditional stamp after the catch", catchPaint + stamp, true},
		{"promise catch then a plain stamp", "await load().catch(e => paint());\n" + stamp, true},
		{"stamp behind if", catchPaint + "if (ready) " + stamp, false},
		{"stamp behind else", catchPaint + "if (a) go(); else " + stamp, false},
		{"stamp in a ?: arm", catchPaint + "ok ? " + stamp + " : 0;", false},
		{"stamp after &&", catchPaint + "ready && (" + stamp + ");", false},
		{"stamp after ||", catchPaint + "cached || (" + stamp + ");", false},
		{"stamp inside a loop", catchPaint + "for (;;) " + stamp, false},
		{"stamp in a nested block", catchPaint + "{ " + stamp + " }", false},
		{"stamp inside the try block", "try { await load(); " + stamp + " } catch (e) { paint(); }\n", false},
		{"stamp in a function declared after the catch", catchPaint + "function later() { " + stamp + " }", false},
		{"catch rethrows", "try { await load(); } catch (e) { paint(); throw e; }\n" + stamp, false},
		{"promise catch hands back Promise.reject", "await load().catch(e => Promise.reject(e));\n" + stamp, false},
		{"promise catch block returns Promise.reject", "await load().catch(e => { return Promise.reject(e); });\n" + stamp, false},
		{"promise catch returns only in a nested callback", "await load().catch(e => { list.forEach(i => { return; }); });\n" + stamp, true},
		{"block catch throws only in a nested callback", catchNested + stamp, true},
		{"catch returns", "try { await load(); } catch (e) { return; }\n" + stamp, false},
	}
	for _, tt := range tests {
		if got := swallowsThenStamps(tt.body); got != tt.want {
			t.Errorf("%s: swallowsThenStamps = %v, want %v", tt.name, got, tt.want)
		}
	}
}
