# Antigravity CLI Integration — Phase 4: CodeGraph & Bench Parity

This document details the architecture, design choices, integration mechanics, and verification evidence for **Phase 4: CodeGraph & Bench Parity** in the Google Antigravity CLI (`agy`) integration.

---

## 1. Architectural Overview: CodeGraph Compatibility & Reconciliation

In Gentle AI, structural codebase exploration (repo maps, symbol references, call graphs, impact analysis) must route through **CodeGraph** before broad filesystem searches.

### 1.1 Compatibility Model (`codeGraphReconciled`)
In `internal/components/communitytool/codegraph_contract.go`, Antigravity CLI is registered as a **reconciled agent**:
```go
var codeGraphCompatibilityTable = map[model.AgentID]codeGraphCompatibility{
    ...
    model.AgentAntigravityCLI: reconciledCompatibility(model.AgentAntigravityCLI, ""),
}
```

Unlike native agents where the third-party upstream `codegraph install` command configures the agent directly, Antigravity CLI's plugin architecture requires Gentle AI to manage the MCP configuration internally under `~/.gemini/config/plugins/gentle-ai/mcp_config.json`.

### 1.2 Canonical Tool Wiring Detection
Antigravity CLI requires strict command and argument validation. In `hasCodeGraphToolWiring`:
```go
if adapter.Agent() == model.AgentAntigravityCLI {
    path := adapter.MCPConfigPath(homeDir, "codegraph")
    data, err := os.ReadFile(path)
    return path, err == nil && hasCanonicalAntigravityCodeGraphServer(data)
}
```

The configuration is valid if and only if:
- The command is `codegraph` (or `codegraph.exe` on Windows).
- The arguments are strictly `["serve", "--mcp"]`.
- The configuration is housed inside the plugin's `mcp_config.json`.

### 1.3 Atomic Reconciler (`ReconcileAntigravityCLICodeGraph`)
When `NeedsAntigravityCLICodeGraphReconcile(homeDir)` detects that Antigravity CLI is installed but missing or having non-canonical CodeGraph wiring:
1. It reads `mcp_config.json` (or creates a new root object if absent).
2. It preserves all other existing MCP servers (e.g., `engram` or custom user servers).
3. It upserts the canonical `codegraph` configuration.
4. It writes atomically via `filemerge.WriteFileAtomic` to avoid race conditions.
5. If the configuration was already canonical, it returns `Changed: false` for strict idempotency.

---

## 2. Sync and Install Wiring

### 2.1 Managed Sync Step (`internal/cli/sync.go`)
During `gentle-ai sync`, `codeGraphGuidanceSyncStep.Run()` checks:
```go
if status.CLI == communitytool.AvailabilityAvailable && communitytool.NeedsAntigravityCLICodeGraphReconcile(s.homeDir) {
    reconciled, err := communitytool.ReconcileAntigravityCLICodeGraph(s.homeDir)
    if err != nil {
        return fmt.Errorf("sync Antigravity CLI CodeGraph wiring: %w", err)
    }
    if s.changedFiles != nil && reconciled.Changed {
        *s.changedFiles = append(*s.changedFiles, reconciled.Files...)
    }
}
```

### 2.2 Fresh Install Step (`internal/cli/run.go`)
During `gentle-ai install`, `communityToolInstallStep.Run()` passes `s.agents` to `installCommunityToolWithHomeAndAgents`. This ensures that even before a project state (`state.json`) is persisted on disk, the installer knows `model.AgentAntigravityCLI` was selected and configures CodeGraph in the fresh plugin directory.

---

## 3. Benchmark Parity: Journey `j4500`

To ensure Antigravity CLI maintains full feature parity and regression safety with the rest of Gentle AI's supported agents, we added benchmark journey `j4500-antigravity-cli-lifecycle-parity` in `bench/journeys_antigravity_cli.go`.

### 3.1 Lifecycle Journey Workflow
The journey tests the complete lifecycle of Antigravity CLI in a sandboxed environment:
1. **Fixture Setup**:
   - Initializes a base Git repository.
   - Creates a mock `agy` executable in the sandbox's PATH.
   - Deploys a canary file in `~/.config/antigravity/settings.json` representing an existing Antigravity Desktop IDE installation.
2. **Public Sync Deploy**:
   - Executes `gentle-ai sync --agents antigravity-cli`.
   - Validates that `rules/AGENTS.md` is populated with `<!-- gentle-ai:persona -->` and `<!-- gentle-ai:sdd-orchestrator -->`.
   - Validates that subagents, satellites (`orchestrator-*.md`), support contracts (`support/`), and review chains (`chains/`) are created.
   - Asserts that the desktop canary (`.config/antigravity/settings.json`) was NOT modified.
3. **Idempotency Check**:
   - Re-runs `gentle-ai sync --agents antigravity-cli`.
   - Asserts that zero bytes changed across rules and plugin bundle assets.
4. **Uninstall Cleanup**:
   - Executes `gentle-ai uninstall --agents antigravity-cli --components sdd --yes`.
   - Asserts that all satellites, support contracts, chains, and subagents are removed.
   - Asserts that the desktop canary is still 100% intact (proving zero cross-contamination between Desktop IDE and CLI).

---

## 4. Key Discoveries & Gotchas

### 4.1 Independent Go Module for `bench/`
- The `bench/` directory contains its own `go.mod` (`module github.com/gentleman-programming/gentle-ai/bench`).
- Running tests inside `bench/` requires targeting the bench module root rather than invoking `go test ./bench/...` from the repository root.

### 4.2 Test Hermeticity Guard (`gemini.LookPathOverride`)
- In `internal/components/communitytool/tool_test.go`, if the host machine has `gemini` in its PATH, `DetectStatus` could detect `AgentGeminiCLI` and alter expected test assertions.
- Added `gemini.LookPathOverride = func(string) (string, error) { return "", fmt.Errorf("gemini disabled in communitytool tests") }` in `tool_test.go` to hermetically isolate tests from host binaries.

---

## 5. Empirical Verification Evidence

All tests pass cleanly:

```bash
# Community Tool CodeGraph tests
$ go test ./internal/components/communitytool/... -run "Antigravity|CodeGraph"
ok      github.com/gentleman-programming/gentle-ai/v3/internal/components/communitytool 0.370s

# CLI CodeGraph sync and install tests
$ go test ./internal/cli -run "TestCodeGraphGuidanceSyncStepReconcilesAntigravityCLI|TestCommunityToolInstallStepPassesAgentsToInstaller"
ok      github.com/gentleman-programming/gentle-ai/v3/internal/cli 0.044s

# Bench journeys uniqueness and manifest consistency
$ cd bench && go test -v . -run "TestJourneyIDsAreUniqueAcrossSourceFiles|TestRegisteredJourneysMatchTheManifest"
=== RUN   TestJourneyIDsAreUniqueAcrossSourceFiles
--- PASS: TestJourneyIDsAreUniqueAcrossSourceFiles (0.00s)
=== RUN   TestRegisteredJourneysMatchTheManifest
--- PASS: TestRegisteredJourneysMatchTheManifest (0.00s)
PASS
ok      github.com/gentleman-programming/gentle-ai/bench 0.010s
```

---

## 6. Associated Work Units & Commits

This phase was implemented and verified across the following atomic conventional commits:
1. `feat(communitytool,cli): wire CodeGraph MCP reconciliation and guidance for antigravity-cli` — CodeGraph compatibility registration, atomic reconciler in `mcp_config.json`, and sync/install wiring.
2. `feat(bench): add j4500 lifecycle parity journey for antigravity-cli` — automated lifecycle parity journey `j4500` testing sync, idempotency, uninstallation, and desktop canary isolation.
3. `docs(antigravity-cli): document codegraph integration mechanics and benchmark lifecycle parity` — architectural documentation, reconciler mechanics, and test evidence.

