# Feature: restore Antigravity CLI CodeGraph wiring (rebase drift)

## Objective

Restore the CodeGraph community-tool wiring for Antigravity CLI that the fork's
Phase 4 integration shipped and the 2026-09-26 release-policy commit (d16e806b)
silently dropped by rewriting the compatibility table: agy was recategorized from
`reconciledCompatibility` to `excludedCompatibility`, so a clean install deploys
no codegraph MCP server and no codegraph guidance even when the user selects the
CodeGraph community tool (state.json records `community_tools: ["codegraph"]`
while the plugin ships without it).

## Problem

Empirically verified on the user's TUI clean install (2026-09-28):
- `~/.gemini/config/plugins/gentle-ai/mcp_config.json` has only context7+engram.
- `rules/AGENTS.md` lost the `gentle-ai:codegraph-guidance` section (14,281 B vs
  17,378 B pre-clean-install).
- Headless probe: CodeGraph guidance phrase absent from model context; zero
  codegraph MCP tools.
- `ReconcileAntigravityCLICodeGraph` (documented in docs/antigravity-cli/SUMMARY.md
  Phase 4) does not exist in the current tree; the pre-rebase implementation lives
  in `backup/antigravity-cli-integration-pre-odd-rebase-20260925` at 69e05931.

The old installation only had CodeGraph because pre-exclusion files survived
syncs; the clean install rewrote them per the current (regressed) code.

## Why

User report flow: CodeGraph was one of the four original complaints. The
always-on rules fix survives the clean install (ODD/TDD/RDD verified in
context), but CodeGraph regressed on clean installs because of this drift.

## Scope

Resurrection of the pre-rebase implementation, adapted verbatim where possible:

- `internal/components/communitytool/codegraph_contract.go`:
  - table: `model.AgentAntigravityCLI: reconciledCompatibility(model.AgentAntigravityCLI, "")`
    (target "" — agy's MCP config is our plugin file; upstream codegraph gets no target),
  - `codeGraphToolWiringPaths`: agy case → the plugin's `MCPConfigPath`,
  - `hasCodeGraphToolWiring`: agy branch reading the plugin mcp config with
    `hasCanonicalAntigravityCodeGraphServer`.
- `internal/components/communitytool/codegraph_guidance.go`: resurrect
  `ReconcileAntigravityCLICodeGraph(+WithAgents)` and
  `NeedsAntigravityCLICodeGraphReconcile(+WithAgents)` from 69e05931.
  The generic `InjectCodeGraphGuidance` then covers agy's rules/AGENTS.md
  section automatically once agy is compatible.
- `internal/components/communitytool/tool.go`: call the agy reconcile inside
  `InstallWithHome` in both branches (repair and full install), next to the
  OpenCode reconcile, so the TUI clean-install flow wires the server.
- `internal/cli/sync.go`: gated agy reconcile block in
  `codeGraphGuidanceSyncStep.Run` (pre-rebase line 1011 pattern) with
  changed-file accounting, for drift correction on later syncs.
- Tests: move agy from the excluded pin to reconciled in
  `codegraph_contract_test.go`; behavior-first tests for the reconciler
  (create/idempotent/preserve-servers/needs-detection); agy wiring detection.

Out of scope:
- j4500 bench changes (no corpus pin exists for the exclusion; the sandbox has
  no codegraph CLI, so the journey cannot drive this path).
- Docs: SUMMARY.md/README already describe the Phase 4 wiring; after this fix
  they are truthful again.

## Constraints

- Per-file rule cap and always-on budgets unaffected (mcp_config.json is not a
  rule file; the guidance section reuses the existing managed marker).
- Ordering is safe: every writer on the plugin mcp_config.json merges/upserts
  (context7+engram overlays merge; reconciler upserts one server), so the
  install plan's community-tool step and the agy plugin import converge in any
  order.
- `detectedCodeGraphTargets` only forwards `codeGraphNative` targets, so agy
  ("" target) never reaches the upstream `codegraph install --target` list.
- Resurrection principle: reuse the pre-rebase code verbatim unless the current
  tree changed a helper it touches.

## Acceptance criteria

- With agy selected and codegraph CLI available, `InstallWithHome`/sync leave
  `~/.gemini/config/plugins/gentle-ai/mcp_config.json` containing the codegraph
  server alongside context7 and engram, idempotently.
- `rules/AGENTS.md` carries the `gentle-ai:codegraph-guidance` managed section.
- `NeedsAntigravityCLICodeGraphReconcile` false once wired.
- Focused communitytool tests green; contract test pin updated; `go build ./...`.

## Checks

- Focused: `go test ./internal/components/communitytool/...`
- Full: `go build ./... && go test ./internal/cli/... -run 'CodeGraph|Antigravity'`
- Runtime: re-run the user's wiring (sync or reinstall) + headless probe finds
  the CodeGraph guidance phrase and codegraph MCP tools again.

## Tasks

- [x] T1: contract — table entry + wiring paths + wiring detection (RED→GREEN) — RED: strategy pin test wanted reconciled, code said excluded; contract test updated and green
- [x] T2: guidance — resurrect reconciler + needs functions (RED→GREEN) — resurrected verbatim from 69e05931; 6 behavior tests green (create/idempotent/preserve-servers/malformed-recovery/needs-detection/wiring-paths)
- [x] T3: tool.go install flow calls the agy reconcile in both branches (repair + full install)
- [x] T4: sync step gated reconcile block (mirrors OpenCode pattern with changed-file accounting)
- [x] T5: build + focused/full tests — go build clean; communitytool failure set identical to clean tree by name (16 environmental Pi CodeGraph tests reading the live install); cli CodeGraph/AntigravityCLI failures identical to clean tree (2, environmental); bench declarations green
- [x] T6: local delivery (sync) + headless probe evidence — binary 3.8.1-087f3f19 installed (backup ~/.local/bin/gentle-ai.bak-pre-codegraph-fix); sync changed exactly the two expected files: plugin mcp_config.json now carries codegraph+context7+engram, and rules/AGENTS.md regained the gentle-ai:codegraph-guidance section (17,378 B, matching the pre-drift install); headless probe: CodeGraph ordering phrase FOUND in context and codegraph_explore available under gentle-ai_codegraph.
- [x] T7: work-unit commits + review assess — commit 087f3f19 (429 authored lines; the user's standing "commit local, PR later" strategy applied over the 400-line ask). Assess: medium, review_due (slice_budget_reached); consent relayed and granted; lineage review-0334ccc3b0cfa930 (lens review-reliability) closed **approved** with one advisory SUGGESTION (reconciler's malformed-config replacement relies on caller-owned rollback; both shipped flows snapshot first; doc-comment follow-up recorded) and acknowledged — authority burned.

## Progress

(2026-09-28) Review of the user's clean install exposed the drift; user
authorized the fix ("si"). Pre-rebase implementation located at 69e05931.

(2026-09-28) All 7 tasks complete. CodeGraph wiring restored end to end:
contract + resurrected reconciler + install/sync wiring, review approved and
acknowledged, local installation delivered and probe-verified. Follow-ups:
(1) advisory SUGGESTION: document the reconciler's caller-owned rollback
contract, (2) PR packaging decision still deferred, (3) context7 (npx) MCP
failure remains a separate defect.
