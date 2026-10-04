# Antigravity IDE vs Antigravity CLI — Integration Comparison

This document explains how gentle-ai integrates the two Antigravity surfaces:
the IDE (desktop app, `model.AgentAntigravity`) and the CLI (terminal,
`model.AgentAntigravityCLI`, "agy"). They share a name and a vendor but are
two completely different integrations in gentle-ai.

## Who owns what

| | IDE (`AgentAntigravity`) | CLI (`AgentAntigravityCLI`) |
|---|---|---|
| Integrated by | Upstream (Gentle-Programming) | The fork (ariasbruno) |
| Fork touches it | No — standing rule | Yes — entire integration |

## Adapter paths

| Path | IDE | CLI |
|---|---|---|
| ConfigPath (catalog) | `~/.gemini/antigravity-cli` (COLLISION — open flag) | `~/.gemini/antigravity-cli` |
| GlobalConfigDir | `~/.gemini/config` (if migrated) → `~/.gemini/antigravity` → variant dir | `~/.gemini/antigravity-cli` (fixed) |
| VariantDir | `~/.gemini/antigravity-desktop` or `~/.gemini/antigravity-cli` | N/A |
| SystemPromptFile | `~/.gemini/GEMINI.md` (StrategyAppendToFile) | Plugin `rules/AGENTS.md` |
| SkillsDir | `<variantDir>/skills/` | Plugin `skills/` |
| SubAgentsDir | `""` (none — FileSubAgents: false) | Plugin `agents/` |
| MCPConfigPath | `<variantDir>/mcp_config.json` | Plugin `mcp_config.json` |

## Capability manifest

| Feature | IDE | CLI |
|---|---|---|
| FileSubAgents | **false** | **true** (10 native agents) |
| Skills | true | true |
| SystemPrompt | true | true |
| MCP | true | true |
| RDD runtime | **No** (not in closed set) | **Yes** (in closed set) |
| Review transport | **None** (unsupported) | `antigravitycli_in_process` |

## RDD (the biggest behavioral difference)

The IDE is **not** in the closed RDD set (`internal/model/rdd.go`):
- Its routing render uses the **non-RDD** Close step (no native review, no kill
  switch, no refuter vocabulary).
- Its orchestrator carries **no** Review Execution Contract, no Provider Defect
  Handoff, no Delegated Verification Gate RDD variant.
- It has **no review transport** — no adapter in `review_transport_capability.go`.

The CLI **is** in the closed RDD set:
- Full RDD vocabulary in its guidance.
- In-process review transport: fresh `agy --sandbox` subprocess per reviewer.
- Review models resolved via `~/.gemini/antigravity-cli/gentle-ai/review-models.json`.

## Guidance delivery

| | IDE | CLI |
|---|---|---|
| Routing | Full RenderRouting (ladder + triggers + TDD, non-RDD variant) merged with the orchestrator → append to `GEMINI.md` | ODD protocol only (~4.8KB) as dedicated always_on rule |
| Orchestrator | Merged into the routing (the v4-canonical delivery for every non-Pi, non-CLI agent) | Transpiled from gentle-shell canon → plugin rule with review contract filled |
| Satellites | Not received (no transpiled canon) | 3 files at plugin root (delegation ~36.6KB, memory, skills) — read on-demand |
| Frontmatter | N/A (no trigger-based rules) | `trigger: always_on` required on every rule file |
| Doctrine source | One: gentle-ai repo (RenderRouting + RenderOrchestratorWithSource) | Two: gentle-shell canon (transpiled) + ODD protocol from gentle-ai |

## Components

| Component | IDE | CLI |
|---|---|---|
| Skills | From internal catalog (selection-driven) | From gentle-shell canon (transpiled, bundle-owned, 13 entries) |
| Engram | Plugin-owned registration with cooperative transfer lock (#797, #1635) | MCP server in plugin mcp_config.json + PreInvocation hook forcing 14 tools |
| Permissions | NIL — IDE manages via UI (Artifact Review Policy) | Deny/allow overlay in settings.json |
| Context7 | Global variantDir mcp_config.json (serverUrl HTTP remote) | **Plugin mcp_config.json** (npx stdio, junto con engram y codegraph) |
| CodeGraph | Standard flow (codegraph install --target) | Reconciled in plugin mcp_config.json |
| Hooks | None — IDE has its own lifecycle | 5 events (hooks.json + hook.sh/cmd, fail-open) |
| Plugin tree | None | Full tree via DeployPluginTree + agy plugin install |
| Subagents | None (FileSubAgents: false) | 10 native (explore, verify, worker, jd-fix, judges, 4R lenses) |
| Chains | None | 4r-review.chain.md |
| Support contracts | None | strict-tdd.md + strict-tdd-verify.md |
| Provenance | N/A | PROVENANCE.json + drift gate + CI |

## Uninstall

The IDE uninstall removes context7 from the global mcp_config.json and cleans
managed catalog skills and GEMINI.md sections. The CLI uninstall removes the
entire plugin tree (rules, agents, skills, chains, hooks, mcp_config, manifest
entry, routing parts).

## agy runtime artifacts (not managed by gentle-ai)

When agy loads a plugin, it generates runtime data outside the plugin tree:
- `~/.gemini/antigravity-cli/plugin_data/` — agy's per-plugin runtime state.
- `~/.gemini/antigravity-cli/mcp/<plugin>_<mcp>/` — namespaced MCP server
  working directories (the `<plugin>_<server>` convention landed in agy 1.2.2
  to prevent collisions). For our plugin: `gentle-ai_engram/` and
  `gentle-ai_codegraph/`.

These are agy runtime artifacts — gentle-ai does not write or manage them.
They appear after `agy plugin install` and persist across sessions. The
uninstall removes the plugin tree but may leave these runtime dirs; agy's own
`/plugin` uninstall command handles them.
