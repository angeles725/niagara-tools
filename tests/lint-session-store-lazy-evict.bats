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
