package niagaratest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// signingProfileXML is a LocalSigningProfile file as the signing plugin reads it (java.util.Properties XML).
func signingProfileXML(validity, refresh string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<!DOCTYPE properties SYSTEM "http://java.sun.com/dtd/properties.dtd">` + "\n<properties>\n")
	b.WriteString(`<entry key="niagara.signing.storetype">JCEKS</entry>` + "\n")
	if validity != "" {
		b.WriteString(`<entry key="niagara.signing.validity">` + validity + "</entry>\n")
	}
	if refresh != "" {
		b.WriteString(`<entry key="niagara.signing.refresh">` + refresh + "</entry>\n")
	}
	b.WriteString(`<entry key="niagara.signing.profileType">com.tridium.gradle.plugins.signing.profile.LocalSigningProfile</entry>` + "\n")
	b.WriteString("</properties>\n")
	return b.String()
}

func TestSigningProfileRefusal(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		props   string // the gradle root's gradle.properties ("" = none)
		uhProps string // GRADLE_USER_HOME/gradle.properties ("" = none)
		profile string // signing/niagara.signing.xml ("" = none)
		want    string // "" = accepted, else a refusal fragment
	}{
		{"the failing harness profile 30/15", "niagaraSigningProfile=signing/niagara.signing.xml\n", "", signingProfileXML("30", "15"),
			"niagara.signing.validity=30, niagara.signing.refresh=15"},
		{"plugin defaults 365/183", "niagaraSigningProfile=signing/niagara.signing.xml\n", "", signingProfileXML("365", "183"), ""},
		{"31 days apart is refused (day 31 of a month)", "niagaraSigningProfile=signing/niagara.signing.xml\n", "", signingProfileXML("214", "183"), "refresh=183"},
		{"32 days apart is accepted", "niagaraSigningProfile=signing/niagara.signing.xml\n", "", signingProfileXML("215", "183"), ""},
		{"no entries: the plugin defaults apply", "niagaraSigningProfile=signing/niagara.signing.xml\n", "", signingProfileXML("", ""), ""},
		{"refresh at or above validity", "niagaraSigningProfile = signing/niagara.signing.xml\n", "", signingProfileXML("183", "183"), "validity=183"},
		{"no profile property: the plugin's default profile", "# niagaraSigningProfile=x\nniagaraSigningAlias=a\n", "", "", ""},
		{"no gradle.properties", "", "", "", ""},
		{"profile file missing", "niagaraSigningProfile=signing/niagara.signing.xml\n", "", "", "cannot read"},
		{"non-integer validity", "niagaraSigningProfile=signing/niagara.signing.xml\n", "", signingProfileXML("a year", "183"), "not a whole number"},
		{"GRADLE_USER_HOME property wins over the root's", "niagaraSigningProfile=signing/ok.xml\n",
			"niagaraSigningProfile=" + "@ROOT@/signing/niagara.signing.xml\n", signingProfileXML("30", "15"), "validity=30"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root, uh := t.TempDir(), t.TempDir()
			if tc.props != "" {
				write(t, filepath.Join(root, "gradle.properties"), tc.props)
			}
			if tc.uhProps != "" {
				write(t, filepath.Join(uh, "gradle.properties"), strings.ReplaceAll(tc.uhProps, "@ROOT@", root))
				write(t, filepath.Join(root, "signing", "ok.xml"), signingProfileXML("365", "183"))
			}
			if tc.profile != "" {
				write(t, filepath.Join(root, "signing", "niagara.signing.xml"), tc.profile)
			}
			got := signingProfileRefusal(root, uh)
			switch {
			case tc.want == "" && got != "":
				t.Errorf("refused an acceptable profile: %s", got)
			case tc.want != "" && !strings.Contains(got, tc.want):
				t.Errorf("refusal %q lacks %q", got, tc.want)
			}
		})
	}
}

func TestSigningProfileRefusalNonLocalType(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	write(t, filepath.Join(root, "gradle.properties"), "niagaraSigningProfile=p.xml\n")
	xml := strings.Replace(signingProfileXML("30", "15"), "LocalSigningProfile", "RestrictedSigningProfile", 1)
	if err := os.WriteFile(filepath.Join(root, "p.xml"), []byte(xml), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := signingProfileRefusal(root, t.TempDir()); got != "" {
		t.Errorf("a profile type that never mints a key was refused: %s", got)
	}
}
