#!/usr/bin/env bats
# count-retro-deltas.bats — G39 (METHODOLOGY K34): a retro's delta rows against its Delta count, deltas header and
# INDEX row; the unit cases live in go/internal/retrodeltas. [ev: retro review2-signing-build-route Δ6]

setup() {
  T="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit/toolbelt/count-retro-deltas.sh"
  D="$BATS_TEST_TMPDIR/retros"
  mkdir -p "$D"
  printf '%s\n' '<!-- review-status: pending -->' '<!-- module: kit -->' '<!-- deltas: 2 -->' '# 2026-10-27 · kit · x' '' \
    '**Delta count**: 2' '' '## Proposed kit deltas (propose-never-apply)' '' '| # | Delta | Target | Evidence |' '|---|---|---|---|' \
    '| Δ1 | a | b | c |' '| Δ2 | d | e | f |' > "$D/2026-10-27-x.md"
  printf '%s\n' '| Retro | Module | Date | Status | Deltas |' '|---|---|---|---|---|' '| 2026-10-27-x.md | kit | 2026-10-27 | pending | 2 |' > "$D/INDEX.md"
}

@test "CRD1: a retro whose rows, Delta count, header and INDEX row agree exits 0 with a TOTAL line" {
  run "$T" "$D/2026-10-27-x.md"
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "$D/2026-10-27-x.md  rows=2  count=2  header=2  index=2  OK" ]
  [ "${lines[1]}" = "TOTAL  rows=2  files=1  mismatches=0" ]
}

@test "CRD2: a Delta count that is not the row count is a MISMATCH, exit 1" {
  sed -i 's/^\*\*Delta count\*\*: 2$/**Delta count**: 3/' "$D/2026-10-27-x.md"
  run "$T" "$D/2026-10-27-x.md"
  [ "$status" -eq 1 ]
  [ "${lines[0]}" = "$D/2026-10-27-x.md  rows=2  count=3  header=2  index=2  MISMATCH" ]
}

@test "CRD3: usage, and a retro with no INDEX.md beside it, exit 3" {
  run "$T"
  [ "$status" -eq 3 ]
  rm "$D/INDEX.md"
  run "$T" "$D/2026-10-27-x.md"
  [ "$status" -eq 3 ]
  [[ "$output" == *"no INDEX.md beside"* ]]
}
