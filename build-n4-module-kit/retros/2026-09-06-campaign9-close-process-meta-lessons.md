<!-- review-status: folded -->
<!-- kit-retro -->
# Campaign 9 close: meta-lessons from the close process

Date: 2026-09-06 · Module: kit · SDD: build-n4-module-campaign9

> Drafted by investigador1 + lead for the campaign-9 close. Campaign 9 ran as: kit lints PR2/PR3/PR10 (demand-scope,
> silent-protection, ext-writable-shape), client PRs PR1/PR6/PR6b/PR8/PR9/PR11 (rotation, servlet guards, config login,
> CR-3/CP-1 alarms, write-path rows), kit doctrine PR12, and this close PR13. Lessons 1–19 are from investigador1's
> evidence map; W2/W3 lessons are lead additions from the second close read. Each lesson has an `[ev:]` token.
> `[ev: retro campaign8-close-process-meta-lessons]`

## Meta-lessons

1. **Stale-checkout reads are the campaign's recurring defect class.** The C9 design (D1/D8) cited `4f5f1c7`; the
   evidence map repeated D8a's stale silent-zero claim; the S20 apply-package rev 1 copied line numbers from the design
   text. Fix that stuck: ONE read-only worktree at the chain tip, and "state the tree you read" on every cite.
   **Proposed delta:** `METHODOLOGY.md` §Evidence discipline — name the worktree and commit on every file cite; a
   stale sha invalidates the cite. `[ev: corpus B820 §820.3]`

2. **Verify content, then COUNT the anchor.** Verifying that `:230` is `pickMostHoursOn` is NOT verifying that it is
   `:230` at the tip. Re-anchor = re-count. **Proposed delta:** `BUILD-LOOP.md` evidence-anchor rule — after confirming
   a symbol's identity, re-count its line at the current tip before recording the cite. `[ev: corpus B820]`

3. **Executable RED wins over prose.** The design's `Decision decide(...)` vs the RED's `int decide(...)`; the boolean
   `rotationMakeBeforeBreak` vs the RED's `int rotationMode`; CL6 400 vs the RED's 401. When prose and compile contract
   disagree, the RED is the spec. **Proposed delta:** `METHODOLOGY.md` K-rule addition — when a design paragraph and
   a RED signature disagree, the RED is authoritative; prose is a hypothesis. `[ev: corpus B820 §820.3]`

4. **Harness-only pins are DECLARED in the RED header, not skip-gated.** The WSL file has zero `@Ignore/Assume`; the
   verify gate counts against the header coverage map, and a WSL run never reports harness-only pins green.
   **Proposed delta:** `BUILD-LOOP.md` §RED discipline — harness-only pins live in the RED header, never as skip-gated
   bats cases. `[ev: corpus B820]`

5. **A design decision must be re-applied to every gate it touches.** The rotation clock moved to `rotSinceMs[]` but
   gate 8 kept `cmdSince[out]`, silently reintroducing the ROT16 bug the section claimed to close. After changing a
   primitive, grep every use of the old one. **Proposed delta:** `BUILD-LOOP.md` §5 — after every structural rename,
   grep the old identifier across the source tree before finalizing the gate. `[ev: corpus B820 §820.3]`

6. **The null-Context write is a security class, not an audit gap.** `ComplexSlotMap.set:662` gates BOTH the
   `AuditEvent` and `user.checkWrite`; `set(prop, v, null)` bypasses Niagara permission enforcement. Kit doctrine: a
   servlet write ALWAYS passes a user-bearing Context; a lint candidate for C10. **Proposed delta:** `types/dashboard.md`
   §write-surface — a servlet write MUST pass a non-null user-bearing Context; `set(prop, v, null)` is a security defect.
   `[ev: corpus B830 §830.4]`

7. **`BUser` IS a `Context` — no wrapper needed.** The client already relied on the cast. Small API facts save whole
   seams. **Proposed delta:** `types/logic-authoring.md` §slot-types-ext-writes — note `BUser implements Context`; no
   wrapper pattern needed. `[ev: corpus B830 §830.2]`

8. **Lockout accounting is caller-invoked.** `validate()` has no side effects; a module that forgets
   `authenticateFailed` ships a password oracle. **Proposed delta:** `types/dashboard.md` §auth — explicit note that
   `authenticateFailed` must be called on each failed attempt; omitting it enables brute-force. `[ev: corpus B830 §830.3]`

9. **Never leak the scheme.** An unsupported authenticator answers 401, not 400. **Proposed delta:** `types/dashboard.md`
   §auth — a 400 on an unrecognized auth scheme reveals the scheme space; respond 401 unconditionally. `[ev: corpus B830 §830.4]`

10. **A tool failure is not a zero.** The first niagara-help pass failed on zsh word-splitting (`$H` unsplit); the zeros
    were recorded only from the successful re-run. **Proposed delta:** `BUILD-LOOP.md` evidence-anchor rule — a tool
    invocation that exits non-zero produces no verdict; re-run with the exact fixed invocation before recording counts.
    `[ev: corpus B830 §830.8]`

11. **Public-repo publication is a user decision, not a peer relay.** The classifier blocked a push to a repo that had
    become public mid-session; the decision was taken by Cristian in-session, then pushes resumed. Check `gh repo view`
    visibility before the first push of a session. **Proposed delta:** `BUILD-LOOP.md` §6 — before the first push of
    a session, verify repo visibility with `gh repo view --json visibility`. `[ev: corpus B829]`

12. **Identity vs session are different columns.** `config_session` is an opaque id (NULL for surface B because the
    `AuditEvent` has no session field); the operator goes in the identity column. **Proposed delta:** `types/dashboard.md`
    §audit — distinguish identity column (operator name) from session column (opaque id, nullable). `[ev: corpus B830 §830.4]`

13. **Second-read cadence works when the reader holds the source.** Design → second read → 14 edits → validator PASS
    in one loop; the reads that caught defects were the ones that re-ran the grep at the tip. **Proposed delta:**
    `METHODOLOGY.md` §Evidence discipline — a second read at the tip is a gate step, not optional QA. `[ev: corpus B820]`

14. **Repo-fixed ≠ deployed — every fix claim carries the deployed version.** The station runs older module versions
    while the repo tip is ahead; the "silent zero" and the defrost `time<=0` bug are live on the panel despite being
    fixed in the repo. Rule: a claim "X is fixed" names the version where it is fixed AND the version that runs; the
    deploy chain is a precondition of every live gate. **Proposed delta:** `BUILD-LOOP.md` §6 — every claim "X is fixed"
    must name BOTH the fix version and the deployed version. `[ev: corpus B815 §815.10]`

15. **Resume from the uncommitted tree, never from the package.** A worker cut by a session limit had 111 lines of the
    core already in the worktree; the resume rule: read the uncommitted tree first and continue from it. Restarting from
    the apply-package re-derives and drifts — the tree is the state, the package was the plan. **Proposed delta:**
    `ORCHESTRATION.md` §Recovery — a resumed worker reads the uncommitted worktree first; the apply-package is input
    history, not the live state. `[ev: corpus B820 §820.3]`

16. **A negative claim needs a grep that actually asked the question.** "coolOnSensorFault is not in WRITABLE" was
    stated from a grep whose pattern never contained that name; the entry is at `write-server.mjs:94`. Before writing
    "X is absent", show the query that would have matched X. **Proposed delta:** `METHODOLOGY.md` §Evidence discipline —
    a negative-absence claim must show the grep that would have matched X if present. `[ev: corpus B820 §820.3]`

17. **Fixture-fitting production code is a defect, not a fix.** PR1 guarded `pickLeastHoursOff` with
    `cmdSince[k] != 0` and switched staging to `autoOnCount` — both to make rotation fixtures pass, both ungated
    production changes. When a fixture fails, the first hypothesis is the fixture is wrong, not that production semantics
    must move. **Proposed delta:** `BUILD-LOOP.md` §5 — a production line changed to satisfy a test is reviewed as a
    production change with its own spec, or reverted; the first hypothesis on a failing fixture is a fixture defect.
    `[ev: corpus B820 §820.3]`

18. **A golden protects only the cases its sequence exercises — name the uncovered axes.** ROT5's byte-identical golden
    passed while `onCount→autoOnCount` silently changed HAND-present staging, because the golden `demandSeq` is
    all-AUTO. A byte-identical claim must state the axes the trace does NOT vary; the fix is a second golden per
    uncovered axis, not a stronger assertion on the same trace. **Proposed delta:** `METHODOLOGY.md` §Kit maintenance —
    a golden's coverage claim must list uncovered axes; a change that touches those axes requires a new golden, not
    a stronger assertion on the original. `[ev: corpus B820 §820.3]` `[ev: retro campaign8-close-process-meta-lessons Δ11]`

19. **A lazy stamp that lags in the safe direction can still INVERT a lifecycle contract.** PR1's rotation clock lagged
    conservatively (rotation later, never earlier) — which looked safe — yet the enable-edge stamp of OFF units plus a
    stage-up that never re-stamped gave an OFF-at-enable unit a stale clock, making it eligible to rotate at ~0 runtime:
    the exact opposite of ROT16. "Conservative lag" is not "correct"; a lifecycle timer must be stamped at the event it
    measures (the command), never reconstructed lazily from a different clock. **Proposed delta:** `types/logic.md`
    §RT-control-logic — a lifecycle timer is stamped at the exact event it measures; a lag that looks safe in isolation
    can invert the contract when combined with a stale-clock edge case. `[ev: corpus B820 §820.3]`

## W2/W3 — Lead additions (second close read)

20. **TTL written and never compared is a dead field.** A TTL value set in the config and stored in the write-server
    session is correct only if the comparison that enforces it actually runs. Confirm every stored value has a read path
    that gates the logic it was stored for. **Proposed delta:** `types/dashboard.md` §auth — a session TTL must be
    compared on every subsequent request; a stored-but-never-compared TTL is a dead field and provides no expiry.
    `[ev: corpus B830 §830.3]`

21. **Comment-satisfiable pins hide real defects — strip `//` and `/* */` before matching.** A bats `contains` pin
    passes when the matched string appears in a comment (`// WARN`, `/* lint-silent-protection */`). Strip single-line
    and block comments before the assertion. Affects four known pins: CRA3s, SP4, EW10 wording, CLW6/CLW4.
    **Proposed delta:** `BUILD-LOOP.md` §5 — assertion patterns that match identifiers or warn-strings must run on
    comment-stripped source; record the strip step in the pin. `[ev: corpus B820 §820.3]`

22. **A `HashMap` on a shared servlet field is a concurrency defect.** A BWebServlet is instantiated once per station;
    a `HashMap` field is shared across all concurrent requests. Use a thread-safe map or per-request state. **Proposed
    delta:** `types/dashboard.md` §servlet-guards — servlet fields shared across requests must use thread-safe
    collections; a raw `HashMap` field is a concurrency defect. `[ev: corpus B829]`

23. **The pre-push hook needs BUILD-STATE.md in the pushed range when a retro is added.** A branch push is not proof;
    the hook evaluates the whole PR range on main — `BUILD-LOOP.md` §108 envelope-pairing. When a retro row is added
    to INDEX.md, the BUILD-STATE envelope must also change in the same push range, or the gate fails.
    **Proposed delta:** `BUILD-LOOP.md` §7 — adding an INDEX row requires a BUILD-STATE envelope change in the same
    push range; the pre-push hook validates the pair. `[ev: corpus B820 §820.3]`

24. **Workers never edit QA REDs (K13).** A QA-owned RED branch is off-limits to workers; any pin update in a QA RED
    is the QA role's responsibility. **Proposed delta:** `ORCHESTRATION.md` §Roles — reinforce K13: a worker that finds
    a QA RED mismatch files the mismatch as a blocker, never edits the RED directly. `[ev: corpus B820 §820.3]`

25. **Workers never write outside their own worktree (K12).** A worker operating in one worktree must not modify files
    in another worktree or in the main checkout, even for convenience. **Proposed delta:** `ORCHESTRATION.md` §Roles —
    reinforce K12: a worker's write scope is exactly its assigned worktree; any cross-worktree write is a defect.
    `[ev: corpus B820 §820.3]`
