<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 6 -->
# 2026-10-16 · kit · n4-course-handson-services-history-threading

**Session**: hands-on Lab 6 (Work Order Service, Jackson + REST), Lab 9 (multithreading variants), Lab 10 (BSimpleJob history-to-CSV) and Certification Exam P2 (`BHistoryExportService`), run live on 4.15.6.20; kit lints re-run on the course sources.
**Delta count**: 6

## What happened
The service labs and exam P2 exercised the threading, job, history and file-export doctrine that the 2026-10-15
doc-mined retros rewrote from the course text. Running the code live confirms most of those proposals; running the
kit's `lint-timers.sh` on the Lab 9 sources shows that the `jdk-thread` FAIL, which one of those proposals wants
to re-scope, never fires on Tridium-style source at all. The exam also gave a minimal, exporter-free history
export recipe that wrote `history.csv` into the station's `shared/` folder.

## Evidence
- [CERT-live] Exam P2 `BHistoryExportService` (`CertificationExam/Cristian_Angeles_Cert/…/BHistoryExportService.java`):
  `export` action `flags = Flags.ASYNC`; `doExport(Context context)` resolves
  `BOrd.make("history:" + getHistoryId().encodeToString() + "|bql:select *").get(this, context)` to
  `BITable<BHistoryRecord>`, opens `BajaFileUtil.createFileToOverwrite(getOutputFile(), context)` with default ORD
  `file:^history.csv`, and writes in one try-with-resources over
  `new PrintWriter(new OutputStreamWriter(file.getOutputStream(), StandardCharsets.UTF_8))` and `table.cursor()`,
  reading audit columns by name (`historyRecord.get("operation")`, `"target"`, `"slotName"`). Verified live:
  `shared/history.csv` present in the submitted station `[ev: engram obs 11009]` `[ev: engram obs 11478]`.
- [CERT] The kit's `BQL from code` section exists (`types/logic-authoring.md:187`) but has no history-export or
  file-export recipe.
- [CERT-live] Kit lints on a copy of the exam `-rt/src` (2026-10-16): `lint-null-context-write.sh` exit 0, 0 rows;
  `lint-bql-string-concat.sh` exit 0 with `WARN … BHistoryExportService.java:126  bql concat: BOrd historyOrd = BOrd.make("history:" + getHistoryId().encodeToString() + "|bql:select *");`.
- [CERT-live] Lab 9 (verified live `[ev: engram obs 11379]`): `BEngineThreadExecutor` is an `ASYNC` action that sleeps
  10 s in `doExecute()`; the lab text states it "run[s] asynchronously, but on the control engine thread" and
  freezes the engine thread while the Workbench session stays responsive
  (`Lab Instructions.txt:1644-1647`, `:1711-1721`) [CERT-doc].
- [CERT-live] `toolbelt/lint-timers.sh <copy of training-rt/src>` → 0 rows, exit 0, although
  `BDedicatedThreadExecutor.java:86` has `new Thread(() -> {` and `BExecutorServiceExecutor.java:14,103,111`
  import `Executors.newSingleThreadExecutor`, create it in `started()` and `shutdownNow()` it in `stopped()`.
  The same two files with the `public class X` / `  extends BComponent` header joined on one line both produce
  `FAIL jdk-thread`. Cause: the check requires `class B… (extends|implements) B…` on one line
  (`lint-timers.sh:480-481`); Tridium and wizard code put `extends` on the next line.
- [CERT-live] `training-rt.jar` `META-INF/module.xml` grants `NETWORK_COMMUNICATION` (`hosts="*"`, `ports="3001"`),
  `REFLECTION` and `MANAGE_EXECUTION`; the module is `CODESIGN`-signed and ran under high mode
  `[ev: engram obs 11148]`; Lab 6 posted a work order to the local json-server (stored id `Q9djx-ksnQE`) with
  `uberjar("com.fasterxml.jackson.core:jackson-databind:2.21.2")` (`training-rt.gradle.kts:55`), 1107 Jackson classes
  from three artifacts in the 2.5 MB jar `[ev: engram obs 11155]`. The kit still says "never … `Executors.*`, or
  `new Thread(...)`" (`types/logic.md:344`).
- [CERT-live] Lab 10 `BHistoriesToCsvJob extends BSimpleJob`: loop checks `if (!isAlive()) throw new JobCancelException();`
  (:105-107), reports `progress(…)` (:109) and writes with `createFileToOverwrite` (:112); Lab 10 verified live
  `[ev: engram obs 11379]`.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Add a "history → CSV without an exporter" recipe from the exam: `history:<id.encodeToString()>\|bql:select *` resolved with the caller's Context, `BITable<BHistoryRecord>` + `TableCursor` in try-with-resources, `createFileToOverwrite(ord, cx)` on a `file:^…` ORD (lands in the station `shared/` folder), explicit `UTF_8`, columns by name for audit histories. Confirms `n4-course-drivers-bql-ords-files` Δ1 (Context passed, cursor closed) and Δ5 (`file:^`, `createFileToOverwrite`, try-with-resources, explicit charset) live. Refine `lint-bql-string-concat.sh`: it WARNs on this recipe (`BHistoryExportService.java:126 bql concat`) although the only concatenated value is the encoded history id in the `history:` scheme and the BQL after `\|bql:` is a literal; exempt a concatenation whose dynamic part ends before `\|bql:` and whose BQL tail is a string literal. Lint: YES (precision). Acceptance: the exam file gives 0 rows; a fixture concatenating a variable AFTER `\|bql:` still WARNs; `lint-null-context-write.sh` stays clean on it (exit 0 today). | `types/logic-authoring.md` § BQL from code (:187) + new § File export | `[ev: code CertificationExam/…/BHistoryExportService.java]` |
| Δ2 | Confirms `n4-course-threading-jobs-doctrine` Δ3: `Flags.ASYNC` moves the work off the caller's Fox thread onto the engine thread (Lab 9 `BEngineThreadExecutor` freezes the engine for 10 s while Workbench stays live), and it is the simplest technique the exam accepted for a short export. Keep that Δ3's lint (ASYNC action with no `post(` override doing I/O) at WARN, word it "acceptable for short bounded work; for long or unbounded work override `post()` to a worker/executor", and never promote it to FAIL. Lint: YES (that Δ3). Acceptance: the lint WARNs on exam `BHistoryExportService` and Lab 9 `BEngineThreadExecutor` and exits 0. | `types/actions.md` § 2 · Threading model + the Δ3 lint | `[ev: code Lab Instructions.txt:1644-1647]` |
| Δ3 | Refine `n4-course-threading-jobs-doctrine` Δ2 (re-scope `jdk-thread`): first make the class-header detection multi-line (join the `class` line with the following `extends`/`implements` lines, or match `extends B…` anywhere before the first `{`), otherwise the re-scoped FAIL/WARN never runs on Tridium-style source and its bats only prove one-line fixtures. Lint: YES. Acceptance: `lint-timers.sh` on the Lab 9 sources emits a row for `BDedicatedThreadExecutor` and `BExecutorServiceExecutor` (WARN once that Δ2 lands, since `MANAGE_EXECUTION` is granted); a two-line-header bats fixture is added; mutation "revert to single-line header match" turns it green-silent again. | `toolbelt/lint-timers.sh` § jdk-thread (:474-486) | `[ev: live 2026-10-16 lint-timers.sh on Lab 9 sources]` |
| Δ4 | Confirms `n4-course-threading-jobs-doctrine` Δ1 and `n4-course-build-deps-permissions-signing` Δ2 live: a dedicated `Thread` and an `ExecutorService` created in `started()` and `shutdownNow()` in `stopped()` load and run in a signed module that requests `MANAGE_EXECUTION` (type station); `REFLECTION` for Jackson databind needs the signed module (it ran under high mode); `NETWORK_COMMUNICATION` can be narrowed to one port (`ports="3001"`). Rewrite `types/logic.md:344` accordingly. Lint: no new one. Acceptance: the § text cites these three grants and the Lab 9 lifecycle lines. | `types/logic.md` § Tridium rt idioms (:344) | `[ev: live 2026-10-16 unzip training-rt.jar META-INF/module.xml]` |
| Δ5 | Confirms `n4-course-authoring-files-and-libraries` Δ1 (Path 0 `uberjar()`) live and adds two facts: `uberjar` pulls the coordinate's transitive artifacts too (databind brought core and annotations, 1107 classes, +2.5 MB), and the result is a multi-release jar that the kit gate currently rejects (see `2026-10-16-n4-course-handson-build-signing-windows` Δ3). Lint: no. Acceptance: the third-party-libraries § lists both facts with this module as the example. | `types/third-party-libraries.md` § Two packaging paths | `[ev: code N4_Labs/…/training-rt.gradle.kts:55]` |
| Δ6 | Confirms `n4-course-threading-jobs-doctrine` Δ6(b) job contract live (`isAlive()` → `JobCancelException`, `progress()`, `createFileToOverwrite` per file in a re-runnable job); its proposed WARN (a `BSimpleJob` `run(` loop with no `isAlive()`) is clean on Lab 10. Lint: no new one. Acceptance: the Background-jobs § cites `BHistoriesToCsvJob.java:105-112` as the worked example. | `types/logic-authoring.md` § Background jobs (:212) | `[ev: code N4_Labs/…/BHistoriesToCsvJob.java:105-112]` |

## Lessons
- A lint that never fires on the reference style it is meant to judge is a silent pass; run each lint on Tridium
  sample code before trusting a zero.
- `ASYNC` is a thread hop to the engine, not a background thread; it is fine for a short export and wrong for a
  long one.
- A history export needs no exporter type: an ORD with a `bql:` pipe, a cursor and a file ORD under `file:^`.
- The course's JDK-thread and executor code runs in production shape once `MANAGE_EXECUTION` is granted; the kit's
  absolute ban is contradicted by a live module.


## Fold record (2026-10-24, quality round 10)
- Δ1 FOLDED (doc): `types/logic-authoring.md` § File export (quality round 8 U5). Lint half owed as Go lint G69
  `bql-concat-history-prefix` (BUILD-STATE open issue, `[deferred]` in `ledger.d/2026-10-23-r8-u5-c59-c62.md`).
- Δ2 FOLDED (doc): `types/actions.md` § 2, short bounded work on a plain `ASYNC` action. Lint half: the acceptance
  cases are added to the owed Go lint G28 `async-blocking-action` (`ledger.d/2026-10-24-r10-course-fold-services.md`).
- Δ3 FOLDED: `toolbelt/lint-timers.sh` jdk-thread header spans lines up to the first `{` (pinned in its bats).
- Δ4 FOLDED: `types/logic.md` § Tridium rt idioms, off-engine work ladder with `MANAGE_EXECUTION`.
- Δ5 FOLDED: `types/third-party-libraries.md` § multi-release jars, transitive pull made explicit; the "kit gate
  rejects multi-release jars" half is superseded: retro n4-course-signed-jars-gate-fix Δ3 made the gate skip
  `META-INF/versions/**`.
- Δ6 FOLDED: `types/logic-authoring.md` § Background jobs, `BHistoriesToCsvJob` worked example.

---
**Status**: FOLDED 2026-10-24 — Δ2 and the Δ5 token by this commit, Δ1/Δ3/Δ4/Δ6 by quality round 8; lint halves owed as G28 and G69; INDEX row flipped to folded.
