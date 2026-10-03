#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/lint-session-store-lazy-evict.sh
# A static session/token Map evicted only lazily (query-time) or via explicit logout, with no
# sweep-on-insert or scheduled purge, is the UXS7 session-fixation shape.
# [ev: retro live-diagnosis-hardening-deltas Δ5]

setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  SSL="$KIT/toolbelt/lint-session-store-lazy-evict.sh"
  mkdir -p "$TMPDIR_T/Mod/src/com/x"
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "SSL-usage: no arg exits 3" { run "$SSL"; [ "$status" -eq 3 ]; }

@test "SSL-nondir: a non-directory arg exits 3" { run "$SSL" "$TMPDIR_T/nope"; [ "$status" -eq 3 ]; }

@test "SSL1: static Map put on login, removed only via explicit logout -> WARN" {
  {
    printf 'class E {\n'
    printf '  private static final Map<String, Entry> SESSIONS = new ConcurrentHashMap<String, Entry>();\n'
    printf '  void login(String id) {\n    SESSIONS.put(id, new Entry());\n  }\n'
    printf '  void logout(String id) {\n    SESSIONS.remove(id);\n  }\n'
    printf '}\n'
  } > "$TMPDIR_T/Mod/src/com/x/E.java"
  run "$SSL" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]]
  [[ "$output" == *"lint-session-store-lazy-evict"* ]]
  [[ "$output" == *"SESSIONS"* ]]
}

@test "SSL1-strict: --strict promotes the WARN to exit 1" {
  {
    printf 'class E {\n'
    printf '  private static final Map<String, Entry> SESSIONS = new ConcurrentHashMap<String, Entry>();\n'
    printf '  void login(String id) {\n    SESSIONS.put(id, new Entry());\n  }\n'
    printf '  void logout(String id) {\n    SESSIONS.remove(id);\n  }\n'
    printf '}\n'
  } > "$TMPDIR_T/Mod/src/com/x/E.java"
  run "$SSL" --strict "$TMPDIR_T/Mod"
  [ "$status" -eq 1 ]
}

@test "SSL2: an INSTANCE-scope (non-static) map is clean -- static is required" {
  {
    printf 'class ConfigSession {\n'
    printf '  private final Map<String, Entry> map = new HashMap<String, Entry>();\n'
    printf '  synchronized String issue(String id, String user) {\n    map.put(id, new Entry(user, 0));\n    return user;\n  }\n'
    printf '  synchronized void revoke(String id) {\n    map.remove(id);\n  }\n'
    printf '}\n'
  } > "$TMPDIR_T/Mod/src/com/x/ConfigSession.java"
  run "$SSL" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
  # Named mutation: drop the `static` requirement -> SSL2 false-WARNs on this fixture.
}

@test "SSL3: sweep-on-insert in the SAME method as put() suppresses the WARN" {
  {
    printf 'class F {\n'
    printf '  private static final Map<String, Entry> SESSIONS = new ConcurrentHashMap<String, Entry>();\n'
    printf '  void login(String id) {\n'
    printf '    for (String k : SESSIONS.keySet()) if (expired(k)) SESSIONS.remove(k);\n'
    printf '    SESSIONS.put(id, new Entry());\n'
    printf '  }\n}\n'
  } > "$TMPDIR_T/Mod/src/com/x/F.java"
  run "$SSL" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

@test "SSL4: no static Map field at all is clean" {
  printf 'class G {\n  void login(String id) {\n    doStuff(id);\n  }\n}\n' > "$TMPDIR_T/Mod/src/com/x/G.java"
  run "$SSL" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

@test "SSL-awkfail: an unreadable source file is an env error (exit 3, named on stderr), never a clean pass" {
  # Mutation: SSL-awkfail -- ignoring the awk exit status reports the unreadable file as clean (exit 0).
  mkdir -p "$(dirname "$TMPDIR_T/Mod/src/com/x/U.java")"
  printf 'x\n' > "$TMPDIR_T/Mod/src/com/x/U.java"
  chmod 000 "$TMPDIR_T/Mod/src/com/x/U.java"
  if [ -r "$TMPDIR_T/Mod/src/com/x/U.java" ]; then chmod 644 "$TMPDIR_T/Mod/src/com/x/U.java"; skip "running as root: chmod 000 does not block reads"; fi
  run "$SSL" "$TMPDIR_T/Mod"
  chmod 644 "$TMPDIR_T/Mod/src/com/x/U.java"
  [ "$status" -eq 3 ]
  [[ "$output" == *"Mod/src/com/x/U.java"* ]]
}

@test "SSL-finderr: an unreadable sub-directory is an env error (exit 3, named on stderr), never a clean pass" {
  # Mutation: SSL-finderr -- ignoring the find exit status skips the unreadable directory's files and exits 0.
  # [ev: issue #226 R3-find-error-still-fail-open]
  mkdir -p "$TMPDIR_T/Mod/src/com/x/locked"
  printf 'class A {}\n' > "$TMPDIR_T/Mod/src/com/x/A.java"
  printf 'class B {}\n' > "$TMPDIR_T/Mod/src/com/x/locked/B.java"
  chmod 000 "$TMPDIR_T/Mod/src/com/x/locked"
  if [ -r "$TMPDIR_T/Mod/src/com/x/locked" ]; then chmod 755 "$TMPDIR_T/Mod/src/com/x/locked"; skip "running as root: chmod 000 does not block reads"; fi
  run "$SSL" "$TMPDIR_T/Mod"
  chmod 755 "$TMPDIR_T/Mod/src/com/x/locked"
  [ "$status" -eq 3 ]
  [[ "$output" == *"Mod/src/com/x/locked"* ]]
}

@test "SSL5: an unrelated Clock.schedule does not count as a purge when only logout removes" {
  # Mutation: SSL5 -- file-wide co-occurrence (any Clock.schedule + any remove) hides the logout-only shape.
  {
    printf 'class E {\n'
    printf '  private static final Map<String, Entry> SESSIONS = new ConcurrentHashMap<String, Entry>();\n'
    printf '  void started() {\n    Clock.schedule(this, BRelTime.makeSeconds(5), refresh, null);\n  }\n'
    printf '  public void doRefresh() {\n    repaint();\n  }\n'
    printf '  void login(String id) {\n    SESSIONS.put(id, new Entry());\n  }\n'
    printf '  void logout(String id) {\n    SESSIONS.remove(id);\n  }\n'
    printf '}\n'
  } > "$TMPDIR_T/Mod/src/com/x/E.java"
  run "$SSL" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"E.java:10"* ]]
  [[ "$output" == *"SESSIONS"* ]]
}

@test "SSL6: a scheduled action whose do<Action>() body removes from the map is a purge (clean)" {
  {
    printf 'class E {\n'
    printf '  private static final Map<String, Entry> SESSIONS = new ConcurrentHashMap<String, Entry>();\n'
    printf '  void started() {\n    Clock.schedulePeriodically(this, BRelTime.makeMinutes(1), sweep, null);\n  }\n'
    printf '  public void doSweep() {\n    for (String k : SESSIONS.keySet()) if (expired(k)) SESSIONS.remove(k);\n  }\n'
    printf '  void login(String id) {\n    SESSIONS.put(id, new Entry());\n  }\n'
    printf '  void logout(String id) {\n    SESSIONS.remove(id);\n  }\n'
    printf '}\n'
  } > "$TMPDIR_T/Mod/src/com/x/E.java"
  run "$SSL" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}
