package blackbox

import (
	"bufio"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// The fake station every Rig serves, and the credentials the server reads from the environment.
const (
	StationName = "FakeStation"
	User        = "admin"
	Password    = "s3cret-pw-value"
	Folder      = "station:|slot:/Folder" // the ORD of the station's Folder (handle "3")
)

// Rig is ToolTestCase: a fake station, the server under test configured for it, and the state
// directory a writes-allowed server keeps its journal and audit in.
type Rig struct {
	T        testing.TB
	Station  *boxstation.Station
	Srv      *Server
	Opts     Options
	StateDir string
}

// NewRig starts a fake station and a read-only server for it over plain HTTP; configure, when
// not nil, edits the options first (Rig.WriteOptions turns writes on).
func NewRig(t testing.TB, configure func(r *Rig, o *Options)) *Rig {
	t.Helper()
	r := &Rig{T: t, Station: boxstation.New(User, Password, StationName), StateDir: filepath.Join(t.TempDir(), "state")}
	r.Station.Start(t)
	r.Opts = Options{AllowHTTP: true, Stations: map[string]string{StationName: r.Station.URL},
		Env: map[string]string{"MCP_N4_USER": User, "MCP_N4_PASSWORD": Password}}
	if configure != nil {
		configure(r, &r.Opts)
	}
	r.Srv = Start(t, r.Opts)
	return r
}

// WriteOptions turns o into the writes-allowed options of the kit's WriteTestCase: the rig's
// state dir, Folder as the write scope and a budget of 200 writes.
func (r *Rig) WriteOptions(o *Options) {
	o.AllowWrites, o.StateDir, o.WriteScopes, o.MaxWrites = true, r.StateDir, []string{Folder}, 200
}

// Restart closes the server and starts another with the rig's options edited by configure.
func (r *Rig) Restart(configure func(o *Options)) {
	r.T.Helper()
	r.Srv.Close()
	if configure != nil {
		configure(&r.Opts)
	}
	r.Srv = Start(r.T, r.Opts)
}

// Connect opens the session (n4_connect to StationName) and returns its result.
func (r *Rig) Connect(extra Args) map[string]any {
	r.T.Helper()
	args := Args{"station": StationName}
	for k, v := range extra {
		args[k] = v
	}
	return r.Srv.OK("n4_connect", args)
}

// ConnectVerified connects with expected_station, which writes require.
func (r *Rig) ConnectVerified() map[string]any {
	r.T.Helper()
	return r.Connect(Args{"expected_station": StationName})
}

// Lines are the JSON lines of state file name (audit.jsonl, journal.jsonl); none when absent.
func (r *Rig) Lines(name string) []map[string]any {
	r.T.Helper()
	f, err := os.Open(filepath.Join(r.StateDir, name))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		r.T.Fatal(err)
	}
	defer f.Close()
	var out []map[string]any
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 64<<20)
	for sc.Scan() {
		var line map[string]any
		if err := json.Unmarshal(sc.Bytes(), &line); err != nil {
			r.T.Fatalf("%s: %v", name, err)
		}
		out = append(out, line)
	}
	return out
}

// Children are the names of the Folder's children.
func (r *Rig) Children() []string {
	var out []string
	r.Station.Do(func(st *boxstation.Station) {
		for _, c := range st.ByHandle("3").Children {
			out = append(out, c.Name)
		}
	})
	return out
}
