#!/usr/bin/env bats
# scaffold-module: a fresh scaffold of every type passes the kit's own checks with no row it could avoid, and
# carries what its first build and deploy need (a .gitignore). [ev: retro build-kit-bench-session Δ4]
# [ev: retro build-kit-bench-session Δ8]
# Shared setup (the Go binary, KIT, SC, the fixture and golden paths, OUT): tests/lib/scaffold-setup.bash.

load lib/scaffold-setup

# stand_in_jar <artifact-dir> <jar>: a jar that holds only the artifact's module.palette (when it has one),
# so verify-module --src runs its source checks (facets, facets-req) and its palette checks without a build.
stand_in_jar() {
  local d
  d="$(mktemp -d "$BATS_TEST_TMPDIR/jar.XXXXXX")"
  mkdir -p "$d/META-INF"
  printf '<module/>\n' > "$d/META-INF/module.xml"
  if [ -f "$1/module.palette" ]; then cp "$1/module.palette" "$d/"; fi
  (cd "$d" && zip -qr "$2" .)
}

@test "TC-REP1: every scaffold type passes report-module --lint-only with zero FAIL and zero WARN" {
  # Mutation: TC-REP1 -- the logic fixture's arm() without its getInterval().getMillis() > 0 guard FAILs
  # lint-delays (unknown getter before slotomatic, facet-floor WARN after), and a type key without
  # .displayName leaves slot-coverage at 0.0% WARN.
  for t in logic dashboard driver; do
    run "$SC" ModuleA "$OUT/$t" --type "$t" --vendor Example
    [ "$status" -eq 0 ]
    run "$KIT/toolbelt/report-module.sh" --lint-only "$OUT/$t/ModuleA/ModuleA"
    [ "$status" -eq 0 ] || { echo "$t: exit $status"; echo "$output"; false; }
    if [ "$t" = dashboard ]; then
      # The login template reads Niagara's session and servlet-support classes, which are com.tridium (private)
      # API with no javax.baja equal: the only WARN rows allowed are those imports (types/dashboard.md § The
      # module ships its own login: record them in the module README, pin the vendorVersion floor).
      [[ "$output" == *" · 0 FAIL · "*"->  CLEAN"* ]] || { echo "$t:"; echo "$output"; false; }
      other="$(printf '%s\n' "$output" | grep '  WARN  ' | grep -v 'lint-private-api-import  .*com\.tridium\.' || true)"
      [ -z "$other" ] || { echo "$t: $other"; false; }
      [[ "$output" == *"WARN  lint-private-api-import  LoginRequest.java"* ]]
    else
      [[ "$output" == *" · 0 FAIL · 0 WARN · "*"->  CLEAN"* ]] || { echo "$t:"; echo "$output"; false; }
    fi
  done
}

@test "TC-REP2: every scaffold artifact passes verify-module's source and palette checks with no WARN" {
  # Mutation: TC-REP2 -- the setpoint of a fixture without facets WARNs facets-req; a palette root of
  # t="b:Folder" WARNs palette-root.
  local t a n rows=0
  for t in logic dashboard driver; do
    run "$SC" ModuleA "$OUT/$t" --type "$t" --vendor Example
    [ "$status" -eq 0 ]
    for a in "$OUT/$t/ModuleA/ModuleA"/ModuleA-*; do
      n="${a##*/}"
      stand_in_jar "$a" "$BATS_TEST_TMPDIR/$n.jar"   # verify-module reads <module-dir>/<jar basename>/src
      run "$KIT/toolbelt/verify-module.sh" "$BATS_TEST_TMPDIR/$n.jar" --src "$OUT/$t/ModuleA/ModuleA"
      # the source checks ran on this artifact (a PASS row each), so a missing row is not a silent pass
      [[ "$output" == *"PASS  facets-req  "*"$a/src"* ]] || { echo "$t $n:"; echo "$output"; false; }
      [[ "$output" != *"WARN  facets-req"* && "$output" != *"FAIL  facets "* ]] || { echo "$t $n:"; echo "$output"; false; }
      [[ "$output" != *"WARN  palette-root"* ]] || { echo "$t $n:"; echo "$output"; false; }
      rows=$((rows + 1))
      rm -f "$BATS_TEST_TMPDIR/$n.jar"
    done
  done
  [ "$rows" -eq 5 ]   # logic -rt, dashboard -rt -ux, driver -rt -wb
}

@test "TC-GI1: every scaffold type emits a .gitignore at the gradle root naming the build outputs" {
  # Mutation: TC-GI1 -- a fixture without .gitignore leaves .gradle/ and build/ untracked after the first build.
  for t in logic dashboard driver; do
    run "$SC" ModuleA "$OUT/$t" --type "$t" --vendor Example
    [ "$status" -eq 0 ]
    gi="$OUT/$t/ModuleA/.gitignore"
    [ -f "$gi" ] || { echo "no .gitignore ($t)"; false; }
    for p in .gradle/ build/ _backups/ .last-deploy-sha; do
      grep -qxF "$p" "$gi" || { echo "$t: $p not in .gitignore"; cat "$gi"; false; }
    done
  done
}
