<!-- review-status: pending -->
<!-- kit-retro -->
# Campaign 2 close: the promotion process's own meta-lessons

Date: 2026-09-04 · Module: kit · SDD: build-n4-module-continuity Campaign 2 (PRs #17-#21, v0.8.2→0.9.3)

Campaign 2 promoted the 42 mined [CERT] lessons (logic L3-L22, UX U1-U10, build/deploy/schema)
into the kit core across 5 serialized doc PRs + one gate-hardening PR (#18, the §7 third exit). Doing
it surfaced three process lessons a FUTURE promotion campaign should inherit — captured here so they are
not re-learned the hard way.

## Meta-lessons (proven across PR-A/A2/B/C)

1. **The fold-vs-revert (split-retro) rule** — a retro whose lessons span multiple PRs must not be marked
   `folded` until ALL its lessons are in core. When a remaining lesson has a LATER PR that will carry it,
   revert its INDEX row to `pending`; when it has NO later home, fold it now (in the current PR) — reverting
   would ORPHAN the lesson. This rule now lives in the `kit` self-section of `BUILD-STATE.md`; it resolved
   fase3 #2, process-timers #2, and editing-base64 #2.

2. **The doc-vs-script folded-completeness rule** — a lesson whose whole content is a rule/checklist item is
   FULLY folded by documenting it (row → `folded`). A lesson that asks for a SCRIPT or GATE behavior change
   is only PARTLY folded by documenting its rule: the implementation is still owed, so its row STAYS
   `pending` and the impl is logged as a `kit` self-section open_issue (a tracked future MINOR feature PR).
   Do not mark a retro `folded` just because its prose landed. This kept 6 impls honestly tracked
   (B4/B6/B7/B8/B10 + soft-start's build.sh clean-lock message) instead of silently dropped.

3. **Conservative adversarial fidelity grading catches what a green suite cannot** — the bats gate stayed
   84/0 through every promotion PR, yet QA's per-lesson fidelity grade (each fold diffed against its source
   retro) caught FIVE folded over-claims (fase3 #2, process-timers #2, editing-base64 #2, soft-start #3,
   fan-mode-defrost #5). A promotion is a documentation act whose correctness is faithfulness-to-source and
   fold-completeness — neither is a runtime property, so it needs a human/adversarial reader, not only a
   passing test. Budget an independent fidelity pass on every promotion PR.

## Proposed kit deltas

- These three rules should guide any future promotion campaign; (1) and (2) are already enforced in the
  `kit` self-section open_issue discipline. Consider a one-line pointer to them from `BUILD-LOOP.md` §7 the
  next time §7 is edited (do NOT add it speculatively now).

## Self-verify

| Claim | Marker | Evidence |
|---|---|---|
| Campaign 2 folded logic+UX+build/deploy/schema doc lessons | [CERT] | CHANGELOG v0.8.2-v0.9.3; retros/INDEX.md folded rows |
| The fold-vs-revert rule is recorded and was applied | [CERT] | BUILD-STATE.md kit self-section; PR#19/#20 revert+fold fixes |
| 6 script-impls stay pending + tracked | [CERT] | BUILD-STATE.md kit open_issue (B4/B6/B7/B8/B10 + soft-start) |
| Fidelity grading caught 5 over-claims a green suite missed | [CERT] | QA verdicts on PR#19/#20/#21 (gate 84/0 throughout) |

Connections: [[2026-09-04-kit-continuity-and-retro-gate-campaign]]; [[2026-09-04-gate-exit-taxonomy-promotion]].
