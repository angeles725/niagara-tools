<!-- ledger: kit -->
<!-- date: 2026-10-27 -->
<!-- last-change-note: kit-backlog B5.7: lint-absolute-claims (G38) reports absolute claims that cite only corpus or retro inference -->
log: 2026-10-27 · kit-backlog B5.7 (G38, C51) → new report `lint-absolute-claims.sh <types-dir>...` (`go/internal/lint/rules/absoluteclaims`): INFO rows for each line holding never/always/silently/forbidden/throws whose bullet or paragraph cites no code/course/devguide/CERT/javap tag and no `.java:` line; exit 0 (a report). `METHODOLOGY.md` K33 points to it; `tests/lint-absolute-claims.bats` AC1-AC2. On the kit's own `types/` at this commit: 423 rows (282 never, 58 throws, 44 silently, 34 always, 5 forbidden; 206 with no tag). [ev: retro n4-course-as-exemplar-source Δ3]
