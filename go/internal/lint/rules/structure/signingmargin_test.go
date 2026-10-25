package structure

import (
	"path/filepath"
	"strings"
	"testing"
)

// marginProfile is a java.util.Properties XML signing profile; an empty profileType leaves the entry out.
func marginProfile(validity, refresh, profileType string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<!DOCTYPE properties SYSTEM "http://java.sun.com/dtd/properties.dtd">` + "\n<properties>\n")
	b.WriteString(`<entry key="niagara.signing.validity">` + validity + "</entry>\n")
	b.WriteString(`<entry key="niagara.signing.refresh">` + refresh + "</entry>\n")
	if profileType != "" {
		b.WriteString(`<entry key="niagara.signing.profileType">com.tridium.gradle.plugins.signing.profile.` + profileType + "</entry>\n")
	}
	b.WriteString("</properties>\n")
	return b.String()
}

// marginTree is <dir>/.git, the gradle root <dir>/Group (settings.gradle.kts, and gradle.properties when props is
// set) and the module root <dir>/Group/Mod with one clean -rt profile. It returns the module root.
func marginTree(t *testing.T, dir, props string, extra map[string]string) string {
	t.Helper()
	files := map[string]string{
		".git/HEAD":                           "x\n",
		"Group/settings.gradle.kts":           "val gradlePluginVersion: String = providers.gradleProperty(\"niagaraPluginVersion\").getOrElse(\"7.6.17\")\n",
		"Group/Mod/Foo-rt/build.gradle.kts":   "plugins { id(\"com.tridium.niagara-module\") }\n",
		"Group/Mod/Foo-rt/module-include.xml": "<types><type name=\"Foo\" class=\"com.x.Foo\"/></types>\n",
		"Group/Mod/Foo-rt/module.lexicon":     "Foo.displayName=Foo\n",
		"Group/Mod/Foo-rt/module.palette":     "<p n=\"Foo\" t=\"x:Foo\"/>\n",
	}
	if props != "" {
		files["Group/gradle.properties"] = props
	}
	for k, v := range extra {
		files[k] = v
	}
	write(t, dir, files)
	return filepath.Join(dir, "Group", "Mod")
}

// TestSigningMargin is L16: the gradle root's gradle.properties names, through niagaraSigningProfile, a
// LocalSigningProfile whose validity - refresh is below 32 days (retro r10-signing-margin-lint Δ1).
func TestSigningMargin(t *testing.T) {
	t.Parallel()
	const props = "niagaraSigningProfile=signing/niagara.signing.xml\n"
	const at = "Group/signing/niagara.signing.xml"
	cases := []struct {
		name     string
		props    string
		profile  string // "" = no profile file
		wantCode int
		want     []string // fragments of stdout ("" never)
		wantErr  string   // fragment of stderr
		noL16    bool
	}{
		{name: "30/15 FAILs naming both values and the reason", props: props, profile: marginProfile("30", "15", "LocalSigningProfile"),
			wantCode: 1, want: []string{"FAIL  lint-structure", "gradle.properties  L16: ", "niagara.signing.validity=30, niagara.signing.refresh=15",
				"re-minted at every signing", "validity - refresh >= 32"}},
		{name: "365/183 passes", props: props, profile: marginProfile("365", "183", "LocalSigningProfile"), wantCode: 0, noL16: true},
		{name: "214/183 (margin 31) FAILs", props: props, profile: marginProfile("214", "183", "LocalSigningProfile"),
			wantCode: 1, want: []string{"L16: ", "niagara.signing.validity=214, niagara.signing.refresh=183"}},
		{name: "215/183 (margin 32) passes", props: props, profile: marginProfile("215", "183", "LocalSigningProfile"), wantCode: 0, noL16: true},
		{name: "no profileType entry is the LocalSigningProfile default", props: props, profile: marginProfile("30", "15", ""),
			wantCode: 1, want: []string{"L16: "}},
		{name: "a profile type that mints no key is not checked", props: props, profile: marginProfile("30", "15", "RestrictedSigningProfile"),
			wantCode: 0, noL16: true},
		{name: "no profile declared: no row", props: "niagaraSigningAlias=a\n# niagaraSigningProfile=signing/niagara.signing.xml\n",
			profile: marginProfile("30", "15", "LocalSigningProfile"), wantCode: 0, noL16: true},
		{name: "no gradle.properties: no row", profile: marginProfile("30", "15", "LocalSigningProfile"), wantCode: 0, noL16: true},
		{name: "a malformed value is the read error, exit 3", props: props, profile: marginProfile("a year", "183", "LocalSigningProfile"),
			wantCode: 3, wantErr: "niagara.signing.validity=\"a year\" is not a whole number of days", noL16: true},
		{name: "a value beyond the plugin's int is the read error", props: props, profile: marginProfile("-9223372036854775808", "9223372036854775807", "LocalSigningProfile"),
			wantCode: 3, wantErr: "is not a whole number of days", noL16: true},
		{name: "not a properties XML file is the read error", props: props, profile: "<properties><entry key=\"a\">", wantCode: 3,
			wantErr: "not a properties XML file", noL16: true},
		{name: "a declared profile that does not exist WARNs, margin unchecked", props: props, wantCode: 0,
			want: []string{"WARN  lint-structure", "L16: ", "does not exist"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			extra := map[string]string{}
			if tc.profile != "" {
				extra[at] = tc.profile
			}
			code, out, errOut := run(marginTree(t, t.TempDir(), tc.props, extra))
			if code != tc.wantCode {
				t.Errorf("exit %d, want %d\nstdout:\n%s\nstderr:\n%s", code, tc.wantCode, out, errOut)
			}
			for _, w := range tc.want {
				if !strings.Contains(out, w) {
					t.Errorf("stdout lacks %q:\n%s", w, out)
				}
			}
			if tc.noL16 && strings.Contains(out, "L16") {
				t.Errorf("unexpected L16 row:\n%s", out)
			}
			if tc.wantErr != "" && !strings.Contains(errOut, tc.wantErr) {
				t.Errorf("stderr lacks %q:\n%s", tc.wantErr, errOut)
			}
		})
	}
}
