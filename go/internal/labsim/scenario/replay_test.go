package scenario

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/labsim/regmap"
)

const replayCSV = "Timestamp,Value,Status\n" +
	"2026-11-15T20:02:00.000-06:00,30,{ok}\n" +
	"2026-11-15T20:01:30.000-06:00,99,{fault}\n" +
	"2026-11-15T20:01:00.000-06:00,20,{ok}\n" +
	"2026-11-15T20:00:00.000-06:00,10,{ok}\n"

// replayScenario writes a scenario with one replayed point and its CSV beside
// it, and returns the scenario's path.
func replayScenario(t *testing.T, behaviour string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "supply.csv"), []byte(replayCSV), 0o600); err != nil {
		t.Fatal(err)
	}
	text := `{"devices": [{"name": "d", "unit": 1, "points": [
	  {"name": "supply", "table": "input_register", "address": 0, "type": "int16", "behaviour": ` + behaviour + `}]}]}`
	path := filepath.Join(dir, "s.json")
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAReplayReadsItsFileBesideTheScenarioAndBuilds(t *testing.T) {
	t.Parallel()
	f, faults, err := Load(replayScenario(t, `{"kind": "replay", "file": "supply.csv", "speed": 60, "end": "loop"}`))
	if err != nil || len(faults) != 0 {
		t.Fatalf("load: %v, faults %q", err, faults)
	}
	p := &f.Devices[0].Points[0]
	if p.Initial() != 10 {
		t.Errorf("initial = %v, want the first ok record 10", p.Initial())
	}
	if got := p.Behaviour.Series().Skipped; got != 1 {
		t.Errorf("skipped = %d, want the fault record", got)
	}
	u := f.Build().Units[1]
	// Speed 60: the record of 20:01 shows after one second; the loop is 3 recorded minutes.
	for at, want := range map[time.Duration]uint16{0: 10, time.Second: 20, 2 * time.Second: 30, 3 * time.Second: 10} {
		if got := u.ReadWords(regmap.InputRegisters, 0, 1, at); got[0] != want {
			t.Errorf("supply at %v = %d, want %d", at, got[0], want)
		}
	}
}

func TestAReplayWithoutOptionsHoldsTheLastRecordAtSpeedOne(t *testing.T) {
	t.Parallel()
	f, faults, err := Load(replayScenario(t, `{"kind": "replay", "file": "supply.csv"}`))
	if err != nil || len(faults) != 0 {
		t.Fatalf("load: %v, faults %q", err, faults)
	}
	u := f.Build().Units[1]
	for at, want := range map[time.Duration]uint16{59 * time.Second: 10, time.Minute: 20, time.Hour: 30} {
		if got := u.ReadWords(regmap.InputRegisters, 0, 1, at); got[0] != want {
			t.Errorf("supply at %v = %d, want %d", at, got[0], want)
		}
	}
}

func TestAReplayNamesEachFaultOnce(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	big := filepath.Join(dir, "big.csv")
	if err := os.WriteFile(big, []byte("time,value\n2026-11-15T20:00:00Z,40000\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]struct{ behaviour, want string }{
		"no file":           {`{"kind": "replay"}`, "replay needs file"},
		"a missing file":    {`{"kind": "replay", "file": "absent.csv"}`, `replay file "absent.csv": open `},
		"a zero speed":      {`{"kind": "replay", "file": "supply.csv", "speed": 0}`, "replay needs speed above 0"},
		"a negative speed":  {`{"kind": "replay", "file": "supply.csv", "speed": -2}`, "replay needs speed above 0"},
		"an unknown end":    {`{"kind": "replay", "file": "supply.csv", "end": "stop"}`, `replay end "stop" is neither "hold_last" nor "loop"`},
		"a negative offset": {`{"kind": "replay", "file": "supply.csv", "start_offset_ms": -1}`, "replay needs start_offset_ms of 0 or more"},
		"an offset past the series": {`{"kind": "replay", "file": "supply.csv", "start_offset_ms": 120001}`,
			"replay start_offset_ms 120001 is past the end of the series (120000 ms)"},
		"a value the type cannot hold": {`{"kind": "replay", "file": "` + filepath.ToSlash(big) + `"}`,
			"behaviour replay runs over 40000 to 40000, which does not fit int16"},
	} {
		_, faults, err := Load(replayScenario(t, c.behaviour))
		if err != nil || len(faults) != 1 || !strings.Contains(faults[0], c.want) || !strings.HasPrefix(faults[0], `device 1 "d", point 1 "supply": `) {
			t.Errorf("%s: %v, faults = %q, want one holding %q", name, err, faults, c.want)
		}
	}
}

func TestABadReplayFileIsOneFaultNamingRowAndColumn(t *testing.T) {
	t.Parallel()
	path := replayScenario(t, `{"kind": "replay", "file": "supply.csv"}`)
	bad := "time,value\n2026-11-15T20:00:00Z,1\n2026-11-15T20:01:00Z,abc\n"
	if err := os.WriteFile(filepath.Join(filepath.Dir(path), "supply.csv"), []byte(bad), 0o600); err != nil {
		t.Fatal(err)
	}
	_, faults, _ := Load(path)
	want := `replay file "supply.csv": row 3, column 2 (value): "abc" is not a number`
	if len(faults) != 1 || !strings.Contains(faults[0], want) {
		t.Errorf("faults = %q, want one holding %q", faults, want)
	}
}

func TestAReplayIsRefusedOnAWritablePointAndOnABit(t *testing.T) {
	t.Parallel()
	f := parse(t)
	f.Devices[0].Points[2].Behaviour = &Behaviour{Kind: KindReplay, File: "x.csv"}
	f.Devices[0].Points[1].Behaviour = &Behaviour{Kind: KindReplay, File: "x.csv"}
	faults := Validate(f)
	if len(faults) != 2 || !strings.Contains(faults[0], "a bit cannot have behaviour \"replay\"") ||
		!strings.Contains(faults[1], "writable point cannot have behaviour \"replay\"") {
		t.Errorf("faults = %q", faults)
	}
}
