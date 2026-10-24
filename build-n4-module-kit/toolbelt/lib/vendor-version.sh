#!/usr/bin/env bash
# lib/vendor-version.sh — the ONE reader and comparator for a module's own vendorVersion.
# Sourced only (never invoked directly) by toolbelt/build.sh (deployed-baseline drift gate) and scripts/ng-deploy.sh
# (version-bump precheck). Fragment rule: edit THIS shared fragment, never re-implement the read or the compare in
# a consumer — mirrors lib/slotomatic-skip.sh's fragment rule (follow-up to kit #437).
#
# vv_module_version <jar> — prints the MODULE's own vendorVersion: the one on the <module ...> root tag of
#   META-INF/module.xml, NOT a <dependency name="baja" vendorVersion="..."/> floor (METHODOLOGY.md "two version
#   concepts"). Two OR more numeric components (Niagara accepts an X.Y module version, chihuahua "1.3"), plus any
#   suffix. Prints nothing (status 0) when the jar or the version cannot be read: callers fail closed on empty.
vv_module_version() {
  { unzip -p "$1" META-INF/module.xml 2>/dev/null \
      | grep -oE '<module [^>]*vendorVersion="[0-9]+(\.[0-9]+)+[^"]*"' \
      | head -1 | sed -E 's/.*vendorVersion="([^"]*)".*/\1/'
  } || true
}

# vv_cmp <a> <b> — prints -1 (a < b), 0 (same version) or 1 (a > b), always status 0. Each dotted component
#   compares as a NUMBER (1.10 > 1.9), a missing component is 0 (1.3 == 1.3.0, 1.03 == 1.3). A non-numeric suffix
#   after the dotted number (1.3-rc) must match exactly; a differing suffix, an empty side or a version with no
#   leading number prints "incomparable" — the caller decides (build.sh: a changed version; ng-deploy: exit 52).
#   One awk, no per-component fork.
vv_cmp() {
  awk -v a="$1" -v b="$2" 'BEGIN {
    sa = a; sub(/^[0-9]+(\.[0-9]+)*/, "", sa); na = substr(a, 1, length(a) - length(sa))
    sb = b; sub(/^[0-9]+(\.[0-9]+)*/, "", sb); nb = substr(b, 1, length(b) - length(sb))
    if (na == "" || nb == "" || sa != sb) { print "incomparable"; exit }
    la = split(na, x, "."); lb = split(nb, y, "."); n = (la > lb) ? la : lb
    for (i = 1; i <= n; i++) {
      if ((x[i] + 0) < (y[i] + 0)) { print -1; exit }
      if ((x[i] + 0) > (y[i] + 0)) { print 1; exit }
    }
    print 0
  }'
}
