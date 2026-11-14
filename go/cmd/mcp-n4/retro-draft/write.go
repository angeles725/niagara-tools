package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/angeles725/niagara-tools/go/internal/atomicfile"
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/retro"
)

// writeOpts are the --write flags (formerly mcp-n4-kit/tools/new_retro.py).
type writeOpts struct {
	stateDir, station, date, since, outDir string
	force, dryRun                          bool
}

// refused is a refusal: exit 3, its message on stderr after "retro-draft: ".
type refused struct{ msg string }

func (r refused) Error() string { return r.msg }

// writeRetro drafts the retro and writes it with its pending INDEX row (or, with --dry-run,
// prints both and writes nothing). Exit: 0 written or printed; 3 refused; 1 an I/O error.
func writeRetro(o writeOpts, now time.Time, s cli.Stdio) int {
	msg, err := doWrite(o, now)
	var r refused
	switch {
	case errors.As(err, &r):
		fmt.Fprintln(s.Err, "retro-draft: "+r.msg)
		return 3
	case err != nil:
		fmt.Fprintln(s.Err, "retro-draft: "+err.Error())
		return 1
	}
	fmt.Fprint(s.Out, msg)
	return 0
}

func doWrite(o writeOpts, now time.Time) (string, error) {
	if err := retro.CheckStation(o.station); err != nil {
		return "", refused{err.Error()}
	}
	date := o.date
	if date == "" {
		date = now.UTC().Format("2006-01-02")
	}
	if err := retro.CheckDate(date); err != nil {
		return "", refused{err.Error()}
	}
	markdown, deltas, err := draft(o.stateDir, o.station, date, o.since)
	if err != nil {
		return "", err
	}
	name := retro.FileName(date, o.station)
	row := retro.IndexRow(name, o.station, date, deltas)
	if o.dryRun {
		return markdown + "\nINDEX row: " + row + "\n", nil
	}
	path, index := pyJoin(o.outDir, name), pyJoin(o.outDir, "INDEX.md")
	if st, err := os.Stat(index); err != nil || !st.Mode().IsRegular() {
		return "", refused{index + " not found: create the retros directory with its INDEX.md first"}
	}
	if _, err := os.Stat(path); err == nil && !o.force { // follows links, as os.path.exists
		return "", refused{path + " already exists: pass --force to rewrite it"}
	}
	old, err := os.ReadFile(index)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(old) {
		return "", fmt.Errorf("%s is not UTF-8", index)
	}
	if err := atomicfile.Write(path, []byte(markdown)); err != nil {
		return "", err
	}
	if err := atomicfile.Write(index, []byte(retro.IndexRows(string(old), name, row))); err != nil {
		return "", err
	}
	return fmt.Sprintf("retro  %s  written (%d delta(s), pending)\n", path, deltas), nil
}

// draft is the retro markdown and its delta count; a --since that is no timestamp is refused.
func draft(stateDir, station, date, since string) (string, int, error) {
	out, err := retro.DraftOn(stateDir, station, date, since)
	var ve *pyjson.ValueError
	var typed interface{ PyType() string }
	switch {
	case errors.As(err, &ve):
		return "", 0, refused{ve.Msg}
	case errors.As(err, &typed):
		return "", 0, errors.New("internal error (" + typed.PyType() + ")")
	case err != nil:
		return "", 0, err
	}
	v, _ := out.Get("markdown")
	markdown := v.(string)
	return markdown, retro.CountDeltas(markdown), nil
}

// pyJoin joins like Python's os.path.join for one relative name, so the paths printed are
// the ones the caller typed (no cleaning).
func pyJoin(dir, name string) string {
	if dir == "" || strings.HasSuffix(dir, "/") {
		return dir + name
	}
	return dir + "/" + name
}

// defaultOutDir is the kit's retros directory: go/bin/retro-draft -> mcp-n4-kit/retros.
func defaultOutDir() string {
	exe, err := os.Executable()
	if err != nil {
		return filepath.Join("mcp-n4-kit", "retros")
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	return filepath.Join(filepath.Dir(exe), "..", "..", "mcp-n4-kit", "retros")
}
