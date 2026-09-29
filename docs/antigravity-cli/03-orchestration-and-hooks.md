# Antigravity CLI Integration — Phase 3: Orchestration & Lifecycle Hooks

This document details the architecture, implementation, bug fixes, and verification evidence for **Phase 3: Orchestration & Lifecycle Hooks** in the Google Antigravity CLI (`agy`) integration.

---

## 1. Architectural Overview & Design Philosophy

Antigravity CLI is designed to run in a continuous interactive terminal loop with subagent spawning (`invoke_subagent`). In Gentle AI, substantial work must be coordinated rather than dumped entirely into an oversized prompt.

### 1.1 The Thin Orchestrator Pattern
Unlike traditional monolithic IDE prompt injections that inline hundreds of lines of phase guidelines and continuation contracts into the system prompt, Antigravity CLI adopts **Gentle Pi's thin orchestrator model**:
- The always-on rules stay lean: `rules/AGENTS.md` carries identity, persona boundaries, Engram protocol, and CodeGraph guidance, while `rules/gentle-ai-routing.md` and `rules/gentle-ai-orchestrator.md` (both `trigger: always_on`) carry the ODD/TDD/RDD routing and the coordinator harness.
- Detailed delegation, memory, and skill-registry material is **lazy-loaded on demand** from plugin bundle assets:
  - `orchestrator-delegation.md`, `orchestrator-memory.md`, `orchestrator-skills.md` (satellites at the plugin root, referenced by the orchestrator rule)
  - `support/strict-tdd.md`, `support/strict-tdd-verify.md`
  - `chains/4r-review.chain.md`

> [!NOTE]
> The SDD-era assets (`sdd-orchestrator-workflow.md`, `support/sdd-status-contract.md`, and the `sdd-*` chains) were retired by the ODD migration and are pruned from the plugin tree on every sync.

### 1.2 Bundle Deployment (formerly `pluginBundleProvider`)
The Phase-3 `pluginBundleProvider` capability and its `sdd.Inject()` copy loop were retired with the SDD component. The current mechanism is `Adapter.DeployPluginTree` in `internal/agents/antigravitycli/adapter.go`: it writes the embedded bundle assets, discovers the transpiled subagents through `assets.FS`, writes each `agents/<name>/agent.md`, installs the fail-open hooks, and prunes legacy SDD files — reconciling the installed tree on every install and sync. The `antigravityCLIPluginImportStep` in `internal/cli/run.go` then finalizes the orchestrator rule and performs the one-time native `agy plugin` import.

```go
type pluginBundleProvider interface {
    PluginDir(homeDir string) string
    BundleAssets() [][2]string
}
```

Any adapter implementing this interface automatically participated in:
1. **Atomic bundle injection**: Copying and rendering embedded satellite and contract files into the adapter's plugin directory.
2. **Clean uninstall operations**: Pruning bundle files and empty subdirectories (`support/`, `chains/`) during `gentle-ai uninstall`.

> [!NOTE]
> The interface itself is retired; `BundleAssets()` and `PluginDir()` live on the adapter and still drive bundle deployment and uninstall cleanup today.

---

## 2. Directory-Based Subagents Deployment

Antigravity CLI requires subagents to reside in dedicated subdirectories containing an `agent.md` file:
```
~/.gemini/config/plugins/gentle-ai/agents/
├── gentle-ai-explore/
│   └── agent.md
├── gentle-ai-worker/
│   └── agent.md
├── gentle-ai-verify/
│   └── agent.md
└── ... (10 ODD subagents total)
```

### 2.1 Subagent Deployment & Verification (ODD era)
The SDD-era copy loop in `internal/components/sdd/inject.go` — including its `sdd-apply`/`sdd-verify` post-injection check — was retired with the SDD component. Today `Adapter.DeployPluginTree` writes every transpiled `agents/<name>/agent.md` from `assets.FS` and prunes legacy subagents, and the coverage guarantees moved to the invariant tests and the driven bench:
- `internal/assets/antigravitycli_assets_test.go` asserts exactly 10 subagents with valid frontmatter on every build.
- Journey `j4500` asserts on every driven run that the deployed plugin tree contains the current worker and reviewer subagents, non-empty and loadable.

### 2.2 Tool Grants (formerly scoped `codegraph_explore` injection)
The Phase-3 `injectCodeGraphToolGrantIntoPrompt` enhancement in `internal/components/sdd/prompts.go` was retired with the SDD component. Subagent tool sets now come entirely from the transpiled `agent.md` frontmatter: the current read-only `gentle-ai-explore` ships `tools: [view_file, list_dir, grep_search, find_by_name]` and does not carry `codegraph_explore`; the parent session keeps the CodeGraph MCP tool (`gentle-ai_codegraph/codegraph_explore`) available through the plugin's `mcp_config.json`.

---

## 3. Lifecycle Hooks & Receipt-Driven Development (RDD)

Antigravity CLI executes lifecycle hooks defined in `~/.gemini/config/plugins/gentle-ai/hooks.json`. Gentle AI injects automated hooks to guarantee session integrity.

### 3.1 Hook Definitions and Execution Shims
During injection (`ensureAntigravityCLIHooks`), Gentle AI deploys:
1. `hooks.json`:
   ```json
   {
     "gentle-ai": {
       "PreInvocation": [
         {
           "type": "command",
           "command": "gentle-ai hook run --agent antigravity-cli --event PreInvocation",
           "timeout": 30
         }
       ],
       "Stop": [
         {
           "type": "command",
           "command": "gentle-ai hook run --agent antigravity-cli --event Stop",
           "timeout": 60
         }
       ]
     }
   }
   ```
2. Cross-platform scripts:
   - `hooks/hook.sh`: POSIX shell script with mode `0755` (`exec gentle-ai hook run --agent antigravity-cli --event "$1"`).
   - `hooks/hook.cmd`: Windows command batch script with mode `0644` (`@echo off\r\ngentle-ai.exe hook run --agent antigravity-cli --event %1\r\n`).

### 3.2 Event Handlers (`internal/cli/antigravity_hook.go`)
- **`PreInvocation` (Turn Start)**:
  - When `invocationNum == 0` (first turn of the session):
    - Triggers silent refresh of `.atl/skill-registry.md` if repository skills changed.
    - Records the initial git repository baseline target identity in `~/.gentle-ai/review-stop-hook/v1/<conversationId>.json`.
  - Emits `{}` so execution proceeds without pause.
- **`Stop` (Turn End & Model Completion)**:
  - If `terminationReason != "model_stop"`, returns `{}`.
  - If `terminationReason == "model_stop"`, evaluates whether unreviewed tracked git changes exist in the workspace since session baseline.
  - If unreviewed candidates are detected, returns a continuation payload:
    ```json
    {
      "decision": "continue",
      "reason": "Receipt-Driven Development: unreviewed candidate changes detected since session baseline. Please run gentle-ai review status preflight before completing."
    }
    ```
  - Records that a reminder was issued to prevent spamming the user on subsequent stops for the same unchanged candidate identity.

---

## 4. Key Gotchas & Bug Fixes

### 4.1 Upstream v3.3.0 OpenCode Compatibility Guard
Upstream `main` received `v3.3.0` which added OpenCode v2 features (`opencode_v2.go`, `opencode_runtime.go`, and `RefreshInstalledOpenCodePlugins` in `internal/components/sdd/inject.go`).
- **Gotcha**: A naive branch merge or file-level checkout (`git checkout feat/antigravity-cli-integration -- internal/components/sdd/`) would have obliterated these upstream v3.3.0 files.
- **Resolution**: Ported strictly the `Antigravity CLI` injection diffs (step 1c, step 3d, subagents directory handling, and hooks deployment), preserving all v3.3.0 OpenCode v2 logic untouched.

### 4.2 Empty `--agent` Flag in `reviewStopHookResolveTargetIdentity`
- **Root Cause**: In `internal/cli/review_stop_hook.go`, `reviewStopHookResolveTargetIdentity` unconditionally constructed arguments as `[]string{"--cwd", root, "--contract", ReviewIntegrationContractV2, "--agent", runtimeAgent, "--next-transition"}`.
- **Impact**: When invoked from `RunAntigravityCLIHook`, `runtimeAgent` is `""`. Passing `--agent ""` caused `runReviewStatus` to fail with an argument parsing error. As a result, `ok` was `false` and baseline identity was never recorded.
- **Fix**:
  ```go
  statusArgs := []string{
      "--cwd", root, "--contract", ReviewIntegrationContractV2,
      "--next-transition",
  }
  if runtimeAgent != "" {
      statusArgs = append(statusArgs, "--agent", runtimeAgent)
  }
  ```

### 4.3 Mise Environment Isolation in `TestEngramPathGuidanceDefault`
- **Root Cause**: `TestEngramPathGuidanceDefault` asserted that the default guidance contains `go/bin`. In environments using `mise` (or custom `GOBIN`/`GOPATH`), the helper dynamically prints the active Mise shims path instead of the hardcoded fallback.
- **Fix**: Added `t.Setenv("GOBIN", "")` and `t.Setenv("GOPATH", "")` at test initialization.

---

## 5. Empirical Verification Evidence

All tests pass cleanly across affected packages:

```bash
$ go test -v ./internal/cli -run "TestAntigravityHook"
=== RUN   TestAntigravityHookPreInvocationBaselineAndSilent
--- PASS: TestAntigravityHookPreInvocationBaselineAndSilent (0.12s)
=== RUN   TestAntigravityHookStopModelStopUnreviewedCandidate
--- PASS: TestAntigravityHookStopModelStopUnreviewedCandidate (0.26s)
PASS
ok      github.com/gentleman-programming/gentle-ai/v3/internal/cli      0.401s

$ go test ./internal/components/sdd/... ./internal/components/uninstall/... ./internal/cli/... -run "Antigravity|Hook"
ok      github.com/gentleman-programming/gentle-ai/v3/internal/components/sdd          0.196s
ok      github.com/gentleman-programming/gentle-ai/v3/internal/components/uninstall   0.114s
ok      github.com/gentleman-programming/gentle-ai/v3/internal/cli                     2.687s
```

All 10 ODD subagents, bundle assets, lifecycle hooks, and uninstall plans are fully verified and ready for production use.

---

## 6. Associated Work Units & Commits

This phase was implemented and verified across the following atomic conventional commits:
1. `feat(sdd,uninstall): support Antigravity CLI bundle assets, nested subagents, and cleanup` — `pluginBundleProvider` interface, directory-based subagents injection, and clean uninstallation.
2. `feat(cli): add antigravity-cli lifecycle hook handler and rdd candidate guard` — `PreInvocation` and `Stop` hooks implementation, platform shims, and RDD candidate detection.
3. `docs(antigravity-cli): document orchestration architecture, lifecycle hooks, and cleanup protocol` — orchestration patterns, hook lifecycle specifications, and test evidence.

