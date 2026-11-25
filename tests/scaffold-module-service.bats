#!/usr/bin/env bats
# scaffold-module --type service and --with-doc: the emitted trees, their lints and their doctrine.
# Shared setup (the Go binary, KIT, SC, OUT): tests/lib/scaffold-setup.bash.

load lib/scaffold-setup

# ---------------------------------------------------------------------------------------------------------
# --type service: a library-fixture root with templates/service-job laid over its -rt part
# (types/service.md; types/logic-authoring.md § Job contract). [ev: course Lab 10 + exam export service]
# ---------------------------------------------------------------------------------------------------------

@test "TC-SVC1: --type service emits the service, its job, a pure class and test, with no library or fixture name left" {
  # Named mutation: drop BModuleAExportJob from templates/service-job/ModuleA-rt/module-include.xml -> the job has no type.
  run "$SC" AcmeExport "$OUT" --type service --vendor Example
  [ "$status" -eq 0 ]
  M="$OUT/AcmeExport/AcmeExport"
  P="$M/AcmeExport-rt/src/com/example/AcmeExport"
  for f in "$P/BAcmeExportService.java" "$P/BAcmeExportExportJob.java" "$P/model/AcmeExportExport.java" \
           "$M/AcmeExport-rt/srcTest/test/com/example/AcmeExport/model/AcmeExportExportTest.java" \
           "$M/AcmeExport-rt/module.palette" "$M/AcmeExport-rt/module.lexicon" "$M/AcmeExport-rt/module-permissions.xml"; do
    [ -f "$f" ] || { echo "missing: $f"; false; }
  done
  # every registered type has its source
  n=0
  while IFS= read -r cls; do
    n=$((n+1))
    [ -f "$M/AcmeExport-rt/src/${cls//.//}.java" ] || { echo "no source for $cls"; false; }
  done < <(grep -o 'class="[^"]*"' "$M/AcmeExport-rt/module-include.xml" | sed 's/class="//; s/"$//')
  [ "$n" -eq 2 ]
  # a component module, not a library: no `type:` header, and the library plumbing is replaced
  run grep -c '^type:' "$M/MODULE-MAP.md"
  [ "$output" -eq 0 ]
  run grep -rIl -e MinimalLibrary -e Angeles -e angeles "$OUT/AcmeExport"
  [ "$status" -eq 1 ]
}

@test "TC-SVC2: the emitted service tree passes lint-module-map and lint-structure with no row, and has no station-test overlay" {
  # Named mutation: list BModuleAExportJob.java in no row of templates/service-job/MODULE-MAP.md -> MM4 FAIL.
  run "$SC" Acme "$OUT" --type service --vendor Example
  [ "$status" -eq 0 ]
  run "$KIT/toolbelt/lint-module-map.sh" "$OUT/Acme/Acme"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  run "$KIT/toolbelt/lint-structure.sh" "$OUT/Acme/Acme"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  [ ! -e "$OUT/Acme/Acme/Acme-rt/moduleTest-include.xml" ]
}

@test "TC-SVC3: the job keeps the job contract: snapshot in the constructor, cooperative cancel, null-context submit, plain file name" {
  # Named mutation: read service.getFileName() inside run() (the live slot) -> the snapshot check fails.
  run "$SC" Acme "$OUT" --type service --vendor Example
  [ "$status" -eq 0 ]
  P="$OUT/Acme/Acme/Acme-rt/src/com/example/Acme"
  job="$P/BAcmeExportJob.java"
  grep -q 'this.fileName = service.getFileName();' "$job"
  grep -q 'if (!isAlive()) throw new JobCancelException();' "$job"
  grep -q 'AcmeExport.isPlainFileName(fileName)' "$job"
  grep -q 'StandardCharsets.UTF_8' "$job"
  grep -q 'createFileToOverwrite' "$job"
  run grep -c 'service\.get' "$job"
  [ "$output" -eq 2 ]   # the constructor's two reads, none in run()
  grep -q 'new BAcmeExportJob(this).submit(null);' "$P/BAcmeService.java"
  grep -q 'getServiceTypes' "$P/BAcmeService.java"
  grep -q 'implements BIRestrictedComponent' "$P/BAcmeService.java"
}

# ---------------------------------------------------------------------------------------------------------
# --with-doc: the -doc help part (types/structure.md § -doc help profile). [ev: course slides-doc, SDK envCtrlDriver-doc]
# ---------------------------------------------------------------------------------------------------------

@test "TC-DOC-A: --with-doc adds the -doc part, the page named by the registered type (no B), a plumbing row; lints stay clean" {
  # Named mutation: name the page doc/Acme-BAcme.html (the class name) -> the page-name check fails.
  run "$SC" Acme "$OUT" --vendor Example --with-doc
  [ "$status" -eq 0 ]
  D="$OUT/Acme/Acme/Acme-doc"
  for f in Acme-doc.gradle.kts module-include.xml module-permissions.xml src/doc/toc.xml src/doc/index.html src/doc/Acme-Acme.html; do
    [ -f "$D/$f" ] || { echo "missing: $f"; false; }
  done
  grep -q 'runtimeProfile.set(doc)' "$D/Acme-doc.gradle.kts"
  grep -q 'moduleName.set("Acme")' "$D/Acme-doc.gradle.kts"
  grep -q 'bajadocs(project(":Acme-rt"))' "$D/Acme-doc.gradle.kts"
  grep -q 'target="Acme-Acme.html"' "$D/src/doc/toc.xml"
  grep -q 'name="Acme"' "$OUT/Acme/Acme/Acme-rt/module-include.xml"   # the registered type name the page is named after
  run grep -c 'Acme-doc/src/doc/Acme-Acme.html' "$OUT/Acme/Acme/MODULE-MAP.md"
  [ "$output" -eq 1 ]
  run "$KIT/toolbelt/lint-module-map.sh" "$OUT/Acme/Acme"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  run "$KIT/toolbelt/lint-structure.sh" "$OUT/Acme/Acme"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "TC-DOC-B: the page follows the type: Network for a driver, Service for a service; the flag position is free" {
  run "$SC" Acme "$OUT" --with-doc --type driver --vendor Example
  [ "$status" -eq 0 ]
  [ -f "$OUT/Acme/Acme/Acme-doc/src/doc/Acme-AcmeNetwork.html" ]
  grep -q 'name="AcmeNetwork"' "$OUT/Acme/Acme/Acme-rt/module-include.xml"
  run "$SC" Svc "$OUT" --vendor Example --type service --with-doc
  [ "$status" -eq 0 ]
  [ -f "$OUT/Svc/Svc/Svc-doc/src/doc/Svc-SvcService.html" ]
  grep -q 'name="SvcService"' "$OUT/Svc/Svc/Svc-rt/module-include.xml"
}

@test "TC-DOC-C: a run without --with-doc emits no -doc part; --with-doc on a library is exit 2 and writes nothing" {
  run "$SC" Plain "$OUT" --vendor Example
  [ "$status" -eq 0 ]
  [ ! -e "$OUT/Plain/Plain/Plain-doc" ]
  run "$SC" Lib "$OUT" --type library --with-doc --vendor Example
  [ "$status" -eq 2 ]
  [ "$output" = "scaffold-module: --with-doc needs a type that registers a component type; library has none" ]
  [ ! -e "$OUT/Lib" ]
}

@test "TC-DOC-D: the usage names --type service, --with-doc and --with-offload" {
  run "$SC"
  [ "$status" -eq 2 ]
  [[ "$output" == *"--type <logic|dashboard|driver|library|service>] [--with-doc] [--with-offload]"* ]]
}
