// Package retroissues is retro-issues (toolbelt/retro-issues.sh): "retro deltas are GitHub issues". It turns the
// mcp-n4 and build-n4-module delta rows of one or more retros into issues so a delta is not lost in a retro file.
// Dry run by default; --apply files them. Every issue is scrubbed against the private client denylist before anything
// leaves the machine, carries a hidden `<!-- retro-delta: <stem-or-alias> Δ<id> -->` marker (the exact dedup key),
// and is searched for by meaning before it is filed: a probable duplicate becomes a comment on the existing issue, a
// closed match is reported as already fixed and never reopened, deltas of several retros that describe one change
// share one issue. A retro outside the kit is named by an alias, so a client-named stem never reaches GitHub.
package retroissues

import (
	"fmt"
	"path/filepath"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/clientdenylist"
	"github.com/angeles725/niagara-tools/go/internal/retrogh"
)

// Exit codes.
const (
	ExitOK    = 0 // planned, or every issue created / already there
	ExitFail  = 1 // an issue refused by the scrub, a gh failure while applying, or open retro-delta issues under --check
	ExitUsage = 3 // usage or environment error; nothing was created
)

// KitRootEnv names the kit root (the toolbelt launcher exports its own ../ directory).
const KitRootEnv = "RETRO_ISSUES_KIT_ROOT"

// Main is retro-issues over the real gh.
func Main(args []string, s cli.Stdio) int { return Command(retrogh.Exec)(args, s) }

type app struct {
	o       options
	s       cli.Stdio
	run     retrogh.Runner
	rules   clientdenylist.Rules
	noRules bool // no denylist file on this machine

	issues    []retrogh.Issue
	markers   map[string]retrogh.Issue
	remoteErr error
}

func (a *app) outf(f string, x ...any) { fmt.Fprintf(a.s.Out, f, x...) }
func (a *app) errf(f string, x ...any) { fmt.Fprintf(a.s.Err, "retro-issues: "+f+"\n", x...) }

func denylistPath(s cli.Stdio) string {
	if p := s.Env("NIAGARA_CLIENT_DENYLIST"); p != "" {
		return p
	}
	return filepath.Join(s.Env("HOME"), ".config", "niagara-private", "client-denylist.txt")
}

// Command is retro-issues with an injectable gh runner.
func Command(run retrogh.Runner) cli.Main {
	return func(args []string, s cli.Stdio) int {
		o, msg := parseArgs(args, s.Env(KitRootEnv))
		if msg != "" {
			fmt.Fprint(s.Err, "retro-issues: "+msg)
			return ExitUsage
		}
		a := &app{o: o, s: s, run: run}
		if o.check {
			return a.checkOpen()
		}
		rules, found, err := clientdenylist.Load(denylistPath(s))
		switch {
		case err != nil:
			a.errf("%v", err)
			return ExitUsage
		case !found && o.apply:
			a.errf("no client denylist ($NIAGARA_CLIENT_DENYLIST, default ~/.config/niagara-private/client-denylist.txt): --apply cannot scrub, nothing was created")
			return ExitUsage
		}
		a.rules, a.noRules = rules, !found
		ds, sk, err := collect(o)
		if err != nil {
			a.errf("%v", err)
			return ExitUsage
		}
		return a.plan(ds, sk)
	}
}

// plan fetches the existing issues, decides every delta, prints the plan and, with --apply, executes it.
func (a *app) plan(ds []delta, sk skipped) int {
	var notes []string
	if a.noRules {
		a.outf("scrub: no denylist on this machine, the issues below are NOT checked\n")
	}
	if len(ds) > 0 {
		if a.issues, a.remoteErr = retrogh.ListLabeled(a.run, retrogh.Label); a.remoteErr != nil {
			if a.o.apply {
				a.errf("cannot list the existing issues, so nothing was created: %v", a.remoteErr)
				return ExitFail
			}
			notes = append(notes, "dedup unchecked: "+a.remoteErr.Error())
		}
		a.markers = retrogh.Markers(a.issues)
	}
	gs, more, err := a.decide(ds)
	if err != nil {
		a.errf("search failed, so nothing was created: %v", err)
		return ExitFail
	}
	for _, n := range append(notes, more...) {
		a.outf("note: %s\n", n)
	}
	for _, t := range sk.tracked {
		a.outf("%s\n", t)
	}
	code := a.report(gs)
	if a.o.apply {
		if c := a.execute(gs); c != ExitOK {
			code = c
		}
	}
	a.outf("%s\n", summary(gs, sk, a.o.apply))
	return code
}

// checkOpen lists the retro-delta issues still open: the deltas nobody has folded yet.
func (a *app) checkOpen() int {
	issues, err := retrogh.ListLabeled(a.run, retrogh.Label)
	if err != nil {
		a.errf("cannot list the retro-delta issues: %v", err)
		return ExitUsage
	}
	open := 0
	for _, is := range issues {
		if !is.Closed() {
			open++
			a.outf("open #%d: %s\n", is.Number, is.Title)
		}
	}
	a.outf("retro-issues: %d open retro-delta issues\n", open)
	if open > 0 {
		return ExitFail
	}
	return ExitOK
}
