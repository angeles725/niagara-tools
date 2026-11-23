# fold-research-retros-2026-11-23 — retro deltas as GitHub issues, and the research-corpus retros

Retros written in the private research corpus during client work (2026-11-13..22) propose mcp-n4 and
build-n4-module deltas that the kit fold audit never saw (it reads only build-n4-module-kit/retros/).
Client names stay out of this public repo: the retros are named R1..R15 here; the private mapping lives
in the research corpus (odd/tasks/fold-research-retros-2026-11-23.md there).

## Specs

- S1 (user, verbatim): "las retros/deltas podrias hacer como una regla para que sean como issues asi como lo hace gentle-ai y gentle-shell, osea para que se tengan en github y no esten escritas de esta forma para que no se pierdan."
- S2 (user, verbatim, accepting the proposal "Un delta para que no vuelva a pasar: que la revisión de integración del kit también busque retros de otros proyectos que mencionen mcp-n4 o build-n4-module"): "Si, haz la campana y tambien el delta."
- S3 (same message): the campaign — verify every pending kit/mcp-n4 delta of R1..R15 against commits, implement what is still missing in batches (mcp-n4, kit), and mark each retro folded.
- S4 (repo policy): no client name in any issue, commit, doc or test of this public repo (client-guard denylist).

## Tasks

| ID | Specs | Route | Status |
|---|---|---|---|
| T1 | S1, S4 | worker | rule + tool: every kit/mcp-n4 retro delta becomes a GitHub issue (dry run default, --apply, idempotent, scrubbed); retro rows carry the issue number; CONTRIBUTING / new-retro template / fold audit follow it |
| T2 | S2 | worker | fold audit also reads retros outside the kit that name mcp-n4 / build-n4-module (configured roots), and reports their uncited deltas |
| T3 | S3 | explore | commit-level audit of R1..R15: landed / pending per delta |
| T4 | S1, S3 | parent | file the pending deltas as issues with the T1 tool (close the 9 open retro issues they duplicate or link them) |
| T5 | S3 | workers | implement mcp-n4 pending deltas (issues closed by PRs) |
| T6 | S3 | workers | implement build-n4-module pending deltas |
| T7 | S3 | parent | flip R1..R15 markers in the research corpus; strict fold audit incl. external roots = 0 |

## Log

- L1 (2026-11-23, user): "Dime, que pendientes de retros/deltas tenemos en mcp-n4 y build-n4-module?? ya no tenemos ninguno??" Audit: kit retros 475 folded / 0 uncited, but ~15 research-corpus retros pending with ~40 mcp-n4 and ~38 kit deltas not located in the kit (concept-level grep, not commit-checked).
- L2 (2026-11-23, user): "Si, haz la campana y tambien el delta. Ahora una cosa, podrias hacer algo, las retros/deltas podrias hacer como una regla para que sean como issues asi como lo hace gentle-ai y gentle-shell, osea para que se tengan en github y no esten escritas de esta forma para que no se pierdan. No se si me este explicando"
- L3: open retro issues already in GitHub before this campaign: #50, #200, #816-#820, #822, #823.
