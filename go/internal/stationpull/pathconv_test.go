package stationpull

import "testing"

func TestNativePath(t *testing.T) {
	cases := []struct{ path, exe, want string }{
		{"/mnt/c/work/lab/station", "/n/bin/plat.exe", `C:\work\lab\station`},
		{"/mnt/d/x", "/n/bin/PLAT.EXE", `D:\x`},
		{"/mnt/c", "/n/bin/plat.exe", `C:\`},
		{"/tmp/lab/station", "/n/bin/plat", "/tmp/lab/station"},
		{"/mnt/c/work/station", "/n/bin/plat", "/mnt/c/work/station"},
	}
	for _, c := range cases {
		got, err := NativePath(c.path, c.exe)
		if err != nil || got != c.want {
			t.Errorf("NativePath(%q, %q) = %q, %v; want %q", c.path, c.exe, got, err, c.want)
		}
	}
}

func TestNativePathRefusesAPathPlatExeCannotReach(t *testing.T) {
	if got, err := NativePath("/home/x/lab/station", "/n/bin/plat.exe"); err == nil {
		t.Fatalf("got %q, want an error: plat.exe cannot write outside a /mnt/<drive> path", got)
	}
}
