#!/usr/bin/env bats
# method-boundary.bats — unit tests for the literal handling of mb_strip and mb_parse (toolbelt/lib/method-boundary.sh).
# mb_strip blanks // and /* */ comments; a // or /* inside a Java string, char literal or text block is
# code, not a comment. Before the fix it was read as a comment and the rest of the code was dropped
# (fail-open in every consumer lint). Each case runs under every awk found on the host (the lints call
# plain `awk`; mawk and gawk --posix are checked when present).
# Mutation: MB-lit -- drop the literal state from mb_strip and MB1, MB2, MB4, MB5 and MB-e2e go red
# Mutation: MB-depth -- count braces inside literals in mb_parse and MB9 goes red
# shellcheck disable=SC2016  # awk program text in single quotes — $ belongs to awk, not shell

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  # shellcheck disable=SC1091
  . "$KIT/toolbelt/lib/method-boundary.sh"
  AWKS=("awk")
  command -v mawk >/dev/null 2>&1 && AWKS+=("mawk")
  command -v gawk >/dev/null 2>&1 && AWKS+=("gawk --posix")
  IN="$BATS_TEST_TMPDIR/in.java"; WANT="$BATS_TEST_TMPDIR/want.txt"
}

# strip_with <awk command> — mb_strip over $IN, one "<line>|<stripped>" row per input line
strip_with() {
  # shellcheck disable=SC2086  # the awk command carries its own flags
  $1 "$MB_AWK"' { l[++n] = $0 } END { mb_strip(l, n, o); for (i = 1; i <= n; i++) print i "|" o[i] }' "$IN"
}

# check — every awk must turn $IN into exactly $WANT
check() {
  local a got
  for a in "${AWKS[@]}"; do
    got=$(strip_with "$a")
    if [ "$got" != "$(cat "$WANT")" ]; then
      printf 'awk=%s\n--- want\n%s\n--- got\n%s\n' "$a" "$(cat "$WANT")" "$got"; return 1
    fi
  done
}

@test "MB1: a // inside a string (URL) is code; the statement after it is kept" {
  printf '%s\n' 'String u = "tcp://host"; doWork();' > "$IN"
  printf '%s\n' '1|String u = "tcp://host"; doWork();' > "$WANT"
  check
}

@test "MB2: a /* inside a string does not open a block comment" {
  printf '%s\n' 'String s = "/*"; a();' 'b();' 'c(); // x */' > "$IN"
  printf '%s\n' '1|String s = "/*"; a();' '2|b();' '3|c(); ' > "$WANT"
  check
}

@test "MB3: char literals '/', '\"' and '\\'' are code; the real // after them is still stripped" {
  printf '%s\n' "char c = '/'; d(); // t" "char q = '\"'; e(); // t" "char a = '\\''; f(); // t" > "$IN"
  printf '%s\n' "1|char c = '/'; d(); " "2|char q = '\"'; e(); " "3|char a = '\\''; f(); " > "$WANT"
  check
}

@test "MB4: escaped quotes and backslashes inside a string; only the real comment is stripped" {
  printf '%s\n' 'String s = "say \"hi\" // not"; g(); // real' 'String b = "a\\"; h(); // c' > "$IN"
  printf '%s\n' '1|String s = "say \"hi\" // not"; g(); ' '2|String b = "a\\"; h(); ' > "$WANT"
  check
}

@test "MB5: a text block spanning lines keeps its // and /* as content; code after it is stripped normally" {
  printf '%s\n' 'String t = """' '  http://x /* not a comment' '  a \""" still inside' '  """; k(); // c' 'm(); // d' > "$IN"
  printf '%s\n' '1|String t = """' '2|  http://x /* not a comment' '3|  a \""" still inside' '4|  """; k(); ' '5|m(); ' > "$WANT"
  check
}

@test "MB6: real // and /* */ comments (one-line and multi-line) are stripped as before" {
  printf '%s\n' 'x(); // c' '/* a' 'b */ y();' 'p(); /* q */ r();' '/** "doc" // */ z();' > "$IN"
  printf '%s\n' '1|x(); ' '2|' '3| y();' '4|p();  r();' '5| z();' > "$WANT"
  check
}

@test "MB7: line count and numbering are preserved; an unterminated string does not leak to the next line" {
  printf '%s\n' 'String s = "abc' 'n(); // c' '' '"x"; /* open' 'still comment "' '*/ q();' > "$IN"
  printf '%s\n' '1|String s = "abc' '2|n(); ' '3|' '4|"x"; ' '5|' '6| q();' > "$WANT"
  check
}

@test "MB8: mb_strip returns the open state at end of file (0 closed, 1 block comment, 2 text block)" {
  local a got
  for a in "${AWKS[@]}"; do
    got=""
    for body in 'a(); "x"; // c' 'b(); /* open' 'String t = """'; do
      printf '%s\n' "$body" > "$IN"
      # shellcheck disable=SC2086
      got="$got$($a "$MB_AWK"' { l[++n] = $0 } END { print mb_strip(l, n, o) }' "$IN")"
    done
    [ "$got" = "012" ] || { echo "awk=$a got=$got"; return 1; }
  done
}

@test "MB9: mb_parse ignores braces inside string and char literals (a \"}\" no longer ends the method early)" {
  # Mutation: MB-depth -- counting the "}" and '}' literals as braces ends method a at line 3 and loses b
  printf '%s\n' 'class A {' '  void a() {' '    String s = "}";' "    char c = '}';" '    x();' '  }' '  void b() {' '    y();' '  }' '}' > "$IN"
  local a got
  for a in "${AWKS[@]}"; do
    # shellcheck disable=SC2086
    got=$($a "$MB_AWK"' { l[++n] = $0 } END { mb_strip(l, n, o); c = mb_parse(o, n, s, e, m)
      for (k = 0; k < c; k++) printf "%s %d %d|", m[k], s[k], e[k] }' "$IN")
    [ "$got" = "a 2 6|b 7 9|" ] || { echo "awk=$a got=$got"; return 1; }
  done
}

@test "MB-e2e: lint-size counts the lines after a \"/*\" string (they were blanked as a comment before)" {
  # Mutation: MB-lit -- with no literal state the 801 lines after the string are blanked and the class passes
  SRC="$BATS_TEST_TMPDIR/Mod/src/com/x"; mkdir -p "$SRC"
  { printf 'package com.x;\npublic class Glob {\n  String g = "/*";\n'
    for ((i = 1; i <= 801; i++)); do printf '  int f%d;\n' "$i"; done; printf '}\n'; } > "$SRC/Glob.java"
  run "$KIT/toolbelt/lint-size.sh" "$BATS_TEST_TMPDIR/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  lint-size  "*"Glob.java:2  class Glob has 805 hand-written lines (> 800"* ]]
}
