<!-- review-status: folded -->
# 2026-09-20 · kit · lint-no-md5-credential-digest (L2 — lint-candidate impl, auto-wired)

**Session**: implement-lint-candidates campaign, L2. Partial promotion of the `no-md5-credential-digest` lint CANDIDATE proposed in the folded retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ13 (documented in types/security.md §9).

**Delta count**: 0 (implementation retro)

## What was done
New `toolbelt/lint-no-md5-credential-digest.sh` (WARN): per-file, flags `MessageDigest.getInstance("MD5")` co-located with a credential context (BPassword/BCredentials/*password*/*credential*/*pin*) — MD5 is broken for credential hashing (use SHA-256+). `# Mutation: NMD2` guard-pin (MATCH). tests/lint-no-md5-credential-digest.bats (7). AUTO-WIRED into report-module.sh §5.21 (per-artifact, any src) + RM21 test, so it runs inside the build.sh chain — not an orphan the agent must remember. BUILD-LOOP §5 updated.

## Self-verify
shellcheck 0; lint bats 7/7; report-module.bats 23/23 (RM21); lint-guard-pins NMD2 MATCH; full suite 0 not-ok; kit-links 10/10; sweep 0. [ev: corpus build-n4-module/automation-audit]

---
**Status**: FOLDED (implementation complete).
