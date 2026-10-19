# Pi CodeGraph agent-scope sync fix

Objective: make `gentle-ai sync` respect the active agent selection for the Pi CodeGraph step and its backup paths, and scope the OpenCode/Antigravity CLI CodeGraph reconcilers to the selected agents. This stops a non-Pi sync from writing `~/.pi` and satisfies the "at minimum" path of #5141.

Branch: `agy/ai-v4.2.0_shell-v4.3.0`
Delivery: local work-unit commit; no push, PR, or release.
Test runner: `go test ./internal/cli ./internal/components/communitytool`; the full suite has known failures outside this change.

Follow-ups (not fixed here): #5108 (gentle-pi native CodeGraph ownership and stale overlay cleanup) and #4794 (install selection persistence and missing-manifest re-adoption).

## Specs

S1. A global `gentle-ai sync` with CodeGraph selected and a selection that does not include Pi MUST NOT stage or run `sync:community-tool:pi-codegraph`. The Pi MCP file and managed Pi overlays MUST remain byte-identical. This is the "or, at minimum" expectation in #5141.

S2. `syncBackupTargetsScoped` MUST include `communitytool.PiCodeGraphPaths` only when `selection.Agents` includes Pi. A non-Pi sync MUST NOT declare Pi overlay paths for rollback.

S3. `codeGraphGuidanceSyncStep` MUST reconcile OpenCode only when the selected agents include OpenCode, and MUST reconcile Antigravity CLI only when the selected agents include Antigravity CLI. An empty agent list preserves the previous detection-based behavior for direct callers and focused tests. This is the scope principle behind #5141.

S4. A Pi-selected sync MUST still run the Pi CodeGraph step and preserve the retired Pi SDD agent release behavior covered by #5219.

S5. The change documents #5141 as the immediate fix and #5108/#4794 as follow-ups. It does not attempt to fix those two issues.

## Tasks

| ID | Specs | Route | Status | Evidence |
| --- | --- | --- | --- | --- |
| T1 | S1-S3 | inline | done | `internal/cli/sync.go`: Pi step and backup gated on `containsAgent(..., model.AgentPi)`; `codeGraphGuidanceSyncStep.agents` scopes OpenCode/AGY reconcilers. |
| T2 | S1-S4 | inline | done | Updated `TestSyncRuntimeAddsCodeGraphStepsOnlyWhenSelected`, `TestRunSyncMigratesLegacyManagedPiCodeGraphSelection`, `TestRunSyncReleasesRetiredPiSDDAgentsFromCodeGraphManifest`; added `TestRunSyncNonPiSelectionDoesNotRefreshPiCodeGraph`. Focused packages pass. |
| T3 | S5 | inline | pending | Feature document and commit. |

## Log

L1. User request (verbatim): "no, solo alacance con follow ups. No te concentres en los issues"

L2. Decision: implement only the selected-agent scope fix. Reference #5141 in the change; record #5108 and #4794 as follow-ups rather than attempting their separate install/native-ownership work.

L3. Implemented the scope-only fix. Targeted sync tests and the full `internal/cli` + `internal/components/communitytool` packages pass. The change satisfies the "or, at minimum" path in #5141: a non-Pi sync no longer stages the Pi CodeGraph step or declares Pi overlay paths, and the CodeGraph guidance step only reconciles the selected OpenCode/AGY agent. #5108 and #4794 remain open follow-ups.

L4. Full `go test ./...` exited 1 with failures outside this change: `internal/cli` `TestRunDoctorToolExecution/canceled_launcher_child` (descendant still running) and a 10-minute package timeout running `TestReviewStartSelectsCanonicalFourRForSubprocessWrapper`; `internal/components/legacyassets` `TestReleasedSDDAgentDigestsMatchGenerator`; `internal/releaseprovenancecmd` `TestPartialReleaseIdentityNeverDowngrades` and `TestCIBuildStillProducesReleaseProvenance`; `internal/shellinstaller` `TestUserFrozenLockAcquisition`. The affected packages pass in isolation (`internal/cli` 381s, `internal/components/communitytool` 0.55s). No regression attributable to this change.
