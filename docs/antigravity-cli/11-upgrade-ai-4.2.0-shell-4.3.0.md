# Incremental update: AI 4.2.0 / Shell 4.3.0 / AGY 1.4.3

Status: implemented, verified and recorded locally within the scope below. The global suite is not green — five packages fail, and they fail identically on the accepted baseline — and authenticated AGY model behavior is not certified. This update preserves the accepted AGY integration; no historical fork reconstruction and no upstream PR.

## Fixed inputs

| Input | Value |
| --- | --- |
| Accepted AGY baseline | `agy/ai-v4.1.1_shell-v4.2.0`, `d4c31ec8d` |
| Prior incorporated AI / merge base | `898d3976e8049a7238eaa7de60bff325c30d2b3d` (AI v4.1.1) |
| AI v4.2.0 | `4acaf4241beae1e0c977c92dfe406a270f8d40de` (peeled release tag; `gh release list` agrees) |
| Prior Shell provenance | `c41ebd591358d393cf19c5ead981f67640a600ba` |
| Shell v4.3.0 | `d5d692a6888da9e9844e7e443ee30991fa4b9d14` |
| Validation runtime | `agy --version`: exit 0, stdout `1.4.3\n`. The target at update time was `1.4.2`; the installed runtime moved to `1.4.3` on 2026-10-20 and every check below was re-run against it |
| Update branch | `agy/ai-v4.2.0_shell-v4.3.0` |

The AGY release source had never been named in this repository: it is `google-antigravity/antigravity-cli`, read with `gh release list --repo google-antigravity/antigravity-cli`, with the changelog in `CHANGELOG.md` on `main` and `agy changelog` in the CLI. Previous validation runtime version was `1.3.4`, three minor releases behind the target. The installed Gentle Shell ships its own pinned `gentle-ai` (`INSTALLER_VERSION = "4.2.0"`, `scripts/gentle-ai-installer.mjs:40`) and runs it unless `GENTLE_SHELL_GENTLE_AI_BIN` overrides it; that override is explicitly deferred for this update.

The `1.4.3` delta is the one the validator needed: `agy plugin validate` now also checks a plugin's hooks, sidecars and component lists, so strict validation covers `hooks.json` and the component counts instead of stopping at skills, subagents and rules. Two further entries touch this integration: subagents and custom agents in Auto mode no longer ask for approval on every command that would leave the terminal sandbox, and the agent keeps track of its most recently started subagents after a conversation is compacted. Under `1.4.3` the deployed plugin validates with zero findings.

## Baseline evidence

Code baseline `d4c31ec8d`. The fork separates from upstream exactly at the incorporated AI release `898d3976e`, so the whole increment is everything after that commit. The upstream range `898d3976e..4acaf4241` is 34 commits, 85 files, +2933/-252. The Shell range `c41ebd59..d5d692a6` is 138 commits, 219 files, +13510/-1849, of which **11 files** reach the AGY bundle, because the generator reads only `assets/` and `skills/`. AGY moved five releases between the previous validation and the target: `1.3.5`, `1.3.6`, `1.4.0`, `1.4.1`, `1.4.2`.

## Impact inventory

Nineteen paths were changed by both sides since the separation — the collision surface, computed against the fork's own authorship rather than a raw tree diff, which would have hidden the distinction. Three of them conflicted textually and were resolved with the human; the other sixteen admit no choice (ten additive, five semantic without markers, one mechanical) and the both-parents test shows that for 18 of the 19 the result differs from **both** parents, so no file was taken whole from one side. The single exception, `.deadcode-baseline.txt`, equals the AI target because the fork's net change on that path is zero.

The merge is a chain of **eight merge commits** — `80d96d7e5`, `81a9ecad3`, `1e3b9c24c`, `96d50af41`, `7590b3f70`, `c54766c03`, `eb4bde9d3`, `3e8aba96c` — one per upstream first-parent slice, each with two parents: the previous fork commit and the upstream commit that closes the slice. Upstream's commits are adopted verbatim and in upstream's order; the chain's final tree is byte-identical to the original single merge's, and the branch's tree is byte-identical to the pre-recut tip apart from the records. The chain exists because the native reviewer refuses a candidate whose complete evidence exceeds its context budget, and it prescribed exactly this split.

Four pinned baselines moved: the deadcode ratchet (248 to 249 lines, the added line being upstream's), the orchestrator module baseline (22 lines, 11 entries, regenerated through the sanctioned `-update` flag), the routing sha256 pin (20 lines, one new runtime entry) and two golden files (one line each).

## Feature coverage and explicit limits

The AGY 1.4.0/1.4.1 plugin-validator rules are now covered against a real deployment: `agy plugin validate --strict` reports no findings on the plugin installed from this branch (skills 12 · agents 10 · rules 13 · hooks 1 · mcpServers 2 · sidecars 0), where the previous bundle reported 11 name-mismatch warnings and exited 1. The 11 mismatches were real and were fixed at their source — the generator now makes the folder the emitted skill's identity and carries the rename through every reference in the generated tree.

One capability is excluded by decision, not by omission: **agy does not take the ODD protocol from the canon.** The canon does not carry it (0 occurrences across Gentle Shell's `assets/`); it is a runtime contract, and the Pi-specific wording upstream 4.2.0 added names harness capabilities agy does not have — the visible `todo` the harness projects, the phase label it infers from session activity, the orchestrator's Mechanisms, the step 7 close, the asset pointers. agy therefore receives the generic protocol in one slim rule alongside the thirteen canon rules, for an always-on budget of about 44 KB. The decision, its rejected alternatives and its citations are in [08-decisions.md §10](08-decisions.md), with the claim row in [10-source-index.md](10-source-index.md).

Explicit limits, all unclaimed: the global suite is not green; authenticated AGY model behavior is not certified; a real Pi install runs `pi install npm:gentle-pi` against the npm registry, which this update does not authorize, so that path is exercised in dry-run only; whether agy claims the AGY 1.3.5 Auto mode is unknown; the Shell-side override of the installed Shell's pinned `gentle-ai` is deferred; and the accepted-baseline record still points at the previous baseline until the user accepts this one.

## Generation and verification

The bundle was regenerated from a **clean clone** at the Shell pin `d5d692a68` — the workspace checkout carries ignored files and the generator rejects a source that is not clean including ignored ones. Two independent generations are identical across path, bytes and provenance: 39 files, normalized tree hash `b2023e2b39138405dffa25b3e1efbf0a76d97435`. The committed tree hashes the same, so **no generated file carries a manual edit**. The drift gate reads the committed tree, not the working copy: `Drift check passed: committed bundle matches provenance-pinned transpilation.` The new Shell 4.3.0 satellite `orchestrator-delegation-reference.md` is carried in the generator's satellite list, the adapter's bundle map and the native rule metadata — an unregistered satellite emits an incomplete bundle with no error.

The thirteen packages the update touches are green with `go test <pkg> -count=1`: `internal/agents/antigravitycli` 0,113s · `internal/agents/pi` 0,025s · `internal/app` 28,848s · `internal/assets` 7,519s · `internal/components` 0,041s · `internal/components/agentguidance` 7,550s · `internal/providercontractbundle` 0,989s · `internal/releasepolicy` 0,004s · `internal/reviewtransaction` 168,178s · `internal/update` 16,740s · `scripts/transpile-antigravitycli` 2,149s · `internal/cli` 508,780s · `bench` 4,796s (its own module: `go test ./bench` from the main module answers `setup failed`, it must be run from inside `bench/`). The build from the final tree is clean: `go install ./cmd/gentle-ai` exit 0, version `4.2.1-0.20261020021501-f97b02128512`.

The whole suite was run with the CI's own recipe — `go run ./internal/gofmtcheck`, `go test ./...`, and inside `bench/`: `go build`, `go vet ./...`, `go test ./...` — on the candidate and on the accepted baseline, with the same environment. The candidate exits **1** in 783s and the baseline exits **1** in 829s, with the **same five failing packages**: `internal/cli` (package timeout), `internal/components/legacyassets` (`TestReleasedSDDAgentDigestsMatchGenerator`, timeout), `internal/releaseprovenancecmd` (`TestPartialReleaseIdentityNeverDowngrades/without_GITHUB_ACTIONS`, `TestCIBuildStillProducesReleaseProvenance` — no `manifest.json`), `internal/reviewtransaction` (package timeout) and `internal/shellinstaller` (`TestUserFrozenLockAcquisition`, four e2e subtests). Every candidate failure is therefore present on the baseline: **none is a regression**. The three package timeouts are an artifact of running both suites at once on four cores — the test in flight when the alarm fires differs on each side — and the same packages pass when run alone.

Nine native reviews were recorded, all approved with the receipt burned: the work unit that normalized the skill names, and the eight slices of the chain. The change that names a high-risk item (the published plugin's skill identity) was additionally checked by an independent verifier, which found no blocker and two missing regression tests, both since added. The store reports zero active lineages.

## Closure and contribution boundary

Compatibility demonstrated locally: the pinned incremental merge as a reviewable chain of real merges, a deterministic generator with a committed drift gate, the AGY skill-identity fix validated by AGY 1.4.2 itself, the plugin tree deployed and compared against the bundle, the full suite compared against a baseline run made the same way, and nine approved native reviews. The accepted AGY integration is preserved; the accepted baseline record is not moved until the user accepts.

The global suite, authenticated AGY model behavior, a real Pi install, the AGY Auto mode, the Shell-side `gentle-ai` override and the deployment decision of the update itself remain unclaimed or explicitly deferred. The upstream contribution question is separate from compatibility and is **not** answered by this record: nothing here has been proposed upstream. No push, pull request, publication, authentication or model invocation occurred; delivery is the user's decision under ordinary repository policy.
