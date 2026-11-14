package main

import (
	"errors"
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// printDraft is draft mode: the retro draft as one JSON object {"markdown", "deltas"}.
func printDraft(o writeOpts, s cli.Stdio) int {
	markdown, deltas, err := draft(o.stateDir, o.station, o.date, o.since)
	var r refused
	switch {
	case errors.As(err, &r):
		fmt.Fprintln(s.Err, "retro-draft: "+r.msg)
		return 3
	case err != nil:
		fmt.Fprintln(s.Err, "retro-draft: "+err.Error())
		return 1
	}
	reply := pyjson.NewObject()
	reply.Set("markdown", markdown)
	reply.Set("deltas", int64(deltas))
	b, err := pyjson.Marshal(reply, pyjson.Options{})
	if err != nil {
		fmt.Fprintln(s.Err, "retro-draft: "+err.Error())
		return 1
	}
	fmt.Fprintln(s.Out, string(b))
	return 0
}
