# Antigravity CLI Integration — Phase 3: Orchestration & Lifecycle Hooks

This document details the architecture, implementation, bug fixes, and verification evidence for **Phase 3: Orchestration & Lifecycle Hooks** in the Google Antigravity CLI (`agy`) integration.

---

## 1. Architectural Overview & Design Philosophy

Antigravity CLI is designed to run in a continuous interactive terminal loop with subagent spawning (`invoke_subagent`). In Gentle AI, substantial work must be coordinated rather than dumped entirely into an oversized prompt.

### 1.1 The Thin Orchestrator Pattern
Unlike traditional monolithic IDE prompt injections that inline hundreds of lines of phase guidelines and continuation contracts into the system prompt, Antigravity CLI adopts **Gentle Pi's thin orchestrator model**:
- The always-on system prompt (`rules/AGENTS.md`) remains lean: it sets identity, persona boundaries, memory protocols, and the Work Routing Ladder.
- Detailed SDD workflows, status continuation contracts, and multi-agent chains are **lazy-loaded on demand** from plugin bundle assets:
  - `sdd-orchestrator-workflow.md`
  - `orchestrator-delegation.md`, `orchestrator-memory.md`, `orchestrator-skills.md`
  - `support/sdd-status-contract.md`, `support/strict-tdd.md`, `support/strict-tdd-verify.md`
  - `chains/sdd-full.chain.md`, `chains/sdd-plan.chain.md`, `chains/sdd-verify.chain.md`, `chains/4r-review.chain.md`

### 1.2 The `pluginBundleProvider` Interface
To support this modular injection cleanly without hardcoding agent-specific hacks into `sdd.Inject()`, we introduced the `pluginBundleProvider` capability:

```go
type pluginBundleProvider interface {
    PluginDir(homeDir string) string
    BundleAssets() [][2]string
}
```

Any adapter implementing this interface automatically participates in:
1. **Atomic bundle injection**: Copying and rendering embedded satellite and contract files into the adapter's plugin directory.
2. **Clean uninstall operations**: Pruning bundle files and empty subdirectories (`support/`, `chains/`) during `gentle-ai uninstall`.

---

## 2. Directory-Based Subagents Deployment

Antigravity CLI requires subagents to reside in dedicated subdirectories containing an `agent.md` file:
```
~/.gemini/config/plugins/gentle-ai/agents/
├── sdd-apply/
│   └── agent.md
├── sdd-verify/
│   └── agent.md
├── gentle-ai-explore/
│   └── agent.md
└── ... (24 subagents total)
```

### 2.1 Nested Injection & Post-Check Verification
In `internal/components/sdd/inject.go`, the subagents copy loop checks `entry.IsDir()`:
- Flat adapters (Cursor, Claude, Kimi) write directly to `agentsDir/<entry.Name()>`.
- `AgentAntigravityCLI` creates `agentsDir/<entry.Name()>/` and writes `agent.md` inside it.
- The post-injection verification validates that critical execution agents (`sdd-apply`, `sdd-verify`) exist and are non-empty:
  ```go
  checkPaths := []string{
      filepath.Join(agentsDir, phase+".md"),
      filepath.Join(agentsDir, phase+".yaml"),
  }
  if adapter.Agent() == model.AgentAntigravityCLI {
      checkPaths = append(checkPaths, filepath.Join(agentsDir, phase, "agent.md"))
  }
  ```

### 2.2 Scoped Tool Grants: `codegraph_explore`
Tool grants must not leak to read-only reviewers or authoring agents. In `internal/components/sdd/prompts.go`, `injectCodeGraphToolGrantIntoPrompt` was enhanced:
```go
if agentID == model.AgentAntigravityCLI && !strings.Contains(prompt[:frontmatterEnd], "name: gentle-ai-explore") {
    return prompt
}
```
For Antigravity CLI, only `gentle-ai-explore` receives the `codegraph_explore` grant in YAML list syntax (`[view_file, ..., codegraph_explore]`).

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

All 24 subagents, bundle assets, lifecycle hooks, and uninstall plans are fully verified and ready for production use.

---

## 6. Associated Work Units & Commits

This phase was implemented and verified across the following atomic conventional commits:
1. `feat(sdd,uninstall): support Antigravity CLI bundle assets, nested subagents, and cleanup` — `pluginBundleProvider` interface, directory-based subagents injection, and clean uninstallation.
2. `feat(cli): add antigravity-cli lifecycle hook handler and rdd candidate guard` — `PreInvocation` and `Stop` hooks implementation, platform shims, and RDD candidate detection.
3. `docs(antigravity-cli): document orchestration architecture, lifecycle hooks, and cleanup protocol` — orchestration patterns, hook lifecycle specifications, and test evidence.

