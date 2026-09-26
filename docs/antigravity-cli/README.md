# Gentle AI™ — Google Antigravity CLI Integration

Welcome to the documentation for the **Google Antigravity CLI (`agy`)** integration in Gentle AI.

This fork introduces Google Antigravity CLI as a first-tier supported agent alongside Claude Code, Codex, OpenCode, and Pi. It implements full architectural parity across prompt injection, nested subagents, satellite contracts, lifecycle hooks, CodeGraph tool reconciliation, benchmark parity, and in-process Receipt-Driven Development (RDD) review transport.

---

## 1. Documentation Index

The integration was designed and implemented across 5 disciplined architectural phases, each backed by authentic technical documentation and empirical test evidence:

| Phase | Topic | Key Deliverables | Document |
|---|---|---|---|
| **Phase 1** | **Foundation & Registration** | Catalog registration, model detection, security overlay, union array merge, plugin MCP configuration | [`01-foundation.md`](01-foundation.md) |
| **Phase 2** | **Asset Transpiler & Guardrails** | Deterministic asset transpiler, 24 subagents, 4 review chains, satellite contracts, strict invariant validation | [`02-asset-transpiler.md`](02-asset-transpiler.md) |
| **Phase 3** | **Orchestration & Lifecycle Hooks** | `pluginBundleProvider` interface, bundle asset injection, nested subagents, PreInvocation and Stop hooks, clean uninstall | [`03-orchestration-and-hooks.md`](03-orchestration-and-hooks.md) |
| **Phase 4** | **CodeGraph & Bench Parity** | Atomic CodeGraph reconciliation, sync/install wiring, journey `j4500` lifecycle parity benchmark | [`04-codegraph-and-benchmarks.md`](04-codegraph-and-benchmarks.md) |
| **Phase 5** | **Review Transport & Capabilities** | In-process review adapter (`agy`), role routing (`review-models.json`), curated presets, interactive TUI model picker | [`05-review-transport.md`](05-review-transport.md) |
| **Phase 6** | **Asset Ownership & Drift Policy** | Skill mirroring with provenance manifest, CI drift test, sync preset-exclusion transparency, 2026-09-22 audit record | [`06-asset-ownership-and-drift.md`](06-asset-ownership-and-drift.md) |
| **Summary** | **Master Technical Summary & References** | End-to-end architecture, parity matrix, CLI specs, lifecycle hooks, official sources, and cross-references | [`SUMMARY.md`](SUMMARY.md) |

---

## 2. Architectural Highlights

### 2.1 Complete Filesystem & Security Boundaries
- **Directory Layout**: Antigravity CLI global configuration lives under `~/.gemini/antigravity-cli/` (`settings.json`, `review-models.json`), while Gentle AI's system prompt, subagents, chains, and contracts are encapsulated inside the plugin bundle under `~/.gemini/config/plugins/gentle-ai/` (`rules/AGENTS.md`, `agents/`, `chains/`, `mcp_config.json`).
- **Zero Cross-Contamination**: Antigravity Desktop IDE settings (`~/.config/antigravity/`) are completely isolated and untouched by CLI operations.
- **Security Policy**: Denies destructive host commands, sensitive directories (`.ssh`, `.gnupg`, `.aws`, `.gemini`), and protects repository integrity.

### 2.2 In-Process Receipt-Driven Development (RDD)
- Unlike host-mediated runtimes that rely on plugin relays, Antigravity CLI captures review evidence via **in-process subprocess execution** (`agy --sandbox --output-format text`).
- Go retains full ownership of prompt materialization, token budgeting, JSON schema validation, and cryptographic receipt admission.
- Review roles (`review-risk`, `review-resilience`, `review-readability`, `review-reliability`, `review-refuter`, `review-validator`) can be mapped individually to specialized models, curated presets, or set to inherit CLI session defaults.

### 2.3 CodeGraph Reconciliation
- Automatically validates and reconciles CodeGraph MCP tool wiring in `~/.gemini/config/plugins/gentle-ai/mcp_config.json`.
- Enforces structural codebase navigation before broad filesystem searches.

---

## 3. Quick Start & Usage

### 3.1 Building and Installing from the Fork

```bash
# Clone the fork
git clone git@github.com:ariasbruno/gentle-ai.git
cd gentle-ai

# Checkout the antigravity-cli branch
git checkout antigravity-cli

# Build and install the gentle-ai binary
go install ./cmd/gentle-ai
```

### 3.2 Setting up Antigravity CLI

```bash
# Run interactive installer and select Google Antigravity CLI
gentle-ai install

# Or sync configuration directly for Antigravity CLI
gentle-ai sync --agents antigravity-cli

# Configure review models and presets (optional)
gentle-ai model-config
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
