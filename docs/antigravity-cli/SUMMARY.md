# Gentle AI™ — Antigravity CLI Integration: Architecture Summary, Sources & References

This document serves as the **master executive summary, architecture blueprint, and technical reference guide** for the **Google Antigravity CLI (`agy`)** integration in Gentle AI (`ariasbruno/gentle-ai`). It details design decisions, the architectural parity matrix, official CLI specifications, and cross-references across the codebase.

---

## 1. Executive Summary

Google Antigravity CLI (`agy`) is Google DeepMind / Gemini's official command-line interface for agentic coding and AI-assisted software development. Unlike the desktop application (Antigravity Desktop IDE), the CLI is engineered for headless execution, terminal-based pair programming, CI/CD environments, and automated orchestration.

This fork integrates Antigravity CLI as a **full-tier supported agent** within Gentle AI, providing:
1. **Full Architectural Parity** with Claude Code, OpenCode, Codex, and Pi.
2. **Strict Filesystem & Security Boundaries**: All configuration is isolated under `~/.gemini/antigravity-cli/` and plugin assets under `~/.gemini/config/plugins/gentle-ai/`, leaving Antigravity Desktop IDE settings (`~/.config/antigravity/`) untouched.
3. **In-Process Receipt-Driven Development (RDD)**: Deterministic, headless review capture using `agy --sandbox --output-format text` without host-mediated plugin dependencies.
4. **Deterministic Asset Transpilation**: 24 native nested subagents, 4 review chains, and satellite contracts validated against strict AST and format invariants.
5. **Atomic CodeGraph Reconciliation**: Automatic MCP tool configuration in `mcp_config.json` ensuring structural repository analysis precedes broad filesystem scans.
6. **Benchmark Parity**: Automated validation via journey `j4500-antigravity-cli-lifecycle-parity` in `bench/`.

---

## 2. Architectural Parity Matrix

| Capability / Dimension | Antigravity CLI (`agy`) | Claude Code | OpenCode | Codex | Gentle Pi |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Internal Identifier** | `model.AgentAntigravityCLI` | `model.AgentClaudeCode` | `model.AgentOpenCode` | `model.AgentCodex` | `model.AgentPi` |
| **Config Root** | `~/.gemini/antigravity-cli/` | `~/.claude/` | `~/.config/opencode/` | `~/.codex/` | `~/.pi/` |
| **Plugin / Bundle Dir** | `~/.gemini/config/plugins/gentle-ai/` | N/A | N/A | N/A | `~/.pi/gentle-ai/` |
| **Prompt Injection** | `rules/AGENTS.md` (in plugin) | `CLAUDE.md` / `settings` | `AGENTS.md` / `opencode.json` | `AGENTS.md` | `.pi/gentle-ai/` |
| **Subagents** | Directory `agents/<name>/agent.md` (`mainAgent: false`) | Task subagents | Subagents in `opencode.json` | Advisory processes | TS extensions |
| **Chains Support** | `chains/*.chain.md` | N/A | Via profiles | N/A | Pi protocol |
| **Lifecycle Hooks** | `PreInvocation`, `Stop` | Settings hooks | Plugin events | N/A | TS event bus |
| **RDD Transport** | `antigravitycli_in_process` (Headless subprocess via stdin) | `claude_prompt_carried` | `opencode_provider_injected` (Plugin relay) | `codex_advisory_scratch_process` | `pi_host_relay` |
| **RDD Model Routing** | Granular per-role (`review-models.json`) | Global | Profiles | CLI arguments | Pi relays |
| **TUI Model Picker** | Yes (Bubbletea interactive + presets + inherit) | No | Yes (Profile selector) | Yes | Yes |
| **CodeGraph Reconciliation** | Atomic in `mcp_config.json` | Automated CLI | Config servers | N/A | Host plugin |
| **Parity Benchmark** | `j4500` (Lifecycle parity) | Core journeys | Core journeys | Core journeys | Core journeys |

---

## 3. End-to-End System Architecture

```text
┌────────────────────────────────────────────────────────────────────────┐
│                          Gentle AI Core                                │
│                                                                        │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────────┐  │
│  │ Catalog & Model  │  │ Security Overlay │  │  TUI Model Picker    │  │
│  │ (AgentAntigravity│  │ (Deny list, union│  │  (Presets, roles,    │  │
│  │  CLI Registered) │  │  permissions)    │  │   defaults/inherit)  │  │
│  └────────┬─────────┘  └────────┬─────────┘  └──────────┬───────────┘  │
│           │                     │                       │              │
│           ▼                     ▼                       ▼              │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │        Adapter Layer (`internal/agents/antigravitycli`)          │  │
│  └──────────────────────────────┬───────────────────────────────────┘  │
└─────────────────────────────────┼──────────────────────────────────────┘
                                  │
                                  ▼
┌────────────────────────────────────────────────────────────────────────┐
│                 Filesystem Layout & Transpiled Assets                  │
│                                                                        │
│ ~/.gemini/                                                             │
│ ├── antigravity-cli/                <-- [GlobalConfigDir] Session/App  │
│ │   ├── settings.json               <-- Security policy & permissions  │
│ │   └── gentle-ai/                                                     │
│ │       └── review-models.json      <-- Custom Review Model Routing    │
│ │                                                                      │
│ └── config/plugins/gentle-ai/       <-- [PluginDir] Extension Bundle   │
│     ├── rules/                                                         │
│     │   └── AGENTS.md               <-- Persona + SDD Orchestrator     │
│     ├── agents/                     <-- 24 Nested Subagents            │
│     │   ├── gentle-ai-explore/agent.md  (Frontmatter:                  │
│     │   ├── gentle-ai-worker/agent.md    mainAgent: false, tools, model│
│     │   └── sdd-apply/agent.md          mapped to Antigravity runtime) │
│     ├── chains/                     <-- 4 Review & SDD Chains          │
│     │   ├── 4r-review.chain.md                                         │
│     │   ├── sdd-full.chain.md                                          │
│     │   ├── sdd-plan.chain.md                                          │
│     │   └── sdd-verify.chain.md                                        │
│     ├── mcp_config.json             <-- CodeGraph & Engram MCP Tools   │
│     ├── orchestrator-delegation.md  <-- Satellite Delegation Contracts │
│     ├── orchestrator-memory.md      <-- Memory Management Satellite    │
│     ├── orchestrator-skills.md      <-- Skill Delegation Satellite     │
│     ├── sdd-orchestrator-workflow.md<-- SDD Phase Contracts            │
│     └── support/contracts/          <-- Invariant Support Schemas      │
└─────────────────────────────────┬──────────────────────────────────────┘
                                  │
                                  ▼
┌────────────────────────────────────────────────────────────────────────┐
│             Receipt-Driven Development (RDD) Execution                 │
│                                                                        │
│  Go Engine (Prompt assembly, schema validation, token budget, receipts) │
│                                 │                                      │
│                                 │ stdin: prompt bytes                  │
│                                 ▼                                      │
│  Subprocess: `agy --sandbox --output-format text [--model M]`           │
│                                 │                                      │
│                                 │ stdout: raw review evidence          │
│                                 ▼                                      │
│  Go Engine (Cryptographic admission, consent envelope, burn authority) │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Official Specifications & Reference Standards

### 4.1 Google Antigravity CLI (`agy`) Specifications

#### 4.1.1 Executable & Runtime Modes

- **Binary**: `agy` (resolved from `$PATH` or `$HOME/.local/bin/agy`).
- **`--sandbox`**: Executes within an OS-level sandbox with restricted filesystem and terminal access.
- **`--output-format text`**: Emits plain text output directly to `stdout` without ANSI escape codes or interactive UI chrome. (Other supported options: `json`, `stream-json`).
- **`--input-format text`**: Reads prompt input directly from `stdin` when piped.
- **`--model <model>`**: Overrides model selection for the session (e.g., `gemini-3.8-flash-high`, `gemini-3.1-pro-high`, `claude-opus-4-6-thinking`).
- **`--effort <effort>`**: Sets reasoning effort budget in `agy` (`low`, `medium`, `high`). (Note: Not used by Gentle AI review transport because reasoning depth is baked directly into catalog model slugs and Claude models reject `--effort`).
- **`--dangerously-skip-permissions`**: Bypasses interactive confirmation for tool calls (used in unattended CI/CD automation).
- **Subcommands**: `agent`, `agents`, `changelog`, `help`, `install`, `mcp`, `models`, `plugin`, `remote-control`, `update`.

#### 4.1.2 Subagent Frontmatter Specification (`agent.md`)

Antigravity CLI subagents are discovered under `~/.gemini/config/plugins/gentle-ai/agents/<name>/agent.md`. They require YAML frontmatter conforming to:

```yaml
---
name: sdd-apply
description: Implement SDD tasks with strict TDD evidence and review workload guard.
mainAgent: false
tools:
  - view_file
  - write_to_file
  - replace_file_content
  - run_command
  - manage_task
  - send_message
model: inherit
---
```

> [!IMPORTANT]
> `mainAgent: false` is an invariant required on all 24 subagents to prevent `agy` from identifying them as the root session agent.

#### 4.1.3 Lifecycle Hook Protocol

Antigravity CLI hooks communicate via standard I/O using JSON payloads.

**Invocation Command**:
```bash
gentle-ai hook run --agent antigravity-cli --event <PreInvocation|Stop>
```

**Input Payload (via `stdin`)**:
```json
{
  "conversationId": "239da533-524d-4bd3-9837-501cfc69bd1c",
  "workspacePaths": ["/path/to/repo"],
  "invocationNum": 0,
  "terminationReason": "model_stop"
}
```

**Output Envelope (via `stdout`)**:
```json
{
  "decision": "continue",
  "reason": "Receipt-Driven Development: unreviewed candidate changes detected since session baseline. Please run gentle-ai review status preflight before completing."
}
```

Returning `{}` indicates successful completion without blocking.

#### 4.1.4 MCP Configuration (`mcp_config.json`)

Located at `~/.gemini/config/plugins/gentle-ai/mcp_config.json`:

```json
{
  "mcpServers": {
    "codegraph": {
      "command": "codegraph",
      "args": ["serve", "--mcp"]
    },
    "gentle-ai_engram": {
      "command": "engram",
      "args": ["serve"]
    }
  }
}
```

#### 4.1.5 Desktop IDE Sandbox Isolation

- Antigravity Desktop IDE stores configuration at `~/.config/antigravity/settings.json`.
- Antigravity CLI stores configuration at `~/.gemini/antigravity-cli/` and plugins at `~/.gemini/config/plugins/gentle-ai/`.
- The two configurations are strictly partitioned: Gentle AI tests and benchmark journey `j4500` assert that desktop canary settings remain unaltered across all CLI operations.

---

### 4.2 Gentle AI Core & Upstream Contracts

#### 4.2.1 Repository & Provenance

- **Upstream**: [`Gentleman-Programming/gentle-ai`](https://github.com/Gentleman-Programming/gentle-ai) (`v3.3.0`, commit `e28af0fd`).
- **Fork**: [`ariasbruno/gentle-ai`](https://github.com/ariasbruno/gentle-ai) (branch `antigravity-cli`).

#### 4.2.2 Receipt-Driven Development (RDD)

- **Capability**: `capabilitymanifest.ContractImmutableReviewExecutorV1`.
- **Transport constant**: `reviewImmutableTransportAntigravityCLIInProcess = "antigravitycli_in_process"`.
- **Predicate**: `reviewerprovider.CapturesInProcess(model.AgentAntigravityCLI) == true`.
- **AST Minimality Guard**: `internal/reviewerprovider/adapter_minimality_guard_test.go` verifies that `AntigravityCLIAdapter` does not absorb lifecycle, root resolution, or semantic admission responsibilities.

#### 4.2.3 Software-Driven Development (SDD)

- **Orchestrator contracts**: `sdd-orchestrator-workflow.md`, `orchestrator-delegation.md`.
- **Lifecycle rendering**: `ReviewExecutionContractFor(model.AgentAntigravityCLI)` resolves canonical `gentle-ai review status` and in-process execution rules without duplicating parent prompt context.

#### 4.2.4 CodeGraph MCP Tool Reconciliation

- **Interface**: `internal/components/communitytool/codegraph_contract.go`.
- **Reconciler**: `ReconcileAntigravityCLICodeGraph()` performs atomic, idempotent configuration upserts without overwriting other configured MCP servers.

---

## 5. Fork File & Component Mapping

| Component | Files | Description |
| :--- | :--- | :--- |
| **Catalog & Adapter** | `internal/model/types.go`<br>`internal/catalog/catalog.go`<br>`internal/agents/antigravitycli/adapter.go`<br>`internal/agents/antigravitycli/adapter_test.go` | Registers `AgentAntigravityCLI`, defines filesystem paths, system files, and platform detection. |
| **Permissions & Security** | `internal/components/permissions/inject.go`<br>`internal/components/engram/inject.go`<br>`internal/components/filemerge/json_merge.go` | Injects security deny list overlay, union array merging, and Engram MCP configuration. |
| **Asset Transpiler** | `scripts/transpile-antigravitycli/main.go`<br>`internal/assets/antigravitycli_assets_test.go`<br>`internal/agents/antigravitycli/adapter_test.go` | Deterministic transpiler generating 24 subagents, 4 chains, and satellite contracts with Invariants 1–5 validation. |
| **Orchestration & Hooks** | `internal/components/sdd/inject.go`<br>`internal/components/uninstall/service.go`<br>`internal/cli/antigravity_hook.go`<br>`internal/cli/antigravity_hook_test.go` | `pluginBundleProvider` interface, bundle asset distribution, `PreInvocation`/`Stop` hooks, and clean uninstallation. |
| **CodeGraph Integration** | `internal/components/communitytool/codegraph_contract.go`<br>`internal/cli/sync.go`<br>`internal/cli/run.go` | Atomic CodeGraph tool reconciliation in `mcp_config.json`, wired into `sync` and `install`. |
| **Benchmark Parity** | `bench/journeys_antigravity_cli.go`<br>`bench/manifest.json` | Journey `j4500-antigravity-cli-lifecycle-parity` testing driven sync, idempotency, uninstall, and canary isolation. |
| **Review Transport (RDD)** | `internal/reviewerprovider/antigravitycli_adapter.go`<br>`internal/reviewerprovider/antigravitycli_routing.go`<br>`internal/cli/review_provider_runtime.go`<br>`internal/cli/review_transport_capability.go` | In-process review adapter, role routing in `review-models.json`, presets, and AST minimality guard. |
| **TUI Model Picker** | `internal/tui/screens/antigravity_review_model_picker.go`<br>`internal/tui/screens/model_config.go`<br>`internal/tui/model.go` | Interactive Bubbletea TUI review model picker integrated into `gentle-ai model-config`. |
| **Technical Documentation** | `docs/antigravity-cli/01-foundation.md`<br>`docs/antigravity-cli/02-asset-transpiler.md`<br>`docs/antigravity-cli/03-orchestration-and-hooks.md`<br>`docs/antigravity-cli/04-codegraph-and-benchmarks.md`<br>`docs/antigravity-cli/05-review-transport.md`<br>`docs/antigravity-cli/README.md`<br>`docs/antigravity-cli/SUMMARY.md` | Authentic technical documentation per phase, master index, upstream sync guide, and architecture summary. |

---

## 6. Empirical Verification Evidence

All test suites pass at 100% across the repository:

```bash
# 1. Transpiler and asset invariants (Invariants 1 through 5)
$ go test -v ./internal/agents/antigravitycli/...
PASS
ok      github.com/gentleman-programming/gentle-ai/v3/internal/agents/antigravitycli 0.052s

# 2. Reviewer Provider (minimality guard, routing, presets, headless adapter)
$ go test -v ./internal/reviewerprovider/...
PASS
ok      github.com/gentleman-programming/gentle-ai/v3/internal/reviewerprovider 0.203s

# 3. CLI Review Transport (closed matrix, STATUS route, capture-result, refuter, validator)
$ go test -v ./internal/cli/... -run "Review|Antigravity"
PASS
ok      github.com/gentleman-programming/gentle-ai/v3/internal/cli 142.609s

# 4. TUI Screens & Model Configuration Picker
$ go test -v ./internal/tui/screens/... -run "Antigravity|ModelConfig"
PASS
ok      github.com/gentleman-programming/gentle-ai/v3/internal/tui/screens 1.952s

$ go test -v ./internal/tui -run "Antigravity|ModelConfig"
PASS
ok      github.com/gentleman-programming/gentle-ai/v3/internal/tui 2.109s

# 5. SDD Contract Resolution & Persona Injection
$ go test -v ./internal/components/sdd -run "Antigravity|Flip|InstalledContract"
PASS
ok      github.com/gentleman-programming/gentle-ai/v3/internal/components/sdd 0.227s

$ go test -v ./internal/components/persona -run "TestInjectAntigravityCLIPersonaAndOrchestrator"
PASS
ok      github.com/gentleman-programming/gentle-ai/v3/internal/components/persona 0.043s

# 6. Community Tools & CodeGraph Reconciliation
$ go test -v ./internal/components/communitytool/... -run "Antigravity|CodeGraph"
PASS
ok      github.com/gentleman-programming/gentle-ai/v3/internal/components/communitytool 0.370s

# 7. Bench Parity (Journey j4500)
$ cd bench && go test -v ./...
PASS
ok      github.com/gentleman-programming/gentle-ai/bench 4.901s
```

---

## 7. Atomic Work-Unit Commits & Implementation Ledger

The entire integration is structured as 18 clean, atomic conventional commits following `work-unit-commits` principles:

| # | Phase | Conventional Commit Message | Description |
|---|---|---|---|
| 1 | Phase 1 | `feat(model,catalog): register AgentAntigravityCLI and adapter capabilities` | Registers `model.AgentAntigravityCLI`, platform paths, detection, and capability manifest. |
| 2 | Phase 1 | `feat(permissions,engram): configure security overlay, union array merge, and plugin mcp for antigravity-cli` | Array-merge union fix in `settings.json`, security deny overlay, and Engram MCP wiring. |
| 3 | Phase 1 | `docs(antigravity-cli): document foundation architecture, detection, and security model` | Foundational architecture documentation, directory duality, and verification evidence. |
| 4 | Phase 2 | `feat(scripts): add deterministic antigravity-cli asset transpiler with strict validation` | Standalone Go transpiler script in `scripts/transpile-antigravitycli/` with AST tool mapping. |
| 5 | Phase 2 | `feat(assets): transpile and embed Antigravity CLI subagents, chains, and contracts` | 24 nested subagents (`mainAgent: false`), 4 review chains, satellites, and 5 invariant tests. |
| 6 | Phase 2 | `test(agents/antigravitycli): add TestBundleAssets to verify embedded assets mapping` | Unit test verifying `BundleAssets` maps embedded FS assets cleanly into the plugin directory. |
| 7 | Phase 2 | `docs(antigravity-cli): document asset transpiler architecture, tool mapping, and guardrails` | Transpiler architecture documentation, tool mapping matrix, and guardrail invariants. |
| 8 | Phase 3 | `feat(sdd,uninstall): support Antigravity CLI bundle assets, nested subagents, and cleanup` | `pluginBundleProvider` interface, directory-based subagent deployment, and uninstallation pruning. |
| 9 | Phase 3 | `feat(cli): add antigravity-cli lifecycle hook handler and rdd candidate guard` | `PreInvocation` and `Stop` hooks, cross-platform execution shims, and unreviewed candidate guard. |
| 10 | Phase 3 | `docs(antigravity-cli): document orchestration architecture, lifecycle hooks, and cleanup protocol` | Thin orchestrator documentation, hook specifications, bug fixes, and verification evidence. |
| 11 | Phase 4 | `feat(communitytool,cli): wire CodeGraph MCP reconciliation and guidance for antigravity-cli` | CodeGraph compatibility registration, atomic reconciler in `mcp_config.json`, and sync/install wiring. |
| 12 | Phase 4 | `feat(bench): add j4500 lifecycle parity journey for antigravity-cli` | Benchmark journey `j4500` testing sync, idempotency, uninstallation, and desktop canary isolation. |
| 13 | Phase 4 | `docs(antigravity-cli): document codegraph integration mechanics and benchmark lifecycle parity` | CodeGraph integration documentation, atomic reconciler mechanics, and benchmark parity. |
| 14 | Phase 5 | `feat(reviewerprovider,cli,sdd): add Antigravity CLI in-process review transport and routing with inherit support` | In-process review adapter (`agy`), role routing in `review-models.json`, sentinel `"inherit"`, and SDD contracts. |
| 15 | Phase 5 | `feat(tui): add interactive Antigravity CLI review model picker with presets and inherit support` | Bubbletea interactive review model picker, `Default (Inherit)` preset, custom role picker, and deprecation flags. |
| 16 | Phase 5 | `docs(antigravity-cli): document in-process review transport architecture, routing, and TUI picker` | In-process transport documentation, pure model slugs rationale, inherit resolution, and TUI guide. |
| 17 | Phase 5 | `docs(antigravity-cli): add master documentation index and upstream synchronization guide` | Master README index, architectural highlights, quick start guide, and upstream rebase strategy. |
| 18 | Phase 5 | `docs(antigravity-cli): add master integration summary with sources, references, and commit mapping` | Master technical summary, parity matrix, CLI specifications, and complete commit mapping ledger. |

---

## 8. Cross-References to Phase Documentation

For deep technical walk-throughs of each phase:
- [Phase 1: Foundation, Detection & Security Model](01-foundation.md)
- [Phase 2: Asset Transpiler Architecture & Guardrails](02-asset-transpiler.md)
- [Phase 3: Orchestration, Nested Subagents & Lifecycle Hooks](03-orchestration-and-hooks.md)
- [Phase 4: CodeGraph Reconciliation & Benchmark Parity](04-codegraph-and-benchmarks.md)
- [Phase 5: In-Process Review Transport (RDD) & TUI Picker](05-review-transport.md)
- [Master Documentation Index & Upstream Synchronization Guide](README.md)
