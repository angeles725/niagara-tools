#!/usr/bin/env bats
# scaffold-module: the login a dashboard module ships (issue #847; types/dashboard.md § The module ships its own login).
# Shared setup (the Go binary, KIT, SC, the fixture and golden paths, OUT): tests/lib/scaffold-setup.bash.

load lib/scaffold-setup

# login_root <type>: scaffold ColdBox (vendor Acme) of <type> and print its -rt directory.
login_scaffold() {
  run "$SC" ColdBox "$OUT/$1" --type "$1" --vendor Acme
  [ "$status" -eq 0 ]
  RT="$OUT/$1/ColdBox/ColdBox/ColdBox-rt"
  P="$RT/src/com/acme/ColdBox"
}

@test "TC-LG1: --type dashboard emits the login template, its activation rule, the asset prefix and the pure tests in -rt" {
  # Named mutation: skip the dashboard-login overlay in overlay.go -> every file below is missing.
  login_scaffold dashboard
  for f in login/BColdBoxLoginTemplate.java login/LoginActivator.java login/LoginRequest.java BColdBoxService.java \
           login/model/LoginActivation.java login/model/LoginAssets.java login/model/LoginPage.java \
           login/model/LoginRedirect.java login/model/LoginView.java login/model/Html.java; do
    [ -f "$P/$f" ] || { echo "missing src/$f"; false; }
  done
  for f in HtmlTest LoginActivationTest LoginAssetsTest LoginPageTest LoginRedirectTest LoginViewTest TypeInitOrderTest; do
    [ -f "$RT/srcTest/test/com/acme/ColdBox/login/model/$f.java" ] || { echo "missing $f"; false; }
  done
  [ -f "$RT/src/rc/coldbox-login/login.css" ]
  [ -f "$RT/src/rc/coldbox-login/login.js" ]
}

@test "TC-LG2: logic, driver and library modules get no login files" {
  for t in logic driver library; do
    login_scaffold "$t"
    run find "$RT" \( -path '*/login' -o -name 'BColdBoxService.java' -o -path '*/rc/coldbox-login' \)
    [ -z "$output" ] || { echo "$t: $output"; false; }
  done
}

@test "TC-LG3: the emitted login carries no template token and no fixture name" {
  login_scaffold dashboard
  run grep -rIlE 'ModuleA|modulea|com\.example|@[A-Z_]+@|Angeles|MinimalDash' \
    "$P/login" "$P/BColdBoxService.java" "$RT/srcTest/test/com/acme/ColdBox/login" "$RT/src/rc/coldbox-login"
  [ "$status" -eq 1 ] || { echo "$output"; false; }
}

@test "TC-LG4: the template declares INSTANCE and names no TYPE (slotomatic writes it), the source-order test reads src/" {
  login_scaffold dashboard
  T="$P/login/BColdBoxLoginTemplate.java"
  grep -q 'public static final BColdBoxLoginTemplate INSTANCE = new BColdBoxLoginTemplate();' "$T"
  run grep -n 'static final Type TYPE' "$T"
  [ "$status" -eq 1 ]
  grep -q 'extends BLoginTemplate' "$T"
  grep -q 'public BOrd resourceToOrd(String path)' "$T"
  grep -q 'LoginAssets.ord(path)' "$T"
  grep -q 'LoginRequest.pointAtDashboard(req)' "$T"
  grep -q 'BDefaultLoginTemplate.INSTANCE.write(req, resp)' "$T"
  O="$RT/srcTest/test/com/acme/ColdBox/login/model/TypeInitOrderTest.java"
  grep -q 'INSTANCE' "$O"
  grep -q 'new File("src")' "$O"
}

@test "TC-LG5: the service applies the rule from stationStarted behind a boolean applyLogin slot (default true) and saves the station" {
  login_scaffold dashboard
  S="$P/BColdBoxService.java"
  grep -q 'extends BAbstractService' "$S"
  grep -q 'name = "applyLogin"' "$S"
  grep -q 'defaultValue = "true"' "$S"
  grep -q 'public void stationStarted()' "$S"
  grep -q 'LoginActivator.apply(getApplyLogin())' "$S"
  grep -q 'BStation.save' "$S"
  grep -q 'getServiceTypes()' "$S"
}

@test "TC-LG6: the activation rule sets only on empty, stock or retired and never overwrites another template" {
  login_scaffold dashboard
  A="$P/login/model/LoginActivation.java"
  grep -q 'STOCK = "web:DefaultLoginTemplate"' "$A"
  grep -q 'KEEP_CUSTOM' "$A"
  grep -q 'RETIRED' "$A"
  grep -q 'webService\|BWebService.getMainService()' "$P/login/LoginActivator.java"
  grep -q 'web.setLoginTemplate(own)' "$P/login/LoginActivator.java"
  T="$RT/srcTest/test/com/acme/ColdBox/login/model/LoginActivationTest.java"
  for c in KEEP_CUSTOM ALREADY OFF 'web:DefaultLoginTemplate' retired; do grep -q "$c" "$T" || { echo "no $c case"; false; }; done
}

@test "TC-LG7: the dashboard path of the login is the one the emitted servlet serves" {
  login_scaffold dashboard
  grep -q 'getServletName() { return "coldbox"; }' "$OUT/dashboard/ColdBox/ColdBox/ColdBox-ux/src/com/acme/ColdBox/ux/route/BColdBoxServlet.java"
  grep -q 'DASHBOARD = "/coldbox/"' "$P/login/model/LoginRedirect.java"
  grep -q 'OURS = "/coldbox-login/"' "$P/login/model/LoginAssets.java"
  grep -q '"module://ColdBox/rc"' "$P/login/model/LoginAssets.java"
  grep -qF 'href=\"/login/coldbox-login/login.css\"' "$P/login/model/LoginPage.java"
}

@test "TC-LG8: the plumbing registers the types, the palette entry, the web-rt dependency and the asset copy" {
  login_scaffold dashboard
  grep -q 'class="com.acme.ColdBox.login.BColdBoxLoginTemplate" name="ColdBoxLoginTemplate"' "$RT/module-include.xml"
  grep -q 'class="com.acme.ColdBox.BColdBoxService" name="ColdBoxService"' "$RT/module-include.xml"
  grep -q 'n="ColdBoxService"' "$RT/module.palette"
  grep -q 'api(":web-rt")' "$RT/ColdBox-rt.gradle.kts"
  grep -q 'include("coldbox-login/\*\*")' "$RT/ColdBox-rt.gradle.kts"
  grep -q '^ColdBoxService.applyLogin=' "$RT/module.lexicon"
  # the ux part is untouched by the login: no login file lands there
  run find "$OUT/dashboard/ColdBox/ColdBox/ColdBox-ux" -ipath '*login*'
  [ -z "$output" ]
}

@test "TC-LG9: the module map lists every login file, so lint-module-map stays silent (0 FAIL, 0 WARN)" {
  login_scaffold dashboard
  run "$KIT/toolbelt/lint-module-map.sh" "$OUT/dashboard/ColdBox/ColdBox"
  [ "$status" -eq 0 ] || { echo "$output"; false; }
  [ -z "$output" ] || { echo "$output"; false; }
  grep -q '^| login | rt |' "$OUT/dashboard/ColdBox/ColdBox/MODULE-MAP.md"
}

@test "TC-LG10: the emitted login's pure tests pass, the order test included (SKIP without javac or junit)" {
  login_scaffold dashboard
  command -v javac >/dev/null 2>&1 || skip "no javac on PATH"
  [ -n "$(find "$HOME/.gradle" -name 'junit-4.13.2.jar' 2>/dev/null | head -1)" ] || skip "junit-4.13.2.jar not in ~/.gradle"
  for c in HtmlTest LoginActivationTest LoginAssetsTest LoginPageTest LoginRedirectTest LoginViewTest TypeInitOrderTest; do
    run "$KIT/toolbelt/run-pure-test.sh" "$RT" "com.acme.ColdBox.login.model.$c"
    [ "$status" -eq 0 ] || { echo "$c: $output"; false; }
    [[ "$output" == *"OK ("* ]]
  done
}

@test "TC-LG11: the order test fails when INSTANCE is declared after TYPE (SKIP without javac or junit)" {
  # The live failure: a type with INSTANCE after TYPE does not load on a station; pure tests and the build stay green.
  login_scaffold dashboard
  command -v javac >/dev/null 2>&1 || skip "no javac on PATH"
  [ -n "$(find "$HOME/.gradle" -name 'junit-4.13.2.jar' 2>/dev/null | head -1)" ] || skip "junit-4.13.2.jar not in ~/.gradle"
  T="$P/login/BColdBoxLoginTemplate.java"
  printf '  public static final Type TYPE = Sys.loadType(BColdBoxLoginTemplate.class);\n' > "$BATS_TEST_TMPDIR/type.txt"
  awk -v f="$BATS_TEST_TMPDIR/type.txt" '/public static final BColdBoxLoginTemplate INSTANCE/ { while ((getline l < f) > 0) print l } { print }' "$T" > "$T.new"
  mv "$T.new" "$T"
  run "$KIT/toolbelt/run-pure-test.sh" "$RT" com.acme.ColdBox.login.model.TypeInitOrderTest
  [ "$status" -eq 1 ]
  [[ "$output" == *"INSTANCE declared after TYPE"* ]]
}
