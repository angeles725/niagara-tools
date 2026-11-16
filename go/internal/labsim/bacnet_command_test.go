package labsim

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/labsim/scenario"
)

func TestCheckPrintsTheObjectListAnIntegratorSees(t *testing.T) {
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
		`BACnet device ahu-1: instance 1001, listen 127.0.0.1:47809, vendor 999 "lab-sim", model "lab-sim", max APDU 1476, COV on`,
		"DEVICE OBJECT TYPE INSTANCE NAME UNITS COMMANDABLE ACCESS",
		"ahu-1 device 1001 ahu-1 - - read-only",
		"ahu-1 binary-output 1 fan_command - yes read/write",
		"ahu-1 binary-value 1 heating_enable - no read/write",
		"ahu-1 binary-input 1 fan_status - no read-only",
		"ahu-1 analog-input 3 outdoor_air_temp degrees-celsius no read-only",
		"ahu-1 analog-input 1 supply_air_temp_x10 no-units no read-only",
		"ahu-1 analog-output 2 fan_speed_pct percent yes read/write",
		"ahu-1 analog-value 1 supply_setpoint_x10 no-units no read/write",
		"ahu-1 analog-value 2 fault_code no-units no read-only",
	} {
		if !rows[want] {
			t.Errorf("no row %q in\n%s", want, out)
		}
	}
	_, out, _ = exec("check", "examples/meter.json")
	for _, want := range []string{"BACnet device meter-slow: instance 2002, listen 127.0.0.1:47810", "COV off", "multi-state-value  1"} {
		if !strings.Contains(out, want) {
			t.Errorf("no %q in\n%s", want, out)
		}
	}
	plain := write(t, `{"devices":[{"name":"d","unit":1,"points":[{"name":"p","table":"coil","address":0}]}]}`)
	if _, out, _ := exec("check", plain); strings.Contains(out, "BACnet") {
		t.Errorf("a scenario with no BACnet device prints a BACnet section:\n%s", out)
	}
}

func TestBACnetAddressesNeedTheFlagToLeaveLoopback(t *testing.T) {
	t.Parallel()
	one, two := 1, 2
	device := func(name, listen string, instance *int) scenario.Device {
		return scenario.Device{Name: name, BACnet: &scenario.BACnetDevice{DeviceInstance: instance, Listen: listen}}
	}
	local := &scenario.File{Devices: []scenario.Device{device("a", "127.0.0.1:47809", &one), {Name: "plain"}}}
	exposed := &scenario.File{Devices: []scenario.Device{device("a", "192.0.2.1:47808", &one), device("b", "127.0.0.1:47810", &two)}}
	if got, err := bacnetAddresses(local, "", false); err != nil || len(got) != 1 || got["a"] != "127.0.0.1:47809" {
		t.Errorf("a loopback scenario: %v, %v", got, err)
	}
	if _, err := bacnetAddresses(exposed, "", false); err == nil || !strings.Contains(err.Error(), "--bacnet-listen a=192.0.2.1:47808") {
		t.Errorf("a scenario that leaves loopback, no flag: %v; want a refusal that names the flag", err)
	}
	if got, err := bacnetAddresses(exposed, "a=192.0.2.1:47808", true); err != nil || got["a"] != "192.0.2.1:47808" || got["b"] != "127.0.0.1:47810" {
		t.Errorf("the same address asked with --bacnet-listen: %v, %v", got, err)
	}
	if got, err := bacnetAddresses(exposed, "b=127.0.0.1:0,a=127.0.0.1:0", true); err != nil || got["a"] != "127.0.0.1:0" || got["b"] != "127.0.0.1:0" {
		t.Errorf("both devices moved by the flag: %v, %v", got, err)
	}
	if got, err := bacnetAddresses(local, "192.0.2.7:47808", true); err != nil || got["a"] != "192.0.2.7:47808" {
		t.Errorf("a bare address for the only BACnet device: %v, %v", got, err)
	}
	for name, c := range map[string]struct {
		f          *scenario.File
		flag, want string
	}{
		"a bare address with two BACnet devices": {exposed, "127.0.0.1:0", "the scenario has 2 BACnet devices"},
		"a device the scenario has not":          {local, "z=127.0.0.1:0", `names "z", which is not a BACnet device`},
		"a device that is not a BACnet device":   {local, "plain=127.0.0.1:0", `names "plain", which is not a BACnet device`},
		"an address without a port":              {local, "a=127.0.0.1", "is not an IPv4 address and a port"},
		"an IPv6 address":                        {local, "a=[::1]:47809", "is not an IPv4 address and a port"},
		"a name":                                 {local, "a=localhost:47809", "is not an IPv4 address and a port"},
		"the same device twice":                  {local, "a=127.0.0.1:1,a=127.0.0.1:2", `names "a" twice`},
	} {
		if got, err := bacnetAddresses(c.f, c.flag, true); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s (%q): %v, %v; want a refusal with %q", name, c.flag, got, err, c.want)
		}
	}
}

func TestRunRefusesToExposeBACnetByAccident(t *testing.T) {
	t.Parallel()
	// 192.0.2.1 is a documentation address this machine does not have: were the
	// guard ever dropped, the bind would fail instead of exposing a test server.
	path := write(t, `{"devices":[{"name":"d","unit":1,"bacnet":{"device_instance":1,"listen":"192.0.2.1:0"},
	  "points":[{"name":"p","table":"coil","address":0}]}]}`)
	if code, out, errs := exec("run", "--listen", "127.0.0.1:0", "--control", "127.0.0.1:0", path); code != 1 || out != "" || !strings.Contains(errs, "--bacnet-listen d=192.0.2.1:0") {
		t.Errorf("exit %d, stdout %q, stderr %q", code, out, errs)
	}
	for _, args := range [][]string{
		{"bacnet-probe", "--addr", "192.0.2.10:47809", "who-is"},
		{"bacnet-probe", "--addr", ":47809", "read", "device,1", "object-name"},
	} {
		if code, out, errs := exec(args...); code != 1 || out != "" || !strings.Contains(errs, "loopback") {
			t.Errorf("%v: exit %d, stdout %q, stderr %q; want a loopback refusal", args, code, out, errs)
		}
	}
	if code, _, errs := exec("run", "--listen", "127.0.0.1:0", "--control", "127.0.0.1:0", "--bacnet-broadcast", "192.0.2.255", ahu); code != 2 || !strings.Contains(errs, "--bacnet-broadcast") {
		t.Errorf("a broadcast address without a port: exit %d, stderr %q", code, errs)
	}
}

func TestAWrongBACnetProbeLineIsAUsageError(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"bacnet-probe"}, {"bacnet-probe", "discover"}, {"bacnet-probe", "who-is", "1"}, {"bacnet-probe", "who-is", "9", "1"},
		{"bacnet-probe", "read", "analog-input,1"}, {"bacnet-probe", "read", "analog_input,1", "present-value"},
		{"bacnet-probe", "read", "analog-input,x", "present-value"}, {"bacnet-probe", "read", "analog-input,1", "value"},
		{"bacnet-probe", "read", "analog-input,1", "present-value", "-1"},
		{"bacnet-probe", "read-multiple", "analog-input,1"}, {"bacnet-probe", "read-multiple", "analog-input,1", "all,nothing"},
		{"bacnet-probe", "write", "analog-output,1"}, {"bacnet-probe", "write", "analog-output,1", "high"},
		{"bacnet-probe", "write", "device,1", "5"}, {"bacnet-probe", "write", "analog-output,1", "5", "--priority", "17"},
		{"bacnet-probe", "write", "binary-output,1", "2"}, {"bacnet-probe", "write", "multi-state-value,1", "1.5"},
		{"bacnet-probe", "subscribe"}, {"bacnet-probe", "subscribe", "analog-input,1", "--mode", "sometimes"},
		{"bacnet-probe", "subscribe", "analog-input,1", "--count", "0"}, {"bacnet-probe", "who-is", "--timeout-ms", "0"},
		{"bacnet-probe", "who-is", "--max-apdu", "100"},
	} {
		if code, out, errs := exec(args...); code != 2 || out != "" || errs == "" {
			t.Errorf("%v: exit %d, stdout %q, stderr %q; want exit 2 and a message", args, code, out, errs)
		}
	}
}

// running is a lab-sim run of the air-handling example on ports of its own.
type running struct {
	t       *testing.T
	out     *lines
	done    chan int
	errs    bytes.Buffer
	control string
	bacnet  string
	logPath string
}

func startRun(t *testing.T) *running {
	t.Helper()
	r := &running{t: t, out: &lines{ch: make(chan string, 64)}, done: make(chan int, 1), logPath: filepath.Join(t.TempDir(), "requests.jsonl")}
	go func() {
		r.done <- Main([]string{"run", "--listen", "127.0.0.1:0", "--control", "127.0.0.1:0", "--bacnet-listen", "127.0.0.1:0", "--log", r.logPath, ahu},
			cli.Stdio{Out: r.out, Err: &r.errs})
	}()
	var ready struct {
		Event, Control string
		BACnet         []struct{ Device, Address string }
	}
	select {
	case line := <-r.out.ch:
		if err := json.Unmarshal([]byte(line), &ready); err != nil || ready.Event != "listening" || len(ready.BACnet) != 1 || ready.BACnet[0].Device != "ahu-1" {
			t.Fatalf("ready line %q: %v", line, err)
		}
	case code := <-r.done:
		t.Fatalf("run ended with exit %d: %s", code, r.errs.String())
	case <-time.After(10 * time.Second):
		t.Fatal("no ready line")
	}
	r.control, r.bacnet = ready.Control, ready.BACnet[0].Address
	return r
}

// stop ends the run and returns its request log.
func (r *running) stop() string {
	r.t.Helper()
	if code, _, errText := exec("stop", "--control", r.control); code != 0 {
		r.t.Fatalf("stop: exit %d, %s", code, errText)
	}
	select {
	case code := <-r.done:
		if code != 0 {
			r.t.Errorf("run exit %d after a stop, stderr %q", code, r.errs.String())
		}
	case <-time.After(10 * time.Second):
		r.t.Fatal("run did not end after the stop")
	}
	logged, err := os.ReadFile(r.logPath)
	if err != nil {
		r.t.Fatal(err)
	}
	return string(logged)
}

func (r *running) probe(args ...string) (int, string, string) {
	return exec(append([]string{"bacnet-probe", "--addr", r.bacnet}, args...)...)
}

func TestRunServesBACnetBesideModbus(t *testing.T) {
	t.Parallel()
	r := startRun(t)
	steps := []struct {
		args []string
		code int
		want string
	}{
		{[]string{"who-is"}, 0, `{"service":"i-am","from":"` + r.bacnet + `","device":"device,1001","max_apdu":1476,"segmentation":"none","vendor":999}`},
		{[]string{"who-is", "1001", "1001"}, 0, `{"service":"i-am","from":"` + r.bacnet + `","device":"device,1001","max_apdu":1476,"segmentation":"none","vendor":999}`},
		{[]string{"read", "device,1001", "object-list", "0"}, 0, `{"object":"device,1001","property":"object-list","index":0,"values":[{"type":"unsigned","value":"15"}]}`},
		{[]string{"read", "device,1001", "object-list", "2"}, 0, `{"object":"device,1001","property":"object-list","index":2,"values":[{"type":"object-identifier","value":"binary-output,1"}]}`},
		{[]string{"read", "analog-input,4", "present-value"}, 0, `{"object":"analog-input,4","property":"present-value","values":[{"type":"real","value":"250"}]}`},
		{[]string{"write", "analog-output,2", "55", "--priority", "8"}, 0, `{"object":"analog-output,2","property":"present-value","priority":8,"written":"55","result":"ok"}`},
		{[]string{"read", "analog-output,2", "priority-array", "8"}, 0, `{"object":"analog-output,2","property":"priority-array","index":8,"values":[{"type":"real","value":"55"}]}`},
		{[]string{"write", "analog-output,2", "null", "--priority", "8"}, 0, `{"object":"analog-output,2","property":"present-value","priority":8,"written":"null","result":"ok"}`},
		{[]string{"read", "analog-output,2", "present-value"}, 0, `{"object":"analog-output,2","property":"present-value","values":[{"type":"real","value":"42.5"}]}`},
		{[]string{"write", "binary-value,1", "1"}, 0, `{"object":"binary-value,1","property":"present-value","written":"1","result":"ok"}`},
		{[]string{"read", "analog-input,99", "present-value"}, 1, `{"object":"analog-input,99","property":"present-value","error":{"class":"object","code":"unknown-object"}}`},
		{[]string{"write", "analog-input,4", "1"}, 1, `{"object":"analog-input,4","property":"present-value","written":"1","error":{"class":"property","code":"write-access-denied"}}`},
		{[]string{"read", "device,1001", "object-list", "--max-apdu", "50"}, 1, `{"object":"device,1001","property":"object-list","abort":"segmentation-not-supported"}`},
	}
	for _, s := range steps {
		if code, got, errText := r.probe(s.args...); code != s.code || got != s.want+"\n" {
			t.Errorf("%v: exit %d, stdout %q, stderr %q; want exit %d, %s", s.args, code, got, errText, s.code, s.want)
		}
	}
	// A commanded output shows its command to the control port and to Modbus;
	// the control port's set moves the relinquish default underneath.
	r.probe("write", "analog-output,2", "60", "--priority", "8")
	ctl := []string{"--control", r.control}
	for _, s := range []struct {
		args []string
		want string
	}{
		{append([]string{"get"}, append(ctl, "ahu-1", "fan_speed_pct")...), "60\n"},
		{append([]string{"set"}, append(ctl, "ahu-1", "fan_speed_pct", "30")...), "60\n"},
		{append([]string{"get"}, append(ctl, "ahu-1", "heating_enable")...), "1\n"},
	} {
		if code, got, errText := exec(s.args...); code != 0 || got != s.want {
			t.Errorf("%v: exit %d, stdout %q, stderr %q; want %q", s.args, code, got, errText, s.want)
		}
	}
	if _, got, _ := r.probe("read", "analog-output,2", "relinquish-default"); !strings.Contains(got, `"value":"30"`) {
		t.Errorf("relinquish-default after the control port's set of 30: %s", got)
	}
	code, all, _ := r.probe("read-multiple", "binary-output,1", "all")
	for _, want := range []string{`"object":"binary-output,1"`, `{"property":"object-name","values":[{"type":"character-string","value":"fan_command"}]}`,
		`{"property":"present-value","values":[{"type":"enumerated","value":"1"}]}`, `{"property":"active-text","values":[{"type":"character-string","value":"Run"}]}`,
		`"property":"priority-array"`, `{"property":"relinquish-default","values":[{"type":"enumerated","value":"1"}]}`} {
		if code != 0 || !strings.Contains(all, want) {
			t.Errorf("read-multiple ALL (exit %d) lacks %s:\n%s", code, want, all)
		}
	}
	if code, got, _ := r.probe("read-multiple", "analog-input,4", "present-value,state-text"); code != 0 ||
		!strings.Contains(got, `{"property":"state-text","error":{"class":"property","code":"unknown-property"}}`) || !strings.Contains(got, `"value":"250"`) {
		t.Errorf("read-multiple with a property the object has not: exit %d, %s", code, got)
	}
	code, status, _ := exec(append([]string{"status"}, ctl...)...)
	for _, want := range []string{`{"protocol":"modbus-tcp","address":"127.0.0.1:`, `{"protocol":"bacnet-ip","address":"` + r.bacnet + `","device":"ahu-1"}`,
		`{"protocol":"control","address":"` + r.control + `"}`} {
		if code != 0 || !strings.Contains(status, want) {
			t.Errorf("status (exit %d) lacks %s:\n%s", code, want, status)
		}
	}
	logged := r.stop()
	for _, want := range []string{
		`"protocol":"bacnet-ip"`, `"device":"ahu-1","link":"original-unicast-npdu","pdu":"unconfirmed-request","service":"who-is","result":"ok","to":["127.0.0.1:`,
		`"service":"read-property","invoke_id":1,"object":"device,1001","property":"object-list","index":0,"result":"ok","read":["15"]`,
		`"service":"write-property","invoke_id":1,"object":"analog-output,2","property":"present-value","priority":8,"written":"55","result":"ok"`,
		`"written":"null","result":"ok"`, `"object":"analog-input,99","property":"present-value","result":"error","error_class":"object","error_code":"unknown-object"`,
		`"result":"abort","reason":"segmentation-not-supported"`, `"service":"read-property-multiple","invoke_id":1,"object":"binary-output,1","detail":"1 objects, 1 references","result":"ok"`,
	} {
		if !strings.Contains(logged, want) {
			t.Errorf("the log lacks %s:\n%s", want, logged)
		}
	}
}

func TestASubscriptionIsNotifiedWhenTheControlPortSetsThePoint(t *testing.T) {
	t.Parallel()
	r := startRun(t)
	out := &lines{ch: make(chan string, 16)}
	var errs bytes.Buffer
	done := make(chan int, 1)
	go func() {
		done <- Main([]string{"bacnet-probe", "--addr", r.bacnet, "subscribe", "analog-input,4", "--lifetime", "120", "--count", "2", "--timeout-ms", "10000"},
			cli.Stdio{Out: out, Err: &errs})
	}()
	next := func(what string) string {
		select {
		case line := <-out.ch:
			return line
		case code := <-done:
			select {
			case line := <-out.ch: // the probe printed it, then ended: both were ready
				done <- code
				return line
			default:
			}
			t.Fatalf("the probe ended (exit %d, %s) before %s", code, errs.String(), what)
		case <-time.After(10 * time.Second):
			t.Fatalf("no %s", what)
		}
		return ""
	}
	if got := next("acknowledgement"); got != `{"object":"analog-input,4","subscribed":"unconfirmed","process":1,"lifetime":120}` {
		t.Errorf("acknowledgement %s", got)
	}
	first := `{"service":"unconfirmed-cov-notification","object":"analog-input,4","device":"device,1001","process":1,"remaining":120,` +
		`"values":[{"property":"present-value","values":[{"type":"real","value":"250"}]},{"property":"status-flags","values":[{"type":"bit-string","value":"0000"}]}]}`
	if got := next("first notification"); got != first {
		t.Errorf("first notification %s\nwant %s", got, first)
	}
	// The value changes behind the protocol; the subscriber hears of it at once.
	if code, _, errText := exec("set", "--control", r.control, "ahu-1", "duct_pressure_pa", "310"); code != 0 {
		t.Fatalf("set: exit %d, %s", code, errText)
	}
	if got := next("notification of the change"); !strings.Contains(got, `{"property":"present-value","values":[{"type":"real","value":"310"}]}`) {
		t.Errorf("notification of the change: %s", got)
	}
	select {
	case code := <-done:
		if code != 0 {
			t.Errorf("the probe ended with exit %d: %s", code, errs.String())
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the probe did not end after its two notifications")
	}
	if code, got, _ := r.probe("subscribe", "analog-input,4", "--mode", "cancel"); code != 0 || got != `{"object":"analog-input,4","subscribed":"cancelled","process":1}`+"\n" {
		t.Errorf("cancel: exit %d, %s", code, got)
	}
	logged := r.stop()
	for _, want := range []string{`"service":"subscribe-cov","invoke_id":1,"object":"analog-input,4","detail":"process 1, unconfirmed, 120 s","result":"ok"`,
		`"service":"unconfirmed-cov-notification","object":"analog-input,4","result":"sent","read":["310","0000"]`, `"detail":"process 1, cancel","result":"ok"`} {
		if !strings.Contains(logged, want) {
			t.Errorf("the log lacks %s:\n%s", want, logged)
		}
	}
}
