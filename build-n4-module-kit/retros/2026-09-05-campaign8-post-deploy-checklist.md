<!-- review-status: folded -->
# Campaign 8 PR13 — post-deploy verification checklist → BUILD-LOOP.md §6.a

**Date:** 2026-09-05
**Module:** kit (self-section)
**PR:** docs/c8-post-deploy-checklist
**Base commit:** 3f666a0

---

## What happened

Campaign 8 PR13 added `### 6.a Post-deploy verification` as an ordered checklist
subsection at the end of `BUILD-LOOP.md §6`. The four steps name the exact scripts
with their current flags: `station-snapshot.sh → triage-console.sh → bog-audit.sh →
report-module.sh`. A fifth bullet states the proxy-link safety (CHECK11) requirement
before operator hand-off. Each step carries a corpus evidence token (`[ev: corpus
B811/B800/B795/B810]` and `[ev: retro campaign7-report-module]`).

`tests/kit-links.bats` gained a new `L7` pin (line 81) that hard-codes the five §6.a
step scripts and asserts each appears in `BUILD-LOOP.md`.

This is a doc-only PR: zero new bats tests other than the L7 kit-links pin (CD2).

---

## Evidence

### Grep-before-fold counts (re-run against base 3f666a0)

| Pattern | Command | Count |
|---------|---------|-------|
| `post-deploy\|snapshot.*triage-console.*bog-audit` | `grep -cE '...' build-n4-module-kit/BUILD-LOOP.md` | 0 |

Pattern had 0 hits before the fold, confirming §6.a did not exist.

### Anchors with line numbers (after fold)

| File | Line | Anchor |
|------|------|--------|
| `build-n4-module-kit/BUILD-LOOP.md` | 65 | `### 6.a Post-deploy verification` subsection header |
| `build-n4-module-kit/BUILD-LOOP.md` | 69 | step 1 — `station-snapshot.sh <station-dir> <out-dir>` `[ev: corpus B811]` |
| `build-n4-module-kit/BUILD-LOOP.md` | 70 | step 2 — `triage-console.sh --package <com.vendor>` `[ev: corpus B800]` |
| `build-n4-module-kit/BUILD-LOOP.md` | 71 | step 3 — `bog-audit.sh <config.bog\|file.xml> --module <MOD>` `[ev: corpus B795]` |
| `build-n4-module-kit/BUILD-LOOP.md` | 72 | step 4 — `report-module.sh <module-root> --console-dir <console-dir>` `[ev: retro campaign7-report-module]` |
| `build-n4-module-kit/BUILD-LOOP.md` | 73 | CHECK11 proxy-link safety hand-off gate `[ev: corpus B810]` |
| `tests/kit-links.bats` | 81 | `@test "L7: every §6.a post-deploy step script is named in BUILD-LOOP.md"` |

### Guard results (task 13.4)

| Guard | Command | Result |
|-------|---------|--------|
| kit-links.bats | `bats tests/kit-links.bats` | 7/7 OK |
| sweep-fold-audit | `sweep-fold-audit.sh --strict retros/INDEX.md build-n4-module-kit` | exit 0, 56 folded, 56 cited, 0 uncited |

---

## Proposed kit deltas

| # | Delta | Status |
|---|-------|--------|
| Δ1 | `BUILD-LOOP.md §6.a` — ordered post-deploy verification subsection (steps 1–4 + CHECK11 gate) | applied (13.2) |
| Δ2 | `tests/kit-links.bats L7` — hard-coded pin for the five §6.a step scripts | applied (13.3) |

---

## Lessons

1. Doc-only PRs require sweep-fold-audit --strict to exit 0 before commit; a new pending retro row does not trigger an uncited-row failure because the audit targets only folded rows.
2. The L5 routing guard (every toolbelt script named in BUILD-LOOP.md or SKILL.md) was already satisfied for all five §6.a scripts; L7 adds a focused post-deploy pin that survives future §6 restructuring.
3. Evidence tokens on individual checklist steps (`[ev: corpus B<n>]`) anchor the source corpus to operator-visible procedure lines, making corpus-to-procedure traceability machine-checkable via grep.
