#!/usr/bin/env bash
# lib/signing-block.sh — the ONE table of the root signing-block requirements (sourced only, never invoked).
# Consumers: preflight.sh sample-sign, build.sh mirror dev-key WARN; lint-structure L15 is the Go rule
# go/internal/lint/rules/structure, whose signBlock mirrors this table entry by entry (pinned by
# tests/lint-structure.bats LS15-sync, which fails on any drift). Each matches
# SIGN_BLOCK_ERE[k] against the root build.gradle.kts code with comments removed; SIGN_BLOCK_NAME[k] names it
# in the consumer's row. [0] (allowDefaultProfile off) is the one that stops the plugin's default-profile
# fallback: without it a missing profile or a failed HSM/token login signs with the self-signed default key
# (issue #266); with it the build fails instead (templates/signing-root.gradle.kts).
# allowDefaultProfile takes the .set(false) call or the Kotlin-DSL `= false` assignment.
# [ev: retro n4-course-signed-jars-gate-fix Δ4] [ev: code devkit-wb.jar!gradle/build.gradle.kts.vm:18]
# shellcheck disable=SC2034  # read by the sourcing script
SIGN_BLOCK_ERE=('allowDefaultProfile[[:space:]]*(\.set\([[:space:]]*false[[:space:]]*\)|=[[:space:]]*false([^[:alnum:]_]|$))'
  'niagaraSigning[[:space:]]*\{' '(^|[^[:alnum:]_])aliases([^[:alnum:]_]|$)' 'signingProfileFile')
# shellcheck disable=SC2034
SIGN_BLOCK_NAME=('allowDefaultProfile.set(false) (or = false)' 'niagaraSigning {' 'aliases' 'signingProfileFile')
