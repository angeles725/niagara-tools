// Command retro-draft drafts the mcp-n4 session retro of a state dir: what the
// n4_session_retro_draft tool drafts, for a station and date the caller names.
//
// Draft mode prints one JSON object, {"markdown": ..., "deltas": N}:
//
//	retro-draft --state-dir DIR --station NAME --date YYYY-MM-DD [--since ISO-8601]
//
// Exit: 0 the draft on stdout; 2 usage; 3 a --since that is no timestamp (its message on
// stderr, after "retro-draft: "); 1 a state dir the draft cannot read.
//
// Write mode (formerly mcp-n4-kit/tools/new_retro.py) writes YYYY-MM-DD-<station>-session.md
// into the retros directory and its pending row into INDEX.md, both atomically. It proposes
// only: it never edits the kit and never stages issues. An existing retro is never
// overwritten without --force; --dry-run prints the retro and its row and writes nothing:
//
//	retro-draft --write --state-dir DIR --station NAME [--date YYYY-MM-DD] [--since ISO-8601]
//	            [--out-dir DIR] [--force] [--dry-run]
//
// --date defaults to today (UTC) and --out-dir to the kit's mcp-n4-kit/retros. Exit: 0
// written or printed; 2 usage; 3 refused (bad station, date or --since, no INDEX.md, the
// retro exists without --force); 1 an I/O error.
package main

import (
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

const usage = "usage: retro-draft --state-dir DIR --station NAME --date YYYY-MM-DD [--since ISO-8601]\n" +
	"       retro-draft --write --state-dir DIR --station NAME [--date YYYY-MM-DD] [--since ISO-8601]\n" +
	"                   [--out-dir DIR] [--force] [--dry-run]"

func main() { cli.Run(run) }

func run(args []string, s cli.Stdio) int {
	fs := flag.NewFlagSet("retro-draft", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	o := writeOpts{}
	fs.StringVar(&o.stateDir, "state-dir", "", "the server's journal/audit directory")
	fs.StringVar(&o.station, "station", "", "the station name the draft names")
	fs.StringVar(&o.date, "date", "", "the draft's date, YYYY-MM-DD")
	fs.StringVar(&o.since, "since", "", "ISO-8601 timestamp; only newer entries count")
	write := fs.Bool("write", false, "write the retro and its INDEX row")
	fs.StringVar(&o.outDir, "out-dir", "", "retros directory with INDEX.md (write mode)")
	fs.BoolVar(&o.force, "force", false, "overwrite an existing retro (write mode)")
	fs.BoolVar(&o.dryRun, "dry-run", false, "print only; write nothing (write mode)")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 || o.stateDir == "" || o.station == "" ||
		(!*write && (o.date == "" || o.outDir != "" || o.force || o.dryRun)) {
		fmt.Fprintln(s.Err, usage)
		return 2
	}
	if *write {
		if o.outDir == "" {
			o.outDir = defaultOutDir()
		}
		return writeRetro(o, time.Now(), s)
	}
	return printDraft(o, s)
}
