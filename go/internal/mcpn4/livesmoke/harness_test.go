package livesmoke

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	mcpcli "github.com/angeles725/niagara-tools/go/internal/mcpn4/cli"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// serverMarker makes the test binary act as the mcp-n4 server: the black-box tests start the
// real server code through it, the way live-smoke starts go/bin/mcp-n4.
const serverMarker = "LIVESMOKE_TEST_AS_MCP_N4"

func TestMain(m *testing.M) {
	if os.Getenv(serverMarker) == "1" {
		os.Exit(mcpcli.Main(os.Args[1:], cli.Stdio{Out: os.Stdout, Err: os.Stderr, Getenv: os.Getenv}))
	}
	os.Exit(m.Run())
}

const (
	password = "pw-Zk93-unique-canary"
	user     = "user-Qx71-unique-canary"
)

// logicTypes are the thermostat types whose components get a computed `out` slot.
var logicTypes = map[string]string{"control:NumericWritable": "baja:StatusNumeric",
	"control:BooleanWritable": "baja:StatusBoolean", "kitControl:GreaterThan": "baja:StatusBoolean"}

// logicStation is the fake station whose thermostat components have an `out` slot and, while
// compute is on, compute Temp > Setpoint after every action.
type logicStation struct {
	*boxstation.Station
	compute bool
}

func newLogicStation(t *testing.T) *logicStation {
	ls := &logicStation{Station: boxstation.New(user, password, "SmokeStation"), compute: true}
	ls.AfterAdd = func(_ *boxstation.Station, n *boxstation.Node) {
		if typ, ok := logicTypes[n.Type]; ok {
			vt := "baja:Double"
			if typ == "baja:StatusBoolean" {
				vt = "baja:Boolean"
			}
			n.Children = append(n.Children, &boxstation.Node{Name: "out", Type: typ, Children: []*boxstation.Node{
				{Name: "value", Type: vt, Value: "false"}, {Name: "status", Type: "baja:Status", Value: "0"}}})
		}
	}
	ls.AfterInvoke = func(st *boxstation.Station, _ string, _ *boxstation.Node) {
		if ls.compute {
			recompute(st.Root())
		}
	}
	ls.Start(t)
	return ls
}

func recompute(root *boxstation.Node) {
	folder := root.Child(Scratch)
	if folder == nil {
		return
	}
	val := func(name string) float64 {
		c := folder.Child(name)
		if c == nil || c.Child("fallback") == nil {
			return 0
		}
		switch v := c.Child("fallback").Child("value").Value.(type) {
		case float64:
			return v
		case string:
			f, _ := strconv.ParseFloat(v, 64)
			return f
		}
		return 0
	}
	state := "false"
	if val("Temp") > val("Setpoint") {
		state = "true"
	}
	for _, name := range []string{"Compare", "Cooling"} {
		if c := folder.Child(name); c != nil && c.Child("out") != nil {
			c.Child("out").Children[0].Value = state
		}
	}
}

// runner is RunnerCase: a logic station, a station home with config.bog, a report path and the
// environment holding the credentials.
type runner struct {
	t                     *testing.T
	fake                  *logicStation
	tmp, home, reportPath string
	env                   []string
}

func newRunner(t *testing.T) *runner {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	r := &runner{t: t, fake: newLogicStation(t), tmp: t.TempDir()}
	r.home = filepath.Join(r.tmp, "home")
	if err := os.Mkdir(r.home, 0o700); err != nil {
		t.Fatal(err)
	}
	bog := filepath.Join(r.home, "config.bog")
	if err := os.WriteFile(bog, []byte("bog"), 0o600); err != nil {
		t.Fatal(err)
	}
	r.fake.ConfigPath = bog
	r.reportPath = filepath.Join(r.tmp, "report.json")
	r.env = append(os.Environ(), "MCP_N4_USER="+user, "MCP_N4_PASSWORD="+password,
		"MCP_N4_GO_BIN="+exe, serverMarker+"=1")
	return r
}

func (r *runner) setEnv(kv string) { r.env = append(r.env, kv) }

func (r *runner) argv(apply bool, extra ...string) []string {
	base := []string{"--station", "SmokeStation=" + r.fake.URL, "--station-home", "SmokeStation=" + r.home,
		"--state-dir", filepath.Join(r.tmp, "state"), "--report", r.reportPath, "--settle", "1",
		"--allow-http-for-tests"}
	if apply {
		base = append(base, "--apply")
	}
	return append(base, extra...)
}

func (r *runner) run(argv []string) (int, string, string) {
	var out, errOut bytes.Buffer
	code := Main(argv, cli.Stdio{Out: &out, Err: &errOut}, r.env)
	return code, out.String(), errOut.String()
}

// report is the JSON report the run wrote.
func (r *runner) report() map[string]any {
	r.t.Helper()
	data, err := os.ReadFile(r.reportPath)
	if err != nil {
		r.t.Fatal(err)
	}
	var rep map[string]any
	if err := json.Unmarshal(data, &rep); err != nil {
		r.t.Fatalf("report is not JSON: %v\n%s", err, data)
	}
	return rep
}

// steps are the report's steps.
func steps(rep map[string]any) []map[string]any {
	var out []map[string]any
	list, _ := rep["steps"].([]any)
	for _, s := range list {
		out = append(out, s.(map[string]any))
	}
	return out
}
