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

# retro build-kit-bench-session Δ21: a mismatch already on the main branch is caught here, not by the next author who
# runs the counter on a retro. The retros before 2026-10-25 predate the header contract and are not read; the two
# named below are known mismatches owed to their own folds.
@test "CRD4: every retro dated after 2026-10-24 agrees with its INDEX row, header and Delta count, bar the two named" {
  local kit retros f b bad
  kit="$BATS_TEST_TMPDIR/kit"
  retros="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit/retros"
  mkdir -p "$kit/retros"
  for f in "$retros"/????-??-??-*.md; do
    b="${f##*/}"
    if [[ "${b:0:10}" > 2026-10-24 ]]; then cp "$f" "$kit/retros/"; fi
  done
  run "$(dirname "$T")/gen-retro-index.sh" "$kit"
  [ "$status" -eq 0 ]
  run "$T" "$kit"/retros/????-??-??-*.md
  [[ "$output" == *"TOTAL  rows="* ]]
  bad="$(printf '%s\n' "$output" | grep ' MISMATCH$' | grep -v -e 'mcpn4-replicate-points' -e 'p1-build-deploy-safety' || true)"
  [ -z "$bad" ] || { printf '%s\n' "$bad" >&2; false; }
}
