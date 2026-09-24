#!/usr/bin/env bash
# lib/fs-type.sh — shared filesystem-type detector for the WSL 9p/drvfs build-location gate.
# Sourced only (never invoked directly) by build.sh (Δ7 build-location precheck) and
# preflight.sh (Δ8 jar-lock SKIP-on-9p/drvfs). Fragment rule: edit THIS shared fragment,
# never re-implement filesystem-type detection separately in a consumer script — mirrors
# lib/method-boundary.sh's fragment rule. VCS-free by design (kit-links.bats L2 scope is
# toolbelt/*.sh only, but this file follows the same no-git discipline regardless).
#
# fs_type_for <path>  — prints the filesystem type backing <path> on stdout (empty string
# if it cannot be determined). Never fails (always exits 0), so a caller under `set -e` can
# safely use it inside a command substitution.
#
# Real detection: `df -T -- <path>` (GNU coreutils; the TYPE column IS the fstype, e.g.
# ext4, 9p, drvfs).
#
# Test override: when N4_FSTYPE_MOUNTS_FILE points at a /proc/mounts-formatted file
# (whitespace-separated columns: device mountpoint fstype ...), the LONGEST matching
# mountpoint PREFIX wins instead of shelling out to `df -T` — so bats tests can simulate a
# 9p/drvfs mount without depending on the host's real mounts.
# [ev: retro panccadia-defrost-sequencing-hmi-reload-deltas Δ7]
# [ev: retro panccadia-defrost-sequencing-hmi-reload-deltas Δ8]
fs_type_for() {
  local path="$1"
  {
    if [ -n "${N4_FSTYPE_MOUNTS_FILE:-}" ] && [ -f "$N4_FSTYPE_MOUNTS_FILE" ]; then
      awk -v p="$path" '
        { if (index(p, $2) == 1 && length($2) > best_len) { best_len = length($2); best_type = $3 } }
        END { print best_type }
      ' "$N4_FSTYPE_MOUNTS_FILE"
    else
      df -T -- "$path" 2>/dev/null | awk 'NR==2 {print $2}'
    fi
  } || true
}

# is_9p_or_drvfs <path>  — true (exit 0) when <path>'s filesystem is a WSL 9p or drvfs mount
# (a Windows-side path reached via /mnt/<drive> in WSL). Never raises — a path that cannot be
# resolved simply reports "not 9p/drvfs" (exit 1), never a false positive.
is_9p_or_drvfs() {
  case "$(fs_type_for "$1")" in
    9p | drvfs) return 0 ;;
    *) return 1 ;;
  esac
}
