package scenario

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/labsim/regmap"
)

const valid = `{
  "name": "bench",
  "unknown_unit": "gateway_exception",
  "devices": [
    {"name": "unit-a", "unit": 1,
     "faults": {"fail_after_reads": 500, "delay_ms": 20, "offline": [{"from_ms": 1000, "to_ms": 2000}]},
     "points": [
      {"name": "run", "table": "coil", "address": 0, "value": 1, "writable": true},
      {"name": "alarm", "table": "discrete_input", "address": 0},
      {"name": "setpoint", "table": "holding_register", "address": 0, "type": "float32", "order": "CDAB", "value": 21.5, "writable": true},
      {"name": "speed", "table": "holding_register", "address": 2, "type": "uint16", "value": 1200},
      {"name": "supply", "table": "input_register", "address": 0, "type": "int16", "value": 180,
       "behaviour": {"kind": "sine", "amplitude": 20, "period_ms": 60000}},
      {"name": "level", "table": "input_register", "address": 1, "type": "uint16",
       "behaviour": {"kind": "ramp", "min": 0, "max": 100, "period_ms": 10000}},
      {"name": "flow", "table": "input_register", "address": 2, "type": "float32", "value": 5,
       "behaviour": {"kind": "random_walk", "seed": 3, "step": 0.5, "interval_ms": 1000, "min": 0, "max": 10},
       "faults": {"fail_after_reads": 0, "delay_ms": 5}}
     ]}
  ]
}`

func parse(t *testing.T) *File {
	t.Helper()
	f, faults := Parse([]byte(valid))
	if len(faults) != 0 {
		t.Fatalf("the base scenario has faults: %v", faults)
	}
	return f
}

func TestParseFillsTheDefaults(t *testing.T) {
	t.Parallel()
	f := parse(t)
	if f.Listen != DefaultListen || f.Control != DefaultControl || f.MaxConnections != DefaultMaxConnections || f.IdleTimeoutMS != DefaultIdleTimeoutMS {
		t.Errorf("defaults = %q %q %d %d", f.Listen, f.Control, f.MaxConnections, f.IdleTimeoutMS)
	}
	p := f.Devices[0].Points
	if p[0].Type != "bit" || p[3].Order != "ABCD" || p[1].Type != "bit" {
		t.Errorf("point defaults: %q %q %q", p[0].Type, p[3].Order, p[1].Type)
	}
	g, faults := Parse([]byte(`{"devices":[{"name":"d","unit":1,"points":[{"name":"p","table":"holding_register","address":0}]}]}`))
	if len(faults) != 0 || g.UnknownUnit != "silent" || g.Devices[0].Points[0].Type != "uint16" {
		t.Errorf("minimal scenario: %+v, faults %v", g, faults)
	}
}

func TestParseReportsWhatIsNotAScenario(t *testing.T) {
	t.Parallel()
	for name, text := range map[string]string{
		"not JSON":          `{"devices": [`,
		"an unknown field":  `{"devices": [], "lisen": "127.0.0.1:1"}`,
		"a second document": valid + `{}`,
		"a wrong type":      `{"devices": [{"name": "d", "unit": "one"}]}`,
	} {
		if _, faults := Parse([]byte(text)); len(faults) != 1 || !strings.HasPrefix(faults[0], "scenario is not valid: ") {
			t.Errorf("%s: faults = %v, want one 'scenario is not valid' message", name, faults)
		}
	}
}

func TestValidateNamesEachFaultOnce(t *testing.T) {
	t.Parallel()
	neg, high := -1, 500.0
	cases := []struct {
		name   string
		break_ func(*File)
		want   string
	}{
		{"no devices", func(f *File) { f.Devices = nil }, "no devices"},
		{"a listen address without a port", func(f *File) { f.Listen = "127.0.0.1" }, `listen "127.0.0.1"`},
		{"a control address off this machine", func(f *File) { f.Control = "0.0.0.0:1503" }, `control "0.0.0.0:1503"`},
		{"an unknown unknown_unit", func(f *File) { f.UnknownUnit = "ignore" }, `unknown_unit "ignore"`},
		{"too many connections", func(f *File) { f.MaxConnections = 5000 }, "max_connections 5000"},
		{"a negative idle timeout", func(f *File) { f.IdleTimeoutMS = -1 }, "idle_timeout_ms -1"},
		{"a device without a name", func(f *File) { f.Devices[0].Name = "" }, "device 1: no name"},
		{"unit 0", func(f *File) { f.Devices[0].Unit = 0 }, "unit 0 is outside 1-255"},
		{"unit 256", func(f *File) { f.Devices[0].Unit = 256 }, "unit 256 is outside 1-255"},
		{"a unit used twice", func(f *File) { d := f.Devices[0]; d.Name = "unit-b"; f.Devices = append(f.Devices, d) }, "unit 1 is already device"},
		{"a device name used twice", func(f *File) { d := f.Devices[0]; d.Unit = 2; f.Devices = append(f.Devices, d) }, `name "unit-a" is already used`},
		{"a device without points", func(f *File) { f.Devices[0].Points = nil }, "no points"},
		{"a negative device fail_after_reads", func(f *File) { f.Devices[0].Faults.FailAfterReads = &neg }, "fail_after_reads -1"},
		{"a negative device delay", func(f *File) { f.Devices[0].Faults.DelayMS = -1 }, "delay_ms -1"},
		{"an offline window that ends before it starts", func(f *File) { f.Devices[0].Faults.Offline[0].ToMS = 1000 }, "offline window 1000-1000"},
		{"a point without a name", func(f *File) { f.Devices[0].Points[0].Name = "" }, "point 1: no name"},
		{"a point name used twice", func(f *File) { f.Devices[0].Points[1].Name = "run" }, `name "run" is already used`},
		{"an unknown table", func(f *File) { f.Devices[0].Points[0].Table = "coils" }, `table "coils"`},
		{"a negative address", func(f *File) { f.Devices[0].Points[0].Address = -1 }, "address -1"},
		{"a two-register point at the last address", func(f *File) { f.Devices[0].Points[2].Address = 65535 }, "address 65535"},
		{"an unknown type", func(f *File) { f.Devices[0].Points[3].Type = "int64" }, `type "int64"`},
		{"a bit in a register table", func(f *File) { f.Devices[0].Points[3].Type = "bit" }, `type "bit" does not fit table "holding_register"`},
		{"a register type in a coil table", func(f *File) { f.Devices[0].Points[0].Type = "uint16" }, `type "uint16" does not fit table "coil"`},
		{"an unknown byte order", func(f *File) { f.Devices[0].Points[2].Order = "big" }, `order "big"`},
		{"a writable input register", func(f *File) { f.Devices[0].Points[4].Writable = true; f.Devices[0].Points[4].Behaviour = nil }, `table "input_register" is read-only`},
		{"a value the type cannot hold", func(f *File) { f.Devices[0].Points[3].Value = 70000 }, "value 70000 does not fit uint16"},
		{"a bit that is neither 0 nor 1", func(f *File) { f.Devices[0].Points[0].Value = 2 }, "value 2 does not fit bit"},
		{"two points on the same register", func(f *File) { f.Devices[0].Points[3].Address = 1 }, `overlaps "setpoint"`},
		{"an unknown behaviour", func(f *File) { f.Devices[0].Points[4].Behaviour.Kind = "square" }, `behaviour "square"`},
		{"a behaviour on a writable point", func(f *File) {
			f.Devices[0].Points[2].Behaviour = &Behaviour{Kind: "sine", Amplitude: 1, PeriodMS: 1000}
		}, "writable point cannot have behaviour"},
		{"a behaviour on a bit", func(f *File) {
			f.Devices[0].Points[1].Behaviour = &Behaviour{Kind: "sine", Amplitude: 1, PeriodMS: 1000}
		}, "a bit cannot have behaviour"},
		{"a sine without a period", func(f *File) { f.Devices[0].Points[4].Behaviour.PeriodMS = 0 }, "sine needs period_ms above 0"},
		{"a sine without an amplitude", func(f *File) { f.Devices[0].Points[4].Behaviour.Amplitude = 0 }, "sine needs amplitude above 0"},
		{"a sine that leaves the type", func(f *File) { f.Devices[0].Points[4].Value = 32760 }, "does not fit int16"},
		{"a ramp without max", func(f *File) { f.Devices[0].Points[5].Behaviour.Max = nil }, "ramp needs min below max"},
		{"a ramp whose min is above max", func(f *File) { f.Devices[0].Points[5].Behaviour.Min = &high }, "ramp needs min below max"},
		{"a ramp without a period", func(f *File) { f.Devices[0].Points[5].Behaviour.PeriodMS = -3 }, "ramp needs period_ms above 0"},
		{"a random walk without a seed", func(f *File) { f.Devices[0].Points[6].Behaviour.Seed = nil }, "random_walk needs a seed"},
		{"a random walk without a step", func(f *File) { f.Devices[0].Points[6].Behaviour.Step = 0 }, "random_walk needs step above 0"},
		{"a random walk without an interval", func(f *File) { f.Devices[0].Points[6].Behaviour.IntervalMS = 0 }, "random_walk needs interval_ms above 0"},
		{"a random walk that starts outside its bounds", func(f *File) { f.Devices[0].Points[6].Value = 11 }, "random_walk needs value within min-max"},
		{"a negative point fail_after_reads", func(f *File) { f.Devices[0].Points[6].Faults.FailAfterReads = &neg }, "fail_after_reads -1"},
		{"a negative point delay", func(f *File) { f.Devices[0].Points[6].Faults.DelayMS = -2 }, "delay_ms -2"},
	}
	for _, c := range cases {
		f := parse(t)
		c.break_(f)
		faults := Validate(f)
		if len(faults) != 1 || !strings.Contains(faults[0], c.want) {
			t.Errorf("%s: faults = %q, want one holding %q", c.name, faults, c.want)
		}
	}
}

func TestValidateReportsEveryFaultOfABrokenScenario(t *testing.T) {
	t.Parallel()
	f := parse(t)
	f.UnknownUnit = "ignore"
	f.Devices[0].Unit = 0
	f.Devices[0].Points[0].Table = "coils"
	if faults := Validate(f); len(faults) != 3 {
		t.Errorf("faults = %q, want the three of them", faults)
	}
}

func TestBuildMakesTheDevicesAndTheirRegisterMaps(t *testing.T) {
	t.Parallel()
	b := parse(t).Build()
	u := b.Units[1]
	if u == nil || len(b.Devices) != 1 || b.Devices[0].Name() != "unit-a" {
		t.Fatalf("Build = %+v", b)
	}
	if got := u.ReadWords(regmap.HoldingRegisters, 0, 3, 0); got[0] != 0x0000 || got[1] != 0x41AC || got[2] != 1200 {
		t.Errorf("holding 0-2 = %04X, want 21.5 as CDAB then 1200", got)
	}
	// The sine swings 20 around its value 180 with a period of 60 s.
	if got := u.ReadWords(regmap.InputRegisters, 0, 1, 15*time.Second); got[0] != 200 {
		t.Errorf("supply at a quarter period = %d, want 200", got[0])
	}
	if got := u.ReadWords(regmap.InputRegisters, 1, 1, 5*time.Second); got[0] != 50 {
		t.Errorf("level at half its ramp = %d, want 50", got[0])
	}
	d := u.Device()
	if !d.Offline(1500*time.Millisecond) || d.Offline(2500*time.Millisecond) {
		t.Error("the offline window was not built")
	}
	if got := d.Delay([]string{"flow"}); got != 20*time.Millisecond {
		t.Errorf("delay = %v, want the device's 20ms (the point asks 5ms)", got)
	}
	if d.Admit([]string{"flow"}, true) {
		t.Error("flow fails after 0 reads, yet a read was admitted")
	}
}

func TestTheShippedExamplesAreValid(t *testing.T) {
	t.Parallel()
	paths, err := filepath.Glob(filepath.Join("..", "examples", "*.json"))
	if err != nil || len(paths) < 2 {
		t.Fatalf("examples: %v, %v; want at least two", paths, err)
	}
	for _, path := range paths {
		f, faults, err := Load(path)
		if err != nil || len(faults) != 0 {
			t.Errorf("%s: %v, faults %q", path, err, faults)
			continue
		}
		if len(f.Build().Units) == 0 {
			t.Errorf("%s: no unit built", path)
		}
	}
	if _, _, err := Load(filepath.Join(t.TempDir(), "absent.json")); !os.IsNotExist(err) {
		t.Errorf("a missing file: %v, want a not-exist error", err)
	}
}
