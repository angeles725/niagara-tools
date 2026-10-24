// Package newretro is new-retro (toolbelt/new-retro.sh): it writes a retro stub, regenerates retros/INDEX.md
// through internal/retroindex and flips retro_pending to true in the named module's BUILD-STATE.src.md
// envelope, in that order, so a generator failure lands nothing.
package newretro

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/gen"
	"github.com/angeles725/niagara-tools/go/internal/retroindex"
)

// Exit codes of new-retro (unchanged from the bash tool).
const (
	ExitOK    = 0 // stub written, index regenerated, envelope flipped
	ExitWrite = 1 // a write or the index regeneration failed
	ExitUsage = 3 // usage/env error or the retro already exists; nothing written
)

// DefaultRootEnv names the kit root used when neither $KIT nor a leading <kit-root> argument gives one; the
// toolbelt/new-retro.sh wrapper exports it as its own ../ directory (the bash script's BASH_SOURCE parent).
const DefaultRootEnv = "NEW_RETRO_DEFAULT_ROOT"

const usage = "usage: new-retro.sh <module|kit> <slug> [--date YYYY-MM-DD] [--deltas N]\n"

// Main is new-retro with today's local date as the default --date.
func Main(args []string, s cli.Stdio) int {
	return Command(func() string { return time.Now().Format("2006-01-02") })(args, s)
}

// Command is new-retro with today as the source of the default --date.
func Command(today func() string) cli.Main {
	return func(args []string, s cli.Stdio) int {
		root, args := kitRoot(args, s)
		r, msg := parse(args, today)
		if msg != "" {
			fmt.Fprint(s.Err, msg)
			return ExitUsage
		}
		return r.write(root, s)
	}
}

// kitRoot is $KIT when it is a directory, else a first argument holding retros/ (made absolute and
// consumed), else DefaultRootEnv, else the cwd.
func kitRoot(args []string, s cli.Stdio) (string, []string) {
	if k := s.Env("KIT"); k != "" && gen.IsDir(k) {
		return k, args
	}
	if len(args) > 0 && gen.IsDir(args[0]) && gen.IsDir(args[0]+"/retros") {
		if abs, err := filepath.Abs(args[0]); err == nil {
			return abs, args[1:]
		}
	}
	if d := s.Env(DefaultRootEnv); d != "" {
		return d, args
	}
	wd, _ := os.Getwd()
	return wd, args
}

type retro struct{ module, slug, date, deltas string }

// parse reads `<module> <slug> [--date D] [--deltas N]...`; a non-empty message is a usage error.
func parse(args []string, today func() string) (retro, string) {
	if len(args) < 2 {
		return retro{}, usage
	}
	r := retro{module: args[0], slug: args[1], date: today(), deltas: "0"}
	if !slugOK(r.slug) {
		return r, "new-retro: slug must be [A-Za-z0-9][A-Za-z0-9._-]+ and >= 6 chars: " + r.slug + "\n"
	}
	for rest := args[2:]; len(rest) > 0; rest = rest[2:] {
		var dst *string
		var valid func(string) bool
		var want string
		switch rest[0] {
		case "--date":
			dst, valid, want = &r.date, isDate, "a YYYY-MM-DD date"
		case "--deltas":
			dst, valid, want = &r.deltas, isCount, "a non-negative integer"
		default:
			return r, "new-retro: unknown option: " + rest[0] + "\n"
		}
		if len(rest) < 2 {
			return r, "new-retro: " + rest[0] + " requires a value\n"
		}
		// Mutation: NR-values -- accepting any value lets --date ../x write outside retros/ and --deltas -1 fail late
		if !valid(rest[1]) {
			return r, "new-retro: " + rest[0] + " must be " + want + ": " + rest[1] + "\n"
		}
		*dst = rest[1]
	}
	return r, ""
}

// slugOK: an alphanumeric, then at least five of [A-Za-z0-9._-], and nothing else.
func slugOK(slug string) bool {
	if len(slug) < 6 || !alnum(slug[0]) {
		return false
	}
	for i := 1; i < len(slug); i++ {
		// Mutation: NR-slug -- checking only slug[1:6] (the bash glob) lets "abcdef/../../x" reach the write
		if c := slug[i]; !alnum(c) && c != '.' && c != '_' && c != '-' {
			return false
		}
	}
	return true
}

// isDate: a calendar date spelled YYYY-MM-DD (it names the retro file, so no path can hide in it).
func isDate(v string) bool {
	_, err := time.Parse("2006-01-02", v)
	return err == nil
}

// isCount: decimal digits only, the deltas header gen-retro-index accepts.
func isCount(v string) bool { return v != "" && strings.Trim(v, "0123456789") == "" }

func alnum(c byte) bool { return c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' }

// write lands the stub, then the regenerated index (the stub is removed again when that fails), then the
// flipped envelope, and prints the three PASS rows.
func (r retro) write(root string, s cli.Stdio) int {
	retros, state := root+"/retros", root+"/BUILD-STATE.src.md"
	file := retros + "/" + r.date + "-" + r.slug + ".md"
	switch {
	case !gen.IsDir(retros):
		fmt.Fprintf(s.Err, "new-retro: retros dir not found: %s\n", retros)
		return ExitUsage
	case !gen.IsRegular(state):
		fmt.Fprintf(s.Err, "new-retro: BUILD-STATE.src.md not found: %s\n", state)
		return ExitUsage
	case gen.IsRegular(file):
		fmt.Fprintf(s.Err, "new-retro: retro already exists: %s\n", file)
		return ExitUsage
	}
	src, err := os.ReadFile(state)
	if err != nil {
		fmt.Fprint(s.Err, "new-retro: failed to stage BUILD-STATE.src.md\n")
		return ExitWrite
	}
	if err := create(file, r.stub()); err != nil {
		fmt.Fprint(s.Err, "new-retro: failed to write retro file\n")
		return ExitWrite
	}
	content, code := retroindex.Generate(root, s.Err)
	if code == retroindex.ExitOK {
		code = retroindex.Output("gen-retro-index", root, content).Write(cli.Stdio{Out: io.Discard, Err: s.Err})
	}
	if code != retroindex.ExitOK {
		_ = os.Remove(file)
		fmt.Fprint(s.Err, "new-retro: gen-retro-index.sh failed; retro removed, INDEX.md and BUILD-STATE.src.md unchanged\n")
		return ExitWrite
	}
	if err := os.WriteFile(state, []byte(Flip(string(src), r.module)), 0o666); err != nil {
		fmt.Fprint(s.Err, "new-retro: failed to update BUILD-STATE.src.md\n")
		return ExitWrite
	}
	fmt.Fprintf(s.Out, "retro    PASS  %s  written\n", file)
	fmt.Fprintf(s.Out, "index    PASS  %s  regenerated\n", retros+"/INDEX.md")
	fmt.Fprint(s.Out, "envelope  PASS  BUILD-STATE.src.md  retro_pending: true\n")
	return ExitOK
}

// create writes a new file; an existing path is an error, never overwritten.
func create(path, content string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
	if err != nil {
		return err
	}
	_, err = f.WriteString(content)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// stub is the retro template with the header block on lines 1-3 and one placeholder row per delta.
func (r retro) stub() string {
	var rows strings.Builder
	n, _ := strconv.Atoi(r.deltas)
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&rows, "| Δ%d | <the change, one line> | `<kit file>` § `<exact heading>` | `[ev: corpus B<n>]` |\n", i)
	}
	row := "| " + r.date + "-" + r.slug + ".md | " + r.module + " | " + r.date + " | pending | " + r.deltas + " |"
	return "<!-- review-status: pending -->\n<!-- module: " + r.module + " -->\n<!-- deltas: " + r.deltas + " -->\n" +
		"# " + r.date + " · " + r.module + " · " + r.slug + "\n\n" +
		"**Session**: <campaign/PR context — one line>\n**Delta count**: " + r.deltas + "\n\n" +
		"## What happened\n<one paragraph: the TRIGGER — what run/change/defect produced this retro>\n\n" +
		"## Evidence\n- <console row / test name / commit / file:line>, each with a token: `[ev: corpus B<n>]` / " +
		"`[ev: <console file>]` / `[ev: <commit sha>]`\n\n" +
		"## Proposed kit deltas (propose-never-apply)\n| Δ | Delta | Target file / § | Token |\n|---|---|---|---|\n" +
		rows.String() + "\n## Lessons\n- <≤5 bullets; each a durable rule, not a narration>\n\n" +
		"---\n**Status**: PENDING — INDEX row generated from the header: `" + row + "`\n"
}
