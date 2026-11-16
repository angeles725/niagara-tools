package labsim

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/labsim/scenario"
)

const ahu = "examples/ahu.json"

// exec runs one lab-sim command and returns its exit code, stdout and stderr.
func exec(args ...string) (int, string, string) {
	var out, errs bytes.Buffer
	code := Main(args, cli.Stdio{Out: &out, Err: &errs})
	return code, out.String(), errs.String()
}

func write(t *testing.T, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "scenario.json")
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCheckPrintsTheRegisterMapAnIntegratorTypesIn(t *testing.T) {
	t.Parallel()
	code, out, errs := exec("check", ahu)
	if code != 0 || errs != "" {
		t.Fatalf("exit %d, stderr %q", code, errs)
	}
	rows := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		rows[strings.Join(strings.Fields(line), " ")] = true
	}
	for _, want := range []string{
		"scenario: air-handling unit",
		"listen: 127.0.0.1:1502 control: 127.0.0.1:1503 unknown unit: silent",
		"DEVICE UNIT POINT TABLE ADDRESS REFERENCE TYPE ORDER ACCESS INITIAL BEHAVIOUR",
		"ahu-1 1 fan_command coil 0 00001 bit - read/write 1 constant",
		"ahu-1 1 freeze_alarm discrete_input 2 10003 bit - read-only 0 constant",
		"ahu-1 1 supply_air_temp_x10 input_register 0 30001 int16 ABCD read-only 180 sine +/-15 / 120000 ms",
		"ahu-1 1 return_air_temp_x10 input_register 1 30002 int16 ABCD read-only 225 random_walk +/-2 / 5000 ms within 200..250, seed 11",
		"ahu-1 1 outdoor_air_temp input_register 2-3 30003-30004 float32 ABCD read-only 5 ramp 5..25 / 600000 ms",
		"ahu-1 1 runtime_hours input_register 6-7 30007-30008 uint32 ABCD read-only 12345 constant",
		"ahu-1 1 fan_speed_pct holding_register 2-3 40003-40004 float32 ABCD read/write 42.5 constant",
		"ahu-1 1 fault_code holding_register 10 40011 uint16 ABCD read-only 0 constant",
	} {
		if !rows[want] {
			t.Errorf("no row %q in\n%s", want, out)
		}
	}
}

func TestCheckPrintsTheFaultsOfADeviceAndTheSixDigitReference(t *testing.T) {
	t.Parallel()
	_, out, _ := exec("check", "examples/meter.json")
	for _, want := range []string{
		"meter-slow (unit 11): every response held 3000 ms",
		"meter-flaky (unit 12): fails after 20 reads; offline 60000-120000 ms",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("no line %q in\n%s", want, out)
		}
	}
	path := write(t, `{"devices":[{"name":"d","unit":1,"points":[
	  {"name":"far","table":"holding_register","address":9999,"faults":{"fail_after_reads":3,"delay_ms":40}}]}]}`)
	_, out, _ = exec("check", path)
	if !strings.Contains(out, "410000") || !strings.Contains(out, "d (unit 1), point far: fails after 3 reads; held 40 ms") {
		t.Errorf("address 9999 must be reference 410000 and the point's faults listed:\n%s", out)
	}
}

func TestCheckListsEveryFaultAndFails(t *testing.T) {
	t.Parallel()
	path := write(t, `{"unknown_unit":"ignore","devices":[{"name":"d","unit":0,"points":[{"name":"p","table":"coils","address":0}]}]}`)
	code, out, errs := exec("check", path)
	if code != 1 || out != "" || strings.Count(errs, "\n") != 4 || !strings.Contains(errs, "3 faults") {
		t.Errorf("exit %d, stdout %q, stderr:\n%s", code, out, errs)
	}
	if code, _, errs := exec("check", filepath.Join(t.TempDir(), "absent.json")); code != 1 || !strings.Contains(errs, "absent.json") {
		t.Errorf("a missing file: exit %d, stderr %q", code, errs)
	}
}

func TestAWrongCommandLineIsAUsageError(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{}, {"serve"}, {"check"}, {"check", "a", "b"}, {"run", ahu, "--port", "1"}, {"run", ahu, "--listen"},
		{"get", "ahu-1"}, {"set", "ahu-1", "fan_command"}, {"set", "ahu-1", "fan_command", "on"},
		{"probe", "--fn", "3"}, {"probe", "--fn", "7", "--address", "0", "--count", "1"},
		{"probe", "--fn", "3", "--address", "0", "--values", "1"}, {"probe", "--fn", "6", "--address", "0", "--count", "1"},
		{"probe", "--fn", "6", "--address", "0", "--values", "70000"}, {"probe", "--fn", "3", "--address", "0", "--count", "1", "--unit", "300"},
	} {
		if code, out, errs := exec(args...); code != 2 || out != "" || errs == "" {
			t.Errorf("%v: exit %d, stdout %q, stderr %q; want exit 2 and a message", args, code, out, errs)
		}
	}
	if code, out, _ := exec("help"); code != 0 || !strings.Contains(out, "lab-sim run") {
		t.Errorf("help: exit %d, %q", code, out)
	}
}

func TestListenAddressNeedsTheFlagToLeaveLoopback(t *testing.T) {
	t.Parallel()
	local := &scenario.File{Listen: "127.0.0.1:1502"}
	exposed := &scenario.File{Listen: "0.0.0.0:1502"}
	if got, err := listenAddress(local, "", false); err != nil || got != "127.0.0.1:1502" {
		t.Errorf("a loopback scenario: %q, %v", got, err)
	}
	if _, err := listenAddress(exposed, "", false); err == nil || !strings.Contains(err.Error(), "--listen 0.0.0.0:1502") {
		t.Errorf("a scenario that leaves loopback, no flag: %v; want a refusal that names the flag", err)
	}
	if got, err := listenAddress(exposed, "0.0.0.0:1502", true); err != nil || got != "0.0.0.0:1502" {
		t.Errorf("the same address asked with --listen: %q, %v", got, err)
	}
	if got, err := listenAddress(local, "192.0.2.7:502", true); err != nil || got != "192.0.2.7:502" {
		t.Errorf("--listen over a loopback scenario: %q, %v", got, err)
	}
	if _, err := listenAddress(local, "nowhere", true); err == nil {
		t.Error("--listen without a port was accepted")
	}
}

func TestRunRefusesToExposeItselfByAccident(t *testing.T) {
	t.Parallel()
	// 192.0.2.1 is a documentation address this machine does not have: were the
	// guard ever dropped, the bind would fail instead of exposing a test server.
	path := write(t, `{"listen":"192.0.2.1:0","devices":[{"name":"d","unit":1,"points":[{"name":"p","table":"coil","address":0}]}]}`)
	if code, out, errs := exec("run", "--control", "127.0.0.1:0", path); code != 1 || out != "" || !strings.Contains(errs, "--listen 192.0.2.1:0") {
		t.Errorf("exit %d, stdout %q, stderr %q", code, out, errs)
	}
	for _, args := range [][]string{
		{"run", "--listen", "127.0.0.1:0", "--control", "0.0.0.0:0", ahu},
		{"get", "--control", "192.0.2.10:1503", "ahu-1", "fan_command"},
		{"status", "--control", ":1503"},
		{"probe", "--addr", "192.0.2.10:502", "--fn", "3", "--address", "0", "--count", "1"},
	} {
		if code, out, errs := exec(args...); code != 1 || out != "" || !strings.Contains(errs, "loopback") {
			t.Errorf("%v: exit %d, stdout %q, stderr %q; want a loopback refusal", args, code, out, errs)
		}
	}
}

// lines hands each line written to it to the test.
type lines struct {
	mu   sync.Mutex
	rest []byte
	ch   chan string
}

func (l *lines) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.rest = append(l.rest, p...)
	for {
		i := bytes.IndexByte(l.rest, '\n')
		if i < 0 {
			return len(p), nil
		}
		l.ch <- string(l.rest[:i])
		l.rest = l.rest[i+1:]
	}
}

func TestRunServesAScenarioUntilItIsStopped(t *testing.T) {
	t.Parallel()
	logPath := filepath.Join(t.TempDir(), "requests.jsonl")
	out := &lines{ch: make(chan string, 64)}
	var errs bytes.Buffer
	done := make(chan int, 1)
	go func() {
		done <- Main([]string{"run", "--listen", "127.0.0.1:0", "--control", "127.0.0.1:0", "--log", logPath, ahu}, cli.Stdio{Out: out, Err: &errs})
	}()
	var ready struct{ Event, Scenario, Listen, Control string }
	select {
	case line := <-out.ch:
		if err := json.Unmarshal([]byte(line), &ready); err != nil || ready.Event != "listening" || ready.Scenario != "air-handling unit" {
			t.Fatalf("ready line %q: %v", line, err)
		}
	case code := <-done:
		t.Fatalf("run ended with exit %d: %s", code, errs.String())
	case <-time.After(10 * time.Second):
		t.Fatal("no ready line")
	}
	ctl, bus := []string{"--control", ready.Control}, []string{"probe", "--addr", ready.Listen, "--unit", "1"}
	steps := []struct {
		args []string
		code int
		want string
	}{
		{append([]string{"get"}, append(ctl, "ahu-1", "fan_speed_pct")...), 0, "42.5\n"},
		{append([]string{"set"}, append(ctl, "ahu-1", "duct_pressure_pa", "310")...), 0, "310\n"},
		{append(bus, "--fn", "4", "--address", "4", "--count", "1"), 0, `{"unit":1,"function":4,"address":4,"quantity":1,"values":[310]}` + "\n"},
		{append(bus, "--fn", "6", "--address", "1", "--values", "60"), 0, `{"unit":1,"function":6,"address":1,"quantity":1}` + "\n"},
		{append([]string{"get"}, append(ctl, "ahu-1", "damper_position_pct")...), 0, "60\n"},
		{append(bus, "--fn", "15", "--address", "0", "--values", "0,1"), 0, `{"unit":1,"function":15,"address":0,"quantity":2}` + "\n"},
		{append(bus, "--fn", "1", "--address", "0", "--count", "2"), 0, `{"unit":1,"function":1,"address":0,"quantity":2,"values":[0,1]}` + "\n"},
		{append(bus, "--fn", "3", "--address", "99", "--count", "1"), 1, `{"unit":1,"function":3,"address":99,"quantity":1,"exception":2}` + "\n"},
		{append([]string{"get"}, append(ctl, "ahu-1", "absent")...), 1, ""},
	}
	for _, s := range steps {
		if code, got, errText := exec(s.args...); code != s.code || got != s.want {
			t.Errorf("%v: exit %d, stdout %q, stderr %q; want exit %d, %q", s.args, code, got, errText, s.code, s.want)
		}
	}
	if code, got, _ := exec(append([]string{"probe", "--addr", ready.Listen, "--unit", "9", "--timeout-ms", "50"}, "--fn", "3", "--address", "0", "--count", "1")...); code != 1 || got != "" {
		t.Errorf("a probe of a unit that stays silent: exit %d, stdout %q; want exit 1", code, got)
	}
	code, status, _ := exec(append([]string{"status"}, ctl...)...)
	for _, want := range []string{`"scenario":"air-handling unit"`, `"requests":6`, `"listen":"` + ready.Listen + `"`, `"name":"ahu-1","unit":1,"points":14,"reads":2,"offline":false`} {
		if code != 0 || !strings.Contains(status, want) {
			t.Errorf("status (exit %d) lacks %s:\n%s", code, want, status)
		}
	}
	if code, _, errText := exec(append([]string{"stop"}, ctl...)...); code != 0 {
		t.Fatalf("stop: exit %d, %s", code, errText)
	}
	select {
	case code := <-done:
		if code != 0 {
			t.Errorf("run exit %d after a stop, stderr %q", code, errs.String())
		}
	case <-time.After(10 * time.Second):
		t.Fatal("run did not end after the stop")
	}
	logged, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(logged), "\n"); got != 6 {
		t.Errorf("%d log lines, want 6:\n%s", got, logged)
	}
	for _, want := range []string{`"function":4,"address":4,"quantity":1,"result":"ok","read":[310]`, `"function":6,"address":1,"quantity":1,"written":[60],"result":"ok"`, `"result":"exception","exception":2`, `"unit":9,"function":3,"address":0,"quantity":1,"result":"no_response","reason":"unknown_unit"`} {
		if !strings.Contains(string(logged), want) {
			t.Errorf("the log lacks %s:\n%s", want, logged)
		}
	}
}
