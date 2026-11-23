// Package signingdoctor is the pre-build check of the signing material a module is re-signed with
// (retro mcpn4-history-batch-tools Δ5): the exported certificate must be the code-signing one, not
// the CA that issued it, and the private key export must be readable by the tool that will use it.
package signingdoctor

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// Exit codes of signing-doctor.
const (
	ExitOK    = 0 // every row PASSes or WARNs
	ExitFail  = 1 // a FAIL row: the material cannot sign
	ExitUsage = 2 // bad arguments or an unreadable file
)

const usage = `usage: signing-doctor [--cert <certificate file>] [--key <private key export file>]
Checks the signing material before a module is built and re-signed: the certificate (PEM or DER) is a
code-signing one, not a CA, and in date; the private key export is in a form OpenSSL 3 can read.
Exit 0 pass / 1 a FAIL row / 2 usage.
`

// Main implements signing-doctor.
func Main(args []string, s cli.Stdio) int {
	fs := flag.NewFlagSet("signing-doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	certPath := fs.String("cert", "", "")
	keyPath := fs.String("key", "", "")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 || (*certPath == "" && *keyPath == "") {
		fmt.Fprint(s.Err, usage)
		return ExitUsage
	}
	// A given but empty option (an unset variable in --cert "$CERT") is refused, never read as
	// "not given": the other file's rows alone would pass the gate without checking it.
	empty := ""
	fs.Visit(func(f *flag.Flag) {
		if f.Value.String() == "" && empty == "" {
			empty = f.Name
		}
	})
	if empty != "" {
		fmt.Fprintf(s.Err, "signing-doctor: --%s is empty\n", empty)
		return ExitUsage
	}
	var rows []Row
	if *certPath != "" {
		data, err := os.ReadFile(*certPath)
		if err != nil {
			fmt.Fprintf(s.Err, "signing-doctor: %v\n", err)
			return ExitUsage
		}
		rows = append(rows, CheckCert(data, time.Now())...)
	}
	if *keyPath != "" {
		data, err := os.ReadFile(*keyPath)
		if err != nil {
			fmt.Fprintf(s.Err, "signing-doctor: %v\n", err)
			return ExitUsage
		}
		rows = append(rows, CheckKey(data)...)
	}
	code := ExitOK
	for _, r := range rows {
		fmt.Fprintf(s.Out, "%-4s  %s  %s\n", r.Status, r.Check, r.Detail)
		if r.Status == "FAIL" {
			code = ExitFail
		}
	}
	return code
}
