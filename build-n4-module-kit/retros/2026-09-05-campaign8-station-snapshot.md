<!-- review-status: pending -->
# Retro: Campaign 8 — station-snapshot.sh local audit-surface snapshot (Campaign 8 PR9)

**Date:** 2026-09-05
**Change:** build-n4-module-campaign8 PR9 — `feat/c8-station-snapshot`
**Scope:** `build-n4-module-kit/toolbelt/station-snapshot.sh` (new); K19 routing to `BUILD-LOOP.md` + `skill/SKILL.md`
**Lead:** Campaign 8 PR9 executor
**RED tip:** `823c032` (SN1–SN4)

---

## What happened

`station-snapshot.sh <station-dir> <out-dir>` copies `config.bog` and every
`console*.txt` from a station directory into `<out-dir>`, records `history/**`
files as pointers (relative path + byte size, never copied), and writes
`manifest.json` with `sha256` + `bytes` per copied file plus a UTC timestamp.
Source directory is opened read-only (`cp` only; no write, no chmod, no temp
file in station-dir). Refuses `<out-dir>` inside `<station-dir>` (exit 3).
Dot-directories are excluded from the console glob and from history walk (D9b).
K19 routing: one line in `BUILD-LOOP.md` (post-deploy section) and one line in
`skill/SKILL.md` (toolbelt list).

---

## Evidence

### TDD evidence

| Phase | Evidence |
|-------|----------|
| RED | Commit `823c032` on `feat/c8-station-snapshot`: SN1–SN4 all failing because `station-snapshot.sh` was absent (exit 127). |
| GREEN | `station-snapshot.sh` implemented; all 4 SN tests turn green; full suite 238/238. |
| REFACTOR | shellcheck exit 0 clean; `_join_entries` helper extracted to produce comma-separated JSON array without trailing comma. |

### Named mutations (all on mktemp copies; original restored after each)

| Mutation | Change applied | Test that flipped | Observed output |
|----------|---------------|------------------|-----------------|
| (a) write to source | Inserted `touch "$STATION_ABS/.snap-marker"` after `mkdir -p "$OUT_DIR"` | SN3 only | `not ok 3 SN3: zero writes to the source` — chmod a-w source blocks the touch; script exits non-zero |
| (b) constant sha256 | Replaced `sha=$(sha256sum …)` with `sha="deadbeef…"` (64 hex zeros) | SN2 only | `not ok 2 SN2: manifest.json carries the real sha256` — grep finds constant, not the file's actual hash |
| (c) copy history/ | Added `cp -r "$STATION_ABS/history" "$OUT_DIR/history"` after mkdir | SN1 only | `not ok 1 SN1: … NOTHING else (history/ decoy excluded)` — `$OUT/history` exists, `[ ! -e "$OUT/history" ]` fails |

### SN5 RED → GREEN proof (lead fix for NTFS/0777 mount defect)

**Defect found:** `cp -p` preserves source mode; all 11 PANCCADIA outputs were executable
(`find <out> -perm -u+x -type f` = 11) because every file under `/mnt/c` is 0777 on WSL.
Contract D10 says outputs are never +x.

**SN5 RED** (mktemp copy of original script, before fix):
```
$ chmod +x "$ST/config.bog" "$ST/console_1.txt" "$ST/console_2.txt"
$ "$SNAP_ORIG" "$ST" "$OUT"
$ find "$OUT" -type f -perm -u+x | wc -l
2
```
`not ok 5 SN5: no output file is executable even when source files are +x`

**Fix:** added `chmod 0644 "$OUT_DIR/$rel"` immediately after `cp -p` in `_snap_file()`.
`cp -p` preserves mtime; the subsequent chmod strips the executable bit without touching timestamps.

**SN5 GREEN** (fixed script):
```
$ find "$OUT" -type f -perm -u+x | wc -l
0
$ stat -c %Y "$ST/config.bog"   # == stat -c %Y "$OUT/config.bog"  (mtime preserved)
1788667832
```
`ok 5 SN5: no output file is executable even when source files are +x (NTFS/0777 mount guard); mtimes preserved`

### Real smoke — PANCCADIA station re-run (fixed script; Windows mount, read-only)

Station: `/mnt/c/Users/equipo/Niagara4.14/OptimizerSupervisor/stations/PANCCADIA`
Out-dir: `mktemp -d` (temp; deleted after run)

```
PASS  station-snap    config.bog  sha256=ffc7804ca4c5e13736a24e9cd5dc47eb0bb5240140cf59f1d817571ffdc2321f bytes=35094
PASS  station-snap    console_backup_260902_0027.txt  sha256=54d429d5822940b5d0438c6b92e49782747788bdcff1e21824df29d19234debe bytes=7681
PASS  station-snap    console_backup_260902_0120.txt  sha256=a5bd68f97f4cd15976ee2060b49d701cf66ee1f1213fb60bb18a9d060b70cd86 bytes=9420
PASS  station-snap    console_backup_260902_1642.txt  sha256=55ef5eb57aeab3c5689aceeaad074dc16cd11a826f7d537cbbdc393082793b2d bytes=11926
PASS  station-snap    console_backup_260902_1658.txt  sha256=f31c0e5097f5aef3e85fdbea1ab59321d205a5d155590a8502de9424b00d0393 bytes=11131
PASS  station-snap    console_backup_260903_0301.txt  sha256=e5152f80fa3d14829ec88aced6a4c55646cab1ab365e7b97c7f2142c20e60d80 bytes=79894
PASS  station-snap    console_backup_260903_1703.txt  sha256=a74c898a639d07227aa167a0bf465d3e64eccf667ee3771399a1878f00a8acaa bytes=262143
PASS  station-snap    console_backup_260903_1704.txt  sha256=d6fcc1e727a76e159581737858dba6ab1e998e71d108e7080c6f455c58bafa33 bytes=4003
PASS  station-snap    console_backup_260903_1858.txt  sha256=619a367571342b508c8351682cc6d06b0ae7047d6a813f291f5ffb535d62f196 bytes=262143
PASS  station-snap    console_backup_260903_1916.txt  sha256=663b233ae079d0ea08b902ad1d42311a32f87223b5b4f8358775f39d00a03931 bytes=262143
PASS  station-snap    console_backup_260904_0028.txt  sha256=a6e9febfde7c7d3259f21f2bcb78696084f526de4c515fa62a1f779bbee4f593 bytes=262143
station-snapshot: 11 file(s) copied; manifest.json written to /tmp/tmp.oLs91hEag2
```

Exit: 0. Files: 1 × config.bog (35 KB) + 10 × console_backup_*.txt (4 KB–256 KB each).
Executable outputs: `find <out> -type f -perm -u+x | wc -l` = **0** (fixed).
File modes: `-rw-r--r--` (0644) on all 11 outputs.
Source unchanged: `find PANCCADIA -newer manifest.json` = empty (no writes).
No history/ directory in this station → pointers array empty.
Temp out-dir deleted afterwards; nothing written under `/mnt/c`.

---

## Proposed kit deltas

| Delta | File | Status |
|-------|------|--------|
| New script | `build-n4-module-kit/toolbelt/station-snapshot.sh` | SHIPPED |
| K19 routing (BUILD-LOOP.md) | Post-deploy section, before triage-console line | SHIPPED |
| K19 routing (skill/SKILL.md) | Toolbelt list, next to schema-risk | SHIPPED |

---

## Design deviations

### D1: manifest format — `.json` not `.txt` (K13: RED wins)

**Design (D10)** says `manifest.txt` with `sha256  relpath  bytes` (space-separated
plain text). **RED (`823c032` SN2)** checks for `manifest.json` with JSON keys
`"ts"` or `"timestamp"`. Since K13 makes the RED authoritative, `manifest.json`
is the implemented format. The JSON shape follows the design's structural intent
(`station`, `ts`, `files[]`, `pointers[]`) while satisfying every SN2 assertion.

### D2: flat output — no `<station>-<stamp>/` subdirectory (K13: RED wins)

**Design (D10)** says `<out-dir>/<station-name>-<UTC stamp>/`. **RED (SN1–SN3)**
checks for files at `$OUT/config.bog` directly (flat in `$OUT`). Since K13 makes
the RED authoritative, the script writes flat to `<out-dir>`. The caller controls
directory naming by choosing `<out-dir>` explicitly (e.g. `outdir="$base/$name-$stamp"`;
`station-snapshot.sh <station> "$outdir"`). The stamp is still recorded in `manifest.json`
via the `ts` field.

---

## Lessons

1. When design and RED disagree on output path structure, the RED's concrete filesystem assertions are the ground truth — K13 resolves this cleanly without ambiguity.
2. Accumulating JSON entries in per-file temp files and joining with awk comma-separation avoids shell variable escaping issues with arbitrary sha256 values and filenames.
3. A `find -newer <manifest>` check after the snapshot confirms zero source writes; the manifest being the last file written makes it a reliable "after" timestamp marker.
4. `cp -p` preserves mtimes from NTFS-mounted Windows directories; the output files show the original Windows timestamps, not the copy time — callers should not depend on output file mtime for ordering.
