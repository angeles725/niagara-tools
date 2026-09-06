<!-- review-status: pending -->
# Campaign 8 PR10 — bog-audit retro

Session: 2026-09-05, branch feat/c8-bog-audit, kit PR10.

---

## D1 — CHECK11 is FAIL, not WARN

tasks.md 10.2 and 10.8 initially described proxy-link-safety as WARN.
The RED bats file (BA9) asserts FAIL and exit 1.  K13 (RED is authoritative)
overrides tasks.md.  Design §D10 also specifies FAIL.

**Proposed kit delta:** When RED bats and tasks.md disagree, record the deviation
in the retro under D-number labels so the proposer can fold the correction
into tasks-authoring guidance.

---

## D2 — CHECK12 is WARN (advisory), not FAIL

tasks.md implied a possible FAIL for servlet-written-slot-is-link-target.
The RED bats file (BA11) and design §D10 both specify WARN (advisory).
The reasoning: servlet writes are ephemeral (link value wins on the next cycle),
so this is an informational notice, not a blocker.

---

## D3 — Direct-parent slot tracking (compound-type sub-slot false-positive)

Niagara compound slots (StatusNumeric, StatusBoolean, etc.) store their
sub-fields as XML children of a non-self-closing `<p>` element:

```xml
<p n='setpoint' f='sL' t='b:StatusNumeric'>
  <p n="value" v="3.0"/>
</p>
```

The parser's `nearest_comp()` helper found ColdRoom_1 as the owner, incorrectly
recording `value` as a direct slot and triggering false CHECK5 FAILs on the
real PANCCADIA bog.

Fix: slot recording was changed to require that the IMMEDIATE parent stack frame
is a `comp` frame (not `other`).  This eliminates sub-slots of compound
properties from the direct-slot dict while preserving correct tracking of real
direct slots.

**Proposed kit delta:** Document the direct-parent-only rule in D10 grammar note.
Any future bog-aware tool should track the stack frame type, not just the nearest
ancestor component.

---

## D4 — wsAnnotation platform slot exclusion

`wsAnnotation` is a Workbench-managed platform slot added at runtime; it is
never declared in module source.  The bog stores it with `f="r"` (READONLY)
in most components, but some CRP:ColdRoom instances stored it without `f=`.
Two fixes applied:
  1. The READONLY `f='r'` flag check already skips wsAnnotation with that flag.
  2. `wsAnnotation` (and `value`, `status`, `displayName`) were added to an
     explicit `_PLATFORM_SLOTS` exclusion set, so they are never checked
     regardless of flags.

---

## D5 — Java numeric literal suffixes in MIN facet regex

Source annotation: `BFacets.make(BFacets.MIN, BDouble.make(0d))`.
The trailing `d` (Java double literal suffix) caused the MIN value regex to
fail (`0d` was not matched by `(-?[0-9.]+)`).

Fix: regex updated to `(-?[0-9.]+)[dDfFlL]?` to accept all Java numeric
suffixes (d, f, l, L).

---

## Named mutations observed

| Mut | Action | Expected | Observed |
|-----|--------|----------|----------|
| (a) | differentialUp corrected to 2.0 | No CHECK3 | No CHECK3 |
| (b) | --strict flag on intervalExpired | CHECK2 FAIL | CHECK2 FAIL |
| (c) | bog-audit.sh removed from BUILD-LOOP.md | L5 FAIL | L5 FAIL (still in SKILL.md — need both) |
| (d) | bog-audit.sh removed from both | L5 FAIL | L5 FAIL confirmed |

---

## Real smoke results

PANCCADIA (`config.bog`, ColdRoomPan module):
- Without --source-dir: 17 CHECK11 FAIL (exactly), 0 CHECK9/10 FAIL.
- With --source-dir: 1 CHECK2 WARN (intervalExpired f='o' vs source f='h'), 0 CHECK5/7 FAIL.

HoneywellMX605132026 (`config.bog`, chihuahua module):
- Without --source-dir: exit 0, 109 chihuahua components inventoried.

---

retro: retros/2026-09-05-campaign8-bog-audit.md (5 deltas, review-status: pending)
