# Incremental AGY update procedure

Preserve the accepted integration and adapt only the changes needed for the pinned release. Read this procedure before planning or executing an update.

## 1. Fix inputs and boundaries

Record the accepted AGY branch and full commit ID, its prior AI upstream reference/commit, the target AI and Shell releases and full commit IDs, the old Shell provenance commit, the previous and validation AGY versions, and the Git merge base. If prior release ancestry is ambiguous, resolve it rather than guessing from the branch name.

The merge base explains textual changes; the prior incorporated release defines the upstream feature inventory. They need not be the same commit. Record provenance and evidence for each input. Remote reads/fetches follow session authorization; a skill invocation does not grant access to a remote host.

Use the last accepted AGY integration, not merely the newest branch or an incomplete review-slices branch. Confirm acceptance with the human when unknown. Preserve dirty files without silently carrying them into another branch. Do not reset, clean, rewrite history, apply a stash, or overwrite an existing branch automatically.

Name the update branch `chore/agy-integration-ai-v{AI}-shell-v{Shell}`, for example `chore/agy-integration-ai-v4.1.1-shell-v4.1.0`. Versions label the attempt; full IDs identify inputs. Inspect an existing same-name branch and ask whether to resume it if its purpose or inputs differ. An AGY-only update with unchanged AI/Shell still needs an explicit attempt decision; do not invent a naming suffix.

Planning stays read-only: return proposed inputs, maps and checks in the response. Create the migration report or implementation tracking only when authorized. An isolated same-clone worktree can protect the original checkout during execution; do not move work outside permitted roots.

## 2. Establish evidence before integration

Record runner/toolchain versions, commands, environment, exact results and known failures. A clean tree is not a functional baseline. Use isolated HOME/config/temp fixtures with valid Git boundaries; do not test against real user settings. For this fork, scrub `PI_CODING_AGENT_DIR` and `GENTLE_PI_AGENT_HOME` from Go test subprocesses to avoid fixtures reaching the live agent home. External fixture paths require explicit permission.

Identify existing preservation tests at the affected AGY seams. Add missing tests before authored behavior changes when a meaningful deterministic RED exists; do not fabricate RED for a merge or a mechanical regeneration. Keep old-behavior checks for shared code touched by adaptations. A baseline failure requires the same check/environment on the comparable base; an untested attribution remains unknown.

## 3. Inventory changes before filtering

Inspect complete changes and release notes for AI from its prior incorporated release to the target, Shell from recorded provenance to target, and AGY from its previous validation version to target. Sources unavailable locally remain missing evidence unless remote access is authorized. Do not filter only by AGY filenames or Git conflicts.

Classify changed public behavior, contracts, defaults, permissions, lifecycle hooks, routing, roles, skills, templates, process boundaries, and native capabilities. Follow the dependencies consumed by AGY even where AGY files did not change. Record excluded groups with reasons; investigate unknown applicability. Reuse a stable seam map and prior evidence rather than re-auditing unchanged historical code.

For each relevant feature record upstream evidence, applicability, affected seam, implementation status, proof and disposition. Use `supported`, `equivalent`, `pending`, `not applicable`, or `unknown`. A support claim needs evidence at the boundary it advertises. Equivalent behavior needs its limits documented; unavailable runtime proof is not support. Optional features may be deferred by an explicit decision; broken required capabilities block closure. Do not implement every new upstream feature automatically.

## 4. Classify ownership and resolve conflicts

| Surface | Treatment |
| --- | --- |
| Uncustomized upstream code | Incorporate upstream intent; check affected AGY consumers. |
| Shared code with AGY adaptations | Reconcile both behavioral intentions. |
| AGY-only code | Preserve unless adaptation or proven replacement is needed. |
| Generated bundle | Change source/generator as needed and regenerate from clean pinned inputs. |
| User-owned configuration/content | Preserve under existing ownership rules; never bulk overwrite. |
| Unknown ownership | Investigate before replacement. |

Classify at the hunk or section level when files mix ownership. A generated-file replacement is expected only when provenance and reproducibility justify it. Git conflicts are textual; a clean merge can still introduce semantic conflicts through changed dependencies, defaults or removed contracts.

For each conflict compare the common base, accepted AGY side and target upstream side. Record both intentions, the chosen outcome and its public regression check. Do not choose whole-file `ours`/`theirs` by default. Previously recorded resolutions or `rerere` are proposals, not proof. If upstream replaces an AGY extension, establish equivalent required behavior before deleting the duplication. Ask the human when resolving a product choice or losing a required capability.

## 5. Integrate without rebuilding history

For authorized execution create the update branch from the accepted AGY tip and merge the fixed AI commit. Keep upstream changes and local adaptations distinguishable for review. Do not start at the upstream tip and copy the fork back, or recreate every historical fork commit to satisfy a reviewer budget.

Regenerate the complete AGY bundle when appropriate: whole-bundle generation is not whole-fork reconstruction. Use the exact clean Shell commit and the current candidate generator. The existing normal source resolver can select an unpinned checkout; explicitly verify its source before generation. Generate twice from identical inputs into separate isolated outputs and compare paths and bytes, including provenance. Do not edit generated outputs by hand.

The baseline generator's `--check` compares committed outputs against recorded provenance. It does not automatically verify a dirty candidate or prove target freshness; check the implementation on the candidate and verify candidate outputs directly when necessary. Reuse the previous failed attempt's fixes only after checking applicability; do not blindly copy its tree or cherry-pick unrelated fixes.

## 6. Verify and close

Run focused affected suites during adaptation, then one final integration check set appropriate to the change. Re-run affected checks after corrections; repeat the full suite only when correction scope requires it. Keep failed, timed-out, skipped and unavailable evidence. Do not claim a suite ran when only compilation or a filtered subset ran. Follow the harness risk-gated independent verification and user-owned native review policy; this skill supplies no verdict or consent.

Review update work units against the previous accepted AGY baseline, not the whole accumulated fork against the new upstream release. If review context is exhausted, surface the exact blocker and choose a smaller meaningful candidate or human-approved policy path; never rebuild history as an automatic escape.

Close compatibility only after required target inputs, preserved behavior, reproducible artifacts and applicable checks are proven, with no unexplained new failures. Report feature coverage separately, including accepted deferrals and unknowns. Native approval does not replace functional tests; structural plugin validation does not prove authenticated model execution. Update the accepted-baseline record only after closure and human acceptance, without implying merge/publication authority.

## 7. Keep official contribution separate

An update branch name does not shrink a PR diff. The initial upstream contribution still contains all integration changes not yet accepted upstream; later updates become smaller only after upstream incorporates them. Assess the actual diff against the agreed upstream target.

Report AGY-only changes, shared adaptations, generated artifacts, tests, docs, unrelated fork changes and dependencies. Use the repository contribution/PR rules if contribution preparation is requested. A one-time reviewed contribution chain is not a recurring migration requirement. Do not start that chain, publish an issue/PR or promise maintainer acceptance during an ordinary update.

## Local sources

- [Current Shell provenance](../../../internal/assets/antigravitycli/PROVENANCE.json)
- [Generator](../../../scripts/transpile-antigravitycli/main.go)
- [Generator tests](../../../scripts/transpile-antigravitycli/main_test.go)
- [Capability declarations, not runtime observations](../../../internal/agents/capabilitymanifest/manifest.go)
- [Verification documentation](../../../docs/antigravity-cli/09-verification.md)
