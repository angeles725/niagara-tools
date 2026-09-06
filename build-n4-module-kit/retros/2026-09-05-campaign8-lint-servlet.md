<!-- review-status: pending -->
# Retro: Campaign 8 — lint-servlet.sh BWebServlet security lint (Campaign 8 PR12, B813)

**Date:** 2026-09-05
**Change:** build-n4-module-campaign8 PR12 — `feat/c8-lint-servlet`
**Scope:** `build-n4-module-kit/toolbelt/lint-servlet.sh` (new); `CsrfXrwOnly.java` fixture; LSV4 pin; K19 routing
**Lead:** Campaign 8 PR12 executor
**RED tip:** `6d4afaf` (LSV1, LSV1c, LSV2, LSV3, LSV5, LSV6, LSV-usage, LSV-prune)

---

## What happened

`lint-servlet.sh <src-dir> [--strict]` scans `*.java` files under `<src-dir>` (dot-dirs
pruned, D9b) that extend `BWebServlet` or declare `doGet`/`doPost`/`service` handlers.
Six checks implemented. LSV4 pin and `CsrfXrwOnly.java` fixture added (not in RED tip).

Method-body extraction via brace-balance counting (same-method scope) isolates each handler;
comment stripping (`//...` to end-of-line) prevents comment text from poisoning pattern matches.

K19 routing: one appended entry in `BUILD-LOOP.md` §5 (pre-gate block, next to `rc-scan`) and
one appended entry in `skill/SKILL.md` toolbelt list. `[ev: retro campaign8-lint-servlet]` in
both routing lines and in the script header.

Final bats: **253/253** green (9 new LSV pins). shellcheck exit 0. All sweep guards exit 0.
kit-links.bats 6/6 (L5 green — `lint-servlet.sh` now named in both routing docs).

---

## Evidence

### TDD evidence

| Phase | Evidence |
|-------|----------|
| RED | Commit `6d4afaf` on `qa/c8-lint-servlet`: LSV1–LSV6 + LSV-usage + LSV-prune all failing because `lint-servlet.sh` was absent (exit 127). |
| GREEN | `lint-servlet.sh` implemented; all 8 original LSV tests turn green; LSV4 pin added and green (9/9). Full suite 253/253. |
| REFACTOR | Comment-stripping fix: two iterations — first for method bodies, second for file-level flags. `\btry\b` → `/(^|[[:space:]])try[[:space:]]*\{/` (gawk `\b` in `~` is not a word boundary). |

### Named mutations (all on mktemp copies; originals untouched)

| Mutation | Change applied | Test flipped | Observed output |
|----------|---------------|--------------|-----------------|
| (a) remove auth rule | `has_write && !has_auth` → `0 && 0` in awk | LSV1 | No FAIL auth row; exit 0 (was 1) |
| (b) remove csrf-xrw-only output | Comment out `printf "%s  csrf-xrw-only...` in awk | LSV4 | No csrf-xrw-only WARN; exit 0, output empty |
| (c) remove parseDouble try check | `if (!in_try)` → `if (0)` | LSV2 | No FAIL input-400; exit 0 (was 1) |
| (d) remove dot-dir prune | `-name '.*' -prune` → `-name 'DISABLED' -prune` | LSV-prune | `FAIL  auth  .deploy-baseline/Stale.java:4  handler writes without getRemoteUser()/getUser() auth gate`; exit 1 |

### Real smokes (read-only; verbatim output)

**Smoke 1: DashboardPan-ux pre-PR#7 (local checkout 4f5f1c7)**

```
WARN  cache-nofinger  com/angeles/DashboardPan/ux/BDashboardServlet.java:428  Cache-Control max-age>0 on rc asset without fingerprint
exit: 0
```

**Smoke 2: DashboardPan-ux origin/main (post-PR#7, `parseFiniteDouble → 400` guard)**

```
WARN  cache-nofinger  BDashboardServlet.java:445  Cache-Control max-age>0 on rc asset without fingerprint
exit: 0
```

**Smoke 3: chihuahua-ux (BChiServlet)**

```
WARN  cache-nofinger  com/angeles/chihuahua/ux/BChiServlet.java:1240  Cache-Control max-age>0 on rc asset without fingerprint
WARN  csrf-xrw-only  com/angeles/chihuahua/ux/BChiServlet.java:1  X-Requested-With guard without CsrfUtil/csrfToken (use Niagara CSRF token) [ev: corpus B813]
exit: 0
```

---

## Proposed kit deltas

| Delta | File | Status |
|-------|------|--------|
| New script | `build-n4-module-kit/toolbelt/lint-servlet.sh` | SHIPPED |
| New fixture | `tests/fixtures/lint-servlet/CsrfXrwOnly.java` | SHIPPED |
| LSV4 pin | `tests/lint-servlet.bats` | SHIPPED |
| K19 routing (BUILD-LOOP.md) | §5 pre-gate block, after rc-scan entry | SHIPPED |
| K19 routing (skill/SKILL.md) | Toolbelt list, after station-snapshot entry | SHIPPED |

---

## Design deviations

### D1: tool is `lint-servlet.sh <src>`, NOT `rc-scan.sh --servlet` (K13: RED wins)

**Tasks.md (12.2/12.5)** said "extend `rc-scan.sh` with `--servlet` mode". **RED (`6d4afaf`)**
calls `lint-servlet.sh "$ONE"` — a standalone script. K13 makes the RED authoritative.
The tool is `lint-servlet.sh`; `rc-scan.sh` is not extended.

### D2: R12.2 step-up re-auth NOT implemented — deferred to C9 S12 (B803)

**Spec R12.2** requires FAIL for "critical write without short-TTL step-up token". B803
confirms Niagara ships no step-up re-auth primitive as of N4.14; every real servlet would
FAIL statically (all write endpoints are reachable without a fresh step-up token). This is a
runtime policy concern, not a statically decidable pattern. Implementing it as FAIL would
produce constant noise with zero actionable fixes. **Decision: deferred to Campaign 9 S12**
when/if a step-up primitive ships or a Niagara-specific proxy pattern is established.

### D3: R12.3 CSRF → WARN not FAIL (B813, K3)

**Spec R12.3** says `X-Requested-With`-only MUST FAIL. **B813** shows the reference servlets
(DashboardPan and chihuahua) use X-Requested-With guards today. Implementing as FAIL would
immediately block real shipped code. K3 (no hard stop over live modules) takes priority;
implemented as `csrf-xrw-only WARN`. Record in tasks.md.

### D4: design expects input-400 FAIL on DashboardPan pre-PR#7 — not observed

**Design** predicted `input-400 FAIL` on BDashboardServlet at 4f5f1c7. Actual: both
`Double.parseDouble` calls in BDashboardServlet are inside `try { ... }` blocks on the same
line (lines 371 and 389). The 5-line lookback correctly identifies the enclosing try and does
NOT emit FAIL. The `parseDouble(String s)` private helper returns 0.0 on bad input (not 400),
but detecting "try/catch that doesn't return 400" requires flow analysis beyond static pattern
matching — left for a future rule. The smoke is honestly reported as WARN cache-nofinger only.

---

## Lessons

1. Java single-line comments contain the exact keywords the security patterns look for; stripping `//.*` from body text before regex matching is mandatory, not optional.
2. `\btry\b` in gawk's `~` operator does not behave as a word boundary — use `/(^|[[:space:]])try[[:space:]]*\{/` to match try-block openings reliably.
3. The 5-line lookback heuristic for try-block detection misses the case where a try/catch catches the exception but returns a safe default instead of 400 — this is a design limitation that requires per-method flow analysis to address.
4. `csrf-xrw-only` WARN (not FAIL) is the correct severity when the reference servlets themselves use the pattern; a tool that FAILs its own corpus has a correctness problem, not a security win.
5. Comment stripping must be applied consistently to both method-body text and file-level flag scans; an inconsistency between the two caused LSV4 to pass when the fixture's own comment mentioned the forbidden token.
