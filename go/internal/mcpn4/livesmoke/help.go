package livesmoke

import "strings"

// rawHelp is what --help prints: argparse's rendering of the parser at 80 columns, its usage
// block first.
const rawHelp = `
usage: live-smoke [-h] [--apply] [--station NAME=URL]
                  [--station-home NAME=PATH] [--credential-env PREFIX]
                  [--insecure-tls NAME] [--state-dir STATE_DIR]
                  [--report PATH] [--settle SECONDS] [--call-timeout SECONDS]
                  [--allow-http-for-tests] [--go-bin PATH]

Live smoke runner for mcp-n4-kit: drives the REAL MCP server over stdio.

options:
  -h, --help            show this help message and exit
  --apply               execute (default: plan only)
  --station NAME=URL
  --station-home NAME=PATH
  --credential-env PREFIX
                        env var prefix holding <PREFIX>_USER/<PREFIX>_PASSWORD
  --insecure-tls NAME
  --state-dir STATE_DIR
  --report PATH         also write the JSON report
  --settle SECONDS      how long to wait for the logic outputs to settle
                        (default 5)
  --call-timeout SECONDS
  --allow-http-for-tests
  --go-bin PATH         Go server binary (default: $MCP_N4_GO_BIN, else
                        go/bin/mcp-n4)
`

// helpText is rawHelp without the newline that opens the literal.
var helpText = rawHelp[1:]

// usageText is the usage block argparse prints before an error.
var usageText = helpText[:strings.Index(helpText, "\n\n")+1]
