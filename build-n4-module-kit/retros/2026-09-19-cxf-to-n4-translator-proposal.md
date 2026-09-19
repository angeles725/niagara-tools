<!-- review-status: pending -->
# Kit proposal — CXF → Niagara N4 control-logic translator (build delta)

> Type: BUILD proposal (new toolbelt capability), not a lint/rule change. Opened 2026-09-19 from the
> `niagara-research` focus `hvac-control-literature` (blocks B1029–B1047). This proposes a well-scoped
> build; it does NOT itself build the tool. Build trigger is a concrete need (below), not this document.

## Problem

There is no way today to turn a standardized, vendor-neutral control sequence (ASHRAE Guideline 36,
expressed in CDL — Control Description Language, ASHRAE Standard 231) into deployable Niagara N4 logic.
Every sequence is hand-wired in Workbench, which is slow, error-prone (the class of defects the kit's
lints already chase), and not reusable across projects.

## What the research established (corpus evidence)

- **CDL is the formal, vendor-neutral equivalent of `kitControl` wiresheet blocks.** A CDL↔kitControl
  block-type mapping already exists: `[Block 1040]` (30-row table: PID↔BPidLoop, Timer, Hysteresis,
  logic/math, Limiter, Switch, Stage…).
- **The CDL→N4 bridge is a demonstrated proof-of-concept**, but no shipped tool: OBC's `codeGeneration`
  spec names Tridium Niagara as a demonstrated CXF translation target `[Block 1044]`. NO downloadable
  Niagara CDL importer exists.
- **The executable first half already exists**: `lbl-srg/modelica-json` parses CDL and emits **CXF**
  (Control eXchange Format, JSON-LD / ASHRAE S231) plus other JSON formats, with a `json2mo` reverse
  path `[Block 1046]`. It has NO Niagara backend, NO CDL→kitControl block map, NO `.bog` emitter.
- G36 sequences worth targeting first: trim-and-respond / reset `[Block 1041]`, AHU economizer &
  staging `[Block 1042]`. Verification methodology (trend-vs-spec) `[Block 1045]`.

**Net: the only missing piece for a CDL→N4 pipeline is a custom `CXF → kitControl/config.bog` translator.**

## Proposed build (scope)

A toolbelt tool (e.g. `toolbelt/cxf-to-n4.sh` + a translator lib) that:

1. **Input**: a CXF/JSON-LD file produced by `modelica-json` from a CDL sequence.
2. **Map**: CXF control blocks + connections → `kitControl` component instances + links, using the
   `[Block 1040]` CDL↔kitControl table as the block-type map (extend the table as gaps appear).
3. **Output**: a `config.bog`-compatible component graph fragment (BOG-XML: `<p h='handle' t='pfx:Type'>`
   nodes + links living in the destination component with `sourceOrd='h:xxxx'`). Reuse the BOG-XML
   grammar + handle-graph engine already in `bog-nav.py` / `bog-audit.sh` (read side) as the reference
   for the WRITE side.
4. **Verify**: gate the generated module through the existing kit gates (`verify-module.sh`,
   `lint-*`), and optionally the OBC-style trend-vs-spec check `[Block 1045]` as a commissioning step.

## Why it benefits us

- Standardized sequence → deployable N4 logic, **reusable across projects** (PANCCADIA, HARBOR, Hilton, …).
- **Consistency + traceability**: sequence specified once, verified, generated → fewer manual-wiring bugs.
- **Leverages existing work**: the hard parse/normalize half is `modelica-json`; the block map is
  `[Block 1040]`; the BOG grammar is `bog-nav`. Groundwork is done.
- A real differentiator for a Niagara integrator (few/none ship CDL→Niagara).

## Non-goals / honest limits

- CDL/G36 is air-side HVAC; **immediate ROI for refrigeration (cold rooms/compressors) is low**. The
  payoff lands when HVAC / datacenter work is concrete.
- `config.bog` WRITE serialization (handles, links, slots, modes) is delicate — this is a real build,
  not a script. Estimate: multi-session, needs a node.js + `modelica-json` env and a real target sequence.
- Keep the kit clean: this is a BUILD tool (mechanics), consistent with the kit's scope; the domain
  knowledge stays in the `niagara-research` corpus, which this tool cites, never ingests.

## Build trigger

Start the build ONLY when there is a concrete HVAC/datacenter CDL sequence to deploy, plus a node.js +
`modelica-json` environment. Until then this stays a `pending` proposal.

## References

Corpus blocks: B1040 (CDL↔kitControl map), B1041/B1042 (G36 sequences), B1044 (codeGeneration/CXF),
B1045 (verification), B1046 (modelica-json), B1047 (s223ToMo). Kit: `bog-nav.py`, `bog-audit.sh`,
`verify-module.sh`, `types/logic.md`.
