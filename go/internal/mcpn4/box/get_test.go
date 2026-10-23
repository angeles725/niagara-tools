package box

import (
	"testing"
)

func TestQuoteAndHeaderInt(t *testing.T) {
	if got, _ := quote("station:|slot:/A B/é~_.-"); got != "station%3A%7Cslot%3A%2FA%20B%2F%C3%A9~_.-" {
		t.Fatalf("quote = %q", got)
	}
	for in, want := range map[string]string{"12": "12", " +1_000 ": "1000", "-0": "0", "1__0": "", "_1": "",
		"1_": "", "0x10": "", "": "", "\xa012\x85": "12", "+-1": ""} {
		n, ok := pyInt(latin1(in))
		got := ""
		if ok {
			got = n.String()
		}
		if got != want {
			t.Errorf("int(%q) = %q, want %q", in, got, want)
		}
	}
}
