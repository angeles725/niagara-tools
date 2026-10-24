<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-24 · kit · go-lint-hot-write

**Session**: niagara-research quality round 10 (Ayudante2 lint-engine owner), branch `r10/hot-write-go` from origin/main 53bfe0c
**Delta count**: 3

## What happened
The sibling lints `toolbelt/lint-persist-hot-write.sh` (PER8) and `lint-changed-hot-write.sh` (RUN8), two awk
programs that each re-implemented the same pieces, are now thin `n4lint` wrappers over two rule packages
(`rules/persisthotwrite`, `rules/changedhotwrite`) that share one helper package, `rules/hotwrite`: the line view,
the changed() callback and its body, the zero-arg callees, the non-transient slot-o-matic properties and the setter
call. Two rule packages, not one, because each keeps its own golden testdata. No new engine field (First mode,
the awk_lint_run walk contract, the K20 row). Byte-identical first, then four truth fixes, each RED-first in one
green commit, all landing in the shared helper so both lints get them:
T1 code view (comments and literal contents out: a Javadoc `{@code execute()}` was execute()'s "declaration", a
"}" in a string ended changed(), a commented-out changed() was the callback, quoted setters were writes); T2 method
bodies from javasrc.Methods (the first `name()` not followed by ";" was the declaration: `if (execute())`,
`pending(tick())`, `this.device() + "..."`); T3 the persist setter at an identifier boundary with blanks before "("
(hotwrite.SetterCall, what changed-hot-write did inline, #151); T4 every property declaration of a line with its own
flags, across lines.

## Evidence
- Goldens: persist 26 cases (9 bats runs, 17 edge trees), changed 27 (10 bats runs, 17 edge trees); RED 12/26 and 12/27 plus TestUnreadable against stub Checks `[ev: go/internal/lint/rules/persisthotwrite/persisthotwrite_test.go TestGolden]`
- Byte identity (stdout, stderr, exit), default and `--strict`: persist Cliente 0 rows, decompiled 103, tests/fixtures 0; changed Cliente 0, decompiled 0, fixtures 0; wrappers identical too. Timings: persist 0.60 s → 0.03 s (Cliente), 8.10 s → 0.30 s (decompiled); changed 1.24 s → 0.06 s, 17.75 s → 0.24 s
- T1: persist Cliente 0 → 14 (all BCompressorControl execute() setters reached from changed() with no cadence guard; the bash read a Javadoc `{@code execute()}` as the declaration): live panccadia-leon CompPan 10 rows (condenser1-3Hours :4158-4160, condenser1-3AutoLocked :4166-4168, faultReset :4054, rotationLeadIndex :4215, rotationLastChangeMs :4216, stagingConfigRevision :4217), stale Leon-Guanjuato CompPan 4 rows; changed Cliente 0 → 1 (stale Leon-Guanjuato BCompressorControl.java:1808, 33 branches, no rate guard; the live changed() has its Clock.millis() gate); decompiled unchanged `[ev: go/internal/lint/rules/persisthotwrite/testdata/truth-code-view-1.txt]`
- T2: persist decompiled 103 → 101 (bacnet-rt BBacnetObject.java:282 and template-rt BTemplateConfig.java:237 were reached through a callee whose "declaration" was a call inside an expression); Cliente unchanged `[ev: go/internal/lint/rules/changedhotwrite/testdata/truth-method-bodies-1.txt]`
- T5 (equal-value skip, after Ayudante1 checked the 10 live rows against the framework): Baja's ComplexSlotMap skips an equal simple value (niagara-research organized/baja/baja/vineflower/com/tridium/sys/schema/ComplexSlotMap.java:1239 setDouble `!BDouble.equals`, :860 setBoolean `oldValue != newValue`, :637 set(Property, BValue) `!oldValue.isSimple() || !oldValue.equals`); hotwrite.Hot now reports a write only when it can dirty the station every cycle. RED truth-equal-value-skip-1 of both rules (modelled on the live code: 7 false shapes, 3 accumulators, a getter-plus-delta, a clock read, a non-simple per-cycle write), bats PHW11 / CHW9. persist Cliente 14 → 6 (live CompPan 10 → 3: condenser1-3Hours stay; AutoLocked edge mirrors, faultReset under its own getter, rotation/revision event mirrors go), decompiled 101 → 53; changed unchanged `[ev: go/internal/lint/rules/persisthotwrite/testdata/truth-equal-value-skip-1.txt]`
- T6 (cadence guard): any `Clock.millis(` on the write's line or the 5 above counted as a guard, so a write of the clock itself and a write after a bare `long now = Clock.millis();` were suppressed (a fail-open); a clock read now guards only from a gating condition. RED truth-clock-guard-1 (port: no row; want lines 8 and 10, lines 11 and 14 guarded), bats PHW12; Cliente 6 → 6, decompiled 53 → 53 `[ev: go/internal/lint/rules/persisthotwrite/testdata/truth-clock-guard-1.txt]`
- What the rules can prove statically: a slot's type from its newProperty default, an enclosing if that tests the slot's getter or a change flag, and a value that is fresh at the call (clock read, getter arithmetic) or comes from a field the tree accumulates by a non-literal delta. What they cannot: that a mirrored model field changes on every call (a field moved only by `++`/`+= 1` or plain assignment counts as event-driven), or what a helper method returns; a non-simple value is always reported because the framework never skips it
- T3, T4: no real-tree row changed `[ev: go/internal/lint/rules/persisthotwrite/testdata/truth-setter-name-1.txt]` `[ev: go/internal/lint/rules/changedhotwrite/testdata/truth-property-flags-1.txt]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | The 3 live panccadia-leon CompPan rows (condenser1-3Hours, accumulated every cycle and written every execute() reached from changed()) are the PER8 shape itself; give them a cadence guard / periodic checkpoint. The other 7 of the first 10 rows were false positives of Baja's equal-value skip (T5) | `Cliente/panccadia-leon/.../BCompressorControl.java` (client finding) | `[ev: go/internal/lint/rules/persisthotwrite/testdata/truth-code-view-1.txt]` |
| Δ2 | Both lints still read only the raw slot-o-matic `Property x = newProperty(` form, not `@NiagaraProperty(flags = ...)`, and only the first changed(Property ...) of a file | `build-n4-module-kit/toolbelt/lint-persist-hot-write.sh` | `[ev: go/internal/lint/rules/hotwrite/hotwrite.go NonTransient]` |
| Δ3 | javasrc.Methods skips one-liner get/set/is accessors (mb_parse I4), so a callee named isX()/getX() whose one-line body writes a slot is not scanned | `go/internal/javasrc/javasrc.go` | `[ev: go/internal/lint/rules/hotwrite/hotwrite.go ZeroArg]` |

## Lessons
- Sibling lints that each re-implement the same parsing share their bugs; one helper package fixed four defects in both at once.
- Truth fixes that look synthetic can open real rows: the Javadoc `{@code execute()}` misread hid a whole family of live PER8 writes from the bash lint.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-fold-lints-a`) — Δ1 (client finding), Δ2 and Δ3 parked as an owed `open_issue:` in its `ledger.d/` fragment, `[deferred:]` in `BUILD-LOOP.md` § 5 kit tooling doctrine (needs code).
