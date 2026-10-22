#!/usr/bin/env bats
# Kit doc pins for nav files, ORDs and tags (register C17, C18, C21 — WU-P0-6). Each pin bans a statement the
# source contradicts, so a later edit cannot bring it back. retros/, odd/ and BUILD-STATE.md are evidence and
# are not scanned.
#   KN1  NavFileDecoder throws unless the root is <nav version="1.0"> (NavFileDecoder.java:130-133, corpus B1307 §1307.2 C1)
#   KN2  nothing auto-discovers .nav files; BUser.navFile names one (corpus B1307 §1307.2 C2, B1317 §1317.2)
#   KN3  a nav: ORD needs a nav-node base: root:|nav:a/b (BNavScheme.java:80-124, corpus B1307 §1307.2 C3)
#   KN4  tag-dictionary API names that do not exist: getRules(), SmartTagRule, tagInfoList.add, NEQL tag::
#        (BSmartTagDictionary.java:71 getTagRules(), BTagDictionary.java:107-114; corpus B1311 §1311.2 C1-C2)
#   KN5  BOrd.make only wraps the string; parsing happens at resolve (BOrd.java:32-34, corpus B1308 §1308.2 C1)
# [ev: retro devguide415-nav-subscriptions-tags-ords Δ5] [ev: retro devguide415-r2-runtime-contracts-formats Δ5]
# [ev: retro devguide415-nav-subscriptions-tags-ords Δ9]

setup() {
  DOCS="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit/types"
}

@test "KN1: every <nav> root in the kit docs carries version=\"1.0\"" {
  run grep -rnE '<nav( [^v][^>]*)?>' "$DOCS"
  [ "$status" -eq 1 ]
}

@test "KN2: no kit doc says Workbench or the station discovers .nav files" {
  run grep -rniE 'discover[a-z]* (the )?`?\.nav`? files' "$DOCS"
  [ "$status" -eq 1 ]
}

@test "KN3: every nav: ORD example has a base (root:|nav:...)" {
  run grep -rnE '`nav:[A-Za-z/]' "$DOCS"
  [ "$status" -eq 1 ]
}

@test "KN4: no kit doc names getRules(), SmartTagRule, tagInfoList.add or a tag:: NEQL operator" {
  run grep -rnE 'getRules\(\)|SmartTagRule\[|tagInfoList\.add|tag::' "$DOCS"
  [ "$status" -eq 1 ]
}

@test "KN5: no kit doc says BOrd.make parses or normalizes" {
  run grep -rnE 'BOrd\.make\(string\)` parses|parse once; store in a field' "$DOCS"
  [ "$status" -eq 1 ]
}
