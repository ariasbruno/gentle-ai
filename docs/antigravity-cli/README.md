# Gentle AI™ — Google Antigravity CLI Integration

Welcome to the documentation for the **Google Antigravity CLI (`agy`)** integration in Gentle AI.

This fork introduces Google Antigravity CLI as a first-tier supported agent alongside Claude Code, Codex, OpenCode, and Pi. It implements full architectural parity across prompt injection, nested subagents, satellite contracts, lifecycle hooks, CodeGraph tool reconciliation, benchmark parity, and in-process Receipt-Driven Development (RDD) review transport.

---

## 1. Documentation Index

The integration was designed and implemented across 5 disciplined architectural phases, each backed by authentic technical documentation and empirical test evidence:

| Phase | Topic | Key Deliverables | Document |
|---|---|---|---|
| **Phase 1** | **Foundation & Registration** | Catalog registration, model detection, security overlay, union array merge, plugin MCP configuration | [`01-foundation.md`](01-foundation.md) |
| **Phase 2** | **Asset Transpiler & Guardrails** | Deterministic asset transpiler, 10 native ODD subagents, 4R review chain, satellite contracts, strict invariant validation | [`02-asset-transpiler.md`](02-asset-transpiler.md) |
| **Phase 3** | **Orchestration & Lifecycle Hooks** | `pluginBundleProvider` interface, bundle asset injection, nested subagents, fail-open PreInvocation, PreToolUse, and Stop hooks | [`03-orchestration-and-hooks.md`](03-orchestration-and-hooks.md) |
| **Phase 4** | **CodeGraph & Bench Parity** | Atomic CodeGraph reconciliation, sync/install wiring, journey `j4500` lifecycle parity benchmark | [`04-codegraph-and-benchmarks.md`](04-codegraph-and-benchmarks.md) |
| **Phase 5** | **Review Transport & Capabilities** | In-process review adapter (`agy`), role routing (`review-models.json`), curated presets, interactive TUI model picker | [`05-review-transport.md`](05-review-transport.md) |
| **Phase 6** | **Asset Ownership & Drift Policy** | Skill mirroring with provenance manifest, CI drift test, sync preset-exclusion transparency | [`06-asset-ownership-and-drift.md`](06-asset-ownership-and-drift.md) |
| **Phase 7** | **24 KB Cap Fix & Dynamic Sync** | Split routing rules into `rules/gentle-ai-routing.md` to beat AGY 24 KB rule cap, and dynamic subagent reconciliation via `assets.FS` | [`SUMMARY.md`](SUMMARY.md) |
| **Summary** | **Master Technical Summary & References** | End-to-end architecture, parity matrix, CLI specs, lifecycle hooks, official sources, and cross-references | [`SUMMARY.md`](SUMMARY.md) |

---

## 2. Architectural Highlights

### 2.1 Complete Filesystem & Security Boundaries
- **Directory Layout**: Antigravity CLI global configuration lives under `~/.gemini/antigravity-cli/` (`settings.json`, `review-models.json`), while Gentle AI's system prompt, subagents, chains, and contracts are encapsulated inside the plugin bundle under `~/.gemini/config/plugins/gentle-ai/` (`rules/AGENTS.md`, `rules/gentle-ai-routing.md`, `agents/`, `chains/`, `mcp_config.json`).
- **Zero Cross-Contamination**: Antigravity Desktop IDE settings (`~/.config/antigravity/`) are completely isolated and untouched by CLI operations.
- **Security Policy**: Denies destructive host commands, sensitive directories (`.ssh`, `.gnupg`, `.aws`, `.gemini`), and protects repository integrity.

### 2.2 In-Process Receipt-Driven Development (RDD)
- Unlike host-mediated runtimes that rely on plugin relays, Antigravity CLI captures review evidence via **in-process subprocess execution** (`agy --sandbox --output-format text`).
- Go retains full ownership of prompt materialization, token budgeting, JSON schema validation, and cryptographic receipt admission.
- Review roles (`review-risk`, `review-resilience`, `review-readability`, `review-reliability`, `review-refuter`, `review-validator`) can be mapped individually to specialized models, curated presets, or set to inherit CLI session defaults.

### 2.3 Modular Rules, Always-On Frontmatter, and the 24,000-Byte Cap
- Antigravity CLI auto-loads only a plugin's `rules/AGENTS.md`; every other rule file is parsed for a YAML frontmatter `trigger` and files without a valid trigger are silently dropped on every turn (`CORTEX_MEMORY_TRIGGER_UNSPECIFIED`). Gentle AI therefore ships every non-AGENTS rule file with `trigger: always_on` frontmatter:
  - `rules/AGENTS.md` (Persona, CodeGraph guidance, and Engram memory protocol — plain markdown, no frontmatter, loaded unconditionally by agy).
  - `rules/gentle-ai-routing.md` (ODD workflow, delegation routing, TDD policy, RDD boundaries, remote authorization — written by sync with always-on frontmatter).
  - `rules/gentle-ai-orchestrator.md` (coordinator harness, delegation ladder, memory and skill registry pointers, and the runtime-rendered Review Execution Contract filled in place of the deterministic insert marker at deploy time).
- **Per-file limit**: Antigravity CLI truncates rule files above 24,000 bytes. All three rule files stay under the cap (the orchestrator rule embeds the ~15 KB rendered review contract inside its ~7 KB body).
- **Aggregate budget**: always-on rules share a 20,000-token rules budget; the three files total well under it.
- The `orchestrator-delegation.md`, `orchestrator-memory.md`, and `orchestrator-skills.md` satellites stay at the plugin root as on-demand reads referenced by the orchestrator rule (the delegation satellite alone exceeds the per-file cap, so it cannot be an always-on rule file).

### 2.4 Dynamic Self-Reconciling Subagent Tree
- `DeployPluginTree` dynamically discovers embedded subagents via `assets.FS.ReadDir` and reconciles `~/.gemini/config/plugins/gentle-ai/agents/`.
- Obsolete or renamed subagents (e.g. legacy SDD agents) are automatically pruned on `gentle-ai sync` without requiring manual file deletion.

### 2.5 Resilient Fail-Open Lifecycle Hooks
- Dispatched via `hooks.json` to `gentle-ai hook run --agent antigravity-cli --event <event>`.
- Implements a 3x identical-call tool loop circuit breaker in `PreToolUse` and candidate change gating in `Stop`.
- Any unhandled event or syntax error yields `{}` and exit code 0, guaranteeing zero workflow lockouts.

---

## 3. Quick Start & Usage

### 3.1 Building and Installing from the Fork

```bash
# Clone the fork
git clone https://github.com/ariasbruno/gentle-ai.git
cd gentle-ai

# Checkout the integration branch
git checkout antigravity-cli-integration

# Build and install the gentle-ai binary
go install ./cmd/gentle-ai
```

### 3.2 Setting up Antigravity CLI

```bash
# Install and register the Antigravity CLI plugin
gentle-ai install --agent antigravity-cli

# Run sync to reconcile all managed assets and rules
gentle-ai sync

# Verify system health
gentle-ai doctor
```

---

## 4. Upstream Synchronization Strategy

To keep this fork up to date with future updates and releases of upstream `Gentleman-Programming/gentle-ai`:

1. **Remote Configuration**:
   ```bash
   git remote add upstream https://github.com/Gentleman-Programming/gentle-ai.git
   git remote -v
   ```

2. **Fetching Upstream**:
   ```bash
   git fetch upstream main --tags
   ```

3. **Rebasing the Clean Integration**:
   Because all Antigravity CLI commits are organized as clean, isolated architectural work units on `antigravity-cli`, rebasing on top of new upstream releases is straightforward:
   ```bash
   git checkout antigravity-cli
   git rebase upstream/main
   ```
   *Note: Always verify test suites after rebase:*
   ```bash
   go test ./...
   cd bench && go test ./...
   ```
