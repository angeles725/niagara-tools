<!-- review-status: pending -->
# 2026-10-16 · kit · n4-course-handson-workflow

**Session**: working method across the hands-on N4 Developer Certification Course (2026-10-12 to 2026-10-16): Labs 1-11, Reviews 1-2, the Certification Exam and the Bonus Lab, with the operator driving Workbench and the agent writing, building and checking code from WSL.
**Delta count**: 4

## What happened
The course was the first sustained run where the agent built modules against a reference implementation that the
evaluator owns. Four working-method lessons came out of it and were corrected by the operator during the run:
mirror the reference and the operator's own files before writing anything, decide "copy the reference whole" only
after a diff, resume from the workspace itself rather than from the newest session note, and keep heavy background
jobs off the machine while Workbench and a station are in use. The build route from WSL also turned out simpler
than the 2026-10-15 testing retro assumed.

## Evidence
- [CERT-live] Lab 4 and Review 1: the agent first gave its own `BTstat` and asserted a palette prefix without
  reading the course reference or the operator's palette; the operator asked why the code was not in the course
  style. Rule recorded: read `Labs/<Lab>` first, then the operator workspace, and say what is not from the course
  `[ev: engram obs 11102]`; the instructor's own signature style (`doRegenerateId(Context context)`) was matched from
  then on `[ev: engram obs 11148]`.
- [CERT-live] Lab 11: the operator asked where each comment came from (wizard, reference or agent) and wants files to
  match the reference; the reference files are CRLF while agent-written files were LF `[ev: engram obs 11477]`.
  Device and ProxyExt were copied whole from the reference; the factory differed only by the wizard `@author`.
- [CERT-live] Bonus Lab: all code is the reference files copied whole — Part 1 a one-line facet edit, Parts 2-4
  replacing the wizard TODO templates — after a diff showed only TODO template, header and `@author` differences
  `[ev: engram obs 11479]`.
- [CERT-live] Resume error: asked "where did we leave off", the agent first recalled an old session summary; the
  operator corrected it, and the rule became "check the course project file mtimes and the latest transcripts,
  not only the newest session_summary" `[ev: engram obs 11477]`.
- [CERT-live] Every course build ran from WSL directly on the Windows drive, e.g.
  `cd /mnt/c/N4-Developer-Training/CertificationExam && cmd.exe /c "gradlew.bat :Cristian_Angeles_Cert-rt:jar"`
  and `cd /mnt/c/N4-Developer-Training/Core-Class/N4_Labs/niagara_dev_home && cmd.exe /c "gradlew.bat :trafficLightDriver-rt:jar"`
  (transcript 2026-10-16), with no scratch copy `[ev: engram obs 11379]`.
- [CERT-live] 2026-10-13: a corpus-nav semantic index build (~560 % CPU, 2.4 GB in WSL) starved Windows so Workbench
  would not open during the course; WSL and Windows share the laptop's cores `[ev: engram obs 11095]`.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Confirms `n4-course-as-exemplar-source` Δ2 (reference-first) live and adds the copy decision: before replacing a generated file with the reference, diff them ignoring line endings, the copyright header, `@author` and the wizard's TODO template lines; copy the reference whole only when the residual diff shows no local customization, otherwise merge by hand; keep the reference's line endings in reference-mirrored files; never add agent comments to them without saying so. Add `toolbelt/diff-vs-reference.sh <ours> <reference>` (`diff --strip-trailing-cr -I '@author' -I 'Copyright'` plus a TODO-line filter) printing the residual diff and exit 0 = safe to copy. Lint: YES (helper). Acceptance: on a fixture pair that differs only by CRLF, header and `@author` the helper exits 0 with no output; one changed statement makes it exit 1 and print that line. | `BUILD-LOOP.md` § 0. Orient (EXEMPLAR bullet, :7) + new `toolbelt/diff-vs-reference.sh` | `[ev: engram obs 11479]` |
| Δ2 | Add to orient: when resuming hands-on work, list the most recently modified files of the working tree (`find <root> -name '*.java' -newer <last BUILD-STATE or summary date>` or `ls -t`) and read the latest transcript before trusting a session summary; report the newest file and its time in the orient line. Lint: no (procedure); a `toolbelt/orient-guard.sh --recent <root>` option is the mechanical form. Acceptance: the orient line template gains `newest: <file> <mtime>`. | `BUILD-LOOP.md` § 0.a Orient from BUILD-STATE | `[ev: engram obs 11477]` |
| Δ3 | Refine `testing-parity-btestng` Δ1 (native Windows route): when the gradle root already lives on a Windows drive, run it in place — `cd /mnt/c/<path> && cmd.exe /c "gradlew.bat <qualified tasks>"` works because cmd.exe inherits a drive-letter cwd; the scratch copy under `%TEMP%` is needed only for sources on ext4 (`\\wsl.localhost` is rejected). Lint: no. Acceptance: the § states both cases with the course command as the example; `run-niagara-test.sh` (that Δ2) skips the copy when the root is under `/mnt/<drive>/`. | `build-verify.md` § Unit tests in WSL + `types/moduleTest.md` | `[ev: transcript 2026-10-16 cmd.exe gradlew.bat on /mnt/c]` |
| Δ4 | Add a shared-host rule: while the operator has Workbench or a station open on the same laptop, do not start CPU- or memory-heavy background jobs (index builds, full suites, parallel batches); defer them or cap them (`nice`, `-j` at half the cores). Lint: partial — `preflight.sh` WARN when `tasklist.exe` shows `wb.exe` or `station.exe` and the command about to run is on a heavy-job list. Acceptance: ORCHESTRATION § Delegation triggers carries the rule; the preflight row fires on a fixture `tasklist` output. | `ORCHESTRATION.md` § 3 Delegation triggers + `toolbelt/preflight.sh` | `[ev: engram obs 11095]` |

## Lessons
- When an evaluator owns a reference implementation, its style is the specification; "equivalent" code costs the
  learner time.
- "Copy the reference whole" is a decision made from a diff, not from trust.
- The workspace is the record of where work stopped; a session note can be older than the last edit.
- A dev laptop shared with a running station is a production-like host for CPU purposes.

---
**Status**: PENDING — INDEX row appended: `| 2026-10-16-n4-course-handson-workflow.md | kit | 2026-10-16 | pending | 4 |`
