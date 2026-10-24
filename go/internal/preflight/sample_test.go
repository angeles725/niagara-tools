package preflight

import (
	"strings"
	"testing"
)

func TestProp(t *testing.T) {
	d := t.TempDir()
	mk(t, d, "p=niagara_homes=/x\n#niagara_home=/y\n  niagara_home \t=  c:\\\\n4\\\\4.15  \r\nniagara_home=/z\n")
	if got := prop(d+"/p", "niagara_home"); got != `c:\n4\4.15` {
		t.Errorf("prop = %q, want the first niagara_home value with \\\\ collapsed", got)
	}
	if got := prop(d+"/p", "niagara_user_home"); got != "" {
		t.Errorf("missing key = %q", got)
	}
}

func TestSampleHomeAndUser(t *testing.T) {
	d := t.TempDir()
	nh := d + "/niagara-4.15.6.20"
	mk(t, d, "niagara-4.15.6.20/", "niagara-4.15.4.24/", "nover/", "gr/settings.gradle.kts")
	for _, c := range []struct{ props, home, user string }{
		{`niagara_home=D:\\N4\\niagara-4.15.6.20` + "\nniagara_user_home=/home/<u>/x\n",
			"FAIL  sample-home   gradle.properties niagara_home /mnt/d/N4/niagara-4.15.6.20 does not exist — retarget it to " + nh,
			"FAIL  sample-user   gradle.properties niagara_user_home has a placeholder: /home/<u>/x"},
		{"niagara_home=" + d + "/niagara-4.15.4.24\nniagara_user_home=/home/u>x<y\n",
			"FAIL  sample-home   gradle.properties niagara_home is 4.15.4.24, the target install is 4.15.6.20 — retarget it to " + nh,
			"PASS  sample-user   niagara_user_home has no <…> placeholder"},
		{"niagara_home=" + d + "/nover\n",
			"WARN  sample-home   gradle.properties niagara_home " + d + "/nover exists, but its version could not be compared (no X.Y.Z in " + d + "/nover) — confirm it is the target install " + nh,
			"WARN  sample-user   no niagara_user_home in " + d + "/gr/gradle.properties — set it to this user's Niagara user home"},
	} {
		mk(t, d, "gr/gradle.properties="+c.props)
		if _, out, _ := pf((&fakeHost{}).host(), nil, "--sample", "--jvm-dir", d, nh, d+"/gr"); !strings.Contains(out, c.home+"\n"+c.user+"\n") {
			t.Errorf("%q:\n%s\nwant\n%s\n%s", c.props, out, c.home, c.user)
		}
	}
}
