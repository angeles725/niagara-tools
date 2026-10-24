# cases.sh — the golden cases, sourced by record.sh: bats-* mirror tests/preflight.bats runs, edge-* are extra.
# new: a fresh case dir $D; fx: copy the bats fixtures in; ENVV: extra environment; EXP: files the run must leave;
# rec <name> <args...>: run and write <name>.txt.
# shellcheck shell=bash disable=SC2016,SC2034  # literal $ in fake scripts; ENVV/EXP are read by record.sh
OK_LSOF='printf "p1\nn/usr/bin/lsof\n"'

# --- tests/preflight.bats ---
new; fx; lsof_fake "$OK_LSOF"; rec bats-pf1-no-pin --jvm-dir "$D/jvm" "$D/nhnp" "$D/gr"
new; fx; lsof_fake "$OK_LSOF"; ENVV=(HOME=/nonexistent); rec bats-pf2-home --jvm-dir "$D/jvm" "$D/nhnp" "$D/gr"
new; fx; rec bats-pf3-winpath --jvm-dir "$D/jvm" 'C:\Niagara\niagara-4.14' "$D/gr"
new; fx; rec bats-pf4-no-lsof --jvm-dir "$D/jvm" "$D/nh" "$D/gr"
drift() { # drift <a-version> <b-version>
  mkdir -p "$D/drift/parent/module-a" "$D/drift/parent/module-b"; cp -r "$D/gr/." "$D/drift/parent/module-a/"
  printf 'defaultModuleVersion("%s")\n' "$1" >> "$D/drift/parent/module-a/settings.gradle.kts"
  printf 'defaultModuleVersion("%s")\n' "$2" > "$D/drift/parent/module-b/build.gradle.kts"
}
new; fx; lsof_fake "$OK_LSOF"; drift 1.0.0 1.0.1; rec bats-drift-warn --jvm-dir "$D/jvm" "$D/nh" "$D/drift/parent/module-a"
new; fx; lsof_fake "$OK_LSOF"; drift 2.1.0 2.1.0; rec bats-drift-none --jvm-dir "$D/jvm" "$D/nh" "$D/drift/parent/module-a"
new; fx; lsof_fake "$OK_LSOF"; rec bats-pf5-no-release --jvm-dir "$D/jvmnr" "$D/nh" "$D/gr"
jars() { mkdir -p "$D/nh2/etc/m2" "$D/nh2/modules"; local i; for i in "$@"; do : > "$D/nh2/modules/$i"; done; }
new; fx; jars Mod1.jar Mod2.jar Mod3.jar Mod4.jar Mod5.jar; lsof_fake "echo call >> \"$D/lsof.calls\""; : > "$D/lsof.calls"
EXP=(lsof.calls); rec bats-jarlock-single-call --jvm-dir "$D/jvm" "$D/nh2" "$D/gr"
new; fx; jars Locked.jar Free.jar; lsof_fake "printf \"p1234\\nn%s\\n\" \"$D/nh2/modules/Locked.jar\""
rec bats-jarlock-detects-locked --jvm-dir "$D/jvm" "$D/nh2" "$D/gr"
new; fx; printf '9p %s 9p rw 0 0\n' "$D/nh" > "$D/mounts"; lsof_fake "echo call >> \"$D/lsof.calls\""; : > "$D/lsof.calls"
EXP=(lsof.calls); rec bats-jarlock-skip-9p --jvm-dir "$D/jvm" "$D/nh" "$D/gr"
new; fx; lsof_fake "$OK_LSOF"; rec bats-jarlock-no-9p --jvm-dir "$D/jvm" "$D/nh" "$D/gr"
new; fx; lsof_fake 'echo "lsof: permission denied" >&2; exit 1'; rec bats-jarlock-lsof-fail --jvm-dir "$D/jvm" "$D/nh" "$D/gr"
bigjars() { # bigjars <locked-index...> (listing names them in reverse order)
  mkdir -p "$D/nh2/etc/m2" "$D/nh2/modules"; local i
  for i in $(seq -f %03g 1 400); do : > "$D/nh2/modules/Mod$i.jar"; done
  lsof_fake "exec awk -v nh=\"$D/nh2\" -v locked=\"$*\" 'BEGIN { n = split(locked, L, \" \"); for (i = 1; i <= 30000; i++) printf \"p%d\\nn/usr/lib/x86_64-linux-gnu/libfile%d.so\\n\", i, i; for (i = n; i >= 1; i--) printf \"p9\\nn%s/modules/Mod%03d.jar\\n\", nh, L[i] }'"
}
new; fx; bigjars 7 250 399; rec bats-jarlock-bulk-order --jvm-dir "$D/jvm" "$D/nh2" "$D/gr"
new; fx; bigjars 1; rec bats-jarlock-bulk-timeout --jvm-dir "$D/jvm" "$D/nh2" "$D/gr"
new; fx; bigjars; rec bats-jarlock-bulk-pass --jvm-dir "$D/jvm" "$D/nh2" "$D/gr"
new; fx; bigjars; lsof_fake 'exit 0'; rec bats-jarlock-bulk-empty --jvm-dir "$D/jvm" "$D/nh2" "$D/gr"
new; fx; jars Mod.jar Other.jar
lsof_fake "printf \"p1\\nn%s\\nn%s.bak\\n%s\\n\" \"$D/nh2/modules/Mod.ja\" \"$D/nh2/modules/Mod.jar\" \"$D/nh2/modules/Other.jar\""
rec bats-jarlock-whole-line --jvm-dir "$D/jvm" "$D/nh2" "$D/gr"
new; fx; mkdir -p "$D/nh2/etc/m2" "$D/nh2/modules/Dir.jar"; lsof_fake "printf \"p1\\nn%s\\n\" \"$D/nh2/modules/Dir.jar\""
rec bats-jarlock-regular-only --jvm-dir "$D/jvm" "$D/nh2" "$D/gr"
new; fx; nhome "$D/nh22" 7.6.22; lsof_fake "$OK_LSOF"; rec bats-plugin-override --jvm-dir "$D/jvm" --plugin-version 7.6.22 "$D/nh22" "$D/gr"
new; fx; nhome "$D/nh22" 7.6.22; lsof_fake "$OK_LSOF"; rec bats-plugin-override-prefix --jvm-dir "$D/jvm" --plugin-version 7.6.2 "$D/nh22" "$D/gr"
new; fx; nhome "$D/nh22" 7.6.22; lsof_fake "$OK_LSOF"; rec bats-plugin-default --jvm-dir "$D/jvm" "$D/nh22" "$D/gr"
new; fx; nhome "$D/nh22" 7.6.22; lsof_fake "$OK_LSOF"; ENVV=(NIAGARA_PLUGIN_VERSION=7.6.22); rec bats-plugin-env --jvm-dir "$D/jvm" "$D/nh22" "$D/gr"
new; fx; lsof_fake "$OK_LSOF"; rec bats-plugin-override-missing --jvm-dir "$D/jvm" --plugin-version 9.9.9 "$D/nh" "$D/gr"
new; rec bats-plugin-usage --plugin-version
# sample_tree <gr> <niagara_home-value> <user-home-value> <signing 1|0> <shipped-build 1|0>
sample_tree() {
  mkdir -p "$1"; cp "$D/gr/settings.gradle.kts" "$1/"
  printf 'niagara_home=%s\nniagara_user_home=%s\n' "$2" "$3" > "$1/gradle.properties"
  printf 'plugins {\n  id("com.tridium.niagara-signing")\n}\n' > "$1/build.gradle.kts"
  if [ "$4" = 1 ]; then
    printf 'signingServices {\n  signingProfileFactory {\n    allowDefaultProfile.set(false)\n  }\n}\n' >> "$1/build.gradle.kts"
    printf 'niagaraSigning {\n  aliases.set(listOf("CodeSigning"))\n  signingProfileFile.set(file("p.xml"))\n}\n' >> "$1/build.gradle.kts"
  fi
  [ "$5" = 0 ] || mkdir -p "$1/trafficLightServer-rt/build/libs"
}
snew() { new; fx; lsof_fake "$OK_LSOF"; SNH="$D/niagara-4.15.6.20"; cp -r "$D/nh" "$SNH"; }
srec() { local n="$1"; shift; rec "$n" --sample --jvm-dir "$D/jvm" "$SNH" "$@"; }
U=/home/u/Niagara4.15/tridium
snew; sample_tree "$D/tls" 'c:\\niagara\\niagara-4.15.4.24' 'C:\\Users\\<username>\\Niagara4.15\\<brand>' 0 1; srec bats-sample-orig "$D/tls"
snew; sample_tree "$D/tls" "$SNH" "$U" 1 0; srec bats-sample-retargeted "$D/tls"
snew; mkdir -p "$D/niagara-4.15.4.24"; sample_tree "$D/tls" "$D/niagara-4.15.4.24" "$U" 1 0; srec bats-sample-version "$D/tls"
snew; sample_tree "$D/tls" 'c:\\niagara\\niagara-4.15.4.24' 'C:\\Users\\<username>\\x' 0 1; rec bats-sample-off --jvm-dir "$D/jvm" "$SNH" "$D/tls"
snew; sample_tree "$D/tls" "$SNH" "$U" 1 0; sed -i 's/allowDefaultProfile.set(false)/allowDefaultProfile.set(true)/' "$D/tls/build.gradle.kts"; srec bats-sample-sign-true "$D/tls"
snew; sample_tree "$D/tls" "$SNH" "$U" 1 0; sed -i '/^plugins/,/^}/!s|^|// |' "$D/tls/build.gradle.kts"; srec bats-sample-sign-line-comment "$D/tls"
snew; sample_tree "$D/blk" "$SNH" "$U" 1 0; sed -i -e '/^signingServices/i /*' -e '$a */' "$D/blk/build.gradle.kts"; srec bats-sample-sign-block-comment "$D/blk"
snew; sample_tree "$D/tls" "$SNH" "$U" 1 0; sed -i '/signingProfileFile/d' "$D/tls/build.gradle.kts"; srec bats-sample-sign-partial "$D/tls"
snew; sample_tree "$D/tls" "$SNH" "$U" 1 0; sed -i '/^niagara_user_home=/d' "$D/tls/gradle.properties"; srec bats-sample-user-missing "$D/tls"
snew; sample_tree "$D/tls" "$D/nh" "$U" 1 0; srec bats-sample-version-unknown "$D/tls"
prepend() { L="$2" awk '{ print } !done && /^}/ { print ENVIRON["L"]; done = 1 }' "$1/build.gradle.kts" > "$1/kts.new"; mv "$1/kts.new" "$1/build.gradle.kts"; }
snew; sample_tree "$D/tls" "$SNH" "$U" 1 0; prepend "$D/tls" 'val jars = fileTree("lib") { include("**/*.jar") }'; srec bats-sample-sign-strings-glob "$D/tls"
snew; sample_tree "$D/url" "$SNH" "$U" 1 0
prepend "$D/url" 'repositories { maven { url = uri("https://repo.example/m2") } }; val q = "a \"/*\" b"; val r = """/* raw"""'; srec bats-sample-sign-strings-url "$D/url"
snew; sample_tree "$D/tls" "$SNH" "$U" 1 0; sed -i 's/allowDefaultProfile.set(false)/allowDefaultProfile = false/' "$D/tls/build.gradle.kts"; srec bats-sample-sign-assign-false "$D/tls"
snew; sample_tree "$D/tls" "$SNH" "$U" 1 0; sed -i 's/allowDefaultProfile.set(false)/allowDefaultProfile = true/' "$D/tls/build.gradle.kts"; srec bats-sample-sign-assign-true "$D/tls"
snew; sample_tree "$D/tls" 'c:\\niagara\\niagara-4.15.4.24' 'C:\\Users\\<username>\\Niagara4.15\\<brand>' 0 1; srec bats-symroot-plain "$D/tls"
snew; sample_tree "$D/tls" 'c:\\niagara\\niagara-4.15.4.24' 'C:\\Users\\<username>\\Niagara4.15\\<brand>' 0 1; ln -s "$D/tls" "$D/tlslink"; srec bats-symroot-link "$D/tlslink"

# --- edge: usage and env ---
new; rec edge-usage-none
new; fx; rec edge-usage-one "$D/nh"
new; fx; rec edge-usage-three "$D/nh" "$D/gr" extra
new; fx; rec edge-usage-unknown-opt -x "$D/nh" "$D/gr"
new; fx; rec edge-usage-dash "-" "$D/gr"
new; rec edge-usage-jvm-no-value --jvm-dir
new; fx; rec edge-usage-option-after "$D/nh" "$D/gr" --sample
new; fx; lsof_fake "$OK_LSOF"; rec edge-double-dash --jvm-dir "$D/jvm" -- "$D/nh" "$D/gr"
new; fx; lsof_fake "$OK_LSOF"; rec edge-double-dash-dashpath --jvm-dir "$D/jvm" -- "-nh" "$D/gr"
new; fx; rec edge-nh-missing --jvm-dir "$D/jvm" "$D/nope" "$D/gr"
new; fx; rec edge-gr-missing --jvm-dir "$D/jvm" "$D/nh" "$D/nope"
new; fx; rec edge-both-missing --jvm-dir "$D/jvm" "$D/nope1" "$D/nope2"
new; fx; : > "$D/afile"; rec edge-nh-is-file --jvm-dir "$D/jvm" "$D/afile" "$D/gr"
new; fx; rec edge-win-gr-backslash --jvm-dir "$D/jvm" "$D/nh" 'mod\root'
new; fx; rec edge-win-both --jvm-dir "$D/jvm" 'C:/n4' 'D:\x'
new; fx; rec edge-win-lowercase-drive --jvm-dir "$D/jvm" 'c:/n4' "$D/gr"
new; fx; rec edge-win-jvm-missing --jvm-dir "$D/none" 'C:\n4' "$D/gr"
# --- edge: jdk8 ---
new; fx; lsof_fake "$OK_LSOF"; mkdir "$D/emptyjvm"; rec edge-jdk-none --jvm-dir "$D/emptyjvm" "$D/nh" "$D/gr"
new; fx; lsof_fake "$OK_LSOF"; rec edge-jdk-dir-missing --jvm-dir "$D/none" "$D/nh" "$D/gr"
new; fx; lsof_fake "$OK_LSOF"; rec edge-jdk-dir-empty-string --jvm-dir "" "$D/nh" "$D/gr"
new; fx; lsof_fake "$OK_LSOF"; mkdir "$D/emptyjvm"; ENVV=(JAVA_HOME="$D/jvm/java-8-openjdk-amd64"); rec edge-jdk-java-home-release --jvm-dir "$D/emptyjvm" "$D/nh" "$D/gr"
new; fx; lsof_fake "$OK_LSOF"; mkdir "$D/emptyjvm"; ENVV=(JAVA_HOME="$D/jvmnr/java-8-openjdk-wsl"); rec edge-jdk-java-home-bin --jvm-dir "$D/emptyjvm" "$D/nh" "$D/gr"
new; fx; lsof_fake "$OK_LSOF"; mkdir -p "$D/emptyjvm" "$D/jh11/bin"; printf 'JAVA_VERSION="11.0.2"\n' > "$D/jh11/release"
cp "$D/jvmnr/java-8-openjdk-wsl/bin/java" "$D/jh11/bin/java"; ENVV=(JAVA_HOME="$D/jh11"); rec edge-jdk-java-home-release-wins --jvm-dir "$D/emptyjvm" "$D/nh" "$D/gr"
new; fx; lsof_fake "$OK_LSOF"; mkdir "$D/emptyjvm"; ENVV=(JAVA_HOME=""); rec edge-jdk-java-home-empty --jvm-dir "$D/emptyjvm" "$D/nh" "$D/gr"
new; fx; lsof_fake "$OK_LSOF"; mkdir -p "$D/mj/a-11" "$D/mj/b-8" "$D/mj/c-8"
printf 'JAVA_VERSION="11.0.2"\n' > "$D/mj/a-11/release"; printf 'JAVA_VERSION="1.8.0_1"\n' > "$D/mj/b-8/release"; printf 'JAVA_VERSION="1.8.0_2"\n' > "$D/mj/c-8/release"
rec edge-jdk-first-of-many --jvm-dir "$D/mj" "$D/nh" "$D/gr"
new; fx; lsof_fake "$OK_LSOF"; mkdir -p "$D/mj/.hidden8" "$D/mj/z/bin"; printf 'JAVA_VERSION="1.8.0_1"\n' > "$D/mj/.hidden8/release"
cp "$D/jvmnr/java-8-openjdk-wsl/bin/java" "$D/mj/z/bin/java"; rec edge-jdk-hidden-skipped-bin-fallback --jvm-dir "$D/mj" "$D/nh" "$D/gr"
new; fx; lsof_fake "$OK_LSOF"; mkdir -p "$D/mj/a/bin"; printf 'echo "1.8.0"\n' > "$D/mj/a/bin/java"; rec edge-jdk-bin-not-exec --jvm-dir "$D/mj" "$D/nh" "$D/gr"
new; fx; lsof_fake "$OK_LSOF"; mkdir -p "$D/mj/a/bin"; printf '#!/usr/bin/env bash\necho "openjdk version \\"11.0.1\\""\n' > "$D/mj/a/bin/java"; chmod +x "$D/mj/a/bin/java"
rec edge-jdk-bin-wrong-version --jvm-dir "$D/mj" "$D/nh" "$D/gr"
new; fx; lsof_fake "$OK_LSOF"; mkdir -p "$D/mj/a/bin"; printf '#!/usr/bin/env bash\necho "openjdk version \\"1.8.0_9\\""\n' > "$D/mj/a/bin/java"; chmod +x "$D/mj/a/bin/java"
rec edge-jdk-bin-stdout --jvm-dir "$D/mj" "$D/nh" "$D/gr"
new; fx; lsof_fake "$OK_LSOF"; mkdir -p "$D/mj"; ln -s "$D/jvm/java-8-openjdk-amd64" "$D/mj/linked"; rec edge-jdk-symlinked-candidate --jvm-dir "$D/mj" "$D/nh" "$D/gr"
new; fx; lsof_fake "$OK_LSOF"; rec edge-jdk-trailing-slash --jvm-dir "$D/jvm/" "$D/nh" "$D/gr"
new; fx; lsof_fake "$OK_LSOF"; mkdir -p "$D/mj/a"; printf 'JAVA_VERSION="1.80"\nJAVA_VERSION=1.8.0\n' > "$D/mj/a/release"; rec edge-jdk-release-near-miss --jvm-dir "$D/mj" "$D/nh" "$D/gr"
# --- edge: plugin-pin ---
new; fx; lsof_fake "$OK_LSOF"; mkdir "$D/gr0"; rec edge-plugin-no-settings --jvm-dir "$D/jvm" "$D/nh" "$D/gr0"
new; fx; lsof_fake "$OK_LSOF"; mkdir "$D/gr0"; printf 'val v = "7.6"\n// "x.y.z"\n' > "$D/gr0/settings.gradle.kts"; rec edge-plugin-no-version --jvm-dir "$D/jvm" "$D/nh" "$D/gr0"
new; fx; lsof_fake "$OK_LSOF"; mkdir "$D/gr0"; printf 'id("a") version "1.2"\nval x = "7.6.17" + "9.9.9"\n' > "$D/gr0/settings.gradle.kts"; rec edge-plugin-first-version --jvm-dir "$D/jvm" "$D/nh" "$D/gr0"
new; fx; lsof_fake "$OK_LSOF"; mkdir -p "$D/nhm/modules"; rec edge-plugin-no-m2 --jvm-dir "$D/jvm" "$D/nhm" "$D/gr"
new; fx; lsof_fake "$OK_LSOF"; nhome "$D/nh" 7.6.170; rm -rf "$D/nh/etc/m2/repository/com/tridium/niagara-module/com.tridium.niagara-module.gradle.plugin/7.6.17"
rec edge-plugin-substring-dir --jvm-dir "$D/jvm" "$D/nh" "$D/gr"
new; fx; lsof_fake "$OK_LSOF"; mkdir -p "$D/n4-7.6.17/etc/m2" "$D/n4-7.6.17/modules"; rec edge-plugin-substring-home --jvm-dir "$D/jvm" "$D/n4-7.6.17" "$D/gr"
new; fx; lsof_fake "$OK_LSOF"; mkdir -p "$D/nhf/etc/m2" "$D/nhf/modules"; : > "$D/nhf/etc/m2/7.6.17"; rec edge-plugin-file-not-dir-override --jvm-dir "$D/jvm" --plugin-version 7.6.17 "$D/nhf" "$D/gr"
new; fx; lsof_fake "$OK_LSOF"; mkdir -p "$D/nhf/etc/m2" "$D/nhf/modules"; : > "$D/nhf/etc/m2/7.6.17.pom"; rec edge-plugin-file-settings --jvm-dir "$D/jvm" "$D/nhf" "$D/gr"
new; fx; lsof_fake "$OK_LSOF"; ENVV=(NIAGARA_PLUGIN_VERSION=9.9.9); rec edge-plugin-flag-beats-env --jvm-dir "$D/jvm" --plugin-version 7.6.17 "$D/nh" "$D/gr"
new; fx; lsof_fake "$OK_LSOF"; ENVV=(NIAGARA_PLUGIN_VERSION=9.9.9); rec edge-plugin-empty-flag-settings --jvm-dir "$D/jvm" --plugin-version "" "$D/nh" "$D/gr"
new; fx; lsof_fake "$OK_LSOF"; mkdir -p "$D/real/m2/x/7.6.17" "$D/nhl/modules" "$D/nhl/etc"; ln -s "$D/real/m2" "$D/nhl/etc/m2"
rec edge-plugin-m2-symlink-override --jvm-dir "$D/jvm" --plugin-version 7.6.17 "$D/nhl" "$D/gr"
new; fx; lsof_fake "$OK_LSOF"; mkdir -p "$D/real/m2/x/7.6.17" "$D/nhl/modules" "$D/nhl/etc"; ln -s "$D/real/m2" "$D/nhl/etc/m2"
rec edge-plugin-m2-symlink-settings --jvm-dir "$D/jvm" "$D/nhl" "$D/gr"
# --- edge: jar-lock ---
new; fx; printf 'ext4 / ext4 rw 0 0\ndrvfs %s drvfs rw 0 0\n' "$D" > "$D/mounts"; rec edge-jarlock-drvfs --jvm-dir "$D/jvm" "$D/nh" "$D/gr"
new; fx; printf '9p / 9p rw 0 0\next4 %s ext4 rw 0 0\n\n  \n' "$D/nh" > "$D/mounts"; lsof_fake "$OK_LSOF"; rec edge-jarlock-longest-prefix --jvm-dir "$D/jvm" "$D/nh" "$D/gr"
new; fx; printf '9p %s/n 9p rw 0 0\n' "$D" > "$D/mounts"; lsof_fake "$OK_LSOF"; rec edge-jarlock-string-prefix --jvm-dir "$D/jvm" "$D/nh" "$D/gr"
new; fx; printf 'x %s\n' "$D/nh" > "$D/mounts"; lsof_fake "$OK_LSOF"; rec edge-jarlock-mount-two-fields --jvm-dir "$D/jvm" "$D/nh" "$D/gr"
new; fx; mkdir -p "$D/nh2/etc/m2" "$D/nh2/modules"; lsof_fake "$OK_LSOF"; rec edge-jarlock-no-jars-pass --jvm-dir "$D/jvm" "$D/nh2" "$D/gr"
new; fx; mkdir -p "$D/nh2/etc/m2"; lsof_fake "$OK_LSOF"; rec edge-jarlock-no-modules-dir --jvm-dir "$D/jvm" "$D/nh2" "$D/gr"
new; fx; mkdir -p "$D/nh2/etc/m2" "$D/nh2/modules"; lsof_fake 'exit 0'; rec edge-jarlock-no-jars-empty --jvm-dir "$D/jvm" "$D/nh2" "$D/gr"
new; fx; jars A.jar .h.jar b.JAR; ln -s "$D/nh2/modules/A.jar" "$D/nh2/modules/L.jar"; ln -s "$D/nope.jar" "$D/nh2/modules/M.jar"
lsof_fake "printf \"n%s\\nn%s\\nn%s\\nn%s\\n\\n\\n\" \"$D/nh2/modules/.h.jar\" \"$D/nh2/modules/b.JAR\" \"$D/nh2/modules/L.jar\" \"$D/nh2/modules/A.jar\""
rec edge-jarlock-glob-shapes --jvm-dir "$D/jvm" "$D/nh2" "$D/gr"
new; fx; jars A.jar; lsof_fake 'printf "\n\n\n"'; rec edge-jarlock-blank-listing --jvm-dir "$D/jvm" "$D/nh2" "$D/gr"
new; fx; jars A.jar B.jar; lsof_fake "printf \"n%s\" \"$D/nh2/modules/B.jar\""; rec edge-jarlock-noeol-listing --jvm-dir "$D/jvm" "$D/nh2" "$D/gr"
new; fx; jars A.jar; lsof_fake "printf \"n%s\\n\" \"$D/nh2/modules/A.jar\"; exit 1"; rec edge-jarlock-lsof-exit1-held --jvm-dir "$D/jvm" "$D/nh2" "$D/gr"
new; fx; jars A.jar; lsof_fake 'printf "%s\n" "$@" > "'"$D"'/args"; printf "p1\n"'; EXP=(args); rec edge-jarlock-lsof-args --jvm-dir "$D/jvm" "$D/nh2" "$D/gr"
new; fx; jars A.jar; printf 'not executable\n' > "$D/fakebin/lsof"; rec edge-jarlock-lsof-not-exec --jvm-dir "$D/jvm" "$D/nh2" "$D/gr"
# --- edge: version-drift ---
deep() { mkdir -p "$D/w/p/me"; cp "$D/gr/settings.gradle.kts" "$D/w/p/me/"; printf 'defaultModuleVersion("%s")\n' "$1" > "$D/w/p/me/build.gradle.kts"; }
sib() { mkdir -p "$(dirname "$1")"; printf '%s\n' "$2" > "$1"; }
new; fx; lsof_fake "$OK_LSOF"; deep 2.0.0; sib "$D/w/p/a/x.gradle.kts" 'defaultModuleVersion("1.9.9")'; sib "$D/w/p/b/b.gradle.kts" 'defaultModuleVersion("2.0.10")'
sib "$D/w/p/c/c.gradle.kts" 'defaultModuleVersion("10.0.0")'; sib "$D/w/p/.h/h.gradle.kts" 'defaultModuleVersion("9.0.0")'
rec edge-drift-many-siblings --jvm-dir "$D/jvm" "$D/nh" "$D/w/p/me"
new; fx; lsof_fake "$OK_LSOF"; deep 1.0.01; sib "$D/w/p/a/x.gradle.kts" 'defaultModuleVersion("1.0.1")'; rec edge-drift-leading-zero-mine --jvm-dir "$D/jvm" "$D/nh" "$D/w/p/me"
new; fx; lsof_fake "$OK_LSOF"; deep 1.0.1; sib "$D/w/p/a/x.gradle.kts" 'defaultModuleVersion("1.0.01")'; rec edge-drift-leading-zero-sib --jvm-dir "$D/jvm" "$D/nh" "$D/w/p/me"
new; fx; lsof_fake "$OK_LSOF"; deep 1.0.0; sib "$D/w/q/x.gradle.kts" 'defaultModuleVersion("3.0.0")'; sib "$D/v/y.gradle.kts" 'defaultModuleVersion("4.0.0")'
rec edge-drift-levels --jvm-dir "$D/jvm" "$D/nh" "$D/w/p/me"
new; fx; lsof_fake "$OK_LSOF"; deep 1.0.0; sib "$D/w/p/a/d1/d2/d3.gradle.kts" 'defaultModuleVersion("5.0.0")'; sib "$D/w/p/b/d1/d2/d3/d4.gradle.kts" 'defaultModuleVersion("6.0.0")'
rec edge-drift-maxdepth --jvm-dir "$D/jvm" "$D/nh" "$D/w/p/me"
new; fx; lsof_fake "$OK_LSOF"; deep 1.0.0; mkdir -p "$D/w/p/a/dir.gradle.kts"; sib "$D/w/p/a/z.gradle.kts" '// defaultModuleVersion was "2.0.0"'
sib "$D/w/p/b/b.gradle.kts" "$(printf 'version = "9.9.9"\ndefaultModuleVersion(v)\ndefaultModuleVersion("1.5.0") // "8.8.8"')"
rec edge-drift-line-shapes --jvm-dir "$D/jvm" "$D/nh" "$D/w/p/me"
new; fx; lsof_fake "$OK_LSOF"; mkdir -p "$D/w/p/me"; cp "$D/gr/settings.gradle.kts" "$D/w/p/me/"; sib "$D/w/p/a/x.gradle.kts" 'defaultModuleVersion("9.0.0")'
rec edge-drift-mine-none --jvm-dir "$D/jvm" "$D/nh" "$D/w/p/me"
new; fx; lsof_fake "$OK_LSOF"; deep 1.0.0; sib "$D/w/p/a/x.gradle.kts" 'defaultModuleVersion("2.0.0")'; rec edge-drift-trailing-slash --jvm-dir "$D/jvm" "$D/nh" "$D/w/p/me/"
new; fx; lsof_fake "$OK_LSOF"; deep 1.0.0; mkdir -p "$D/w/p/me/sub"; sib "$D/w/p/me/sub/s.gradle.kts" 'defaultModuleVersion("0.1.0")'; rec edge-drift-own-sub --jvm-dir "$D/jvm" "$D/nh" "$D/w/p/me"
new; fx; lsof_fake "$OK_LSOF"; deep 1.0.0; mkdir -p "$D/real/r"; sib "$D/real/r/r.gradle.kts" 'defaultModuleVersion("7.0.0")'; ln -s "$D/real/r" "$D/w/p/lnk"
rec edge-drift-symlink-sibling --jvm-dir "$D/jvm" "$D/nh" "$D/w/p/me"
new; fx; lsof_fake "$OK_LSOF"; deep 1.0.0; sib "$D/w/p/b/b.gradle.kts" 'defaultModuleVersion("2.0.0")'; mkdir -p "$D/w/p/b/locked"; chmod 000 "$D/w/p/b/locked"
rec edge-drift-unreadable --jvm-dir "$D/jvm" "$D/nh" "$D/w/p/me"
new; fx; lsof_fake "$OK_LSOF"; mkdir -p "$D/w/p/me"; cp "$D/gr/settings.gradle.kts" "$D/w/p/me/"; printf 'defaultModuleVersion("1.0.0")' > "$D/w/p/me/build.gradle.kts"
sib "$D/w/p/a/x.gradle.kts" 'defaultModuleVersion("1.0.0")'; rec edge-drift-noeol-mine --jvm-dir "$D/jvm" "$D/nh" "$D/w/p/me"
# --- edge: sample ---
snew; sample_tree "$D/tls" 'D:\\N4\\niagara-4.15.6.20' "$U" 1 0; mkdir -p "$D/mntd"; srec edge-sample-drive-missing "$D/tls"
snew; sample_tree "$D/tls" "$SNH" "$U" 1 0; rm "$D/tls/gradle.properties"; srec edge-sample-no-properties "$D/tls"
snew; sample_tree "$D/tls" "$SNH" "$U" 1 0; printf '  niagara_home \t=  %s  \r\nniagara_home=/other\nniagara_user_home = %s\t\n' "$SNH" "$U" > "$D/tls/gradle.properties"; srec edge-sample-prop-spacing "$D/tls"
snew; sample_tree "$D/tls" "$SNH" '/home/<u>/x' 1 0; srec edge-sample-user-placeholder "$D/tls"
snew; sample_tree "$D/tls" "$SNH" '/home/u>x<y' 1 0; srec edge-sample-user-reversed-brackets "$D/tls"
snew; sample_tree "$D/tls" "$SNH" "$U" 1 0; printf 'niagara_homes=/x\n#niagara_home=/y\n' > "$D/tls/gradle.properties"; srec edge-sample-key-shapes "$D/tls"
new; fx; lsof_fake "$OK_LSOF"; mkdir -p "$D/niagara-4.15.6.20"; sample_tree "$D/tls" "$D/niagara-4.15.6.20" "$U" 1 0; rec edge-sample-target-unversioned --sample --jvm-dir "$D/jvm" "$D/nh" "$D/tls"
snew; sample_tree "$D/tls" "$D/niagara-4.15.6.20.1.2.3" "$U" 1 0; mkdir -p "$D/niagara-4.15.6.20.1.2.3"; srec edge-sample-many-version-tokens "$D/tls"
snew; sample_tree "$D/tls" "$SNH" "$U" 1 0; rm "$D/tls/build.gradle.kts"; srec edge-sample-no-root-kts "$D/tls"
snew; sample_tree "$D/tls" "$SNH" "$U" 0 0; mkdir -p "$D/tls/.gradle/build" "$D/tls/a/build/b/build" "$D/tls/b-rt/build" "$D/tls/b/build" "$D/tls/x/y/build" "$D/tls/x/y/z/build"
mkdir -p "$D/tls/c" "$D/tls/build"; : > "$D/tls/c/build"; srec edge-sample-build-shapes "$D/tls"
snew; sample_tree "$D/tls" "$SNH" "$U" 0 0
printf 'signingServices { signingProfileFactory { allowDefaultProfile.set( false ) } }\n/* a /* b */ niagaraSigning {\n aliases_x signingProfileFile\n"""\n// raw """ aliases\n' >> "$D/tls/build.gradle.kts"; srec edge-sample-sign-awk-shapes "$D/tls"
snew; sample_tree "$D/tls" "$SNH" "$U" 0 0; printf 'val s = "a\\\\"; niagaraSigning { aliases signingProfileFile }\nallowDefaultProfile=false;\n' >> "$D/tls/build.gradle.kts"; srec edge-sample-sign-escape "$D/tls"
snew; sample_tree "$D/tls" "$SNH" "$U" 0 0; printf 'allowDefaultProfile = falsey\nniagaraSigning{\n.aliases\nsigningProfileFile\n' >> "$D/tls/build.gradle.kts"; srec edge-sample-sign-falsey "$D/tls"
snew; sample_tree "$D/tls" "$SNH" "$U" 1 1; srec edge-sample-winpath-skips 'C:\x'
# --- glob order: bash sorts "<name>/" (with the slash), so a-b/ sorts before a/ ---
new; fx; lsof_fake "$OK_LSOF"; deep 1.0.0; sib "$D/w/p/a/x.gradle.kts" 'defaultModuleVersion("2.0.0")'; sib "$D/w/p/a-b/x.gradle.kts" 'defaultModuleVersion("3.0.0")'
sib "$D/w/p/a.b/x.gradle.kts" 'defaultModuleVersion("4.0.0")'; rec edge-glob-slash-order-drift --jvm-dir "$D/jvm" "$D/nh" "$D/w/p/me"
new; fx; lsof_fake "$OK_LSOF"; mkdir -p "$D/mj/j" "$D/mj/j-x"; printf 'JAVA_VERSION="1.8.0_1"\n' > "$D/mj/j/release"; printf 'JAVA_VERSION="1.8.0_2"\n' > "$D/mj/j-x/release"
rec edge-glob-slash-order-jdk --jvm-dir "$D/mj" "$D/nh" "$D/gr"
