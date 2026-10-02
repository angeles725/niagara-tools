#!/usr/bin/env bats
# RED-FIRST pins for lint-manual-labels.sh (fold-2026-10-02-pending-retros WU11).
#   Every quoted UI label in an operator manual ("…", “…”, «…») must exist in the UI source
#   (rc/ html/js, lexicon, properties, java) — WARN-only (labels may be paraphrased),
#   --strict promotes a WARN to exit 1. A renamed label the manual still quotes is the
#   drift this lint catches. [ev: retro operator-manual-lockstep Δ3]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  ML="$KIT/toolbelt/lint-manual-labels.sh"
  T="$(mktemp -d)"
  mkdir -p "$T/ui/rc" "$T/rt" "$T/docs"
  printf '<button id="save">Guardar</button>\n<label>Temp. de corte</label>\n' > "$T/ui/rc/index.html"
  printf 'var title = "Modo   manual";\n' > "$T/ui/rc/app.js"
  printf 'pressure=Presi\\u00f3n de succi\\u00f3n\n' > "$T/rt/module.lexicon"
}
teardown() { rm -rf "$T"; }

@test "ML1: every quoted label of a markdown manual exists in the UI source -> exit 0, no WARN" {
  printf '# Manual\n\nPress "Guardar" to store the value.\n' > "$T/docs/manual.md"
  run "$ML" "$T/docs/manual.md" "$T/ui"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
  [[ "$output" == *"checked=1 missing=0"* ]]
}

@test "ML2: a label renamed in the UI but still quoted by the manual -> WARN naming the label and its line, exit 0" {
  printf '# Manual\n\nPress "Guardar".\nSet the "Corte" value.\n' > "$T/docs/manual.md"
  run "$ML" "$T/docs/manual.md" "$T/ui"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  lint-manual-labels"* ]]
  [[ "$output" == *"manual.md:4"* ]]
  [[ "$output" == *'"Corte"'* ]]
  [[ "$output" != *'"Guardar"'* ]]
  [ "$(printf '%s\n' "$output" | grep -c '^WARN')" -eq 1 ]
}

@test "ML3: --strict promotes a missing label to exit 1" {
  printf 'Set the "Corte" value.\n' > "$T/docs/manual.md"
  run "$ML" --strict "$T/docs/manual.md" "$T/ui"
  [ "$status" -eq 1 ]
  [[ "$output" == *"WARN  lint-manual-labels"* ]]
}

@test "ML4: a label with accents is found in a lexicon written with \\u escapes -> no WARN" {
  printf 'Read the “Presión de succión” gauge.\n' > "$T/docs/manual.md"
  run "$ML" "$T/docs/manual.md" "$T/ui" "$T/rt"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

@test "ML5: html manual — entity-quoted labels are checked, tag attribute values are not labels" {
  printf '<html><body>\n<p>Press &ldquo;Guardar&rdquo; and see <a href="other-page.html">the page</a>.</p>\n<p>Open &laquo;Diagn&oacute;stico&raquo;.</p>\n</body></html>\n' > "$T/docs/manual.html"
  run "$ML" "$T/docs/manual.html" "$T/ui"
  [ "$status" -eq 0 ]
  [[ "$output" != *"other-page.html"* ]]
  [[ "$output" == *'"Diagnóstico"'* ]]
  [[ "$output" == *"manual.html:3"* ]]
  [ "$(printf '%s\n' "$output" | grep -c '^WARN')" -eq 1 ]
}

@test "ML6: label match ignores case and collapses whitespace" {
  printf 'Choose "modo manual".\n' > "$T/docs/manual.md"
  run "$ML" "$T/docs/manual.md" "$T/ui"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

@test "ML7: usage and env errors exit 3" {
  run "$ML"
  [ "$status" -eq 3 ]
  [[ "$output" == *"usage"* ]]
  run "$ML" "$T/docs/missing.md" "$T/ui"
  [ "$status" -eq 3 ]
  printf 'x\n' > "$T/docs/manual.md"
  run "$ML" "$T/docs/manual.md" "$T/no-such-dir"
  [ "$status" -eq 3 ]
}

@test "ML8: accent-insensitive; a label is found as a word-boundary prefix of a unit, never mid-word" {
  printf '<span>Limite succion baja</span>\n<span>Rearranque 1</span>\n' > "$T/ui/rc/more.html"
  printf 'Set "Límite succión baja" and "Rearranque". Do not press "Guard".\n' > "$T/docs/manual.md"
  run "$ML" "$T/docs/manual.md" "$T/ui"
  [ "$status" -eq 0 ]
  [[ "$output" == *'"Guard"'* ]]
  [[ "$output" != *'"Rearranque"'* ]]
  [[ "$output" != *'"Límite succión baja"'* ]]
  [ "$(printf '%s\n' "$output" | grep -c '^WARN')" -eq 1 ]
}

@test "ML9: --strict on a clean manual exits 0 (strict promotes WARN rows only)" {
  printf '# Manual\n\nPress "Guardar".\n' > "$T/docs/manual.md"
  run "$ML" --strict "$T/docs/manual.md" "$T/ui"
  [ "$status" -eq 0 ]
  [[ "$output" == *"checked=1 missing=0"* ]]
}

@test "ML10: the header Row format matches the emitted WARN and SUMMARY prefixes" {
  printf 'Set the "Corte" value.\n' > "$T/docs/manual.md"
  run "$ML" "$T/docs/manual.md" "$T/ui"
  [ "$status" -eq 0 ]
  warn_prefix=$(printf '%s\n' "$output" | grep '^WARN' | sed 's/  [^ ]*:[0-9].*//')
  sum_prefix=$(printf '%s\n' "$output" | grep '^SUMMARY' | sed 's/  checked=.*//')
  grep -qF "#   $warn_prefix  <manual>:<line>" "$ML"
  grep -qF "#   $sum_prefix  checked=<n>" "$ML"
}
