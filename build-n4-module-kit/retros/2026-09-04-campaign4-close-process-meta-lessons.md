<!-- review-status: pending -->
<!-- kit-retro -->
# Campaign 4 close: the corpus-exhaustion campaign's own meta-lessons

Date: 2026-09-04 · Module: kit · SDD: build-n4-module-continuity Campaign 4 (PRs #29-#31, v0.13.3→0.14.0)

Campaign 4 folded the last mined group (the T/process lessons + the WB/UX/organization corpus) and shipped
the curated `corpus-index.md`, exhausting the 42-lesson mined corpus. Doing it surfaced three genuinely-new
process lessons (the promotion + gate + fidelity lessons of Campaigns 2-3 already live in the kit). Captured
here so a future kit-maintenance campaign inherits them.

## Meta-lessons (proven across C4-PR1..PR3)

1. **Verify kit coverage against a WORKTREE off origin/main, never the local `main` checkout — it can sit
   stale.** In C4-PR1 a coverage grep ran against the base repo's local `main`, which was 12 PRs / one whole
   campaign behind origin/main (v0.5.0 vs the real head) because the ff-only merges landed on origin without a
   local pull. It nearly drove a fold against pre-campaign files. The tell was a line-number mismatch between
   the grep hit and the file I then opened. Rule: do coverage checks in a worktree created off `origin/main`;
   after a run of origin-side merges, `git checkout main && git pull --ff-only` (or `git merge --ff-only
   origin/main`) so the base checkout can't mislead the next session. This is this campaign's own theme —
   stale state misleading work — hitting our own workspace.

2. **A PARTIAL promotion is a legitimate gate shape the §7 promotion exit does not yet recognize.** A PR that
   folds some lessons of a retro while the retro stays `pending` (its other halves owed to a later PR) flips
   NO INDEX row — so it produces no in-range `INDEX.md` diff. The `.githooks/pre-push` promotion exit requires
   `Retro: promotion (…)` AND an in-range INDEX diff, so it FALSE-NEGATIVES a real partial promotion (C4-PR2
   was one: it folded B751/B752 but corpus-index stayed pending for T7a). The blanket-escape guard is still
   right (a promotion trailer with no structural anchor must fail). Fix (tracked as a gate-hardening
   open_issue, not done in C4): accept an in-range BUILD-STATE.md diff — a partial promotion stamps its owed
   open_issue in BUILD-STATE — as an ALTERNATIVE structural anchor to the INDEX diff, and still run `sweep`
   for ledger coherence on that branch.

3. **Grep EVERY kit file for a lesson before folding it — its proposed target file is not the only place it
   may already live.** In C4-PR1 the plan was to fold T4 (live-anchor verification) into BUILD-LOOP §6; a
   first-pass grep of only BUILD-LOOP.md showed it absent, so a bullet was added — then a grep of
   `build-verify.md` found T4 already folded there (the 4-layer stack, `[ev: hidden-actions · T4]`). The
   duplicate was reverted before commit. A lesson's mined "target" is a suggestion; the fold may already sit
   in a sibling guide. Grep the whole `build-n4-module-kit/**/*.md` for the rule (and its `[ev:]` tag) before
   folding, or you double-fold noise.

## Proposed kit deltas

- These three rules guide a future kit-maintenance campaign; consider a one-line pointer from the
  METHODOLOGY "Kit maintenance — retro promotion discipline" section the next time it is edited (do NOT add
  speculatively now). Lesson 2's fix is already logged as the gate-hardening open_issue in the `kit`
  self-section of `BUILD-STATE.md`.

## Self-verify

| Claim | Marker | Evidence |
|---|---|---|
| Campaign 4 exhausted the 42-lesson mined corpus | [CERT] | CHANGELOG v0.13.3-v0.14.0; retros/INDEX.md all-folded; corpus-index.md |
| The local main checkout was a whole campaign stale | [CERT] | it read v0.5.0 while origin/main was v0.13.x; fixed by ff-only sync |
| The §7 promotion exit false-negatives a partial promotion | [CERT] | .githooks/pre-push line 43 (grep + has_index>=1); C4-PR2 flipped no row |
| T4 was nearly double-folded; caught by grepping build-verify.md | [CERT] | build-verify.md live cold-boot smoke `[ev: hidden-actions · T4]`; the BUILD-LOOP §6 dup was reverted |

Connections: [[2026-09-04-campaign3-close-process-meta-lessons]]; [[2026-09-04-campaign2-promotion-process-meta-lessons]].
