<!-- review-status: pending -->
# 2026-09-26 · CompPan · comppan-auto-lock-indicator

**Session**: `Cliente/panccadia-leon` branch `feat/comppan-auto-lock-indicator` (from b8250f9): latched "auto-locked (no start)" indicator, CompPan 2.6.0 + DashboardPan 2.8.0; RDD `review-2c01aa3b04752f9e` approved (1 lens), handed off to the Downloads origin.
**Delta count**: 7

## What happened
An operator asked whether the T3 "auto-off after a sustained proof-of-run fault"
leaves the compressor marked as locked out for not starting. It did not. `fault[k]`
requires `cmd[k]`, so the alarm clears on the scan after the adapter writes
`condenserNMode = MODE_OFF`. The only trace left is one `CpLog.info` line, and the
dashboard shows an automatic protective lockout exactly like a manual "Apagar". The
2026-09-24 T3 design reused the manual OFF path to avoid a new state (user decision
T2), but nobody asked what the operator sees AFTER the automatic action. This
session added a persisted, latched `condenserNAutoLocked` (cleared only when the
operator moves the mode off OFF) and a "Bloqueado: no arrancó" badge. The native
reliability review approved it with 4 non-blocking findings. Two of them name
repeatable kit gaps: a negative-criterion test that is vacuous with the feature
disabled, and a new link-in indicator whose unlinked default is indistinguishable
from "false".

## Evidence
- Self-clearing alarm: `CompressorControl.java:294` (`fault[k] = measured && cmd[k] && ...`) and the T3 adapter write `BCompressorControl.java:2687-2697`. `[ev: ae98e15]`
- Latch + restart seed: `9c2bd0a` (`autoLocked[]` in the pure model, `seedAutoLocked()` mirroring `seedHours()` before the first `execute()`). `[ev: 9c2bd0a]`
- Review finding R3-w35d-vacuous-manual-apagar (WARNING, deterministic): W35d runs with the default cfg (`autoOffOnProofFault=false`), so it passes whatever the latch does. `[ev: odd/tasks/comppan-auto-lock-indicator.md § Review]`
- Review finding R3-unlinked-autolocked-reads-false (WARNING): new `compNAutoLocked` link-in defaults to `BStatusBoolean(false)` with ok status, so a missing station link shows "not locked". `[ev: 2456d41]`
- Review finding R3-adapter-seed-mirror-unproved (SUGGESTION): nothing proves that the seed runs before the first mirror write. `[ev: 9c2bd0a]`
- Writer hand-back carried no elapsed time. The 13.7 min figure came from the harness task notification, not from the report. `[ev: retro 2026-09-26-change-tier-time-budgets.md § Measured run]`
- Handoff = push the branch to the Downloads origin + switch that working tree + re-check versions and one UI string. This was done by hand and is recorded nowhere. `[ev: 40a3414]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **Doctrine: every AUTOMATIC protective action must leave a latched, operator-visible trace that is distinct from the equivalent manual action, and that only the operator clears.** A self-clearing alarm plus an automatic mode write erases its own cause. Extends 2026-09-02 comppan-fase1 lesson 5 ("a latched fault needs an operator-owned exit") from the fault to the ACTION. The design checklist for any auto-lockout, auto-trip, or auto-mode-change must name the latch slot, its clear condition, and its HMI rendering, or explicitly waive them. | `METHODOLOGY.md` § `Domain correctness` + `skill/SKILL.md` § `Decision Gates — pick the module type` (design checklist item, see 2026-09-25 Δ7) | `[ev: ae98e15]`; `[ev: 9c2bd0a]` |
| Δ2 | **A test of a negative acceptance criterion ("X never sets Y") must run with the feature ENABLED, and must include the near-miss case.** Here, the operator writes OFF while the fault is accumulating but before the delay expires. A negative test under a disabled flag is vacuous. Apply the "mutation must flip a fixture" discipline: drop the guard, and the test must go red. | `METHODOLOGY.md` § `Conformance rules — lintable vs advisory` | `[ev: odd/tasks/comppan-auto-lock-indicator.md § Review R3-w35d]` |
| Δ3 | **A new link-in indicator slot must be fail-visible when unlinked.** Either default its status to `{null}`/stale so the reader's existing "Sin datos" path renders, or add an explicit "linked?" probe that the dashboard shows. An ok/false default makes a missing commissioning link look like a healthy "no". Pair this with the 2026-09-25 Δ4 read-only oBIX audit: an unlinked indicator target is a commissioning FAIL row, not a WARN. | `METHODOLOGY.md` § `rt (components)` + `BUILD-LOOP.md` § `6.b Commissioning-verify requirement` | `[ev: 2456d41]`; `[ev: retro 2026-09-25-panccadia-commissioning-lessons.md Δ4]` |
| Δ4 | **Name the "seed from persisted slot before the first mirror" pattern once, with an ordering test.** `seedHours()` (T4) and `seedAutoLocked()` are the same pattern: the pure model is recreated on every start, the adapter re-seeds it from the persisted slots, a one-shot guard applies, and only then is `execute()` allowed to mirror state back. Document it as a named rt pattern with a source-structural ordering test (seed call precedes the first mirror in `started()`/`atSteadyState()`), because the NRE blocks a behavioral adapter test (baja-offline boundary). | `METHODOLOGY.md` § `Schema / upgrade safety` + `BUILD-LOOP.md` § `4.b Test layer` | `[ev: 9c2bd0a]`; `[ev: retro 2026-09-21-live-diagnosis-hardening-deltas.md PER8]` |
| Δ5 | **A feature doc for an opt-in (default-off) feature must carry, in its deploy checklist, the exact enabling slot and its current live value.** `autoOffOnProofFault` defaults to false, so the new indicator is inert until someone enables it. That fact reached the operator only because the parent happened to mention it at handoff. | `BUILD-LOOP.md` § `6. Deploy (station) — operator` | `[ev: odd/tasks/comppan-auto-lock-indicator.md]` |
| Δ6 | **The writer hand-back must report elapsed time per phase** (discovery/mapping, RED, GREEN, build, docs). The tier budgets of `2026-09-26-change-tier-time-budgets` Δ1 can only be tuned from measured phases. This run's total came from the harness, and the split (how much was re-mapping the rt -> dashboard path) is unknown. | `ORCHESTRATION.md` § `8. Per-run retro/ticket loop` | `[ev: retro 2026-09-26-change-tier-time-budgets.md § Measured run]` |
| Δ7 | **Codify the two-checkout handoff as a checklist step.** Build in the ext4 clone, `git push origin <branch>` to the Downloads (Windows) origin, `git switch <branch>` in that working tree (it must be clean apart from known untracked archives), then re-verify `defaultModuleVersion` in each touched group and one new UI string. This makes "what the deploy laptop receives" an observed fact, not an assumption. | `BUILD-LOOP.md` § `6. Deploy (station) — operator` | `[ev: 40a3414]` |

## Lessons
- Reusing a manual path for an automatic action saves a state but costs the operator the "why"; the latch is the price of reuse.
- A negative test is only as strong as the configuration it runs under; test the enabled, near-miss case.
- A link-in indicator that defaults to "healthy" hides a missing link; make absence visible.
- The same persistence pattern twice (hours, latch) is a named pattern waiting to be written down.
- Handoff between two checkouts is a deploy step and deserves the same verification as a build.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-26-comppan-auto-lock-indicator.md | CompPan | 2026-09-26 | pending | 7 |`
