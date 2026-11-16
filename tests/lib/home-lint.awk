# home-lint.awk — the SH2 lint of tests/shell-hygiene.bats (run by its _home_lint).
# awk -v names="<script basenames>" -f home-lint.awk <bats-file>...: one "<file>:<line>: ..." row per command
# that runs a named script with neither --home nor HOME=; exit 1 when there is any.
#
# A script is run when its path (or a variable assigned that path in the same file) stands in command position: first on the command or after run, bash, sh, env,
# exec, time, timeout, their flags and VAR=value prefixes. A path passed to cp, grep, cat or a test is not, and
# a here-document body is data.
function strip(tok) { gsub(/^[{(!]+|^\$\(|[;)}]+$/, "", tok); gsub(/"/, "", tok); return tok }
function script_of(tok,   i, v) {           # the named script this token is, or ""
  tok = strip(tok)
  for (i = 1; i <= n_names; i++) if (tok ~ ("(^|/)scripts/" name_re[i] "$")) return name[i]
  v = tok; sub(/^\$\{?/, "", v); sub(/\}$/, "", v)
  return (tok ~ /^\$/ && (FILENAME SUBSEP v) in var) ? var[FILENAME, v] : ""
}
function launcher(tok) {
  return tok ~ /^(run|bash|\/bin\/bash|sh|\/bin\/sh|env|exec|time|timeout|!|\{|then|do|else|&&|\|\||\||;)$/ || tok ~ /(;|\$\(|\()$/
}
BEGIN { n_names = split(names, name, " "); for (i = 1; i <= n_names; i++) { name_re[i] = name[i]; gsub(/\./, "\\.", name_re[i]) } }
FNR == 1 { held = ""; here = "" }
here != "" { if ($0 == here) here = ""; next }               # a here-document body is data, not a command
{ line = held $0; start = (held == "") ? FNR : start; held = "" }
line ~ /\\$/ { sub(/\\$/, " ", line); held = line; next }       # a continued command is one command
line ~ /^[ \t]*#/ { next }
match(line, /(^|[^<])<<-?[ \t]*["\047]?[A-Za-z_][A-Za-z0-9_]*["\047]?/) { here = substr(line, RSTART, RLENGTH); gsub(/[<"\047 \t-]/, "", here) }
{
  nt = split(line, t, /[ \t]+/)
  if (match(line, /^[ \t]*[A-Za-z_][A-Za-z0-9_]*=[^ \t]+[ \t]*$/)) {    # NAME=<path>: remember the variable
    eq = index(line, "="); v = line; sub(/^[ \t]*/, "", v); v = substr(v, 1, index(v, "=") - 1)
    s = script_of(substr(line, eq + 1)); gsub(/[ \t]/, "", s)
    if (s != "") var[FILENAME, v] = s
    next
  }
  for (k = 1; k <= nt; k++) {
    if ((s = script_of(t[k])) == "") continue
    for (p = k - 1; p >= 1 && (t[p] ~ /^-/ || t[p] ~ /^[A-Za-z_][A-Za-z0-9_]*=/ || t[p] ~ /^[0-9.]+s?$/); p--) ;
    if (p >= 1 && t[p] != "" && !launcher(t[p])) continue
    if (line ~ /(^|[ \t])--home([ \t=]|$)/ || line ~ /(^|[ \t])HOME=/) continue
    printf "%s:%d: %s runs without --home or HOME=: it works on the real home directory\n", FILENAME, start, s
    bad = 1
  }
}
END { exit bad }
