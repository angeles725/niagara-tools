#!/usr/bin/env bash
# Shapes that must NOT be reported (FPL2): one awk per file, a fork per FILE on a non-loop
# variable, `||`, comments, heredoc and single-quoted awk programs, a one-liner loop piped after `done`.
awk_lint_each '/x/ { print }'
while IFS= read -r f; do
  result=$(awk -f main.awk "$f")
  printf '%s\n' "$result" | grep -q '^FAIL' && FAILED=1
done < "$1"
while IFS= read -r line; do
  case "$line" in *Clock.schedule*) n=$((n + 1)) ;; esac
  # printf '%s' "$line" | grep -q only-a-comment
  [ -n "$line" ] || printf 'empty\n' | cat
  printf '%s\n' "$line" || grep -q never-piped "$1"
  awk '{ printf "%s", $line | "grep x" }' "$1"
done < "$1"
cat > main.awk <<'AWKEOF'
{ while (getline line < f) do_x(); }
{ printf "%s", $line | "sort" }
AWKEOF
awk '
  { n = split("if else do while read line done", kw, " ") }
' "$1"
x=$(printf '%s\n' "$x" | while read -r b; do printf '0x%s ' "$b"; done | sed 's/ $//')
printf '%s %s %s\n' "$x" "$n" "$FAILED"
