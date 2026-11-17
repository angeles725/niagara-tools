package blackbox

// n4_connect and n4_describe_session (port of test_server.TestConnect and TestDescribeSession;
// the non-UTF-8 credential test is pinned by session.TestANonUTF8CredentialIsRefusedWithoutItsPosition).

import (
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestConnectReturnsStationIdentityAndMode(t *testing.T) {
	r := NewRig(t, nil)
	want := map[string]any{"station_name": StationName, "base_url": r.Station.URL, "root_handle": "2",
		"mode": "read-only", "version": "4.14.0.162", "version_source": "/obix/about/",
		"tier": "A", "tier_writes": "allowed"}
	if got := r.Connect(nil); !reflect.DeepEqual(got, want) {
		t.Errorf("connect = %v", got)
	}
}

func TestConnectReportsWritesAllowedMode(t *testing.T) {
	r := NewRig(t, func(r *Rig, o *Options) { o.AllowWrites, o.StateDir = true, r.StateDir })
	if got := r.Connect(nil)["mode"]; got != "writes-allowed" {
		t.Errorf("mode %v", got)
	}
}

func TestCredentialsComeFromTheOperatorConfiguredEnvPrefix(t *testing.T) {
	r := NewRig(t, func(_ *Rig, o *Options) {
		o.CredentialEnv, o.Env = "LAB", map[string]string{"LAB_USER": User, "LAB_PASSWORD": Password}
	})
	if got := r.Connect(nil)["station_name"]; got != StationName {
		t.Errorf("station_name %v", got)
	}
}

func TestCredentialEnvIsNotAToolArgument(t *testing.T) {
	r := NewRig(t, nil)
	var schema map[string]any
	for _, tool := range listTools(r.Srv) {
		if tool["name"] == "n4_connect" {
			schema, _ = tool["inputSchema"].(map[string]any)
		}
	}
	var props []string
	properties, _ := Dig(schema, "properties").(map[string]any)
	for name := range properties {
		props = append(props, name)
	}
	sort.Strings(props)
	if !reflect.DeepEqual(props, []string{"expected_station", "expected_version", "station"}) ||
		!reflect.DeepEqual(schema["required"], []any{"station"}) {
		t.Errorf("properties %v, required %v", props, schema["required"])
	}
	// an unknown argument is refused before any request (retro row M11), never used
	code, msg := rpcRefusal(t, r.Srv, "n4_connect", Args{"station": StationName, "base_url": "https://evil.example"})
	if code != -32602 || msg != "unknown argument 'base_url' for n4_connect" || r.Station.Requests() != 0 {
		t.Errorf("base_url: code %d %q, %d requests", code, msg, r.Station.Requests())
	}
}

func TestHTTPStationIsRefusedAtStartWithoutTheTestFlag(t *testing.T) {
	r := NewRig(t, nil)
	o := r.Opts
	o.AllowHTTP = false
	_, err := StartErr(t, o)
	var se *StartupError
	if !errors.As(err, &se) || !strings.Contains(se.Stderr, "https") || r.Station.Requests() != 0 {
		t.Errorf("err %v, requests %d", err, r.Station.Requests())
	}
}

func TestUnknownStationNameIsRefusedAndNamesTheFlag(t *testing.T) {
	r := NewRig(t, nil)
	text := r.Srv.Err("n4_connect", Args{"station": "other"})
	if !strings.Contains(text, "--station") || !strings.Contains(text, StationName) || r.Station.Requests() != 0 {
		t.Errorf("text %q, requests %d", text, r.Station.Requests())
	}
}

func TestNoConfiguredStationIsRefusedAndNamesTheFlag(t *testing.T) {
	r := NewRig(t, func(_ *Rig, o *Options) { o.Stations = nil })
	text := r.Srv.Err("n4_connect", Args{"station": StationName})
	if !strings.Contains(text, "--station NAME=URL") || r.Station.Requests() != 0 {
		t.Errorf("text %q, requests %d", text, r.Station.Requests())
	}
}

func TestExpectedStationMatchConnects(t *testing.T) {
	if got := NewRig(t, nil).Connect(Args{"expected_station": StationName})["station_name"]; got != StationName {
		t.Errorf("station_name %v", got)
	}
}

func TestDescribeBeforeConnectReportsDisconnected(t *testing.T) {
	want := map[string]any{"connected": false, "station_name": nil, "base_url": nil, "mode": "read-only",
		"server_version": serverVersion, "configured_stations": []any{StationName}, "version": nil,
		"version_source": nil, "tier": nil, "tier_writes": nil}
	if got := withoutServer(t, NewRig(t, nil).Srv.OK("n4_describe_session", nil)); !reflect.DeepEqual(got, want) {
		t.Errorf("describe = %v", got)
	}
}

func TestDescribeAfterConnect(t *testing.T) {
	r := NewRig(t, nil)
	r.Connect(nil)
	want := map[string]any{"connected": true, "station_name": StationName, "base_url": r.Station.URL,
		"mode": "read-only", "server_version": serverVersion, "configured_stations": []any{StationName},
		"version": "4.14.0.162", "version_source": "/obix/about/", "tier": "A", "tier_writes": "allowed"}
	if got := withoutServer(t, r.Srv.OK("n4_describe_session", nil)); !reflect.DeepEqual(got, want) {
		t.Errorf("describe = %v", got)
	}
}

func TestDescribeListsOnlyNamesNeverUrlsOrCredentials(t *testing.T) {
	r := NewRig(t, func(_ *Rig, o *Options) {
		o.Stations = map[string]string{"b": "http://user:pw@h/", "a": "http://h2/"}
	})
	out := r.Srv.OK("n4_describe_session", nil)
	if !reflect.DeepEqual(out["configured_stations"], []any{"a", "b"}) || strings.Contains(jsonText(out), "pw") {
		t.Errorf("describe = %v", out)
	}
}

// withoutServer is a describe reply without its server block (describe_server_test.go pins it).
func withoutServer(t *testing.T, out map[string]any) map[string]any {
	t.Helper()
	if _, ok := out["server"].(map[string]any); !ok {
		t.Errorf("no server block: %v", out)
	}
	delete(out, "server")
	return out
}
