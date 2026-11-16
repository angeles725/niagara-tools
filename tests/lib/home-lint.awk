# home-lint.awk — the SH2 lint of tests/shell-hygiene.bats (run by its _home_lint).
# awk -v names="<script basenames>" -f home-lint.awk <bats-file>...: one "<file>:<line>: ..." row per command
# that runs a named script with neither --home nor HOME=; exit 1 when there is any.
#
# A script is run when its path (or a variable assigned that path in the same file, also after local or export
# or inside a one-line function) stands in command position: first on the command, first inside $( ) or
# backticks, or after run, bash, sh, env, exec, time, timeout, if, elif, while, until, !, {, then, do, else,
# &&, ||, |, a token ending in ; ( or $(, their flags and VAR=value prefixes. A path passed to cp, grep, cat or a
# test is not, an assignment is not, and a here-document body is data. The --home and HOME= tests read the
# command without its trailing comment. The forms the lint is held to: tests/lib/home-lint-forms.tsv.
#
# Accepted limits (the lint stays silent; a review catches them): a run through command, nice, stdbuf,
# xargs, eval, /usr/bin/bash or an unknown helper; an array command; a name built as scripts/$NAME; a path
# without scripts/ (cd scripts; ./x.sh); a single-quoted path; the Go binary run directly; a here-document fed
# to a shell; HOME= set by an earlier command of the same line; --home "$HOME" (the real home, spelled out).
function strip(tok) { gsub(/^[{(!]+|^\$\(|[;)}]+$/, "", tok); gsub(/"/, "", tok); return tok }
function script_of(tok,   i, v) {           # the named script this token is, or ""
  tok = strip(tok)
  for (i = 1; i <= n_names; i++) if (tok ~ ("(^|/)scripts/" name_re[i] "$")) return name[i]
  v = tok; sub(/^\$\{?/, "", v); sub(/\}$/, "", v)
  return (tok ~ /^\$/ && (FILENAME SUBSEP v) in var) ? var[FILENAME, v] : ""
}
function launcher(tok) {
  return tok ~ /^(run|bash|\/bin\/bash|sh|\/bin\/sh|env|exec|time|timeout|if|elif|while|until|!|\{|then|do|else|&&|\|\||\||;)$/ || tok ~ /(;|\$\(|\()$/
}
function uncomment(s,   i, c, q) {          # s without its trailing comment (a # that starts a word, unquoted)
  for (i = 1; i <= length(s); i++) {
    c = substr(s, i, 1)
    if (q == "" && c == "#" && (i == 1 || substr(s, i - 1, 1) ~ /[ \t]/)) return substr(s, 1, i - 1)
    if (c == "\\" && q != "\047") i++
    else if (q == "" && (c == "\"" || c == "\047")) q = c
    else if (c == q) q = ""
  }
  return s
}
BEGIN { n_names = split(names, name, " "); for (i = 1; i <= n_names; i++) { name_re[i] = name[i]; gsub(/\./, "\\.", name_re[i]) } }
FNR == 1 { held = ""; here = "" }
here != "" { if ($0 == here) here = ""; next }               # a here-document body is data, not a command
{ line = held $0; start = (held == "") ? FNR : start; held = "" }
line ~ /\\$/ { sub(/\\$/, " ", line); held = line; next }       # a continued command is one command
line ~ /^[ \t]*#/ { next }
{ line = uncomment(line) }                                    # a comment that says --home protects nothing
match(line, /(^|[^<])<<-?[ \t]*["\047]?[A-Za-z_][A-Za-z0-9_]*["\047]?/) { here = substr(line, RSTART, RLENGTH); gsub(/[<"\047 \t-]/, "", here) }
{
  nt = split(line, t, /[ \t]+/)
  for (k = 1; k <= nt; k++) {
    tok = t[k]; inner = 0
    if (tok ~ /\$\(|`/) { sub(/.*(\$\(|`)/, "", tok); inner = 1 }   # first word of a $( ) or `...`
    else if (tok ~ /^[A-Za-z_][A-Za-z0-9_]*=/) {                    # NAME=<path>: remember the variable
      if ((s = script_of(substr(tok, index(tok, "=") + 1))) != "") var[FILENAME, substr(tok, 1, index(tok, "=") - 1)] = s
      continue
    }
    if ((s = script_of(tok)) == "") continue
    if (!inner) {
      for (p = k - 1; p >= 1 && (t[p] ~ /^-/ || (t[p] ~ /^[A-Za-z_][A-Za-z0-9_]*=/ && t[p] !~ /;$/) || t[p] ~ /^[0-9.]+s?$/); p--) ;
      if (p >= 1 && t[p] != "" && !launcher(t[p])) continue
    }
    if (line ~ /(^|[ \t])--home([ \t=]|$)/ || line ~ /(^|[ \t])HOME=/) continue
    printf "%s:%d: %s runs without --home or HOME=: it works on the real home directory\n", FILENAME, start, s
    bad = 1
  }
}
END { exit bad }
