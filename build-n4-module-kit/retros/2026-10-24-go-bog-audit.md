<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 9 -->
# 2026-10-24 · kit · go-bog-audit

**Session**: niagara-research quality round 10, branch `r10/bog-audit-go` from origin/main 40c9786c
**Delta count**: 9

## What happened
`toolbelt/bog-audit.sh` (889 lines: bash argument loop + an embedded python3 line scanner, CHECK1-CHECK20) is now
`go/internal/bogaudit` + `go/cmd/bog-audit` and a thin wrapper that execs it via `go_tool_bin`; python3 is no longer
needed. It is its own command, not an n4lint rule: the operand is one bog file (not a source root), `--module` repeats,
rows are `CHECK<n>  <STATUS>  <path>  <detail>` with a trailing parse-time line, and exit 3 also means "no component of
the named modules" — the engine's row, walk and empty-root contracts do not fit without contortions. Shared code only:
the bog is read once by `xmlsrc.Open` (R8-D12; a zipped .bog through its file.xml member), which gained one field,
`Element.Empty`; the Java sources are read with `htmltext.Decode` (python's text read), `javasrc.ParenDepth` and, after
truth fix 3, `javasrc.Strip`. Byte-identical first, then six truth fixes RED-first, then a cross-check against
niagara-go `bog-nav` (internal/bog) that found three of them.

## Evidence
- Goldens recorded from the python script (stdout, stderr, exit; the parse-time line normalized): 23 bats runs
  (bog-audit.bats + commissioning-verify.bats through a recording wrapper), 14 kit bog fixtures and 31 edge trees
  (every argument shape, zip / bare xml / empty / non-zip / no file.xml, every CHECK's finding and clean shape, repr
  quoting, float formats, SlotPath `$20`/`$u00ef` names, empty-element and composite quirks, source walk order, dot
  directories, CRLF, multi-file classes). RED per commit: each CHECK group's cases failed before its file
  `[ev: go/internal/bogaudit/golden_test.go]`
- Byte identity old vs new on real stations: PANCCADIA config.bog (32 rows bog-alone; 982 with
  `--source-dir Cliente/panccadia-leon`; each module alone; `--strict`; zipped and bare file.xml) and the Harbor HM_BMS /
  HM_Central backups (`--module control`, `kitControl`, `baja`, four modules: 7-2423 rows). Timing: PANCCADIA 469 → 37 ms,
  with sources 570 → 110 ms, HM_BMS 7065 → 383 ms `[ev: toolbelt/bog-audit.sh]`
- Declared differences: python3 not required (BA7); a malformed file.xml is `bog-audit: <bog>:<line>:<col>: <msg>`
  exit 3 where the line scanner audited what its regex saw (BA7c, truth-declared-malformed); entity-decoded values and
  start tags split over lines (truth-declared-entities-multiline); a valueless hasDefrost no longer crashes with an
  AttributeError traceback (truth-declared-valueless-hasdefrost); a `.xml` path holding a zip is unzipped; an
  unreadable bare `.xml` is `cannot read <path>: <reason>`; non-ASCII decimal digits are not python digits
- Truth fixes, each a `testdata/truth-*.txt` case plus a bats pin with a `# Mutation:` line (rows PANCCADIA bog-alone /
  with sources / HM_Central control+kitControl):
  1. missing or unlistable `--source-dir` → exit 3 (BA13; the recorded case had 22 CHECK5/CHECK7 FAIL ghosts)
  2. CHECK4 reads `Flags.SUMMARY | Flags.TRANSIENT` and `Flags.SUMMARY|Flags.TRANSIENT` (BA14; 5 of the 70 TRANSIENT
     slots in panccadia-leon were invisible; rows 0 → 0, none persisted)
  3. source read on code: an annotation or `.set(` in a comment declares nothing (BA15; 982 → 982)
  4. `b:ConversionLink` is a link (BA16; PANCCADIA 19 links, HM_BMS 9, HM_Central 120 were invisible; HM_Central CHECK11
     FAIL 396 → 421, CHECK14 WARN 1540 → 1499)
  5. an empty-element `<p h=...  />` is a component (BA17; PANCCADIA 862 vs 662 components, HM_BMS 36132 vs 34799;
     real rows unchanged, 5 edge goldens became truth-empty-component-*)
  6. a stored composite property (`<p n='setpoint' t='b:StatusNumeric'>` + nested value) is a slot (BA18; PANCCADIA
     with sources CHECK6 WARN 949 → 843)
- Δ9 counts, python vs Go per CHECK on the real bogs: only CHECK6 (pan-src 949 → 843), CHECK11 (central 396 → 421)
  and CHECK14 (central 1540 → 1499) differ; every other CHECK/status count is equal
- bog-nav cross-check (`TestCrossCheckDump` vs `bog-nav --json tree|links|slot`, niagara-go 61def0f2): after fixes 4-6
  handles, paths, types and links agree exactly on all three bogs (PANCCADIA 862/444, HM_BMS 36132/844, HM_Central
  7732/2133) and slot names agree on PANCCADIA's 862 components except naming. Remaining disagreements, explained:
  (a) bog-nav prints names `$`-decoded (`object tags`, `ntpl:desc`), the audit keeps the escaped ord form
  (`object$20tags`) — presentation, both right; (b) bog-nav reads a composite slot's nested value (310 slots on
  PANCCADIA), the audit does not — see Δ1

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **CHECK8 is blind on real bogs.** Writable priority inputs are stored as composites (`<p n='in8' t='b:StatusBoolean'><p n="value" v="true"/><p n="status" v="40"/></p>`; 27 on PANCCADIA, 28 on HM_BMS); the check reads only empty-element `in1..in8` values, so it never fires. A fix reads the nested value and treats the null status bit as inactive; the status encoding (`v="40"` = 0x40 null?) needs a source citation first. | `go/internal/bogaudit` CHECK8 | `[ev: go/internal/bogaudit/bogchecks.go]` |
| Δ2 | **bog-nav `hoa` ignores the null status (niagara-go follow-up).** It reads the nested value of `in8` and reports OVERRIDE for `value=true status=40`, a null (inactive) input. | niagara-go internal/bognav cmdHoa | `[ev: go/internal/bogaudit/crosscheck_test.go]` |
| Δ3 | **The PANCCADIA smokes pin a bog that no longer exists.** BA-smoke-check11 / SL-smoke-panccadia / c8-close SC16 expect 17 CHECK11 FAIL; the current station bog gives 2 (python and Go alike). Re-pin to a frozen fixture copy, never to the live station file. | `tests/bog-audit.bats`, `tests/c8-close.bats` | `[ev: tests/bog-audit.bats]` |
| Δ4 | **commissioning-verify's SKIP text names python3.** `bog-audit env fault (exit 3) — check python3 availability and bog path` is stale; exit 3 is now usage / bog / module. | `toolbelt/commissioning-verify.sh` | `[ev: toolbelt/commissioning-verify.sh]` |
| Δ5 | **CHECK6 is mostly noise.** 843 WARN rows on PANCCADIA are declared slots at their default (never saved). Restrict it to non-default-able slots or make it opt-in. | `toolbelt/bog-audit.sh` header CHECK6 | `[ev: go/internal/bogaudit/srcchecks.go]` |
| Δ6 | **Duplicate class files merge silently.** `--source-dir` over a client folder merges every `B<Class>.java` copy (live, stale, salvage checkouts) into one class, last write wins; run it on one checkout's module sources. | `toolbelt/bog-audit.sh` header (source scope) | `[ev: go/internal/bogaudit/source.go]` |
| Δ7 | **Source checks key on the file name.** A class is its `.java` file name, so test sources (`*Test.java`) are scanned too; harmless today, but `.set("X",` in a test feeds CHECK12. Skip `srcTest/`. | `go/internal/bogaudit/source.go` | `[ev: go/internal/bogaudit/source.go]` |
| Δ8 | **A brief's "malformed bog is an xmlsrc error" was a behaviour change, not a port detail.** The python scanner never refused a bog; the positioned error is listed as a declared difference with its own case. | go/README.md § XML rules | `[ev: go/internal/bogaudit/testdata/truth-declared-malformed.txt]` |
| Δ9 | **Cross-check a port against a sibling parser, not only against its original.** Byte identity proved the port faithful to the python and blind to the same three things; the bog-nav diff found them in one pass. | go/README.md § Adding a lint rule step 5 | `[ev: go/internal/bogaudit/crosscheck_test.go]` |

## Lessons
- A recorder that names cases from `$BATS_TEST_NAME` must make names unique: three CHECK20 tests collapsed into one
  file until re-recorded.
- An env-gated dump test (`BOGAUDIT_DUMP`) is the cheapest honest cross-check: it reads the graph the checks read.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-24-go-bog-audit.md | kit | 2026-10-24 | pending | 9 |`
