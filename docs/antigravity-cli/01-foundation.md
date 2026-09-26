# Antigravity CLI Integration — Phase 1: Foundation & Architecture

This document records the foundational architecture, design decisions, security model, and verification evidence for integrating **Google Antigravity CLI (`agy`)** as a first-class, fully supported agent (`TierFull`) within the Gentle AI ecosystem.

---

## 1. Problem Statement & Motivation

Google Antigravity CLI (`agy`) is Google's terminal-native agentic development harness. Unlike the Antigravity Desktop IDE (which relies on editor windows and implicit workspace state), the CLI version operates on:
- Dedicated terminal sessions with SQLite-backed conversation transcripts (`~/.gemini/antigravity-cli/conversations/<id>.db`).
- A centralized application data directory (`~/.gemini/antigravity-cli`).
- A shared plugin architecture (`~/.gemini/config/plugins/<plugin-name>`).
- Built-in multi-agent delegation via `invoke_subagent`.

Prior to this integration, Gentle AI had partial support for the desktop IDE (`AgentAntigravity`), but no support for the CLI (`agy`). The goal of Phase 1 is to establish the agent identity, directory hierarchy, capability manifest, security policy, and persistence overlays without modifying or breaking existing agent adapters.

---

## 2. Directory Layout & Architecture Duality

Antigravity CLI employs an intentional separation of concerns between its runtime session data and its plugin configurations:

```
~/.gemini/
├── antigravity-cli/                   # [GlobalConfigDir] User & Settings Root
│   ├── settings.json                 # Permissions overlay (allow/deny rules)
│   └── gentle-ai/
│       └── review-models.json        # Review role model routing
│
└── config/plugins/gentle-ai/          # [PluginDir] Gentle AI Plugin Bundle
    ├── plugin.json                   # Antigravity CLI plugin manifest
    ├── mcp_config.json               # MCP servers configuration (Engram, CodeGraph)
    ├── rules/AGENTS.md               # System prompt / orchestrator rules
    ├── agents/                       # 24 nested subagent definitions
    ├── chains/                       # Multi-agent review & SDD chains
    ├── hooks.json                    # PreInvocation and Stop lifecycle hooks
    └── hooks/                        # Platform execution shims (hook.sh / hook.cmd)
```

### Key Design Choice: Plugin Isolation
Rather than injecting prompts and subagent definitions into the global `~/.gemini/antigravity-cli` folder, Gentle AI installs itself as a modular plugin under `~/.gemini/config/plugins/gentle-ai/`. This provides:
1. **Zero Contamination**: Clean uninstall and upgrade without touching personal CLI preferences.
2. **Deterministic Discovery**: Antigravity CLI automatically discovers skills, subagents, and MCP servers located inside enabled plugins.

---

## 3. Capability Manifest & Agent Registration

In `internal/model/types.go`, we registered:
```go
const AgentAntigravityCLI AgentID = "antigravity-cli"
```

The adapter is implemented in `internal/agents/antigravitycli/adapter.go` and implements the `agents.Adapter` contract:
- **Tier**: `model.TierFull`.
- **Detection**: Probes for binary `agy` via `exec.LookPath` and directory existence of `~/.gemini/antigravity-cli`.
- **System Prompt Strategy**: `model.StrategyAppendToFile` targeting `rules/AGENTS.md`.
- **MCP Strategy**: `model.StrategyMCPConfigFile` targeting `mcp_config.json`.
- **Subagents**: Supported via directory-based Markdown definitions (`SupportsSubAgents() = true`).
- **Auto-install**: Returns `SupportsAutoInstall() = false`, providing clear instructions pointing to Google's official installer.

In `internal/agents/capabilitymanifest/manifest.go`, the manifest is registered with byte-stable SHA256 verification and explicitly advertises review transport capabilities.

---

## 4. Security & Permissions Overlay Engine

### The Array-Merge Overwrite Bug & Fix
Antigravity CLI enforces tool and command permissions via `settings.json` under `permissions.allow`, `permissions.ask`, and `permissions.deny`:
```json
{
  "permissions": {
    "allow": ["command(git status)", "command(git diff)"],
    "deny": ["command(sudo rm -rf /)"]
  }
}
```

During testing, we discovered a systemic defect in `internal/components/filemerge/json_merge.go`: the generic object merger replaced slices wholesale (`result[key] = overlayValue`). When Gentle AI injected permissions into an existing `settings.json`, it wiped out all user-configured rules.

**The Fix**:
In `internal/components/filemerge/json_merge.go`, we introduced union-merging for permission rule arrays:
- Existing user-defined `allow`, `ask`, and `deny` rules are preserved.
- Injected rules are unioned into the array without duplicates.
- Order is preserved, and missing keys are created cleanly.

---

## 5. Engram MCP Integration

Gentle AI provisions the Engram persistent memory server into Antigravity CLI via `internal/components/engram/inject.go`:
1. **Plugin Manifest (`plugin.json`)**:
   ```json
   {
     "$schema": "https://antigravity.google/schemas/v1/plugin.json",
     "name": "gentle-ai",
     "description": "Gentle-AI — ecosystem, frameworks, and workflows for AI coding agents."
   }
   ```
2. **MCP Server Wiring (`mcp_config.json`)**:
   Unlike sandboxed agents that require restricted tools, Antigravity CLI receives the full Engram tool suite (`args: ["mcp"]`).
3. **Idempotency**: Successive calls to `Inject()` verify zero drift and report `Changed: false`.

---

## 6. Verification Evidence

The entire foundation layer passes automated unit and boundary tests:

```bash
# Model, Capability Manifest, Adapter, and Catalog tests
go test -v ./internal/model/... ./internal/agents/capabilitymanifest/... ./internal/agents/antigravitycli/... ./internal/catalog/...
# Pass: 100%

# File merge and Permission overlay tests
go test -v ./internal/components/filemerge/... ./internal/components/permissions/...
# Pass: 100%

# Engram MCP injection and idempotency
go test -v ./internal/components/engram/ -run TestInjectAntigravityCLI
# Pass: 100%
```

---

## 7. Associated Work Units & Commits

This phase was implemented and verified across the following atomic conventional commits:
1. `feat(model,catalog): register AgentAntigravityCLI and adapter capabilities` — registers `model.AgentAntigravityCLI`, catalog capabilities, and adapter detection.
2. `feat(permissions,engram): configure security overlay, union array merge, and plugin mcp for antigravity-cli` — fixes array merge in `settings.json` and wires Engram MCP.
3. `docs(antigravity-cli): document foundation architecture, detection, and security model` — foundational documentation and verification evidence.

