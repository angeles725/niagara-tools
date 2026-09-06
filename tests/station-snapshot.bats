#!/usr/bin/env bats
# RED-FIRST pins for station-snapshot.sh (campaign 8 PR9). Local mode: snapshot a running station's
# audit surface (config.bog + console*.txt) into an out-dir with a sha256 manifest, WITHOUT writing a
# single byte back to the live station dir. Feeds the post-deploy "snapshot the station and audit it"
# workflow (bog-audit reads the copied config.bog; triage-console reads the copied console files).
#
# SURFACE: station-snapshot.sh <station-dir> <out-dir>
#   copies ONLY config.bog + console*.txt; writes <out-dir>/manifest.json { file: {sha256, bytes}, ts }.
#   exit: 0 ok · 1 a required file missing/unreadable · 3 usage/env.
#
# RED today: station-snapshot.sh does not exist -> every pin fails for the right reason (tool absent).
# NAMED MUTATION (post-green): drop the sha256 from the manifest -> SN2 (manifest hash matches the file) fails.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  SNAP="$KIT/toolbelt/station-snapshot.sh"
  ST="$BATS_TEST_TMPDIR/station"; OUT="$BATS_TEST_TMPDIR/out"
  mkdir -p "$ST/history"
  # config.bog is a tiny zip (real .bog is a zip) — generated, never a committed binary
  echo "<bajaObjectGraph/>" > "$BATS_TEST_TMPDIR/_bog.xml"
  ( cd "$BATS_TEST_TMPDIR" && zip -q "$ST/config.bog" _bog.xml )
  printf 'INFO station started\n' > "$ST/console_1.txt"
  printf 'WARNING something\n'    > "$ST/console_2.txt"
  printf 'old history data\n'     > "$ST/history/old.log"   # decoy — must NOT be copied
}

@test "SN1: copies config.bog + both console*.txt and NOTHING else (history/ decoy excluded), exit 0" {
  run "$SNAP" "$ST" "$OUT"
  [ "$status" -eq 0 ]
  [ -f "$OUT/config.bog" ] && [ -f "$OUT/console_1.txt" ] && [ -f "$OUT/console_2.txt" ]
  [ ! -e "$OUT/history" ] && [ ! -e "$OUT/old.log" ]
}

@test "SN2: manifest.json carries the real sha256 of each copied file + a timestamp" {
  run "$SNAP" "$ST" "$OUT"
  [ "$status" -eq 0 ]
  [ -f "$OUT/manifest.json" ]
  bogsum=$(sha256sum "$ST/config.bog" | cut -d' ' -f1)
  grep -q "$bogsum" "$OUT/manifest.json"                 # the manifest hash MATCHES the actual file
  grep -q "console_1.txt" "$OUT/manifest.json" && grep -q "console_2.txt" "$OUT/manifest.json"
  grep -qiE '"(ts|timestamp)"' "$OUT/manifest.json"
}

@test "SN3: zero writes to the source — a READ-ONLY station dir still snapshots (exit 0), source unchanged" {
  before=$(cd "$ST" && find . -type f -exec sha256sum {} \; | sort)
  chmod -R a-w "$ST"
  run "$SNAP" "$ST" "$OUT"
  st=$status
  chmod -R u+w "$ST"                                      # restore so bats can clean up
  [ "$st" -eq 0 ]
  after=$(cd "$ST" && find . -type f -exec sha256sum {} \; | sort)
  [ "$before" = "$after" ]                               # not one byte changed in the live station
}

@test "SN4: missing out-dir argument -> exit 3 (usage)" {
  run "$SNAP" "$ST"
  [ "$status" -eq 3 ]
}

@test "SN5: no output file is executable even when source files are +x (NTFS/0777 mount guard); mtimes preserved" {
  chmod +x "$ST/config.bog" "$ST/console_1.txt" "$ST/console_2.txt"
  run "$SNAP" "$ST" "$OUT"
  [ "$status" -eq 0 ]
  exec_count=$(find "$OUT" -type f -perm -u+x | wc -l)
  [ "$exec_count" -eq 0 ]                                  # no output is executable
  # mtimes preserved despite mode strip
  src_mtime=$(stat -c %Y "$ST/config.bog")
  dst_mtime=$(stat -c %Y "$OUT/config.bog")
  [ "$src_mtime" = "$dst_mtime" ]
}
