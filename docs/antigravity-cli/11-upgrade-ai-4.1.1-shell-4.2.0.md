# Incremental update: AI 4.1.1 / Shell 4.2.0 / AGY 1.3.4

Status: in progress. Compatibility and feature coverage are not certified yet. This update preserves the accepted AGY integration and does not reconstruct the historical fork or prepare an upstream PR.

## Fixed inputs

| Input | Value |
| --- | --- |
| Accepted AGY baseline | `antigravity-cli-v4`, `5048df524016bf0e37305229fafa1642f8e75d56` |
| Prior incorporated AI / merge base | `5140c5f55baf91763198eb0bfca019c015e3b015` |
| AI v4.1.1 | `898d3976e8049a7238eaa7de60bff325c30d2b3d` (peeled release tag) |
| Prior Shell provenance | `0da07ce949952268b290bf751f22ad1202a40fc3` |
| Shell v4.2.0 | `c41ebd591358d393cf19c5ead981f67640a600ba` |
| Validation runtime | `agy --version`: exit 0, stdout `1.3.4\n` |
| Update branch | `chore/agy-integration-ai-v4.1.1-shell-v4.2.0` |

Previous validation runtime version: not independently established. AGY's observed changelog supplies release-specific evidence, not authenticated model proof. Target Shell is a clean detached source worktree; do not substitute the earlier Shell 4.1.0 attempt.

## Baseline evidence

Code baseline `5048df524`; passive workflow commit `30926f096` adds no Go changes. `go test ./... -count=1 -timeout=12m -json` ran under Go 1.27.1 with private HOME/XDG/TMP, in-project Go caches, offline module resolution and a minimal environment excluding live Pi agent-home variables.

Result: exit **1**, elapsed **513.083 seconds**, stderr empty. Preserve these eight leaf failures (parent/package failure events are not extra cases):

| Leaf test | Observed failure |
| --- | --- |
| `TestInstallThenSyncConverges/codex` | Codex version subprocess fails: mise config is untrusted. |
| `TestCodexReinstallKeepsConfigTOML` | Same Codex version/mise failure. |
| `TestCodeGraphInstallThenSyncConverges/codex` | Same Codex version/mise failure. |
| `TestInstallLeavesNoDanglingSharedReferences/codex` | Same Codex version/mise failure. |
| `TestSyncCodexGentlemanConvergesWithHooksJSON` | Same Codex version/mise failure. |
| `TestGoldenPersona_Windsurf_Gentleman` | Golden mismatch, first difference at byte 4715. |
| `TestGoldenCombined_Windsurf` | Golden mismatch, first difference at byte 4715. |
| `TestGoldenPersona_Antigravity_Gentleman` | Golden mismatch, first difference at byte 4715. |

The suite has environmental/native-tool dependencies despite fixture isolation. Do not change host trust settings or install Codex to hide this evidence. Candidate failures must be compared against this exact command/environment; a new failure is not baseline by assertion.

Raw local receipts: authorized `/tmp/gentle-ai-agy-411-shell420-01a136ee/evidence/baseline.{env.json,result.json,stdout.jsonl,stderr.txt}`. They contain exact commands, results and outputs; the summary here is portable evidence, not a claim that the baseline passed.

## Impact inventory

Inventory before filtering: 831 changed AI paths over the fixed AI range and 409 changed Shell paths over the original provenance range. Complete local Git logs/diffs and expected merge-simulation exit 1 are retained in `.git/agy-upgrade/update420/impact-evidence.json`.

| Group | AGY applicability / treatment | Required evidence |
| --- | --- | --- |
| Shared AI install/sync, scoped paths, routing and permissions | Reconcile upstream contracts with AGY extensions; preserve user ownership. | Affected package tests and AGY public install/sync lifecycle. |
| AI review-provider contracts and routing | Preserve Go-owned bindings and AGY transport behavior; capability declarations alone are not proof. | Public adapter/routing tests; report real native/model proof separately. |
| Shell roles, core, phase/support files and skills | Regenerate from the fixed clean source with native translations. | Two-run path/byte equality, provenance and structural plugin validation. |
| Shell Pi SDK/extensions/session plumbing | Not transplanted into AGY runtime. Guidance may still affect translated contracts. | Residue/semantic checks; no invented AGY tools. |
| Other AI agents, OS installers, telemetry and catalog updates | Inherit through merge, not port into AGY. Check shared consumers if resolutions touch them. | Final suite comparison; do not treat all inherited changes as new AGY features. |
| AGY 1.3.4 CLI changes | Native runtime boundaries; preserve explicit model/effort semantics and inspect plugin/agent discovery. | Isolated version/help/plugin checks; no authentication or model calls authorized. |

Ten textual conflicts: `bench/journeys.go`, `bench/journeys_id_collision_test.go`, `bench/testdata/journeys.manifest`, `internal/cli/run.go`, `internal/components/agentguidance/inject.go`, `inject_test.go`, `routing.go`, `internal/components/filemerge/json_merge_test.go`, `internal/components/permissions/inject_test.go`, `internal/tui/model.go`.

Earlier merge `82da47c43` has exactly the same accepted AGY and target AI parents. Its conflict resolutions are reusable proposals under matching inputs, not permission to copy the whole previous tree. Validate each affected intention and preserve independently added upstream and AGY tests/journeys.

## Feature coverage (pending candidate checks)

| Feature/change | Status / disposition | Evidence |
| --- | --- | --- |
| Explorer `web_enable` | Required native disposition pending: optional Pi activation, not a new AGY search tool. | Target Shell explorer declares it; strict baseline-generator preview exits 1 and omits it. Native `search_web`/`read_url_content` projection must remain. |
| Updated risk-gated verification and foreground/native delegation guidance | Pending translated semantic checks. | Shell canonical inputs; must not regress to older broad replacements or claim Pi session/background tools. |
| AGY plugin list `--json`, complete installed list/version/enabled state | Native 1.3.4 feature; integration discovery checks pending. | Observed 1.3.4 changelog and help; no installation proof implied. |
| AGY default effort for base model names | Native 1.3.4 feature; retain explicit routing configuration. Authenticated execution unknown. | Observed changelog; no real model run authorized. |
| Unknown headless `--agent` failure and conflicting print flags | Native 1.3.4 behavior; check adapter implications without starting a model request. | Observed changelog: exit 1 for unknown agent, exit 2 for conflicting print flags. |
| Plugin root `agents.json`, skill discovery and plugin MCP access | Native inherited capabilities; evaluate generated layout rather than adding every optional format. | Observed 1.3.2/1.3.3 notes; generated plugin validation pending. |
| MCP OAuth, Remote Control and terminal/UI fixes | Runtime-managed; no AGY transpiler change established. | Changelog evidence, not exercised here. |

Do not label pending or unknown features as supported. Real installation, authentication and model/native receipt proof are outside the current grant.

## Generation and verification

Baseline-generator strict preview against Shell 4.2.0: **exit 1**, stderr reports `web_enable` omitted. Preview output is partial and not an accepted bundle. Keep `preview.*` receipts; correct required translation before rerunning.

Candidate checks pending: affected package suites, two clean-source generations and byte/provenance comparison, AGY plugin validation, applicable public lifecycle journey, final full suite against the baseline, independent high-risk verification and native risk-gated review. The committed-only `--check` gate must not be represented as validating dirty candidate outputs.

## Closure and contribution boundary

Compatibility: pending. Coverage: pending as classified above. Accepted baseline has not advanced. No push, PR, installation, authentication or model execution has occurred. Official upstream contribution readiness is not evaluated; it is a separate task, not a requirement to reconstruct this fork.
