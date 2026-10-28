<!-- ledger: kit -->
<!-- date: 2026-10-28 -->
<!-- last-change-note: kit-backlog B10.5: lint-charset-default warns text bytes with the platform default charset -->
log: 2026-10-28 · kit-backlog B10.5 (course coverage run A, L1) → new lint `lint-charset-default.sh [--strict] <src-root>` (`go/internal/lint/rules/charsetdefault`): WARN CD1 no-argument `getBytes()`, CD2 a byte-array `new String` with one or three arguments, CD3 a one-argument `InputStreamReader`/`OutputStreamWriter`, CD4 any `FileReader`/`FileWriter`; a name typed both byte[] and otherwise in one file is not judged. `METHODOLOGY.md` § rt (components) points to it; `tests/lint-charset-default.bats` CD1-CD3. Smoke: 20 client module src roots, 0 WARN (1 K20 ERROR on an empty `-wb/src`). [ev: retro course-coverage Δ19]
