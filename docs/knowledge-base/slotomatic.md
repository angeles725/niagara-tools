# Slot-O-Matic Patterns — Niagara N4

When to run slotomatic, the AUTO GENERATED CODE region rules, and the
coordinated-edit pattern for safe slot removal.

---

## Card 1: When to run slotomatic and how to verify

**When to run**: Any time you add, modify, or remove a `@NiagaraProperty`,
`@NiagaraType`, `@NiagaraAction`, `@NiagaraTopic`, or `@NiagaraSingleton`
annotation on a `BComponent`. If you only changed method bodies (no annotation
changes), slotomatic is NOT needed.

**How to run** (from WSL — the Windows-only claim is a myth, see wsl-build-gotchas.md):
```bash
cd /path/to/chihuahua
NIAGARA_HOME=/mnt/c/Niagara/iC-Niagara-4.13.2.18 \
./gradlew \
  -Pniagara_home=/mnt/c/Niagara/iC-Niagara-4.13.2.18 \
  "-Pniagara_user_home=/mnt/c/Users/equipo/Niagara4.13/iSMA CONTROLLI" \
  -Porg.gradle.java.installations.paths=/usr/lib/jvm/java-8-openjdk-amd64 \
  :chihuahua-rt:slotomatic
```

**How to verify** the run succeeded: check that the class hash comment changed.
Slotomatic embeds a deterministic hash in a comment at the top of each AUTO region:
```java
/*@ $com.angeles.chihuahua.components.BChiUp(1160081383)$ @*/
```
A hash of `0` means slotomatic did NOT finish processing that file (usually due
to a compilation error or missing dependency). A real number means success.
If the hash did not change from the last run AND the annotations are stable,
the output is idempotent — that is expected behavior.

**Cosmetic gotcha**: slotomatic propagates `@NiagaraProperty` javadoc comments
to the generated Slot/Getter/Setter (3x duplication). Clean up decorative
comments in the annotation before running slotomatic if you don't want noise.

---

## Card 2: AUTO GENERATED CODE region rules

The regions between `//region Property "xxx"` and `//endregion` are owned
exclusively by slotomatic. Rules:

- **NEVER hand-edit AUTO GENERATED CODE regions.** If you edit them manually,
  the next slotomatic run will overwrite your changes without warning.
- **NEVER delete an AUTO region without also removing the annotation.** Slotomatic
  will re-create the region on the next run, leaving dangling code.

**ONE allowed exception**: Slot **REMOVAL** (see Card 3 below).

---

## Card 3: Coordinated-edit pattern for slot removal

Removing a slot (`@NiagaraProperty`, etc.) requires a coordinated single-commit
edit because the build is RED between the annotation removal and the AUTO region
deletion if done in two separate commits.

**4-step recipe** (do this in a single commit, not two):

1. **Remove the annotation** from the top of the Java file:
   ```java
   // Delete this line:
   @NiagaraProperty
   public BDouble manualSetpoint = new BDouble(0.0);
   ```

2. **Manually remove the AUTO GENERATED CODE block** for that slot:
   ```java
   // Delete from here:
   //region Property "manualSetpoint"
   ...getter, setter, slot declaration...
   //endregion
   // To here (inclusive).
   ```
   This is the ONE exception to the "never hand-edit AUTO region" rule —
   you are deleting it, not editing it.

3. **Run a clean build** to confirm GREEN:
   ```bash
   ./gradlew :chihuahua-rt:clean :chihuahua-rt:jar <same -P overrides>
   ```
   If RED, the coordinated edit left dangling references — fix before proceeding.

4. **Run slotomatic** to update the class hash (it will regenerate the remaining
   slots and update the file-level hash comment):
   ```bash
   ./gradlew :chihuahua-rt:slotomatic <same -P overrides>
   ```
   Commit all files together: annotation removal + manual region deletion +
   slotomatic output = one `chore(slotomatic): regen <Class> after <slot> removal` commit.

**Idempotency rationale**: slotomatic is deterministic when annotations are stable.
Running it twice with no annotation change produces byte-for-byte identical output.
The non-deterministic gotcha is comment propagation — slotomatic copies javadoc
from the annotation to generated methods, which can vary if comments change.

---

---

## Card 4: ng-deploy.sh integration (v0.3.0+)

From v0.3.0, `ng-deploy.sh` integrates slotomatic as an opt-in step and adds
a passive annotation-change heuristic.

### Flags

| Flag | Effect |
|------|--------|
| `--with-slotomatic` | Run `:MODULE-rt:slotomatic` BEFORE `build_jars` (modes A/C only; ignored on mode B) |
| `--strict-slotomatic` | Abort with exit 15 if annotation changes are detected without `--with-slotomatic` |

### SLOTOMATIC_DETECTION env var

Controls the passive heuristic (`warn`|`strict`|`off`, default: `warn`):

| Value | Behavior |
|-------|----------|
| `warn` | WARN to stderr if `@Niagara*` annotation changes detected (no abort) |
| `strict` | Abort exit 15 if annotation changes detected without `--with-slotomatic` |
| `off` | Disable detection entirely (no WARN, no abort) |

### Detection mechanics

`detect_annotation_changes()` runs before `build_jars` (modes A/C only):
1. Reads `.last-deploy-sha` (written post-verify) as the baseline commit.
2. Falls back to `HEAD~1` if the file is absent, empty, or contains an invalid SHA.
3. Runs `git diff <baseline>..HEAD -- '*/src/com/**/*.java'` and filters with
   `grep -E '^[+-][[:space:]]*@Niagara(Type|Property|Action|Topic|Singleton)'`.
4. Returns 0 if matches found (annotation change detected), 1 otherwise.

**False-positive edge case**: comment-only lines beginning with `@Niagara*`
(e.g., in javadoc) will trigger the heuristic. The `warn` default is intentionally
non-aborting for this reason. If your project has many `@Niagara*` comments, set
`SLOTOMATIC_DETECTION=off` and rely on `--with-slotomatic` explicitly.

### .last-deploy-sha

Written to `$(pwd)/.last-deploy-sha` after every successful verify. Silent if
git is absent. **Add this to your consumer module's `.gitignore`**:
```
.last-deploy-sha
```

If the consumer's `.gitignore` does not exclude it, the file will show as
untracked on every deploy. It cannot be enforced upstream but is documented here.

### Decision table (detection × flags)

| Changes detected | --with-slotomatic | --strict-slotomatic | Action |
|---|---|---|---|
| no | 0 | 0 | nada |
| no | 1 | 0 | run slotomatic (idempotent) |
| no | * | 1 | nada (strict sin findings) |
| sí | 0 | 0 | WARN + continúa |
| sí | 1 | 0 | run slotomatic (sin WARN — ya corrigen) |
| sí | 0 | 1 | exit 15 |
| sí | 1 | 1 | run slotomatic (sin WARN) |

### Mode B behavior

Mode B is ux-only (no rt subproject is built). Detection is skipped entirely
and `--with-slotomatic` is silently ignored with a WARN:
`[ng-deploy] WARN --with-slotomatic ignored for mode B (ux-only)`.

← Back to [GOTCHAS index](../GOTCHAS.md)
