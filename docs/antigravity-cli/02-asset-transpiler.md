# Antigravity CLI Integration — Phase 2: Deterministic Asset Transpiler

This document details the architecture, design decisions, tool-mapping engine, and verification invariants for the **deterministic asset transpiler** that translates subagents, chains, and contracts from [Gentle Pi (`gentle-shell`)](https://github.com/Gentleman-Programming/gentle-shell) into embedded Go assets for Google Antigravity CLI.

---

## 1. Principle: Zero Manual Edits

In a multi-agent coding ecosystem, manually maintaining 10 ODD subagents, the 4R review chain, and multiple satellite contracts across different agent harnesses is an anti-pattern. Prompts inevitably drift, instructions become inconsistent, and bug fixes in upstream agents fail to propagate.

**The Golden Rule**:
> Never manually edit files under `internal/assets/antigravitycli/`.
> When a prompt, tool grant, or workflow contract changes in Gentle Pi, update `gentle-shell` and run the transpiler to deterministically regenerate the embedded assets.

---

## 2. Transpiler Engine Architecture

The transpiler is implemented as a standalone Go CLI tool in `scripts/transpile-antigravitycli/main.go`.

### Source Resolution Hierarchy
The transpiler locates the upstream `gentle-shell` source tree in order:
1. `--source-dir <path>` command-line flag.
2. Environment variables: `GENTLE_SHELL_DIR` or `GENTLE_PI_DIR`.
3. Sibling local checkouts: `../gentle-shell` or `../gentle-pi`.
4. Remote fallback: Shallow git clone of `https://github.com/Gentleman-Programming/gentle-shell` to a temporary directory (automatically purged on exit).

### AST & Tool Mapping Matrix
Gentle Pi and Google Antigravity CLI expose different primitive tool names for file operations and execution. The transpiler parses the YAML frontmatter of each agent and performs deterministic tool translation:

| Gentle Pi Tool | Antigravity CLI Tool | Description |
|---|---|---|
| `read` | `view_file` | Read text and binary files within viewport bounds |
| `edit` | `replace_file_content` | Precision chunk replacements in existing files |
| `write` | `write_to_file` | Create or overwrite files |
| `glob` | `find_by_name` | File and directory pattern discovery |
| `grep` | `grep_search` | Ripgrep-backed regex/literal text searching |
| `bash` | `run_command` | Shell execution with foreground/background management |
| `subagent_run` | `invoke_subagent` | Native agent delegation |
| MCP tools (e.g. `mem_*`, `codegraph_*`) | Pass-through | Directly mapped via MCP configuration |

---

## 3. Key Design Choices & Gotchas

### Hiding Subagents from the Interactive Picker (`mainAgent: false`)
**Problem**: In Antigravity CLI, any agent definition located in a plugin's `agents/` directory automatically appears in the interactive `/agents` menu and `agy agents` CLI listing. Displaying all 10 specialized subagents (e.g., `jd-judge-a`, `review-risk`) cluttered the user's agent selection and invited accidental manual invocation of agents designed strictly for machine delegation.

**Solution**:
Antigravity CLI supports the `mainAgent: false` frontmatter property. The transpiler injects `mainAgent: false` into every transpiled subagent:
```yaml
---
name: gentle-ai-explore
description: Read-only exploration and mapping for generic non-SDD work.
tools: [view_file, list_dir, grep_search, find_by_name]
mainAgent: false
---
```
This keeps the interactive `/agents` UI clean for the user while leaving all 10 subagents 100% discoverable and callable by the orchestrator via `invoke_subagent`.

### Strict Validation Mode (`--strict`)
When executed with `--strict` (in CI or during pre-commit checks), the transpiler enforces:
- Zero unmapped tools.
- Non-empty descriptions and valid YAML fences.
- Mandatory `mainAgent: false` presence.
- Strict error codes (`exit 1`) upon encountering any upstream drift.

---

## 4. Go Embed Integration (`embed.FS`)

The transpiled assets are stored in `internal/assets/antigravitycli/` and registered into the binary via `internal/assets/assets.go`:
- `agents/`: 24 subagent Markdown definitions.
- `chains/`: Multi-agent execution chains (`4r-review`).
- `support/`: SDD status and Strict TDD contracts.
- Satellites: `orchestrator-delegation.md`, `orchestrator-memory.md`, `orchestrator-skills.md`, `sdd-orchestrator-workflow.md`.

---

## 5. Architectural Invariants (The 5 Guardrails)

In `internal/assets/antigravitycli_assets_test.go`, we established 5 automated invariant tests that run on every build:

1. **Invariant 1: Subagents Census**: Asserts that exactly 10 subagents are present in `embed.FS` and that no duplicate or missing agents exist. (The SDD-era census was 24; the ODD migration reduced the set and the invariant moved with it.)
2. **Invariant 2: Frontmatter Fidelity**: Validates that every subagent contains non-empty name and description, a valid tools array, and `mainAgent: false`.
3. **Invariant 3: Satellites Integrity**: Proves that all 3 orchestrator satellite files exist and exceed minimum non-trivial byte sizes.
4. **Invariant 4: Support Contracts**: Proves that Strict TDD and status contracts exist and match specifications.
5. **Invariant 5: Chains Integrity**: Asserts that all 4 execution chain definitions exist, declare valid sequence steps, and reference real agents.

---

## 6. Verification Evidence

```bash
# Deterministic transpilation run
go run ./scripts/transpile-antigravitycli --strict
# Output: Successfully transpiled Antigravity CLI assets from Gentle Pi.

# Verify all 5 Architectural Invariants
go test -v ./internal/assets -run TestAntigravityCLI_
# === RUN   TestAntigravityCLI_Invariant1_Subagents
# --- PASS: TestAntigravityCLI_Invariant1_Subagents (0.00s)
# === RUN   TestAntigravityCLI_Invariant2_Frontmatter
# --- PASS: TestAntigravityCLI_Invariant2_Frontmatter (0.00s)
# === RUN   TestAntigravityCLI_Invariant3_Satellites
# --- PASS: TestAntigravityCLI_Invariant3_Satellites (0.00s)
# === RUN   TestAntigravityCLI_Invariant4_SupportContracts
# --- PASS: TestAntigravityCLI_Invariant4_SupportContracts (0.00s)
# === RUN   TestAntigravityCLI_Invariant5_Chains
# --- PASS: TestAntigravityCLI_Invariant5_Chains (0.00s)
# PASS

# Verify Adapter Bundle Assets mapping
go test -v ./internal/agents/antigravitycli -run TestBundleAssets
# PASS
```

---

## 7. Associated Work Units & Commits

This phase was implemented and verified across the following atomic conventional commits:
1. `feat(scripts): add deterministic antigravity-cli asset transpiler with strict validation` — Go transpiler script in `scripts/transpile-antigravitycli/` with AST tool mapping.
2. `feat(assets): transpile and embed Antigravity CLI subagents, chains, and contracts` — transpiled subagents, chains, satellites, and architectural invariant guardrails in `internal/assets/`.
3. `test(agents/antigravitycli): add TestBundleAssets to verify embedded assets mapping` — unit test verifying `BundleAssets` maps embedded FS assets cleanly.
4. `docs(antigravity-cli): document asset transpiler architecture, tool mapping, and guardrails` — comprehensive transpiler and invariant documentation.

