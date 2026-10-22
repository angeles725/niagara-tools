package hostbudget

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFile creates path (and its parents) with content.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// cgroup writes <root>/<name>/pids.current and pids.max ("" omits the file).
func cgroup(t *testing.T, root, name, cur, max string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
		t.Fatal(err)
	}
	if cur != "" {
		writeFile(t, filepath.Join(root, name, "pids.current"), cur+"\n")
	}
	if max != "" {
		writeFile(t, filepath.Join(root, name, "pids.max"), max+"\n")
	}
}

// proc writes a fake /proc with threads-max and a loadavg whose 4th field is running/total.
func proc(t *testing.T, root, threadsMax, total string) {
	t.Helper()
	writeFile(t, filepath.Join(root, "sys/kernel/threads-max"), threadsMax+"\n")
	writeFile(t, filepath.Join(root, "loadavg"), "0.50 0.40 0.30 2/"+total+" 12345\n")
}

// sparse creates a file of apparent size n bytes that uses (almost) no blocks.
func sparse(t *testing.T, path string, n int64) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := f.Truncate(n); err != nil {
		t.Fatal(err)
	}
}

const gb = int64(1_000_000_000)

// healthy is a config whose every resource passes; each case changes one thing.
func healthy(t *testing.T) (Config, Probe) {
	t.Helper()
	dir := t.TempDir()
	cg, pr, cache := filepath.Join(dir, "cg"), filepath.Join(dir, "proc"), filepath.Join(dir, "gocache")
	cgroup(t, cg, "init.scope", "700", "48129")
	proc(t, pr, "320861", "900")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	c := Config{CgroupRoot: cg, ProcRoot: pr, PidsPct: 50, DiskPct: 90, GocacheGB: 20, GocacheDir: cache,
		GocacheMaxEntries: 1000}
	p := Probe{Statfs: func(string) (Disk, error) { return Disk{Used: 10, Avail: 90}, nil }}
	return c, p
}

// rowFor returns the row whose resource is res.
func rowFor(rows []Row, res string) (Row, bool) {
	for _, r := range rows {
		if r.Resource == res {
			return r, true
		}
	}
	return Row{}, false
}

func TestCheckPids(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, cur, max string
		want           Status
		wantRow        bool
	}{
		// Mutation: HB-pids-fail -- comparing against the warn threshold only reports 62% as WARN.
		{"above half fails (the 2026-10-22 incident shape)", "30000", "48129", Fail, true},
		// Mutation: HB-pids-ge -- using >= instead of > fails the exact half.
		{"exact half is not a FAIL", "24064", "48128", Warn, true},
		{"one above half fails", "24065", "48128", Fail, true},
		{"exact 40% passes", "19251", "48128", Pass, true}, // 1925100 vs 1925120: below the line
		{"above 40% warns", "19252", "48128", Warn, true},
		// Mutation: HB-pids-max -- parsing "max" as a number reports the unlimited slices.
		{"max is unlimited and skipped", "99999", "max", Pass, false},
		{"garbled pids.max warns", "10", "lots", Warn, true},
		{"zero pids.max with tasks fails", "1", "0", Fail, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c, p := healthy(t)
			cgroup(t, c.CgroupRoot, "user.slice", tc.cur, tc.max)
			rows, err := Check(c, p)
			if err != nil {
				t.Fatal(err)
			}
			r, ok := rowFor(rows, "pids:user.slice")
			if ok != tc.wantRow {
				t.Fatalf("row present = %v, want %v; rows %v", ok, tc.wantRow, rows)
			}
			if ok && r.Status != tc.want {
				t.Errorf("status = %v, want %v (%s)", r.Status, tc.want, r)
			}
			if ok && tc.max != "lots" {
				if want := tc.cur + "/" + tc.max; r.Used+"/"+r.Limit != want {
					t.Errorf("row %q, want used/limit %s", r, want)
				}
			}
		})
	}
}

func TestCheckPidsRowFormat(t *testing.T) {
	t.Parallel()
	c, p := healthy(t)
	cgroup(t, c.CgroupRoot, "init.scope", "30000", "48129")
	rows, err := Check(c, p)
	if err != nil {
		t.Fatal(err)
	}
	r, _ := rowFor(rows, "pids:init.scope")
	if got := r.String(); !strings.HasPrefix(got, "FAIL pids:init.scope 30000/48129") {
		t.Errorf("row = %q, want the PASS|WARN|FAIL <resource> <used>/<limit> shape", got)
	}
}

func TestCheckCgroupWithoutPidsFilesIsSkipped(t *testing.T) {
	t.Parallel()
	c, p := healthy(t)
	cgroup(t, c.CgroupRoot, "dev-mqueue.mount", "", "")
	writeFile(t, filepath.Join(c.CgroupRoot, "cgroup.procs"), "1\n") // a root file, not a cgroup
	rows, err := Check(c, p)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if strings.Contains(r.Resource, "dev-mqueue") || strings.Contains(r.Resource, "cgroup.procs") {
			t.Errorf("unexpected row %q", r)
		}
	}
}

func TestCheckUnreadableCgroupRootIsAnError(t *testing.T) {
	t.Parallel()
	c, p := healthy(t)
	c.CgroupRoot = filepath.Join(c.CgroupRoot, "does-not-exist")
	if _, err := Check(c, p); err == nil {
		t.Fatal("Check on a missing cgroup root returned no error")
	}
}

func TestCheckThreads(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, max, total string
		want             Status
	}{
		{"healthy", "320861", "900", Pass},
		{"above half of threads-max fails", "1000", "501", Fail},
		{"above 40% warns", "1000", "401", Warn},
		{"garbled loadavg warns", "1000", "x", Warn},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c, p := healthy(t)
			proc(t, c.ProcRoot, tc.max, tc.total)
			rows, err := Check(c, p)
			if err != nil {
				t.Fatal(err)
			}
			r, ok := rowFor(rows, "threads")
			if !ok || r.Status != tc.want {
				t.Errorf("threads row %q (present %v), want %v", r, ok, tc.want)
			}
		})
	}
}

func TestCheckThreadsUnreadableProcWarns(t *testing.T) {
	t.Parallel()
	c, p := healthy(t)
	c.ProcRoot = filepath.Join(c.ProcRoot, "nope")
	rows, err := Check(c, p)
	if err != nil {
		t.Fatal(err)
	}
	if r, ok := rowFor(rows, "threads"); !ok || r.Status != Warn {
		t.Errorf("threads row %q (present %v), want WARN", r, ok)
	}
}

func TestCheckGocache(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		files map[string]int64
		want  Status
		hint  bool
	}{
		// Mutation: HB-gocache -- summing allocated blocks instead of apparent size misses a sparse 21 GB file.
		{"21 GB sparse cache warns with the clean hint", map[string]int64{"00/a-d": 21 * gb}, Warn, true},
		{"sizes add up across subdirectories", map[string]int64{"00/a-d": 11 * gb, "ff/b-d": 10 * gb}, Warn, true},
		{"19 GB passes", map[string]int64{"00/a-d": 19 * gb}, Pass, false},
		{"exactly the limit passes", map[string]int64{"00/a-d": 20 * gb}, Pass, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c, p := healthy(t)
			for name, n := range tc.files {
				sparse(t, filepath.Join(c.GocacheDir, name), n)
			}
			rows, err := Check(c, p)
			if err != nil {
				t.Fatal(err)
			}
			r, ok := rowFor(rows, "gocache")
			if !ok || r.Status != tc.want {
				t.Fatalf("gocache row %q (present %v), want %v", r, ok, tc.want)
			}
			if got := strings.Contains(r.String(), "go clean -cache"); got != tc.hint {
				t.Errorf("hint present = %v, want %v in %q", got, tc.hint, r)
			}
		})
	}
}

func TestCheckGocacheMissingDirIsEmpty(t *testing.T) {
	t.Parallel()
	c, p := healthy(t)
	c.GocacheDir = filepath.Join(c.GocacheDir, "never-built")
	rows, err := Check(c, p)
	if err != nil {
		t.Fatal(err)
	}
	if r, ok := rowFor(rows, "gocache"); !ok || r.Status != Pass {
		t.Errorf("gocache row %q (present %v), want PASS", r, ok)
	}
}

func TestCheckGocacheWalkIsBounded(t *testing.T) {
	t.Parallel()
	c, p := healthy(t)
	c.GocacheMaxEntries = 3
	for _, n := range []string{"a", "b", "c", "d", "e"} {
		writeFile(t, filepath.Join(c.GocacheDir, n), "x")
	}
	rows, err := Check(c, p)
	if err != nil {
		t.Fatal(err)
	}
	r, ok := rowFor(rows, "gocache")
	if !ok || r.Status != Warn || !strings.Contains(r.String(), "walk stopped") {
		t.Errorf("gocache row %q (present %v), want a WARN naming the stopped walk", r, ok)
	}
}

func TestCheckGocacheUnresolvedWarns(t *testing.T) {
	t.Parallel()
	c, p := healthy(t)
	c.GocacheDir = ""
	rows, err := Check(c, p)
	if err != nil {
		t.Fatal(err)
	}
	if r, ok := rowFor(rows, "gocache"); !ok || r.Status != Warn {
		t.Errorf("gocache row %q (present %v), want WARN", r, ok)
	}
}

func TestCheckDisk(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		used, avail uint64
		want        Status
	}{
		{"95% fails", 95, 5, Fail},
		// Mutation: HB-disk-ge -- using >= fails a mount at exactly 90%.
		{"exactly 90% warns", 90, 10, Warn},
		{"85% warns", 85, 15, Warn},
		{"exactly 80% passes", 80, 20, Pass},
		{"50% passes", 50, 50, Pass},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c, _ := healthy(t)
			c.Mounts = []string{"/mnt/c"}
			p := Probe{Statfs: func(path string) (Disk, error) {
				if path == "/mnt/c" {
					return Disk{Used: tc.used * uint64(gb), Avail: tc.avail * uint64(gb)}, nil
				}
				return Disk{Used: 1, Avail: 9}, nil
			}}
			rows, err := Check(c, p)
			if err != nil {
				t.Fatal(err)
			}
			r, ok := rowFor(rows, "disk:/mnt/c")
			if !ok || r.Status != tc.want {
				t.Errorf("disk row %q (present %v), want %v", r, ok, tc.want)
			}
			if root, ok := rowFor(rows, "disk:/"); !ok || root.Status != Pass {
				t.Errorf("/ must always be checked: row %q (present %v)", root, ok)
			}
		})
	}
}

func TestCheckDiskStatfsErrorWarnsAndDedupesRoot(t *testing.T) {
	t.Parallel()
	c, _ := healthy(t)
	c.Mounts = []string{"/", "/gone"}
	p := Probe{Statfs: func(path string) (Disk, error) {
		if path == "/gone" {
			return Disk{}, errors.New("no such file")
		}
		return Disk{Used: 1, Avail: 9}, nil
	}}
	rows, err := Check(c, p)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, r := range rows {
		if r.Resource == "disk:/" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("disk:/ rows = %d, want 1 (rows %v)", n, rows)
	}
	if r, ok := rowFor(rows, "disk:/gone"); !ok || r.Status != Warn {
		t.Errorf("disk:/gone row %q (present %v), want WARN", r, ok)
	}
}

func TestStatusString(t *testing.T) {
	t.Parallel()
	for s, want := range map[Status]string{Pass: "PASS", Warn: "WARN", Fail: "FAIL"} {
		if s.String() != want {
			t.Errorf("%d.String() = %q, want %q", s, s.String(), want)
		}
	}
}
