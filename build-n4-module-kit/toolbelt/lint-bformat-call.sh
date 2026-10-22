#!/usr/bin/env bash
# lint-bformat-call.sh — flags BFormat special calls written in a form BFormat never parses: %time(<arg>)% and
# %user% fall through to reflection and print %err:...% instead of a time or a user name.
#
# BFormat matches time() and user() only as the exact calls "time()" and "user()" (S/util/BFormat.java:443-444):
# time() takes NO argument and returns a BAbsTime to chain (%time().toDateString%), and %user% without the
# parentheses is a reflective call that fails. See types/logic.md § BFormat pattern reference.
# [ev: retro devguide415-r2-runtime-contracts-formats Δ6] [ev: corpus B1317 §1317.3]
#
# Rules (Java: // and /* */ comments blanked first; other files read as-is):
#   BFC1 time-arg — `%time(` followed by anything but `)`
#   BFC2 user     — `%user%`
# Scope: *.java, *.lexicon, *.bog (plain-XML bog), *.palette — where display formats and facets live.
#
# Usage:  lint-bformat-call.sh [--strict] <src-root>
#   Row:  WARN  lint-bformat-call  <file>:<line>  <detail>
#   Exit: 0  no WARN (or WARN without --strict) · 1  any WARN under --strict · 3  usage/env, an unscannable source file or a sub-directory find cannot enter
# VCS-free by design (kit-links L2).
# Mutation: BF1 -- dropping the time-argument rule passes %time(HH:mm)% clean
# Mutation: BF2 -- dropping the bare-user rule passes %user% clean
# Mutation: BF-ok -- dropping the not-`)` guard WARNs the valid %time()% and %time().toDateString%
# Mutation: BF-comment -- not blanking Java comments reports a pattern quoted in a comment
set -u
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/method-boundary.sh"
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/awk-lint.sh"
awk_lint_parse lint-bformat-call one 'usage: lint-bformat-call.sh [--strict] <src-root>' "$@"
printf '%s\n' "$MB_AWK" > "$AL_TMP/method-boundary.awk"
# shellcheck disable=SC2034  # read by awk_lint_run (lib/awk-lint.sh)
AL_AWK_ARGS=(-f "$AL_TMP/method-boundary.awk")

cat > "$AL_TMP/main.awk" << 'AWKEOF'
{ raw[++n] = $0 }
END {
  if (FILE ~ /\.java$/) mb_strip(raw, n, ln); else for (i = 1; i <= n; i++) ln[i] = raw[i]
  for (i = 1; i <= n; i++) {
    if (ln[i] ~ /%time\([^)]/)
      printf "WARN  lint-bformat-call  %s:%d  BFC1 time() takes no argument -- %%time(<fmt>)%% goes to reflection and prints %%err:...%%; chain instead: %%time().toDateString%% (types/logic.md BFormat)\n", FILE, i
    if (ln[i] ~ /%user%/)
      printf "WARN  lint-bformat-call  %s:%d  BFC2 %%user%% is a reflective call that fails -- write %%user()%% (types/logic.md BFormat)\n", FILE, i
  }
}
AWKEOF

awk_lint_run -name '*.java' -o -name '*.lexicon' -o -name '*.bog' -o -name '*.palette'
