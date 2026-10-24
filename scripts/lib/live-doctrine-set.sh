# live-doctrine-set.sh — the LIVE build-n4-module doctrine set, defined once: the cards whose every evidence
# pointer must resolve under `kit-cite-resolve --strict` (history cards — retros/, ledger.d/, BUILD-STATE.md —
# are not in it). Sourced by .githooks/pre-push; any later caller sources it too instead of a copy.
#   LIVE_DOCTRINE_RE  ERE over repository-relative paths: the live cards
#   LIVE_CITE_TOOL_RE ERE over repository-relative paths: the resolver itself (a change re-checks the set)
#   live_doctrine_files  stdin: repository-relative paths, one per line → stdout: the live ones
# shellcheck shell=bash
# shellcheck disable=SC2034  # the two patterns are consumed by the sourcing script
LIVE_DOCTRINE_RE='^(build-n4-module-kit/(types/[^/]+|METHODOLOGY|BUILD-LOOP|ORCHESTRATION|SOURCES|build-verify|corpus-index|README|skill/SKILL|toolbelt/INDEX)\.md|go/README\.md|CONTRIBUTING\.md|README\.md)$'
LIVE_CITE_TOOL_RE='^(go/internal/(citeparse|citeresolve)/|go/cmd/kit-cite-resolve/|build-n4-module-kit/toolbelt/kit-cite-resolve\.sh$)'
live_doctrine_files() { grep -E "$LIVE_DOCTRINE_RE" || true; }
