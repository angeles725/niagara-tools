# shellcheck shell=bash
# cases.sh — the golden cases record.sh writes: the tests/retro-loop.bats runs (bats-*) and edge cases (edge-*).
# A case the Go port changed on purpose is a truth-* case (hand-written) and is no longer recorded here.
# Sourced by record.sh, which defines new/kit/put/rec, D (the case dir) and ENVV (the case's environment).
# shellcheck disable=SC2034  # ENVV is read by record.sh's rec

# --- tests/retro-loop.bats (KIT=<kit>, today's date) ---
new; kit kit; ENVV=(KIT="$D/kit"); rec bats-rl1-stub kit rl1slug
new; kit kit; ENVV=(KIT="$D/kit"); put kit/retros/2026-01-02-rl2slug.md 'first run\n'; rec bats-rl2-second-run kit rl2slug
new; kit kit; ENVV=(KIT="$D/kit"); rec bats-rl5-deltas3 kit rl5slug --deltas 3
RL7='<!-- build-state.v1 -->\nmodule: kit\nretro_required: true\nretro_pending: false\n<!-- /build-state.v1 -->\n<!-- build-state.v1 -->\nmodule: ModuleA\nretro_required: true\nretro_pending: false\n<!-- /build-state.v1 -->\n<!-- build-state.v1 -->\nmodule: ModuleB\nretro_required: true\nretro_pending: false\n<!-- /build-state.v1 -->\n'
new; kit kit; ENVV=(KIT="$D/kit"); put kit/BUILD-STATE.src.md "$RL7"; rec bats-rl7-kit kit rl7-kit-slug
new; kit kit; ENVV=(KIT="$D/kit"); put kit/BUILD-STATE.src.md "$RL7"; rec bats-rl7-modulea ModuleA rl7-crp-slug
new; kit kit; ENVV=(KIT="$D/kit"); rec bats-rl8-five-chars kit abcde
new; kit kit; ENVV=(KIT="$D/kit"); rec bats-rl8-six-chars kit abcdef
new; kit kit; ENVV=(KIT="$D/kit")
printf '| 2026-01-02-rl9slug.md | kit | 2026-01-02 | pending | 0 |\n' >> "$D/kit/retros/INDEX.md"; rec bats-rl9-stale-row kit rl9slug
new; kit kit; ENVV=(KIT="$D/kit"); put kit/retros/2026-01-01-legacy.md '<!-- review-status: pending -->\n# legacy\n'
rec bats-rl10-generator-fails kit rl10slug
new; kit kit; ENVV=(KIT="$D/kit"); rm "$D/kit/retros/INDEX.md"; rec bats-rl11-no-index kit rl11slug

# --- kit root resolution ---
new; kit kit; rec edge-root-positional "$D/kit" kit slugabc --date 2026-03-04
new; kit kit; ENVV=(KIT=kit); rec edge-root-kit-relative kit slugabc
new; kit kit; ENVV=(KIT="$D/nope"); rec edge-root-kit-not-a-dir "$D/kit" kit slugabc
new; kit kit; kit other; ENVV=(KIT="$D/kit"); rec edge-root-kit-beats-positional "$D/other" kitmod slugabc
new; kit kit; mkdir -p "$D/noretros"; rec edge-root-positional-without-retros "$D/noretros" kit slugabc
new; kit .; rec edge-root-default kit slugabc
new; rec edge-root-default-no-retros kit slugabc
new; kit kit; rm "$D/kit/BUILD-STATE.src.md"; ENVV=(KIT="$D/kit"); rec edge-no-build-state kit slugabc
new; kit kit; rm "$D/kit/BUILD-STATE.src.md"; mkdir "$D/kit/BUILD-STATE.src.md"; ENVV=(KIT="$D/kit")
rec edge-build-state-is-a-dir kit slugabc
# edge-empty-retros-and-state (an empty BUILD-STATE.src.md) is now the hand-edited truth-no-envelope-section.

# --- arguments ---
new; kit kit; ENVV=(KIT="$D/kit"); rec edge-usage-none
new; kit kit; ENVV=(KIT="$D/kit"); rec edge-usage-one kit
new; kit kit; rec edge-usage-root-then-one "$D/kit" kit
new; kit kit; ENVV=(KIT="$D/kit"); rec edge-date-no-value kit slugabc --date
new; kit kit; ENVV=(KIT="$D/kit"); rec edge-deltas-no-value kit slugabc --deltas
new; kit kit; ENVV=(KIT="$D/kit"); rec edge-unknown-option kit slugabc --force
new; kit kit; ENVV=(KIT="$D/kit"); rec edge-option-before-slug kit --date 2026-03-04 slugabc
new; kit kit; ENVV=(KIT="$D/kit"); rec edge-options-last-wins kit slugabc --date 2026-03-04 --deltas 1 --date 2026-05-06 --deltas 2
new; kit kit; ENVV=(KIT="$D/kit"); rec edge-deltas-leading-zeros kit slugabc --deltas 007
# edge-deltas-negative (--deltas -1, exit 1 in the bash) is now the hand-edited truth-deltas-negative.
# edge-date-escapes-retros (--date ../x wrote outside retros/ in the bash) is now the hand-edited truth-date-escapes-retros.
new; kit kit; ENVV=(KIT="$D/kit"); rec edge-slug-bad-first-char kit _abcdef
new; kit kit; ENVV=(KIT="$D/kit"); rec edge-slug-charset kit A0.b_c-d
# edge-slug-space-after-six (kit 'abcdef ghi', exit 0 in the bash) is now the hand-edited truth-slug-space-after-six.
# edge-module-with-specials (module 'Mod $x `y`' with no section of its own) is now the hand-edited
# truth-module-with-specials: an envelope that does not flip is no PASS.

# --- the BUILD-STATE.src.md flip ---
new; kit kit; ENVV=(KIT="$D/kit")
put kit/BUILD-STATE.src.md 'retro_pending: false\n<!-- build-state.v1 -->\nmodule:   kit   # main kit\nretro_pending:false\nretro_pending: false\n<!-- /build-state.v1 -->\nretro_pending: false\n'
rec edge-flip-module-comment-first-only kit slugabc
new; kit kit; ENVV=(KIT="$D/kit")
put kit/BUILD-STATE.src.md '<!-- build-state.v1 -->\n## kit  \nmodule: other\nretro_pending: falsey # false\n<!-- build-state.v1 -->\n## kit\nretro_pending: false'
rec edge-flip-heading-then-module-noeol kit slugabc
new; kit kit; ENVV=(KIT="$D/kit")
put kit/BUILD-STATE.src.md '<!-- build-state.v1 -->\r\nmodule: kit\r\nretro_pending: false\r\n<!-- build-state.v1 -->\nmodule: other\n## kit\nretro_pending: false\n<!-- build-state.v1 -->\nmodule:kit\t\n\nretro_pending:\tfalse\n'
rec edge-flip-crlf-and-module-wins kit slugabc
# edge-flip-already-true-and-outside (the kit section already true) is now the hand-edited truth-flip-already-true.
