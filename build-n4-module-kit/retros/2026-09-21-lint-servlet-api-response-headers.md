<!-- review-status: folded -->
# 2026-09-21 · kit · lint-servlet api-response-headers check + stale-doc fixes

**Session**: build-n4-module "what's-left" audit follow-up. `security.md §311` documented an `api-response-headers` lint candidate "not yet in lint-servlet.sh" — the last genuine unimplemented per-module lint. Implemented as a CHECK inside the existing `lint-servlet.sh` (already auto-wired into report-module.sh §5.18), so it needs no new wiring and runs in the build.sh chain.

**Delta count**: 0 (implementation retro)

## What was done
- New `api-response-headers` WARN check in `toolbelt/lint-servlet.sh`: a BWebServlet handler that writes a body (`getWriter()`/`getOutputStream()`) with no `X-Content-Type-Options` / `setApiHeaders`/`applyHeaders` anywhere in the file → WARN (defense-in-depth; the global TridiumSecurityFilter usually covers it, per B1133/UXS1). `# Mutation: LSV7` (MATCH); tests LSV7 + LSV7-clean in tests/lint-servlet.bats.
- Stale-doc fixes: security.md §311 (api-response-headers now implemented), security.md §419 (no-md5-credential-digest now implemented, #138), dashboard.md :173 (lint-lexicon-ascii already exists).

## Self-verify
shellcheck 0; lint-servlet.bats 13/13; report-module.bats 25/25; full suite 0 not-ok; lint-guard-pins --strict 0; sweep 0. [ev: corpus build-n4-module/automation-audit]

---
**Status**: FOLDED. Kit lint surface complete for the per-module flow.
