# Feature: Antigravity CLI always-on rules injection

## Objective

Make the Gentle AI ODD workflow, TDD policy, RDD boundaries, delegation rules, and
orchestrator instructions actually reach the Antigravity CLI model context by default,
and remove the permission friction that blocks default CodeGraph lazy-init.

## Problem

Empirical diagnosis (2026-09-27, probes documented in Engram observation 2037/2093):

1. `rules/gentle-ai-routing.md` (entire ODD protocol, delegation triggers, TDD policy,
   RDD switch policy) is silently dropped by agy 1.2.12 on every turn. agy only
   auto-loads a plugin's `rules/AGENTS.md`; other rule files without YAML frontmatter
   parse as trigger `CORTEX_MEMORY_TRIGGER_UNSPECIFIED` and are rejected
   (log: `Invalid rule trigger`, 843 occurrences; headless probes confirmed routing
   phrases absent from context while AGENTS.md phrases are present).
2. `orchestrator.md` (+ `-delegation`, `-memory`, `-skills`) deployed at the plugin
   root is never read by agy (root is not a discovery location), so the coordinator
   harness and the Review Execution Contract never reach the model. The
   `<!-- antigravity-review-execution-contract:insert -->` marker inside it was never
   filled by anything (`ReviewExecutionContractFor` exists, unwired).
3. CodeGraph guidance IS injected and the `gentle-ai_codegraph` MCP tool works, but
   in unindexed projects the canonical lazy-init command `gentle-ai codegraph init`
   is not in the `settings.json` allow list, so every default attempt hits a
   permission prompt and the agent falls back to plain file tools.

Validated mechanism (temp probe plugin, created and removed): a plugin `rules/*.md`
file with frontmatter `trigger: always_on` + `description` is injected always-on by agy.

## Why

User report: ODD, RDD, TDD, and CodeGraph never run in Antigravity CLI unless
explicitly requested. Direct harness failure: the harness disciplines exist on disk
but never enter the model context.

## Scope

In scope:
- `internal/components/filemerge`: idempotent YAML frontmatter prepend helper + tests.
- `internal/components/agentguidance`: always-on frontmatter for the antigravity-cli
  routing rule file (only for `AgentAntigravityCLI`; never for AGENTS.md carriers).
- `internal/agents/antigravitycli`: BundleAssets maps `orchestrator.md` to
  `rules/gentle-ai-orchestrator.md`; prune legacy root `orchestrator.md`; export the
  review-contract marker const.
- `internal/cli`: after `DeployPluginTree`, fill the review-contract marker from
  `reviewassets.ReviewExecutionContractFor(model.AgentAntigravityCLI)` and prepend
  frontmatter to `rules/gentle-ai-orchestrator.md` (single site, both install and sync).
- `internal/components/permissions`: antigravity CLI overlay gains `allow` entries
  `command(gentle-ai codegraph init)`, `command(gentle-ai review assess)`,
  `command(gentle-ai review status)` (agy command targets match word-by-word prefixes).
- `internal/components/uninstall`: remove the legacy root `orchestrator.md` too.
- `bench/journeys_antigravity_cli.go` (j4500): assert frontmattered routing + filled
  orchestrator rule on first sync, idempotency on second sync, absence of root file.
- `docs/antigravity-cli/README.md` §2.3: document the always-on frontmatter mechanism.

Out of scope (recorded as follow-ups):
- context7 (npx) MCP server failing to load (separate defect).
- Satellites `orchestrator-{delegation,memory,skills}.md` stay at plugin root as
  on-demand reads (delegation is 33.6 KB, over the 24 KB rule-file cap).
- Any change to AGENTS.md section content.

## Constraints

- Per-file rule cap: 24,000 bytes (routing 18.2 KB OK; orchestrator 7.3 KB + rendered
  contract ~15.4 KB ≈ 22.7 KB OK). Aggregate rules budget: 20,000 tokens (fits).
- No import cycle: `internal/agents/antigravitycli` must NOT import
  `internal/components/reviewassets` (reviewassets/install.go imports internal/agents).
  The contract fill lives in `internal/cli` (already imports both).
- Frontmatter only on non-AGENTS rule files; AGENTS.md must stay plain markdown.
- Transpiled embedded assets regenerate from gentle-shell via
  `scripts/transpile-antigravitycli`; keep the marker const single-sourced from the
  adapter package.
- Authored line forecast: ~300 additions+deletions (under the 400 budget; no chain needed).
- ACTUAL (2026-09-27): 679 additions+deletions (225 insertions + 10 deletions in modified
  files + 444 new-file lines, ~65% of which are tests). Exceeds the 400 forecast; delivery
  strategy question (single PR exception vs chain) asked before first commit per rdd-defect-workflow.

## Acceptance criteria

- After `gentle-ai sync`, `~/.gemini/config/plugins/gentle-ai/rules/gentle-ai-routing.md`
  starts with `trigger: always_on` frontmatter and keeps the managed agent-routing section.
- `rules/gentle-ai-orchestrator.md` exists with frontmatter, contains the rendered
  review execution contract, and contains no unfilled insert marker; root
  `orchestrator.md` no longer exists.
- Headless `agy --print` probe from a neutral directory finds the ODD/TDD/RDD phrases
  previously absent.
- `go test ./...` green (focused packages first), j4500 driven run green.
- Uninstall removes the new rule file and the legacy root file.

## Checks

- Focused: `go test ./internal/components/filemerge/... ./internal/components/agentguidance/... ./internal/agents/antigravitycli/... ./internal/cli/... ./internal/components/permissions/...`
- Full: `go build ./... && go test ./...`
- Bench declarations: `cd bench && go test ./...`
- Driven: build binary + `gentle-ai-bench run --binary <bin> --only j4500-antigravity-cli-lifecycle-parity`
- Runtime: `gentle-ai sync --agents antigravity-cli` on real HOME, then `agy --print` probe.
- RDD: `gentle-ai review assess` per work-unit commit (mode: on/global).

## Tasks

- [x] T1: filemerge frontmatter helper (RED→GREEN) with unit tests — 8 tests green, idempotency covered
- [x] T2: agentguidance always-on frontmatter for agy routing rule (RED→GREEN) — RED observed (file started with marker), GREEN with dedicated rule-file test + negative test for all other carriers (pi subtest skipped: carrier resolves to live install)
- [x] T3: adapter remap orchestrator rule + legacy prune + marker const export (RED→GREEN) — RED was compile-undefined const; BundleAssets + DeployPluginTree green; transpiler --strict zero drift (const unified via adapter import)
- [x] T4: cli review-contract fill + frontmatter in plugin import step (RED→GREEN) — RED undefined fn; contract + frontmatter + idempotency (byte-identical second fill) green
- [x] T5: permissions allow entries for codegraph init + review assess/status (RED→GREEN) — union merge preserves user entries; idempotent overlay
- [x] T6: uninstall legacy root orchestrator.md + routing rule cleanup (RED→GREEN) — discovered pre-existing gap: uninstall never removed the agy routing rule (always-on rule would keep firing post-uninstall); fixed for agy; generic agent-routing section cleanup for other agents = follow-up
- [x] T7: bench j4500 assertions + docs §2.3 update — corpus checked, no old-layout pins (issue_3500 is opencode settings, unrelated)
- [x] T8: build, full tests, driven j4500 run — build OK; focused packages green; agentguidance+uninstall failure sets identical to clean tree (20=20, all pre-existing environmental pi/live-install); driven `gentle-ai-bench run --binary /tmp/gentle-ai-fixed --only j4500-antigravity-cli-lifecycle-parity` → **status: completed**
- [ ] T9: local sync + headless agy probe verification (record evidence)
- [ ] T10: work-unit commits + review assess per commit

## Progress

(2026-09-27) Diagnosis complete and empirically validated; user authorized fix
("haz el fix"). T1–T8 complete with RED→GREEN evidence; subagent writer channel
unusable (two stalls, pi-web-access signature) → fallback to inline execution per
Work Routing Ladder. Delivery strategy question pending before first commit (679
authored lines > 400 forecast).
