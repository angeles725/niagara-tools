package scaffoldmodule

import (
	"os"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/signingprofile"
)

// TestTheSigningProfileTemplateTheScaffoldNamesIsUsable (retro build-kit-bench-session Δ6): the template the signing
// line points at exists, names the fully qualified profile class (the short name fails `:jar` with "Could not
// determine class for profile"), holds the store keys and the key-usage command, and is a profile the kit's own
// margin check accepts.
func TestTheSigningProfileTemplateTheScaffoldNamesIsUsable(t *testing.T) {
	const path = "build-n4-module-kit/templates/signing-profile.xml"
	if !strings.Contains(emittedLines, path) {
		t.Fatalf("the signing line does not name %s", path)
	}
	b, err := os.ReadFile("../../../" + path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<entry key="niagara.signing.profileType">com.tridium.gradle.plugins.signing.profile.LocalSigningProfile</entry>`,
		`niagara.signing.storetype`, `niagara.signing.storepath`, `niagara.signing.storepass`, `niagara.signing.keypass.`,
		`-ext EKU=codeSigning -ext KU=digitalSignature`,
	} {
		if !strings.Contains(string(b), want) {
			t.Errorf("the template lacks %q", want)
		}
	}
	p, err := signingprofile.Parse(b)
	if err != nil || !p.Safe() {
		t.Errorf("Parse = %+v, %v; want a profile with a safe validity margin", p, err)
	}
}
