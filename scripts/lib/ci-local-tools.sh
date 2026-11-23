#!/usr/bin/env bash
# shellcheck shell=bash
# lib/ci-local-tools.sh — the pinned tools of scripts/ci-local.sh: shellcheck v0.10.0 and the bats-core version
# ci.yml installs. Sourced only, after lib/sha256.sh, with CACHE (the tarball cache dir) and WORK (a private temp
# dir) set by the caller. Each tool is a sha256-checked tarball cached under CACHE, unpacked into WORK and put
# first on PATH; never installed system-wide, never sudo. Every failure exits 3.
# [ev: retro 2026-10-22-ci-local-replay] (retro 2026-11-15 coordination-stretch C8)

SC_VER=v0.10.0
SC_TAR="shellcheck-${SC_VER}.linux.x86_64.tar.xz"
SC_URL="https://github.com/koalaman/shellcheck/releases/download/${SC_VER}/${SC_TAR}"
# Computed 2026-10-22 from a download of SC_URL (2,404,716 bytes).
SC_SHA256=6c881ab0698e4e6ea235245f22832860544f17ba386442fe7e9d629f8cbedf87

# fetch_pinned <tarball name> <url> <sha256> — make CACHE/<name> hold the pinned bytes (download on first use)
# and print its path. A cached file with another digest is refused (remove it to re-download); a fresh download
# with another digest is deleted. The download is bounded: 10 s to connect and 120 s per try, and no retry starts
# after 60 s (--max-time is per try, so --retry 3 alone allowed four full tries, ~8 minutes); a blackholed network
# ends in exit 3 within about 3 minutes instead of hanging. (retro 2026-11-15 coordination-stretch T32, T36)
fetch_pinned() {
  local tar="$CACHE/$1" src got
  src="$tar"
  if [ ! -f "$tar" ]; then
    src="$tar.partial"
    mkdir -p "$CACHE"
    curl -fsSL --retry 3 --retry-max-time 60 --connect-timeout 10 --max-time 120 -o "$src" "$2" || { rm -f "$src"; printf 'ci-local: download failed: %s\n' "$2" >&2; exit 3; }
  fi
  got="$(sha256_of "$src")"
  if [ "$got" != "$3" ]; then
    [ "$src" = "$tar" ] || rm -f "$src"
    printf 'ci-local: sha256 mismatch for %s (got %s, pinned %s); remove it to re-download\n' "$src" "$got" "$3" >&2
    exit 3
  fi
  [ "$src" = "$tar" ] || mv "$src" "$tar"
}

# provision_shellcheck — verify (and on first use download) the pinned tarball, unpack it into $WORK, PATH it.
provision_shellcheck() {
  sha256_require ci-local || exit 3
  [ "$(uname -s)/$(uname -m)" = Linux/x86_64 ] || { printf 'ci-local: the pinned shellcheck is linux x86_64 only\n' >&2; exit 3; }
  fetch_pinned "$SC_TAR" "$SC_URL" "$SC_SHA256"
  tar -xJf "$CACHE/$SC_TAR" -C "$WORK"
  PATH="$WORK/shellcheck-${SC_VER}:$PATH"
  printf 'ci-local: %s\n' "$(shellcheck --version | grep '^version:')"
}

# bats_pin — set BATS_PIN_VER and BATS_PIN_SHA from the BATS_VER= / BATS_SHA256= lines of the workflow's "Install bats*"
# step (`ci-steps bats-pin`, go/internal/cisteps/batspin.go: the reading is Go, with a test per failure). Both are ""
# when the workflow has no such step. A step whose pin is missing, malformed or set twice exits 3 with the parser's
# message. Needs CI_STEPS (the ci-steps binary) and CI_FILE, which ci-local.sh sets.
bats_pin() {
  local out rc=0
  out="$("$CI_STEPS" bats-pin "$CI_FILE")" || rc=$?
  [ "$rc" -eq 0 ] || exit 3
  BATS_PIN_VER="${out%% *}" BATS_PIN_SHA=""
  [ -z "$out" ] || BATS_PIN_SHA="${out#* }"
}

# provision_bats — run the bats-core version ci.yml installs (bats_pin): CI's bats 1.10.0 refuses a file that bats 1.14
# accepts (a here-document line that starts with the test keyword is read as a duplicate case). The bats on PATH is
# used when it already is that version; otherwise the pinned tarball is fetched and its bin/ put first on PATH. With
# no install step (a fixture workflow) the bats on PATH runs, named on stderr. GNU parallel for -j still comes from
# PATH.
provision_bats() {
  bats_pin
  local ver="$BATS_PIN_VER" sha="$BATS_PIN_SHA"
  if [ -z "$ver" ]; then
    printf 'ci-local: the workflow installs no bats; using the bats on PATH\n' >&2
    return 0
  fi
  if [ "$(bats --version 2>/dev/null)" = "Bats $ver" ]; then
    printf 'ci-local: bats %s (on PATH, the CI pin)\n' "$ver"
    return 0
  fi
  sha256_require ci-local || exit 3
  fetch_pinned "bats-core-v${ver}.tar.gz" "https://github.com/bats-core/bats-core/archive/refs/tags/v${ver}.tar.gz" "$sha"
  tar -xzf "$CACHE/bats-core-v${ver}.tar.gz" -C "$WORK"
  PATH="$WORK/bats-core-${ver}/bin:$PATH"
  printf 'ci-local: bats %s (pinned by ci.yml, from %s)\n' "$ver" "$CACHE"
}
