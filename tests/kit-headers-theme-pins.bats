#!/usr/bin/env bats
# Kit doc pins for web headers and theme modules (register C13, C30 — WU-P0-8). Each pin bans a statement the
# source contradicts, so a later edit cannot bring it back. retros/, odd/ and BUILD-STATE.md are evidence and
# are not scanned; types/ is.
#   KH1  station header providers reach every module servlet: TridiumSecurityFilter on /* of each servlet context
#        (TridiumSecurityFilter.java:22-27, BJettyWebServer.java:1178-1180,1462-1468; corpus B1349 C1), so the
#        "DashboardPan omits X-Frame/nosniff/CSP" gap and a per-servlet CSP snippet (it REPLACES the station
#        policy) are wrong
#   KH2  embedding uses CSP frame-ancestors on the station provider, never X-Frame-Options Any
#        (guide User/web-HttpHeaderProviders.txt:20; corpus B1349 §1349.1)
#   KH3  the themeName def value is the module name (module://<value>/nss), not a display name
#        (CustomThemeModuleManager.java:24,47; corpus B1328 §1328.2 C1)
#   KH4  no "Platform → User Interface → Look and Feel" theme picker exists (corpus B1328 §1328.2 C1)
# [ev: retro devguide415-r3-web-session-headers Δ3] [ev: retro devguide415-r3-web-session-headers Δ5]
# [ev: retro devguide415-r2-web-ui-hmi Δ2]

setup() {
  DOCS="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit/types"
}

@test "KH1: no kit doc names the DashboardPan header gaps ODA2-G1/G2 or ships a per-servlet CSP snippet" {
  run grep -rnE 'ODA2-G[12]|resp\.setHeader\("Content-Security-Policy"' "$DOCS"
  [ "$status" -eq 1 ]
}

@test "KH2: no kit doc recommends X-Frame-Options ALLOW or Any for embedding" {
  run grep -rniE 'X-Frame-Options[^|]*\b(ALLOW-FROM|ALLOWALL)\b|set X-Frame[^.]*\bAny\b' "$DOCS"
  [ "$status" -eq 1 ]
}

@test "KH3: no kit doc calls the themeName def value a display name" {
  run grep -rniE 'value is the display' "$DOCS"
  [ "$status" -eq 1 ]
}

@test "KH4: no kit doc places the theme picker under Platform → User Interface" {
  run grep -rnE 'Platform → User Interface' "$DOCS"
  [ "$status" -eq 1 ]
}
