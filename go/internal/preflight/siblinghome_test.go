package preflight

import (
	"os"
	"testing"
)

func TestCleanHome(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"/opt/n4/", "/opt/n4"}, {"/opt/./n4", "/opt/n4"}, {`C:\Niagara\n4`, "/mnt/c/Niagara/n4"},
		{"d:/N4", "/mnt/d/N4"}, {"/opt/c:x", "/opt/c:x"},
	} {
		if got := homeKey(c.in, ""); got != c.want {
			t.Errorf("homeKey(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestHomeKey(t *testing.T) {
	d := t.TempDir()
	mk(t, d, "real/etc/", "sib/")
	if err := os.Symlink(d+"/real", d+"/link"); err != nil {
		t.Fatal(err)
	}
	real := homeKey(d+"/real", "")
	for _, c := range []struct{ name, pin, base string }{
		{"symlink", d + "/link", ""}, {"trailing slash", d + "/real/", ""}, {"dot segments", d + "/sib/../real", ""},
		{"relative to the sibling root", "../real", d + "/sib"},
	} {
		if got := homeKey(c.pin, c.base); got != real {
			t.Errorf("%s: homeKey(%q, %q) = %q, want %q", c.name, c.pin, c.base, got, real)
		}
	}
	if got := homeKey(d+"/missing/", ""); got != d+"/missing" {
		t.Errorf("a path that does not exist falls back to its cleaned string, got %q", got)
	}
	if got := homeKey(`C:\x\n4`, ""); got != "/mnt/c/x/n4" {
		t.Errorf("drive pin = %q", got)
	}
}
