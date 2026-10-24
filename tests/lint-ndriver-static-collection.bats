#!/usr/bin/env bats
# lint-ndriver-static-collection.bats — G66: a static collection field in a driver network, device or
# point-device-ext class is a WARN row; the rule's golden cases live in go/internal/lint/rules/ndriverstaticcollection.
# [ev: retro n4-course-handson-ndriver Δ7]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-ndriver-static-collection.sh"
  DRV="$BATS_TEST_TMPDIR/acmeDriver"
  cp -r "$BATS_TEST_DIRNAME/fixtures/driver-lints/acmeDriver" "$DRV"
  DEV="$DRV/acmeDriver-rt/src/com/example/drv/BAcmeDevice.java"
  RT="$DRV/acmeDriver-rt/src/com/example/drv"
}

MSG='is shared by every instance -- keep it per instance'

@test "NSC-usage: no argument prints the usage and exits 3" {
  run "$L"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: lint-ndriver-static-collection.sh [--strict] <module-root>..." ]
}

@test "NSC-fixture: the fixture device's static Set WARNs once; --strict exits 1; per instance it is clean" {
  run "$L" "$DRV"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-ndriver-static-collection  $DEV:16  static Set<String> subscribed $MSG" ]
  run "$L" --strict "$DRV"
  [ "$status" -eq 1 ]
  sed -i 's/private static final Set/private final Set/' "$DEV"
  run "$L" --strict "$DRV"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "NSC1: a static List in a subclass of a module-local device base WARNs" {
  # Mutation: NSC1 -- not following module-local superclasses turns this case green (exit 0, no row)
  sed -i 's/private static final Set/private final Set/' "$DEV"
  printf '%s\n' 'package com.example.drv;' 'public class BAcmeSubDevice extends BAcmeDevice {' \
    '  protected static final List<String> q = new ArrayList<>();' '}' > "$RT/BAcmeSubDevice.java"
  run "$L" "$DRV"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-ndriver-static-collection  $RT/BAcmeSubDevice.java:3  static List<String> q $MSG" ]
}

@test "NSC2: an unmodifiable or empty constant, a static scalar and a utility class's static Set are clean" {
  # Mutation: NSC2 -- dropping the unmodifiable/empty initializer exemption WARNs NAMES
  sed -i 's/private static final Set/private final Set/' "$DEV"
  printf '%s\n' 'package com.example.drv;' 'public class BAcmeNetwork2 extends BNNetwork {' \
    '  static final Set<String> NAMES = Collections.unmodifiableSet(new HashSet<>());' \
    '  static final List<String> NONE = Collections.emptyList();' '  static final int MAX = 3;' '}' > "$RT/BAcmeNetwork2.java"
  printf '%s\n' 'package com.example.drv;' 'final class AcmeUtil { static final Set<String> K = new HashSet<>(); }' > "$RT/AcmeUtil.java"
  run "$L" --strict "$DRV"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "NSC-K20: a root with no -rt src/ *.java is the ERROR row, exit 3" {
  rm -r "$DRV/acmeDriver-rt"; mkdir -p "$DRV/acmeDriver-ux/src"; printf 'class A {}\n' > "$DRV/acmeDriver-ux/src/A.java"
  run "$L" "$DRV"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-ndriver-static-collection  $DRV  no *.java under a -rt part's src/ (wrong path: pass the module root)" ]
}
