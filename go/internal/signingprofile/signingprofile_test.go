package signingprofile

import (
	"strings"
	"testing"
)

func xmlProfile(entries string) []byte {
	return []byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
		`<!DOCTYPE properties SYSTEM "http://java.sun.com/dtd/properties.dtd">` + "\n<properties>\n" + entries + "</properties>\n")
}

func entry(k, v string) string { return `<entry key="` + k + `">` + v + "</entry>\n" }

// TestRefusal pins the predicate at its boundary: 31 days apart is refused (a key minted on day 31), 32 is not.
func TestRefusal(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		validity, refresh string
		refused           bool
	}{{"30", "15", true}, {"365", "183", false}, {"214", "183", true}, {"215", "183", false}, {"183", "183", true}, {"", "", false}} {
		var e string
		if tc.validity != "" {
			e = entry("niagara.signing.validity", tc.validity) + entry("niagara.signing.refresh", tc.refresh)
		}
		p, err := Parse(xmlProfile(e))
		if err != nil {
			t.Fatalf("%s/%s: %v", tc.validity, tc.refresh, err)
		}
		why := p.Refusal()
		if (why != "") != tc.refused || p.Safe() == tc.refused {
			t.Errorf("%s/%s: refusal %q, Safe %v, want refused %v", tc.validity, tc.refresh, why, p.Safe(), tc.refused)
		}
		if tc.refused && !strings.Contains(why, "niagara.signing.validity="+tc.validity+", niagara.signing.refresh="+tc.refresh) {
			t.Errorf("refusal does not name both values: %s", why)
		}
	}
}

func TestParse(t *testing.T) {
	t.Parallel()
	if p, err := Parse(xmlProfile(entry("niagara.signing.profileType", "x.RestrictedSigningProfile") + entry("niagara.signing.validity", "bad"))); err != nil || p.Local || !p.Safe() {
		t.Errorf("another profile type: %+v, %v (want not local, days unread)", p, err)
	}
	for _, v := range []string{"a year", "2147483648", "-9223372036854775808", ""} {
		if _, err := Parse(xmlProfile(entry("niagara.signing.validity", v))); err == nil || !strings.Contains(err.Error(), "is not a whole number of days") {
			t.Errorf("validity %q: err %v", v, err)
		}
	}
	if _, err := Parse([]byte("<properties><entry key=\"a\">")); err == nil || !strings.HasPrefix(err.Error(), "not a properties XML file: ") {
		t.Errorf("truncated XML: err %v", err)
	}
}

func TestGradleProperty(t *testing.T) {
	t.Parallel()
	data := []byte("# niagaraSigningProfile=commented\n! also\n  niagaraSigningProfile : first\r\nniagaraSigningProfile second\nother=x\n" +
		strings.Repeat("y", 70000) + "\nniagaraSigningProfile=last\n")
	if v, ok := GradleProperty(data, Property); !ok || v != "last" {
		t.Errorf("got %q %v, want the last definition past a 70000-byte line", v, ok)
	}
	if _, ok := GradleProperty([]byte("niagaraSigningProfileX=1\n"), Property); ok {
		t.Error("a longer key matched")
	}
}
