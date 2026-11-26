package buildmodule

import (
	"os"
	"strings"
	"testing"
)

// A hand-edited profile whose password breaks the XML must not have the password quoted back by the
// parser's error (independent verify, #1121).
func TestNoPrivateKeyMalformedProfileNeverEchoesThePassword(t *testing.T) {
	pw := "Zq9secretPW"
	for name, mid := range map[string]string{
		"bad close tag": "pa</" + pw + "</entry>",
		"bare lt":       pw + "<",
		"valid":         pw, // parses: the password goes to keytool, still never printed
	} {
		f := noKeyBuild(t, `<properties><entry key="niagara.signing.storepath">/k</entry>`+
			`<entry key="niagara.signing.storepass">`+mid+`</entry></properties>`, "")
		if strings.Contains(f.out+f.err, pw) {
			t.Errorf("%s: password in output:\n%s", name, f.err)
		}
		if name != "valid" && !strings.Contains(f.err, "the profile cannot be read (not valid XML)") {
			t.Errorf("%s: stderr %q", name, f.err)
		}
	}
}

// A password with shell and regex metacharacters reaches keytool only through its environment, is never
// printed or on argv, and the process environment is left as it was.
func TestNoPrivateKeyMetacharPasswordReachesKeytoolOnlyByEnv(t *testing.T) {
	pw := `a$(x)'";|\.*[x]+?^`
	esc := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(pw)
	stub := "#!/usr/bin/env bash\necho \"keytool $*\" >> \"LOGDIR/keytool.log\"\n" +
		"printf '%s' \"$BUILD_MODULE_STOREPASS\" > \"LOGDIR/seen.pw\"\n" +
		"echo \"bad: $BUILD_MODULE_STOREPASS\" >&2\nprintf 'acme, Nov 25, 2026, PrivateKeyEntry, \\n'\n"
	t.Setenv("BUILD_MODULE_STOREPASS", "preexisting")
	f := noKeyBuild(t, profileXML(`<entry key="niagara.signing.storepath">/k</entry>`+
		`<entry key="niagara.signing.storepass">`+esc+`</entry>`), stub)
	if f.read(f.dir+"/seen.pw") != pw {
		t.Error("keytool did not get the password through the environment")
	}
	if strings.Contains(f.out+f.err+f.read(f.dir+"/keytool.log"), pw) {
		t.Error("password printed or on argv")
	}
	if os.Getenv("BUILD_MODULE_STOREPASS") != "preexisting" {
		t.Error("process environment changed")
	}
}
