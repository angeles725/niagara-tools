<!-- review-status: pending -->
# Retro: Campaign 8 — rc-scan.sh browser-resource lint (Campaign 8 PR6)

**Date:** 2026-09-05
**Change:** build-n4-module-campaign8 PR6 — `feat/c8-rc-scan`
**Scope:** `build-n4-module-kit/toolbelt/rc-scan.sh` (new); K19 routing to `BUILD-LOOP.md` + `skill/SKILL.md`
**Lead:** Campaign 8 PR6 executor
**RED tip:** `1e391d1` (RC1–RC9)

[ev: corpus B801] [ev: corpus B803]

---

## What shipped

| Check | Label | Verdict | Description |
|-------|-------|---------|-------------|
| `ord-literal` | `ord-literal` | FAIL | `station:|local:|slot:/` or handle `(^|[|"'`])h:[0-9a-fA-F]+` under `rc/` assets |
| `host-literal` | `host-literal` | FAIL | `http://` URI not starting with `www.w3.org/`, or bare IPv4 |
| `bare-catch` | `bare-catch` | WARN (`--strict` FAIL) | `.catch(() => {})` pattern swallowing fetch errors |
| `null-branch` | `null-branch` | WARN (`--strict` FAIL) | `? null :` ternary branch on a process field |
| D9b prune | — | — | Dot-directories (`.deploy-baseline`, `.git`) excluded from file walk |

File scope: `**/rc/**` `*.html *.js *.css`, excludes `rc/ext/**`, `*.min.js`, `srcTest/**`.

---

## TDD evidence

| Phase | Evidence |
|-------|----------|
| RED | Commit `1e391d1` on `feat/c8-rc-scan` (rebased from `qa/c8-rc-scan` `79ed7fd`): RC1–RC9 all failing because `rc-scan.sh` was absent. |
| GREEN | `rc-scan.sh` implemented; all 9 RC tests turn green. Full suite 227/227. |
| REFACTOR | Updated host-literal row to include matched URL in detail (enables mutation (c) to flip RC2's `www.w3.org` assertion). shellcheck exit 0. |

---

## Design deviations

### D1: ORD `h:` pattern — handle scheme, segment-start anchored (lead correction)

**D7 design** specifies pattern `h:`. Initial implementation used `h:/` (false assumption: history ORDs carry a slash). Lead review (ea19275) identified this as a false NEGATIVE for every Niagara handle literal.

**Root cause (corrected):** `h:` is the Niagara **HANDLE** ORD scheme (`h:<hex>`, e.g. `h:45ef7`), not a history path prefix. Handles are station-specific hex identifiers that change per JACE deployment — the most brittle ORD type. They appear after ORD segment separators (`station:|h:45ef7`) or as quoted standalone strings (`"h:45ef7"`). The `h:/` pattern misses ALL handle literals.

**CSS false-positive root cause and fix:** `h:` is a substring of CSS property names ending in `h` (`width:`, `depth:`). In any ORD context, `h:` is always preceded by a segment-boundary character: `|`, `"`, `'`, or a backtick. In CSS, `h:` is always preceded by a letter. The fix anchors the match: `(^|[|"'\`])h:[0-9a-fA-F]+`. `width:100px` has `h:` preceded by `t` — no match. `"h:45ef7"` has `h:` preceded by `"` — FAIL.

**Awk pattern (corrected):**
```
/(station:|local:|slot:\/)|(^|[|"'\`])h:[0-9a-fA-F]+/
```

**RC10 RED -> GREEN proof (mktemp copies):**
- RED (old `h:/` script): `rc-scan.sh handle-fixture` -> exit 0, no output (handle `"h:45ef7"` missed — false negative confirmed)
- GREEN (segment-anchored pattern): `rc-scan.sh handle-fixture` -> `FAIL  rc-scan  rc/handle.js:2  ord-literal: hardcoded ORD literal`, exit 1; `style.css` (`.x{width:100px;depth:1}`) and `page.html` (`<div style="width:100px">`) produce no row (no false positive)
### D2: null-branch WARNs at lines 852–853, not :863 as predicted

**Design predicted** (tasks.md 6.8): `WARN :863 (null branch)`.

**Real file (local checkout 4f5f1c7):** Line 863 reads `const nextDef = (p.nextRemainMs == null) ? null` — the ternary true-branch `? null` continues onto line 864 with the `:` (multi-line expression). The pattern `\?\s*null\s*:` requires `:` on the same line, so line 863 does NOT match.

**Actual smoke hits:**
- Line 852: `const elap = (p.dElapsedMs == null) ? null : p.dElapsedMs + dt;` → WARN ✓
- Line 853: `const rem  = (p.dRemainMs  == null) ? null : Math.max(0, p.dRemainMs - dt);` → WARN ✓

Design predicted one null-branch hit; real smoke produces two (both legitimate display branches). No rule change required; the multi-line form at :863 is NOT a false negative — it is a separate, different pattern shape that the current single-line rule intentionally does not cover. Deviation recorded here.

### D3: ci.yml not present (noted, per PR1 convention)

**tasks.md 6.5** references `ci.yml`. There is no `ci.yml` in this repo. Test coverage is provided by bats (RC1–RC9). Same handling as PR1 task 1.6. [NOTE already in tasks.md.]

---

## Named mutations

These prove each sub-check has independent bite (run on mktemp copies):

| Mutation | Target | Observed output | Effect |
|----------|--------|-----------------|--------|
| (a) Remove ORD rule | `ord` fixture | (no output) | exit 0 — RC1 flips from exit 1 to exit 0 |
| (b) Remove bare-catch rule | `warns` fixture | only null-branch WARN | no `bare-catch` WARN row — RC3 flips |
| (c) Remove W3C namespace exemption | `host` fixture | 3 FAIL rows incl. `www.w3.org/2000/svg` and `www.w3.org/1999/xlink` | RC2 assertion `[[ output != *www.w3.org* ]]` FAILS |
| (d) Remove dot-dir prune | `dotdir` fixture | `FAIL  rc-scan  .deploy-baseline/rc/config.js:1  host-literal: http://127.0.0.1/obix/config/Old/` | RC9 flips from exit 0 to exit 1 |

---

## Real smoke results

### DashboardPan-ux local checkout (4f5f1c7)

Invocation: `rc-scan.sh /home/cristian/modulos_niagara_n4/Cliente/Leon-Guanjuato/Dashboard/DashboardPan/DashboardPan-ux`

```
FAIL  rc-scan  src/rc/index.html:701  host-literal: http://127.0.0.1/obix/config/Nave_Panccadia/Puntos/
WARN  rc-scan  src/rc/index.html:852  null-branch: ? null : branch may blank UI on null field
WARN  rc-scan  src/rc/index.html:853  null-branch: ? null : branch may blank UI on null field
WARN  rc-scan  src/rc/index.html:1298  bare-catch: bare .catch(()=>{}) swallows errors
exit: 1
```

- **FAIL at :701**: hardcoded `http://127.0.0.1/obix/config/Nave_Panccadia/Puntos/` — design predicted, confirmed.
- **WARN at :852 and :853**: two single-line `? null :` display branches — design predicted `:863` but that line is a multi-line expression (`:` on next line); :852 and :853 are the actual matches. Lines moved from design prediction; noted in D2 above.
- **WARN at :1298**: bare `.catch(() => {})` — design predicted, confirmed.
- No ORD literal FAILs: real `index.html` has no `station:` or `slot:/` or handle `h:<hex>` literals.
- W3C namespace URIs at lines 542, 567, 791, 1443 correctly NOT flagged.

Merged-tree smoke (`git show origin/main:<path>`) not run — `git` is VCS-only tooling not available in the `rc/` scan path, and the local checkout is the live artifact used for commissioning. Not run; noted here.

---

## Guard results

```
bats tests/*.bats               : 227/227 passed (RC1-RC9 added = +9 total)
shellcheck toolbelt/rc-scan.sh  : exit 0
sweep-build-state.sh            : exit 0
sweep-fold-audit.sh --strict    : exit 0 (56 folded, 56 cited, 0 uncited; campaign8-rc-scan pending)
tests/kit-links.bats            : 6/6 passed (L5: rc-scan.sh now in BUILD-LOOP.md and skill/SKILL.md)
```

---

## Lessons

1. `h:` in Niagara is the HANDLE ORD scheme (`h:<hex>`, e.g. `h:45ef7`), never `h:/path` — using `h:/` is a false negative for every handle literal. The correct fix anchors `h:` at ORD segment boundaries (`|`, `"`, `'`, backtick) with a hex payload, since CSS property names ending in `h` (e.g. `width:`) are always preceded by a letter, not a segment-boundary character.
2. A multi-line ternary (`? null` with `:` on the next line) does not match a single-line `\?\s*null\s*:` pattern; this is a separate detection shape requiring either a multi-line awk accumulation or a relaxed pattern with known false-positive risk.
3. Including the matched URL in the host-literal row detail (not just a generic label) makes the W3C-exemption named mutation observable via substring assertions, without requiring the test to parse row structure.
4. Dot-directory pruning via `-type d -name '.*' -prune` must be the first branch in the `find` expression; placing it after path-based conditions allows `find` to descend into dot-dirs before the prune fires.
5. Writing the awk program to a temp file with a quoted heredoc (`<< 'AWKEOF'`) eliminates all shell-quoting conflicts with single quotes inside awk regex character classes.
