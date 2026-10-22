#!/usr/bin/env bats
# Kit doc pins for libraries, signing, alarms and the moduleTest route (register C12, C22, C27, C28 — WU-P0-9).
# Each pin bans a statement the source contradicts, so a later edit cannot bring it back. retros/, odd/ and
# BUILD-STATE.md are evidence and are not scanned; types/ and the top-level kit docs are.
#   KL1  okhttp ships in the NRE's bin/ext, not in net-rt (4.15.6.20 bin/ext/okhttp-4.12.0.jar; net-rt.jar has no
#        okhttp class; corpus B1320 §1320.3 C1)
#   KL2  jsonToolkit-wb ships Gson 2.13.2 and the stock MQTT module is abstractMqttDriver (4.15.6.20 pom.properties;
#        corpus B1321 §1321.2, B1323)
#   KL3  BAlarmClass.ackRequired defaults to toOffnormal|toFault|toNormal in 4.14.0.162 and 4.15.6.20 (devguide
#        source alarm-rt BAlarmClass.java:178; vineflower 4.14.0.162 BAlarmClass.java:115)
#   KL4  niagaraTest runs natively on Windows (also from WSL through cmd.exe); the 0-test result is a WSL-side
#        Gradle with plugin 7.6.17 (retro testing-parity-btestng Δ1)
#   KL5  six permission groups require a signed module, not three; any trusted cert works, not only a
#        "Tridium-trusted" one (requiresSignature() in six vineflower 4.14.0.162 classes; corpus B1312 §1312.3 C1-C3)
# [ev: retro devguide415-r2-shipped-libraries Δ2] [ev: retro devguide415-r2-shipped-libraries Δ5]
# [ev: retro devguide415-r2-control-alarms-schedules Δ1] [ev: retro testing-parity-btestng Δ1]
# [ev: retro devguide415-horizons-authoring-security Δ4]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  SCAN=("$KIT/types" "$KIT/build-verify.md" "$KIT/BUILD-LOOP.md" "$KIT/METHODOLOGY.md")
}

@test "KL1: no kit doc says okhttp ships in net-rt" {
  # shellcheck disable=SC2016  # the backticks are Markdown code spans in the pattern, not command substitution
  run grep -rnE 'okhttp3?[^|]*ships in `net-rt`|declare `net-rt` as a module dependency' "${SCAN[@]}"
  [ "$status" -eq 1 ]
}

@test "KL2: no kit doc names Gson 2.9.0 as shipped or an mqtt-driver module" {
  run grep -rnE 'Gson 2\.9\.0|mqtt-driver' "${SCAN[@]}"
  [ "$status" -eq 1 ]
}

@test "KL3: no kit doc says the ackRequired default omits toNormal" {
  run grep -rnE 'default is toOffnormal\|toFault, without toNormal' "${SCAN[@]}"
  [ "$status" -eq 1 ]
}

@test "KL4: no kit doc sends moduleTest to Workbench Run Tests as the only route" {
  run grep -rnE 'run from Workbench → \*\*Tools → Run Tests\*\* until|only a native/JACE Workbench run' "${SCAN[@]}"
  [ "$status" -eq 1 ]
}

@test "KL5: no kit doc names only three signature groups or a Tridium-trusted cert" {
  # shellcheck disable=SC2016  # the backticks are Markdown code spans in the pattern, not command substitution
  run grep -rnE '`ACCESS_CLASS`, `REFLECTION`, and `MBEAN_PERMISSION`|Tridium-trusted cert' "${SCAN[@]}"
  [ "$status" -eq 1 ]
}
