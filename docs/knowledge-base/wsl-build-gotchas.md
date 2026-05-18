# WSL Build Gotchas — Niagara N4 from WSL2

Confirmed anti-patterns when running Niagara N4 gradle builds from WSL2.
Sourced from production build sessions (chihuahua project, 2026-05+).

---

## Card 1: Mandatory `-P` overrides for WSL builds

The checked-in `gradle.properties` is configured for Windows paths.
WSL builds MUST override three properties:

```bash
./gradlew \
  -Pniagara_home=/mnt/c/Niagara/iC-Niagara-4.13.2.18 \
  "-Pniagara_user_home=/mnt/c/Users/equipo/Niagara4.13/iSMA CONTROLLI" \
  -Porg.gradle.java.installations.paths=/usr/lib/jvm/java-8-openjdk-amd64 \
  :chihuahua-rt:clean :chihuahua-rt:jar
```

**Chihuahua values** (reference — copy-paste-adjust for your module):
| Property | Value |
|---|---|
| `niagara_home` | `/mnt/c/Niagara/iC-Niagara-4.13.2.18` |
| `niagara_user_home` | `"/mnt/c/Users/equipo/Niagara4.13/iSMA CONTROLLI"` (quoted — has spaces) |
| `org.gradle.java.installations.paths` | `/usr/lib/jvm/java-8-openjdk-amd64` |

**Generic template**:
```bash
./gradlew \
  -Pniagara_home=/mnt/c/Niagara/<niagara-version> \
  "-Pniagara_user_home=/mnt/c/Users/<username>/Niagara<version>/<vendor-dir>" \
  -Porg.gradle.java.installations.paths=<linux-jdk8-path> \
  :<module>-rt:clean :<module>-rt:jar
```

**Without these**: build fails with `Plugin com.tridium.settings.multi-project not found`
(missing niagara_home) or `URISyntaxException` (Windows path in JDK property).

---

## Card 2: `chmod +x gradlew` is required after cloning on NTFS

**Symptom**: `./gradlew` → `permission denied: ./gradlew`

**Cause**: NTFS (Windows filesystem) does not store POSIX execute bits. When a
repository is cloned into a `/mnt/c/...` directory, git cannot set the exec bit.
The file exists but is not executable.

**Fix**:
```bash
chmod +x gradlew
```

Run this once after cloning. If the repo lives in a WSL-native filesystem
(`/home/...`) the bit is preserved by git and this is not needed.

---

## Card 3: gradlew path one-too-deep anti-pattern

**Symptom**: `./gradlew: command not found` even though the file exists.

**Cause**: Niagara module repos have a nested structure. The `gradlew` script
is in the **module root**, not in a sub-module subdirectory:

```
chihuahua/          ← gradlew lives here (module root)
├── gradlew
├── chihuahua-rt/   ← NOT here
└── chihuahua-ux/   ← NOT here
```

Doing `cd chihuahua/chihuahua-rt && ./gradlew` fails because `gradlew`
does not exist inside `chihuahua-rt/`.

**Fix**: Always `cd` to the module root (one level up from the subprojects):
```bash
cd /home/cristian/modulos_niagara_n4/Cliente/Honeywell/MX60/chihuahua
./gradlew :chihuahua-rt:jar
```

---

## Card 4: Slotomatic runs fine from WSL — the Windows-only claim is a myth

**Symptom**: Docs or comments say "Slot-O-Matic requires Windows" or
"cannot run slotomatic from WSL".

**Reality**: Slotomatic runs fine from WSL2 with the same three `-P` overrides
as any other gradle task. Confirmed in production (chihuahua, 2026-05-13):

```bash
cd /path/to/chihuahua
NIAGARA_HOME=/mnt/c/Niagara/iC-Niagara-4.13.2.18 \
./gradlew \
  -Pniagara_home=/mnt/c/Niagara/iC-Niagara-4.13.2.18 \
  "-Pniagara_user_home=/mnt/c/Users/equipo/Niagara4.13/iSMA CONTROLLI" \
  -Porg.gradle.java.installations.paths=/usr/lib/jvm/java-8-openjdk-amd64 \
  :chihuahua-rt:slotomatic
```

Warnings `srcTest/*.java: Cannot transform a class with no metadata` are benign
(test classes have no slotomatic metadata — expected).

The Windows-only comment in `build-and-deploy.ps1` is obsolete.
See `docs/knowledge-base/slotomatic.md` for full slotomatic workflow.

---

← Back to [GOTCHAS index](../GOTCHAS.md)
