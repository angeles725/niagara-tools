// Command retro-draft prints the mcp-n4 session retro draft of a state dir as one JSON object,
// {"markdown": ..., "deltas": N}: what the n4_session_retro_draft tool drafts, for a station and
// date the caller names. mcp-n4-kit/tools/new_retro.py runs it and writes the retro and its INDEX
// row; the Go server's retro package does the drafting, so the two never differ.
//
// Usage: retro-draft --state-dir DIR --station NAME --date YYYY-MM-DD [--since ISO-8601]
//
// Exit: 0 the draft on stdout; 2 usage; 3 a --since that is no timestamp (its message on stderr,
// after "retro-draft: "); 1 a state dir the draft cannot read.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/retro"
)

func main() { cli.Run(run) }

func run(args []string, s cli.Stdio) int {
	fs := flag.NewFlagSet("retro-draft", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	stateDir := fs.String("state-dir", "", "the server's journal/audit directory")
	station := fs.String("station", "", "the station name the draft names")
	date := fs.String("date", "", "the draft's date, YYYY-MM-DD")
	since := fs.String("since", "", "ISO-8601 timestamp; only newer entries count")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 || *stateDir == "" || *station == "" || *date == "" {
		fmt.Fprintln(s.Err, "usage: retro-draft --state-dir DIR --station NAME --date YYYY-MM-DD [--since ISO-8601]")
		return 2
	}
	out, err := retro.DraftOn(*stateDir, *station, *date, *since)
	var ve *pyjson.ValueError
	var typed interface{ PyType() string }
	switch {
	case errors.As(err, &ve):
		fmt.Fprintln(s.Err, "retro-draft: "+ve.Msg)
		return 3
	case errors.As(err, &typed):
		fmt.Fprintln(s.Err, "retro-draft: internal error ("+typed.PyType()+")")
		return 1
	case err != nil:
		fmt.Fprintln(s.Err, "retro-draft: "+err.Error())
		return 1
	}
	markdown, _ := out.Get("markdown")
	reply := pyjson.NewObject()
	reply.Set("markdown", markdown)
	reply.Set("deltas", int64(retro.CountDeltas(markdown.(string))))
	b, err := pyjson.Marshal(reply, pyjson.Options{})
	if err != nil {
		fmt.Fprintln(s.Err, "retro-draft: "+err.Error())
		return 1
	}
	fmt.Fprintln(s.Out, string(b))
	return 0
}
