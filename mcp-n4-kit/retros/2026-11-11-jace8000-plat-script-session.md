<!-- review-status: pending -->
# 2026-11-11 · kit · jace8000-plat-script-and-dist-session

**Session**: niagara-research JACE-8000 lab (first plat login to a factory remote daemon + OS dist install; full chronology in `niagara-research/docs/bypass-jace8000/sesion-2026-11-11-fabrica-414.md`)
**Delta count**: 4

## What happened
First scripted session against a remote JACE-8000 platform daemon (192.168.1.140:3011, daemon 4.1.27.20, factory state). The kit's known B1432 gotchas (`plat script -f:` rejects Windows paths; stdin works) were extended into a complete working recipe: `option -h:ip:<host>` / `-usr:` / `-pwd:` set as script verbs (B1298.4), run via `Start-Process plat.exe script -echo` with stdin/stdout/stderr redirection from WSL through `powershell.exe`. All remote output lands on stderr; local NRE boot adds ~40 s (registry rebuild) so every call needs a ≥150 s timeout. Baseline gathered with `details`, `ipconfig`, `liststations`, `flist`, and licenses backed up with `fget <src>* <localDir> -o` from `/opt/niagara/security/licenses` (NOT `~/security` — that path does not exist on JACE; `/etc/niagara` neither). A QNX OS dist was installed with `distinstall`, which required discovering the Niagara `FilePath` argument format the hard way: Windows paths give `UnresolvedException: Relative path`, and `!/dir/file` gives `SyntaxException: Illegal double slashes` — the correct form is `!dir/file` (absMode 3, relative to NIAGARA_HOME).

## Evidence
- login OK first try with the script verbs: `option -h:ip:192.168.1.140` + `details` returned full platform summary — `[ev: probe niagara-research/sources/probes/B1433-jace8000-414/]`
- flags before the subcommand print usage instead of connecting (two failed runs) — `[ev: probe plat-err 2026-11-11 01:25-01:27 runs]`
- `fget /opt/niagara/security/licenses/*.license C:/Users/.../jace-baseline -o` wrote byte-identical backups — `[ev: probe jace-baseline sha256 3b7fe488…,e3a2215e…,c798c6aa…]`
- `distinstall !jace-dist-414/tridium-qnx7-n4-titan-am335x.dist -notcp` → 40,264,708 bytes in 7.5 s, "Installing OS Image.", "Rebooting remote host.", "Installation complete." — `[ev: probe plat-err 2026-11-11 05:48-05:49]`
- post-flash anomaly (35+ min: ping OK, all TCP ports closed, power-cycle did not recover) is recorded in the session chronology §8 — `[ev: doc niagara-research/docs/bypass-jace8000/sesion-2026-11-11-fabrica-414.md §8]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Add a ready-made `plat script` transport (bash or ps1 helper): writes the stdin file with `option -h:ip:<host>`/`-usr:`/`-pwd:`, builds the file, runs via `Start-Process -RedirectStandardInput/-Output/-Error`, default timeout 150 s (NRE boot ~40 s), collects stderr as the answer, never logs the password | `mcp_n4` platform tools / new helper | `[ev: probe plat-run.ps1 + plat-run results 2026-11-11]` |
| Δ2 | Cookbook entry for Niagara `FilePath` arguments on plat CLI commands (`distinstall`/`fput`/`fget`/`flist`): `!` glued to the first path segment = NIAGARA_HOME-relative; leading `/` of Windows-style or `!/` double slash both fail with the exact errors seen | `README`/skill § platform cookbook | `[ev: code organized/baja/baja/vineflower/javax/baja/file/FilePath.java + BFileSystem.pathToLocalFile]` |
| Δ3 | Document fget/fput verb grammar in the cookbook (`fget <src>* <localDir> -o`) and the JACE license location `/opt/niagara/security/licenses` (license backup routine should prefer `flist` then `fget`, never assume `~/security`) | `README`/skill § platform cookbook | `[ev: probe plat-session5/plat-session6 outputs]` |
| Δ4 | Redaction rule: session scripts containing `-usr:`/`-pwd:` must never be committed; the recipe uses a 600-mode env file (`~/.config/jace8000/cred.env`) and the doc template refers to it instead of inlining the password (one leak caught and redacted this session) | kit conventions § secrets | `[ev: doc niagara-research docs/bypass-jace8000 session file redaction edit 2026-11-11]` |
