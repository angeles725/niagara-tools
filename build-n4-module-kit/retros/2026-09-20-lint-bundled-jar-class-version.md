<!-- review-status: folded -->
# 2026-09-20 · kit · lint-bundled-jar-class-version (L3 — lint-candidate impl, auto-wired)

**Session**: implement-lint-candidates campaign, L3. Partial promotion of the `bundled-jar-class-version` CANDIDATE from the folded retro module-hardening-failure-modes-deltas Δ11 (BLD2, types/issues-and-gotchas.md).

**Delta count**: 0 (implementation retro)

## What was done
New `toolbelt/lint-bundled-jar-class-version.sh` (FAIL, module-once): scans vendored `*.jar` under the module root (prunes `build/` + dot-dirs), samples up to 20 `.class` entries per jar (skips `META-INF/versions/` for multi-release safety), reads the class-file major version and FAILs if > 52 (Java 8) — a Java 9+ bundled class throws a raw UnsupportedClassVersionError that bypasses catch(Exception) at module load. `# Mutation: BJCV2` (MATCH); tests/lint-bundled-jar-class-version.bats (6). AUTO-WIRED into report-module.sh §12 (module-once) + RM23, so build.sh runs it.

## Self-verify
shellcheck 0; lint bats 6/6; report-module.bats 25/25 (RM23); BJCV2 MATCH; full suite 0 not-ok; kit-links 10/10; sweep 0. [ev: corpus build-n4-module/automation-audit]

---
**Status**: FOLDED. Per-module lint-candidate set complete (L1 subscribe, L2 no-md5, L3 bundled-jar, L5 wb-file-chooser); L4 split-package deferred to project/CI level.
