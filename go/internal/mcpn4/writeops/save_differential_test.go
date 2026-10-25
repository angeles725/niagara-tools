package writeops_test

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/golden"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// pySave runs each case through the real _save_plan (with the write state _process adds to its
// data) and the plan as _process hashes it, applies the case's save to config.bog, then runs
// _save_readback on a fake clock whose sleeps apply the case's later changes. It prints the
// plan, its plan_hash and targets, the read-back, the sleeps and the clock, or the error (class
// and message) where Python raises.
const pySave = `import hashlib, json, os, shutil, sys, types
from mcp_n4 import safety, tools_write

def describe(e):
    return {"type": type(e).__name__, "message": str(e)}

def apply(effects, path):
    for e in effects:
        if e.get("remove") or "write" in e:
            if os.path.isdir(path):
                shutil.rmtree(path)
            elif os.path.lexists(path):
                os.remove(path)
        if "write" in e:
            with open(path, "wb") as fh:
                fh.write(e["write"].encode())
        if "mtime_ns" in e and os.path.lexists(path):
            os.utime(path, ns=(e["mtime_ns"], e["mtime_ns"]))

for line in sys.stdin:
    c = json.loads(line)
    clock = {"now": c["start"], "sleeps": 0}

    def sleep(s, c=c, clock=clock):
        clock["sleeps"] += 1
        clock["now"] += s
        apply(c["polls"].get(str(clock["sleeps"]), []), c["bog"])
    write = types.SimpleNamespace(station_homes=c["homes"], save_timeout=c["timeout"],
                                  save_interval=c["interval"], sleep=sleep,
                                  monotonic=lambda clock=clock: clock["now"])
    sess = types.SimpleNamespace(station_name=c["station"], root_handle=c["root"])
    ctx = types.SimpleNamespace(write=write, session=sess)
    try:
        p = tools_write._IMPLS["n4_save_station"].plan(None, {}, ctx)
        p.data["write"] = write
        inverse = tools_write._inverse_list(p.inverse)
        plan = {"tool": "n4_save_station", "ops": p.ops,
                "inverse": tools_write._hashed_inverse(inverse), "notes": p.notes,
                "station": {"name": c["station"], "base_url": "http://fake.invalid",
                            "session_id": "0" * 32}}
        out = {"plan": plan, "plan_hash": hashlib.sha256(safety.canonical(plan).encode()).hexdigest(),
               "targets": tools_write._data(p).get("targets", {})}
    except Exception as e:
        print(json.dumps({"error": describe(e)}))
        continue
    apply(c["save"], c["bog"])
    try:
        out["readback"] = list(tools_write._IMPLS["n4_save_station"].readback(
            None, {}, p, [c["reply"]], inverse))
    except Exception as e:
        out["readback_error"] = describe(e)
    out["sleeps"], out["now"] = clock["sleeps"], clock["now"]
    print(json.dumps(out))
`

// apply is pySave's apply on the Go side.
func apply(t *testing.T, effects []any, path string) {
	for _, e := range effects {
		eff := e.(*pyjson.Object)
		_, write := eff.Get("write")
		if get(eff, "remove") == true || write {
			if err := os.RemoveAll(path); err != nil {
				t.Fatal(err)
			}
		}
		if write {
			if err := os.WriteFile(path, []byte(get(eff, "write").(string)), 0o666); err != nil {
				t.Fatal(err)
			}
		}
		if ns, ok := eff.Get("mtime_ns"); ok {
			if _, err := os.Lstat(path); err == nil {
				mt := time.Unix(0, ns.(int64))
				if err := os.Chtimes(path, mt, mt); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}

// goSave is pySave for one case on the Go side.
func goSave(t *testing.T, c *saveCase) *pyjson.Object {
	now, sleeps := c.start, 0
	homes := map[string]string{}
	for _, k := range c.homes.Keys() {
		homes[k] = get(c.homes, k).(string)
	}
	w := &plan.State{StationHomes: homes, SaveTimeout: c.timeout, SaveInterval: c.interval,
		Monotonic: func() float64 { return now }}
	w.Sleep = func(s float64) {
		sleeps++
		now += s
		if effects, ok := c.polls.Get(strconv.Itoa(sleeps)); ok {
			apply(t, effects.([]any), c.bogPath)
		}
	}
	sess := &session.Session{StationName: c.station, BaseURL: "http://fake.invalid", SessionID: strings.Repeat("0", 32),
		RootHandle: c.root}
	p, err := writeops.SaveStation.PlanWith(nil, obj(), w, sess)
	if err != nil {
		return obj("error", describe(err))
	}
	hashed, hash, err := plan.Hashed("n4_save_station", p, sess)
	if err != nil {
		return obj("error", describe(err))
	}
	out := obj("plan", hashed, "plan_hash", hash, "targets", pyjson.NewObject())
	apply(t, c.save, c.bogPath)
	requested, accepted, observed, verdict, err := writeops.SaveStation.Readback(nil, obj(), p, []any{c.reply}, p.Inverse)
	if err != nil {
		out.Set("readback_error", describe(err))
	} else {
		out.Set("readback", []any{requested, accepted, observed, verdict})
	}
	out.Set("sleeps", sleeps)
	out.Set("now", now)
	return out
}

// saveCase is one corpus case: where the station home points, the config.bog on disk before
// the plan, what the save does to it and what later polls see.
type saveCase struct {
	home, fixture       string // home: none, empty, dir, slash, slashes, file, missing; fixture: a bog kind
	mtime               int64
	save                []any
	polls               *pyjson.Object
	timeout, interval   float64
	station             string
	root, reply         any
	start               float64
	dir, bogPath, label string
	homes               *pyjson.Object
}

const baseNS = int64(1700000000123456789)

// saveCorpus is every home kind against every config.bog kind and save effect, then odd mtimes,
// timings, later changes, station names, root handles and replies.
func saveCorpus() []saveCase {
	saves := map[string][]any{
		"none":         {},
		"rewrite":      {obj("write", "bog-saved", "mtime_ns", baseNS+5_000_000_000)},
		"same-bytes":   {obj("write", "bog", "mtime_ns", baseNS)},
		"touch-1ns":    {obj("mtime_ns", baseNS+1)},
		"touch-1us":    {obj("mtime_ns", baseNS+1000)},
		"older-bytes":  {obj("write", "older", "mtime_ns", baseNS-5_000_000_000)},
		"older-touch":  {obj("mtime_ns", baseNS-1)},
		"older-1us":    {obj("mtime_ns", baseNS-1000)},
		"same-size":    {obj("write", "gob", "mtime_ns", baseNS)},
		"remove":       {obj("remove", true)},
		"empty":        {obj("write", "", "mtime_ns", baseNS)},
		"remove-write": {obj("remove", true), obj("write", "bog", "mtime_ns", baseNS)},
	}
	order := []string{"none", "rewrite", "same-bytes", "touch-1ns", "touch-1us", "older-bytes", "older-touch", "older-1us", "same-size", "remove", "empty", "remove-write"}
	none := pyjson.NewObject()
	var cases []saveCase
	base := saveCase{fixture: "bog", mtime: baseNS, polls: none, timeout: 1.0, interval: 0.25, station: "FakeStation",
		root: "2", reply: nil}
	for _, home := range []string{"none", "empty", "dir", "slash", "slashes", "file", "missing"} {
		for _, fixture := range []string{"bog", "empty", "absent", "dir", "noperm"} {
			for _, s := range order {
				c := base
				c.home, c.fixture, c.save, c.label = home, fixture, saves[s], home+"/"+fixture+"/"+s
				cases = append(cases, c)
			}
		}
	}
	for _, mt := range []int64{0, 1, 999_999_999, 1_000_000_000, -1_500_000_000, 1700000000999999999, 4102444800000000001, 1 << 62} {
		for _, s := range []string{"none", "touch-1ns", "touch-1us", "rewrite"} {
			c := base
			c.home, c.mtime, c.label = "dir", mt, fmt.Sprintf("mtime %d/%s", mt, s)
			c.save = saves[s]
			if s == "touch-1ns" || s == "touch-1us" {
				d := int64(1)
				if s == "touch-1us" {
					d = 1000
				}
				c.save = []any{obj("mtime_ns", mt+d)}
			}
			cases = append(cases, c)
		}
	}
	later := []*pyjson.Object{none, obj("2", []any{obj("write", "late", "mtime_ns", baseNS+2000)}), obj("3", []any{obj("remove", true)}),
		obj("1", []any{obj("mtime_ns", baseNS+1000)}), obj("4", []any{obj("write", "bog", "mtime_ns", baseNS)}),
		obj("2", []any{obj("remove", true)}, "3", []any{obj("write", "back", "mtime_ns", baseNS+3000)})}
	for _, timing := range [][3]float64{{1.0, 0.25, 0}, {0.3, 0.1, 0}, {0, 0.5, 0}, {0.05, 1.0, 0}, {1.0, 0.3, 7.7}, {-1, 0.5, 0},
		{2.5, 0.5, 1e15}} {
		for i, p := range later {
			c := base
			c.home, c.timeout, c.interval, c.start, c.polls = "dir", timing[0], timing[1], timing[2], p
			c.label = fmt.Sprintf("timing %v/later %d", timing, i)
			cases = append(cases, c)
		}
	}
	for _, v := range []struct {
		station     string
		root, reply any
	}{{"Station A", 5, "x"}, {"é", nil, obj("v", true)}, {"a=b", 2.5, []any{}}, {"", "h:1", false}} {
		for _, home := range []string{"dir", "none"} {
			c := base
			c.home, c.station, c.root, c.reply, c.save = home, v.station, v.root, v.reply, saves["rewrite"]
			c.label = "station " + v.station + "/" + home
			cases = append(cases, c)
		}
	}
	return cases
}

// layout lays out case i under root: its home dir and config.bog, and the line pySave reads.
func (c *saveCase) layout(t *testing.T, root string, i int) string {
	c.dir = filepath.Join(root, fmt.Sprintf("case%03d", i))
	c.bogPath = filepath.Join(c.dir, "config.bog")
	if err := os.RemoveAll(c.dir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		t.Fatal(err)
	}
	var err error
	switch c.fixture {
	case "bog", "noperm":
		err = os.WriteFile(c.bogPath, []byte("bog"), 0o644)
		if c.fixture == "noperm" && err == nil {
			err = os.Chmod(c.bogPath, 0)
		}
	case "empty":
		err = os.WriteFile(c.bogPath, nil, 0o644)
	case "dir":
		err = os.Mkdir(c.bogPath, 0o755)
	}
	if err == nil && c.fixture != "absent" {
		mt := time.Unix(0, c.mtime)
		err = os.Chtimes(c.bogPath, mt, mt)
	}
	if err != nil {
		t.Fatal(err)
	}
	homes := pyjson.NewObject()
	switch c.home {
	case "empty":
		homes.Set(c.station, "")
	case "dir":
		homes.Set(c.station, c.dir)
	case "slash":
		homes.Set(c.station, c.dir+"/")
	case "slashes":
		homes.Set(c.station, c.dir+"//")
	case "file":
		homes.Set(c.station, c.bogPath)
	case "missing":
		homes.Set(c.station, filepath.Join(c.dir, "nowhere"))
	}
	homes.Set("Other", c.dir) // another station's home is never used
	c.homes = homes
	line, err := pyjson.Marshal(obj("homes", homes, "station", c.station, "root", c.root, "reply", c.reply,
		"bog", c.bogPath, "save", c.save, "polls", c.polls, "timeout", c.timeout, "interval", c.interval,
		"start", c.start), pyjson.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return string(line)
}

func TestSavePlanAndReadbackDifferentialPython(t *testing.T) {
	g := golden.Open(t, "save")
	// The cases run in a temp dir under relative paths: the plan_hash covers the home path
	// in the notes, so only a path that is the same on every run keeps the golden replayable.
	t.Chdir(t.TempDir())
	root := "."
	cases := saveCorpus()
	lines := make([]string, len(cases))
	for i := range cases {
		lines[i] = cases[i].layout(t, root, i)
	}
	out := g.Run(pySave, []byte(strings.Join(lines, "\n")+"\n"))
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	sc.Buffer(nil, 1<<22)
	kinds := map[string]int{}
	n := 0
	for ; sc.Scan(); n++ {
		c := &cases[n]
		c.layout(t, root, n) // the same files again: the Python run changed them
		got, err := pyjson.Marshal(goSave(t, c), pyjson.Options{})
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != sc.Text() {
			t.Errorf("case %d %s:\n go     %s\n python %s", n, c.label, got, sc.Text())
		}
		kinds[saveOutcome(sc.Text())]++
	}
	if n != len(cases) {
		t.Fatalf("python answered %d of %d cases", n, len(cases))
	}
	t.Logf("%d cases: %v", n, kinds)
	for _, k := range []string{"verified", "mismatch", "no home", "unreadable before", "unreadable after", "polled"} {
		if kinds[k] == 0 {
			t.Errorf("no case reaches %s", k)
		}
	}
}

// saveOutcome classifies a Python result line, so the corpus provably reaches every branch.
func saveOutcome(line string) string {
	switch {
	case strings.Contains(line, `"error"`):
		return "error"
	case strings.Contains(line, "to check persistence"):
		return "no home"
	case strings.Contains(line, "before the save; check"):
		return "unreadable before"
	case strings.Contains(line, "became unreadable after the save"):
		return "unreadable after"
	case strings.Contains(line, `"mismatch"]`):
		return "mismatch"
	case strings.Contains(line, `"verified"]`) && !strings.Contains(line, `"sleeps": 0,`):
		return "polled"
	case strings.Contains(line, `"verified"]`):
		return "verified"
	}
	return "other"
}
