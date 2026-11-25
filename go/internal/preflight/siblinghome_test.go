package preflight

import "testing"

func TestCleanHome(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"/opt/n4/", "/opt/n4"}, {"/opt/./n4", "/opt/n4"}, {`C:\Niagara\n4`, "/mnt/c/Niagara/n4"},
		{"d:/N4", "/mnt/d/N4"}, {"/opt/c:x", "/opt/c:x"},
	} {
		if got := cleanHome(c.in); got != c.want {
			t.Errorf("cleanHome(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
