<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-24 · kit · go-obix-link-audit

**Session**: niagara-research quality round 10, branch `r10/obix-link-audit-go` from origin/main c894b377
**Delta count**: 2

## What happened
`toolbelt/obix-link-audit.sh` (179 lines: an awk over the wiring map, `grep -o 'display="(Indirect|Direct): ..."'`
over the raw oBIX text, an awk join, and a curl GET with credentials on stdin) became `go/cmd/obix-link-audit`
(`go/internal/obixlinkaudit`) and the script a thin `go/go-tool.sh` wrapper. It is a standalone command, not a
lint (two inputs, a network mode), modelled on `check-facets-card`. The links are read through `internal/xmlsrc`
(R8-D12) and the live GET through `net/http`: credentials from `OBIX_USER`/`OBIX_PASS` into the basic-auth header
only, no redirect followed, HTTP >= 400 fails, 20 s bound, `InsecureSkipVerify` only under `--insecure`, the
http:// warning kept.

Reading the XML instead of grepping its text removed two false results of the bash tool: a malformed response
(truncated dump, mis-nested element) was audited as far as grep could see it (MISSING rows, exit 1, and a MATCH
from a broken file), and a link written as `&#8594;`, `-&gt;`, with single quotes or with spaces around `=` was
not seen at all (false MISSING). The port prints the `path:line:col` error (exit 3) and reads those spellings as
links. An empty 200 body is now exit 3 instead of every row MISSING.

## Evidence
- 38 bash goldens recorded from `tests/fixtures/obix-link-audit` (every bats scenario, usage edges, repeated
  flags, empty values, CRLF map, edge map/xml, no-table maps; `go/internal/obixlinkaudit/testdata/bash-goldens.json`):
  35 byte-identical from the built binary, 3 declared (`truncated`, `badclose`, `entities`) `[ev: go/internal/obixlinkaudit/command_test.go TestBashGoldens]`
- 7 live runs, bash + real curl vs the Go binary, against a local python station: byte-identical except the
  curl-owned `curl: (22) ...` line of the 404 case `[ev: r10 obix-link-audit live parity harness]`
- RED `undefined: DeclaredRows` / `undefined: get` / `undefined: Main` before each Go commit; OLA-malformed and
  OLA-entities not ok on the pre-switch bash script, ok through the wrapper `[ev: tests/obix-link-audit.bats OLA-malformed]`
- Mutations OLA-http, OLA-malformed, OLA-insecure each turn a Go test red `[ev: go/internal/obixlinkaudit/fetch_test.go TestGetVerifiesTheCertificateUnlessInsecure]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **When the shared reader a port must use (R8-D12 `xmlsrc`) itself removes a false result, the port commit carries that difference, declared and pinned, instead of a byte-identical emulation of the old text scan.** Why: a raw-grep emulation in the first commit is the second tokenizer R8-D12 forbids, and dropping the "fix" commit would restore it; "Port in two steps" stays for every difference the reader does not force. | `CONTRIBUTING.md` § 2.2 "Port in two steps" | `[ev: retro go-obix-link-audit]` |
| Δ2 | **A network tool's bats pin behavior against a local 127.0.0.1 stand-in (http and a throwaway self-signed https), never a stub of the client binary.** Why: the curl stubs pinned argv/stdin plumbing (`-k`, `-K -`, `CURL_RC`) that the port no longer has; the stand-in tests passed on the bash script and on the port alike, so one file proved both before the switch. | `CONTRIBUTING.md` § 2.2 Go module | `[ev: tests/obix-link-audit.bats _station]` |

## Lessons
- `grep -o` over XML text is not an attribute reader: entity spellings, quote style and `=` spacing are all the
  same attribute to XML and invisible to the pattern.
- A tool that audits "what it could find" in a broken input reports issues that look real; a parse error must stop
  the audit.

---
**Status**: PENDING — INDEX row generated from the header.
