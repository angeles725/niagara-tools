<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 4 -->
# 2026-10-15 · kit · n4-course-as-exemplar-source

**Session**: N4 Developer Certification Course (Tridium, N4.15.6.20) as a kit source — the course folder, the corpus focus `n4-dev-course` (B1230–B1258), `docs/n4-module-development-course-guide.md`, and the user's live lab project
**Delta count**: 4

## What happened
The course is the only Tridium-authored, end-to-end module-development curriculum we hold, and the corpus digested it in 29
blocks (B1230–B1258) with a 73-rule checklist (B1242 §1242.3), a ranked "what not to copy" list (§1242.4) and 13 proposed
kit deltas (§1242.7). None of it reached the kit: `SOURCES.md` and `corpus-index.md` do not list the course or any of those
blocks (kit grep `B123`/`B124`/`B125`, `n4-dev-course` → 0), and none of the 13 B1242 deltas landed (`allowDefaultProfile`,
`javaToolchains`, `checkSetFlags`, `getMaxOutstandingTransactions`, `MessageFormat` → 0). Mining it found two kit rules that
Tridium's own material reverses (the JDK-thread ban, the BSimpleJob pool claim) and two kit examples no Tridium artifact
matches (the silent-no-op action handler, the class-name palette syntax). While coding the labs live, the user asked that
course code follow the course reference and the user's own lab style first, not an independent design. The course's
reference `.java` files ship without the slot-o-matic region (0 of 47), so the course workflow is "transcribe → `gradlew
slotomatic` → fix errors → `gradlew jar`", run directly with `gradlew`, not through `toolbelt/build.sh`.

## Evidence
- Course scope: `Labs/` (Lab1–Lab11, Bonus Labs, Lab Instructions.pdf 128 p.), `Slides/` (decks 0–10 + Bonus), `Slide Code/` (`slides-rt`, 34 tests), `References/BQL Reference.pdf`, `Review Assignments/` 1–2 — `[ev: course /mnt/c/Users/equipo/Downloads/OneDrive_2026-10-08/N4 Developer Certification Course/]`; corpus synthesis — `[ev: corpus B1242 §1242.3]` `[ev: corpus B1242 §1242.4]` `[ev: corpus B1242 §1242.7]`
- Kit has no course pointer — `[ev: code SOURCES.md]` `[ev: code corpus-index.md:1-12]`; B1242 K13 asked for the `corpus-index.md` row — `[ev: corpus B1242 §1242.7 K13]`
- Reversals found this session: `[ev: code types/logic.md:344]` vs `[ev: course Labs/Lab Instructions.pdf p80]`; `[ev: code types/logic-authoring.md:216-233]` vs `[ev: corpus B1238 §1238.8]`; `[ev: code types/actions.md:24-26]` vs `[ev: corpus B1231 §1231.4]`; `[ev: code types/structure.md:150-165]` vs `[ev: /mnt/c/Niagara/Niagara-4.15.6.20/modules/kitControl-rt.jar!module.palette:133-141]`
- "Example code only and is not intended for use in production" headers on every course file; ranked lab-solution defects — `[ev: course Labs/Lab9 Multithreaded/module-permissions.xml]` `[ev: corpus B1242 §1242.4]`
- Region-less sources: `grep -rl "BEGIN BAJA AUTO"` over `Labs/**/*.java` → 0 of 47; the user's compiled copies carry it (13 files) — `[ev: course Labs/]` `[ev: /mnt/c/N4-Developer-Training/Core-Class/N4_Labs/niagara_dev_home/training/training-rt/src]`; lab step order "transcribe the annotations … Run gradlew slotomatic … This will produce a lot of errors" — `[ev: course Labs/Lab Instructions.pdf p81]`; the kit covers slotomatic-before-jar only inside `build.sh` and K15 — `[ev: code BUILD-LOOP.md:115-123]` `[ev: code METHODOLOGY.md:121]`
- User instruction (live session, 2026-10-15): course code follows the course reference and the user's own lab style first — `[ev: user session 2026-10-15]`; kit orientation reads the SOURCES.md exemplar only — `[ev: code BUILD-LOOP.md:7]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Register the course as a P0 source: in `SOURCES.md` § Exemplar source add the course folder (Labs, Slide Code, BQL Reference, Review Assignments) with the "example code only" caveat; in `corpus-index.md` add an `n4-dev-course` row (B1230–B1258, P0 reads B1242 synthesis + B1238 threading + B1235 permissions + B1236 lexicon/tests + B1237 BQL + B1239 drivers) and `docs/n4-module-development-course-guide.md`. Lint: no. | `SOURCES.md` § Exemplar source + `corpus-index.md` | `[ev: corpus B1242 §1242.7 K13]` |
| Δ2 | Add a reference-first rule to Orient: when the user supplies a reference implementation (course lab, vendor sample, an earlier module of theirs) or has their own code for the same exercise, mirror that reference and the user's style (names, layout, permission file, palette) first; state each deviation with its reason; for course code, apply B1242 §1242.4 "what not to copy" fixes only as named, opt-in changes. Lint: no. | `BUILD-LOOP.md` § 0. Orient (after the EXEMPLAR bullet) + `skill/SKILL.md` step 2 | `[ev: user session 2026-10-15]` |
| Δ3 | Add a K-rule: before keeping or adding an absolute rule ("never", "throws", "silent") whose only evidence is corpus inference, check it against Tridium-authored training/docs when they exist; record the four reversals of this session as its worked examples, and schedule a one-time sweep of `types/*.md` "never"/"forbidden" rules against B1242 §1242.3. Lint: YES (partial) — `sweep-fold-audit.sh`-style report listing absolute-claim lines whose `[ev:]` tags cite only retros/corpus inference (no `code`/`course`/devguide tag), for human review. | `METHODOLOGY.md` § kit-process K-rules | `[ev: corpus B1238 §1238.8]` |
| Δ4 | Extend K15 for reference sources: course/vendor `.java` files arrive without the generated region (course: 0/47), so after transcribing run `gradlew slotomatic` (or `toolbelt/build.sh`) before `jar`; the IDE shows errors until slot-o-matic runs and until superclass modules are added (see `2026-10-15-n4-course-build-deps-permissions-signing.md` Δ1); a bare `gradlew jar` on region-less sources fails. Lint: no (build.sh already runs slotomatic). | `build-verify.md` § Doctrine (raw-gradlew path) + `METHODOLOGY.md` K15 | `[ev: course Labs/Lab Instructions.pdf p81]` |

## Lessons
- A digested source that never reaches the kit index is invisible to the build loop; 29 blocks and 13 proposed deltas sat unused.
- Tridium training outranks corpus inference for doctrine; where they disagree, the kit rule is the one to re-verify.
- For course work, the user's reference and style are the specification; independent redesign is a deviation that must be named.
