# Incremental update: AI 4.1.1 / Shell 4.2.0 / AGY 1.3.4

Status: implemented and verified locally within the scope below; final documentation record being closed. The global suite is not green and authenticated AGY behavior is not certified. This update preserves the accepted AGY integration; no historical fork reconstruction or upstream PR.

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

## Feature coverage and explicit limits

| Feature/change | Status / disposition | Evidence |
| --- | --- | --- |
| Explorer `web_enable` | Implemented native disposition: optional Pi activation, not a fabricated AGY tool. | Public generator RED/GREEN and independent suite; `search_web`/`read_url_content` retained. |
| Updated risk-gated verification and foreground/native delegation guidance | Delivered and structurally checked in native core/phase/role carriers. | Independent public routing tests, reproducible generation and scoped native reviews; actual model adherence remains unknown. |
| AGY plugin list `--json`, complete installed list/version/enabled state | JSON on an empty isolated profile observed; nonempty installed metadata not certified. | Actual AGY1.3.4 exit0/stdout `[]`; no installation performed. |
| AGY default effort for base model names | Native 1.3.4 feature; retain explicit routing configuration. Authenticated execution unknown. | Observed changelog; no real model run authorized. |
| Unknown headless `--agent` failure and conflicting print flags | Native 1.3.4 behavior; check adapter implications without starting a model request. | Observed changelog: exit 1 for unknown agent, exit 2 for conflicting print flags. |
| Plugin root `agents.json`, skill discovery and plugin MCP access | Native inherited capabilities; evaluate generated layout rather than adding every optional format. | Actual AGY1.3.4 validation processed12skills/10agents/1hook. Optional commands/MCP absent in the structural projection and explicitly skipped; not proof of effective MCP access. |
| MCP OAuth, Remote Control and terminal/UI fixes | Runtime-managed; no AGY transpiler change established. | Changelog evidence, not exercised here. |

Do not label pending or unknown features as supported. Real installation, authentication and model/native receipt proof are outside the current grant.

## Generation and verification

Baseline-generator strict preview against Shell 4.2.0: **exit 1**, stderr reports `web_enable` omitted. Preview output is partial and not an accepted bundle; original `preview.*` receipts are retained.

Public generator regressions observed RED then GREEN for native web activation disposition, risk-gated verification, native waiting, worker/verifier phase context, Pi session-tool removal, modular core, retired-workflow translation and byte-preserved non-Markdown resources. Clean-source rejection tests cover tracked/untracked/ignored dirt and unchanged destination data. `--strict --check` now rejects unknown tools instead of silently passing. Existing read/list behavior remains covered. Generator suite: exit 0, package 2.680 seconds.

Final core adds the shared-code guard before `Risk: none`. Two fresh final generations from the clean fixed Shell source produced **38 files with identical paths/bytes/provenance**. This validates reproducibility, not model behavior. The generator help now states clean-source and strict-check semantics; use `--source-dir <clean pinned clone>` and `--strict` explicitly.

New upstream routing guards evaluate AGY's public deployed core and phase/role content rather than falsely requiring all doctrine in its protocol-only routing string. Other-agent assertions are unchanged. The module baseline includes the AGY variant; its sanctioned `-update` refresh was inspected: unchanged prior raw assets and all non-RDD rows, plus the expected 92-byte RDD contract growth from the AGY runtime enum. Ratchet/routing checks without `-update`: exit 0, package 0.082 seconds.

Driven `j4500-antigravity-cli-lifecycle-parity` observed RED on obsolete root satellite paths, then **completed** after checks were adapted to all nine native phase rules, metadata/byte caps, idempotency and cleanup. Product and harness builds and bench declarations passed. This exercises the public product with a fake AGY process; it does not prove model execution.

The first seam sweep passed assets, AGY adapter, reviewerprovider, filemerge, permissions and TUI; its routing/baseline failures were corrected as above. Codex retired-workflow checking still fails on the same mise dependency in a clean AI v4.1.1 reference, confirming an upstream/environment failure rather than an AGY-only assertion.

A native validation probe initially failed because of a wrong import, then a wrong generic rendering API. Those are retained fixture failures, not product compatibility results. The actual AGY pipeline reads the deployed native core, fills its single provider contract marker and adds always_on metadata; the public driven journey already checks its final cap. Corrected structural projection and AGY 1.3.4 validation passed: exit 0, **12 skills, 10 agents and 1 hook** processed; commands/MCP absent and explicitly skipped. The core with the full native contract and production metadata is **23,892 bytes** (108 bytes below 24,000). `agy plugin list --json` returned exit 0, exact stdout `[]\n`, valid JSON under an isolated empty profile. This is not installation, authentication, effective model permission enforcement or real native receipt proof.

Final full candidate suite: **exit 1, subprocess 632.423 seconds, empty stderr** (background wrapper approximately 633 seconds). All eight pre-update leaf failures remain; five additional failures were observed. The two AGY documentation failures (missing catalog row and retired command example) were corrected; focused ratchets now pass (app 0.106s, catalog 0.004s). Two new Codex checks hit the same untrusted mise configuration. The additional `TestOwnershipReuseRejectsLedgerWithoutMutation/hash_number` failure is reproduced on pure AI 4.1.1: exit 1, package 0.008s, with the decoder adding the map key to the error field. At that point production and the security test were unchanged, and that variant had not reached its no-mutation assertions. The later authorized test-only correction and independent recheck below resolved it. The full suite has **not** passed and was not rerun after the focused documentation/test corrections. Exact comparison and reference failures are in `comparison.json` and `reference-ownership.*`.

Independent high-risk verification: **PARTIAL**, no unexplained update-caused blocker. Fresh staged snapshot generator suite passed (2.572s), routing/matrix/docs checks passed (agentguidance 0.186s, app 0.114s, catalog 0.005s), all 38 generated files matched, and the independently driven public journey completed once with zero model runs. The adapter package in the focused regex had no matching tests; it is not independently certified as a full suite. The journey used the previously built product/harness. The initial verdict marked S2/S3/S4 pass and S1 partial. The later blocker-limited recheck resolved the ownership assertion; it did not grant global compatibility or authenticated runtime certification.

The user chose the portable test correction. A minimal independent stdlib wire-schema mirror derives only the map-entry field spelling (`files` or `files.jd-judge-a.md`); production decoding remains unchanged. The test still pins the full wrapped error, type/offset/struct metadata, empty result and before/after file snapshots. Focused ownership rejection/current-acceptance checks passed (0.051s), then the complete reviewassets package passed (0.517s), both exit 0 with empty stderr. The earlier failed receipts remain. No alternate Go toolchain execution or new global-suite PASS is claimed.

The single blocker-limited independent recheck **passed**: public `InstallNativeAgents`, hash_number variant, exit 0, package 0.008s, empty stderr. S5 passes and the S1 ownership proof blocker is resolved; exact rejection metadata, empty result and unchanged file snapshots were observed independently. No production ownership changes, retries or new full verification sweep. Historical global failures remain documented, not a current suite PASS.

Native assessment reports high risk and reviewDue=true: 849 changed paths and approximately 92,686 diff lines, including inherited upstream. Authority inspection was ready with no lineages; ordinary START then stopped preflight with **lens_context_budget_exceeded**, mutation not started and **no review authority created**. No verdict, capture, acknowledgement or recovery exists. Immutable evidence was not truncated, and retrying the same candidate cannot succeed.

The user selected an upstream integration anchor plus bounded AGY adaptation reviews. The anchor was derived from the exact open-merge inputs and only the ten previously verified conflict resolutions, using an alternate index; the prior attempt's whole tree was not copied. Four adaptation commits then passed native reviews and exact acknowledgements, with authority explicitly burned on each. No RDD switch change, publication or recovery was performed.

| Local work unit | Commit | Native review outcome |
| --- | --- | --- |
| Incremental upstream anchor | `434feb5a1` | Explicitly not reaudited by user decision; exact merge parents/tree and final index preservation checked. |
| Generator, public regressions and CLI docs | `31db82239` | `review-6faaf7a83c570aa3`, four lenses, approved and consumed. |
| Shell4.2 bundle and public lifecycle assertions | `2c7934c4e` | `review-d336e73eda7ab1a4`, four lenses, approved and consumed. |
| Native routing and module ratchet | `9520887bc` | `review-833260080241fad9`, two lenses, approved and consumed. |
| Portable ownership proof and compatible living docs | `a89ab4211` | `review-ca0def8718ff2963`, three lenses, approved and consumed. |

R2-001 at `bench/journeys_antigravity_cli.go:221` is an informational readability advisory, not a blocker or correction. No follow-up source change or review rerun was made for it.

Final parent readback: all implementation/generated/test/compatibility-doc bytes equal the preserved final tree; only evolving report/task metadata differs. The reported ownership probe ran once as parent spot check: exit0, package0.007s, empty stderr. `--check --strict` against the clean pinned Shell source now passed against the **committed** bundle: exact stdout `Drift check passed: committed bundle matches provenance-pinned transpilation.\n`, exit0, empty stderr. These checks do not turn the historical full-suite result into PASS.

## Closure and contribution boundary

Compatibility demonstrated locally: pinned incremental source merge, deterministic generator and committed drift, delivered native guidance, fake-AGY public lifecycle, actual AGY1.3.4 structural validation, ownership no-mutation proof, and scoped native review acknowledgements. A clean global suite, authenticated AGY model/tool enforcement, nonempty installed-plugin metadata and unexercised runtime-managed features remain unclaimed.

The accepted `antigravity-cli-v4` branch and old attempts/stash remain unchanged. No push, PR, real installation or authentication occurred. No real AGY model was invoked; native outside views used13Pi host-relay reviewer runs. Official upstream contribution readiness is a separate unevaluated task. Rollback boundaries are the recorded work-unit paths; rollback/publication is not authorized merely by these results.

Risk: item5 (installer/environment integration), plus shared-consumer risk item3 where applicable. Independent verification and bounded native reviews are recorded above; they are not blanket upstream or runtime approval.
