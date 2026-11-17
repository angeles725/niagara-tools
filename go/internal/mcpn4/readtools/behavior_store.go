package readtools

import (
	"os"
	"regexp"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
)

// snapshotFormat names the layout of a behaviour snapshot file.
const snapshotFormat = "mcp-n4.behavior-snapshot/v1"

// snapshotSubdir is where snapshots live under the state dir.
const snapshotSubdir = "snapshots"

// snapshotName is a snapshot name: the file is <state-dir>/snapshots/<name>.json.
var snapshotName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,79}$`)
var snapshotFile = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,79}\.json$`)

// snapshot is one behaviour snapshot as stored.
type snapshot struct {
	name, station, scope, takenAt string
	truncated                     bool
	points                        []pointState
}

func (s *snapshot) json() *pyjson.Object {
	pts := make([]any, len(s.points))
	for i, p := range s.points {
		pts[i] = obj("path", p.Path, "value", p.Value, "status", p.Status)
	}
	return obj("format", snapshotFormat, "name", s.name, "station", s.station, "scope", s.scope,
		"taken_at", s.takenAt, "truncated", s.truncated, "points", pts)
}

// saveSnapshot writes s as a new 0600 file (never over an existing one) and returns its path.
func (t *Tools) saveSnapshot(s *snapshot) (string, error) {
	path, err := t.outputPath(s.name+".json", snapshotSubdir, snapshotFile, "(the snapshot name) ")
	if err != nil {
		return "", err
	}
	data, err := pyjson.Marshal(s.json(), pyjson.Options{})
	if err != nil {
		return "", err
	}
	if err := writePrivate(path, func(f *os.File) error { _, err := f.Write(data); return err }); err != nil {
		return "", writeFailed(path, err)
	}
	return path, nil
}

// loadSnapshot reads the snapshot called name; a missing or foreign file is a ToolError.
func (t *Tools) loadSnapshot(name string) (*snapshot, error) {
	if !snapshotName.MatchString(name) {
		return nil, toolError("snapshot name must be letters, digits, '_' or '-' (up to 80), got %s", pyjson.StrRepr(name))
	}
	path := safety.Join(t.stateDir(), snapshotSubdir, name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, toolError("no snapshot %s (%s): take it first with n4_behavior_snapshot", pyjson.StrRepr(name), path)
	}
	v, err := pyjson.Unmarshal(data)
	o, ok := v.(*pyjson.Object)
	if err != nil || !ok || lookup(o, "format") != snapshotFormat {
		return nil, toolError("%s is not a %s file", path, snapshotFormat)
	}
	s := &snapshot{name: name, truncated: lookup(o, "truncated") == true}
	s.station, _ = lookup(o, "station").(string)
	s.scope, _ = lookup(o, "scope").(string)
	s.takenAt, _ = lookup(o, "taken_at").(string)
	raw, _ := lookup(o, "points").([]any)
	for _, r := range raw {
		p, _ := r.(*pyjson.Object)
		path, _ := lookup(p, "path").(string)
		value, _ := lookup(p, "value").(string)
		status, _ := lookup(p, "status").(string)
		s.points = append(s.points, pointState{path, value, status})
	}
	return s, nil
}
