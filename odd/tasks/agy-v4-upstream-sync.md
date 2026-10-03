# Feature: agy-v4-upstream-sync — Sync antigravity-cli-v4 with upstream/main (v4.0.0) without losing integration

## Objective
Sync the local branch `antigravity-cli-v4` (13 fork commits, base `648806d8`) with `upstream/main`
(v4.0.0+, fetched tip `9dfe17d8`) without losing any Antigravity CLI (agy) integration surface,
following the recorded sync-loss protocol (Engram: `strategy/agy-sync-loss-mechanism`,
`strategy/agy-v4-pr-preparation`, `odd/agy-v4-reland`, `strategy/agy-freeze-and-delivery-rules`).

## Standing rules (binding, from memory)
- **Tripwire BEFORE sync**: the unguarded hook `antigravityCLIPluginImportStep`
  (`internal/cli/run.go:905`, `internal/cli/sync.go:592`) must be asserted by committed tests
  before any rebase attempt (obs 2189). "Sync defensively is not a strategy; sync with a tripwire is."
- **NO push** without explicit user OK. Standing rule 5 (no push of v4-prep work) still governs
  delivery; the v4.0.0 freeze condition is satisfied but push/release stays user-owned.
- Responsibility-divided commits; Pi canon as source; divergences asked first.
- IDE-vs-CLI surface rule: `antigravity` (IDE) is upstream's surface; `antigravity-cli` (agy) is ours.
- **go test env hazard** (obs 2193): this session runs inside gentle-shell, which exports
  `PI_CODING_AGENT_DIR` / `GENTLE_PI_AGENT_HOME` to every child. Running `go test` with those
  vars can make tests escape `t.TempDir()` and destroy real agent-home files. Every `go test`
  invocation in this feature MUST run under `env -u PI_CODING_AGENT_DIR -u GENTLE_PI_AGENT_HOME`.
- Digest commit `2a1bfa02` (17 non-Pi agent digests) must be **RE-DERIVED** under v4's evidence
  budget semantics, not replayed (obs 2144).

## Expected conflict map (obs 2144, merge-tree verified)
- **Textual conflicts (2)**: `README.md`, `internal/agents/capabilitymanifest/manifest_test.go`
- **Semantic minefields (auto-merge danger, verify against both sides)**:
  - `internal/tui/model.go` (upstream SDK consent screen vs fork agy model picker, same flow)
  - `internal/cli/run.go` (upstream OpenCodeSDKConsent variadic vs fork plugin-import step)
  - `internal/cli/sync.go` (upstream PartialSyncError vs fork plugin-import twin)
  - `internal/components/engram/inject.go` (fork args=["mcp"] vs upstream transfer lock)

## Known-good failure baseline (obs 2150, to be re-confirmed in T2)
Full suite leaves exactly 3 failures: 1 pre-existing environmental opencode uninstall-mode test
+ 2 deliberate cap-guard tests (open 24KB design question). Post-rebase suite must match this
list. Any NEW red after the rebase = integration loss caught by the tripwire (success, not failure).

## Tasks
- [x] T1: Tripwire tests — assert install & sync `stagePlan` include the agy plugin-import step
      (`install:antigravity-cli:plugin-import` / `sync:antigravity-cli:plugin-import`); prove the
      guard fires (temporarily remove hook → RED, restore → GREEN); work-unit commit.
- [x] T2: Pre-rebase baseline — env-scrubbed full `go test ./...` (verifier), record the exact
      known-failure list to compare after the rebase.
- [x] T3: Backup branch `backup/antigravity-cli-v4-pre-v4-sync-<date>` + fresh `git fetch upstream`.
- [x] T4: Rebase `antigravity-cli-v4` onto `upstream/main`; resolve the 2 textual conflicts;
      verify the 4 semantic minefields against both sides (no silent drop, no semantic smush).
- [x] T5: Re-derive capabilitymanifest digests under evidence-budget semantics; fix digest tests.
- [x] T6: Post-rebase verification — gofmt/vet/build + env-scrubbed full suite vs T2 baseline
      + 3 ratchets (refusal-resolution, deadcode with GOTOOLCHAIN=go.mod, guard-population).
- [x] T7: Re-measure the 24KB cap question (rendered rule file sizes; recorded 38168 B did not
      reproduce; recomputation gave 22137→22550 B). Report options; the design decision stays open
      for the user.
- [x] T8: Close — report verified outcome, failed/skipped checks, next step. Push/release
      explicitly deferred to the user.

## Evidence
- 2026-10-02: start. Branch state: 13 ahead / 30 behind upstream/main, working tree clean.
  Hooks verified alive at run.go:905 and sync.go:592; zero tests reference `plugin-import` (gap confirmed).
- T1 DONE: `internal/cli/antigravity_cli_plugin_import_guard_test.go` — table-driven, 4 cases
  (install/sync × agy selected/not). GREEN 4/4 with hooks; loss simulation (both hook blocks
  removed, clean syntax) → RED on both positive cases, negative discrimination cases still
  green; hooks restored via `git checkout`, suite back to 4/4 green. gofmt clean, `go vet ./internal/cli/` ok.
  All go test runs executed under `env -u PI_CODING_AGENT_DIR -u GENTLE_PI_AGENT_HOME`.
  Commit: 141b9485 `test(cli): guard the antigravity-cli plugin-import plan hooks with a tripwire`
- T3 DONE: backup branch `backup/antigravity-cli-v4-pre-v4-sync-20261002` @ 141b9485. Fresh fetch:
  upstream/main MOVED past the recorded risk-map tip: `9dfe17d8` → `5140c5f5` (+2: fbfe89ec
  retire external community plugins #5200, 5140c5f5 fix(sync) preserved native agent recovery #5202).
  Divergence now 14 ahead / 32 behind. T4 must recompute the merge-tree against `5140c5f5`, not 9dfe17d8.
- T2 DONE: baseline recorded (verifier, env-scrubbed). `internal/cli` ALL GREEN (327s, default
  timeout; includes tripwire 4/4). The first attempt's `-timeout=180s` cap produced a false hang
  report on this 5.5-min package. Known-failure set is EXACTLY the 3 predicted, nothing else:
  1. `agentguidance` TestInjectRoutingAntigravityCLIUsesDedicatedRuleFile — routing file 65613 B > 24000 cap
  2. `agentguidance` TestInjectRoutingAntigravityCLIRuleFileCarriesAlwaysOnFrontmatter — same 65613 B
  3. `uninstall` TestUninstallOpenCodeFamilyManagedAgents/opencode — environmental (file mode)
  NOTE: measured routing size is now 65613 B, NOT the recorded 38168/22137/22550 figures — T7 must
  re-measure with this as the fresh anchor. ~50 packages ok, 0 build failures.
  Post-rebase comparison contract: same 3 failures and nothing else; anything new = loss caught.
- T4 DONE: rebase onto `5140c5f5` completed, 15/15 commits replayed (13 fork + tripwire + docs).
  3 conflict stops resolved:
  1. manifest_test.go @c209e123: upstream evidence-budget digests (17 agents incl. Pi) + agy row
     re-derived; non-Pi count check 16→17 (message made consistent).
  2. internal/tui/model.go @a93a2568: kept ScreenAntigravityReviewModelPicker (ours), dropped the 5
     opencode-plugin screens upstream retired in fbfe89ec (#5200) — upstream surface, upstream intent.
  3. README.md @df490d31: kept the fork's source-edition `### Installation` section; dropped upstream's
     official-release Windows/PowerShell install block (does not apply to this fork edition).
  4. manifest_test.go @2a1bfa02: trivial Fatalf-message leftover; map auto-resolved via rerere.
  Minefields verified post-rebase: hooks ALIVE at run.go:905 + sync.go:663 (tripwire 4/4 green);
  engram agy special case args=["mcp"] alive at inject.go:135 merged with upstream transfer lock;
  tui agy picker present (7 refs) / opencode plugin screens 0. Full `go build ./...` ok; gofmt clean.
  Rebase note: `git rebase --continue` needs GIT_EDITOR=true in this harness (interactive editor hangs).
- T5 DONE: agy digest re-derived under evidence-budget semantics:
  `AgentAntigravityCLI = sha256:7013f647fe77e95f2bfcfc9f77cdf47351b415702a01be58fdc17c15718d083a`
  (old fork value 92499795... invalid under v4). All 17 upstream agents pass with upstream's own v4
  digests; the fork registration shifts NO other agent digest; routing digest unchanged.
  capabilitymanifest package fully green at both the intermediate and final replayed states.
- T6 DONE (verifier, env-scrubbed): full `go test ./... -count=1` (~14m, ~73 packages) — SAME 3
  baseline failures, ZERO new: the 2 agy cap-guard tests (routing file now 66,834 B > 24,000 cap;
  pre-rebase 65,613 B — upstream content grew it ~1.2 KB) + 1 environmental uninstall opencode mode.
  internal/cli ALL GREEN (491s: tripwire 4/4, refusal-resolution ratchet, guard-population ratchet).
  Deadcode ratchet GREEN (GOTOOLCHAIN=go1.25.10 needed; 3 baselined entries now reachable —
  optional future `--update` tighten). gofmt clean, full build ok. Comparison contract HELD: no loss.
- T7 DONE (fresh measurement, decision deliberately left to the user):
  - Rendered file: `.gemini/config/plugins/gentle-ai/rules/gentle-ai-routing.md`, frontmatter
    `trigger: always_on`, cap check at inject_test.go:1017,1086 (`len(data) >= 24000`).
  - Routing file NOW: **66,834 B** (65,613 B pre-rebase; upstream's +2 commits grew it ~1.2 KB).
    Overflow: **2.78x the 24,000 B cap**. ALL recorded figures (38168 / 22137 / 22550 B) are stale
    — those measured the ORCHESTRATOR rule, and even that file only has ~1.4 KB headroom left.
  - Options (from the recorded design question, updated with fresh math):
    (a) split into multiple always-on rule files — at 66.8 KB this now needs 3 files, not 2;
    (b) slim agy-specific routing at the transpiler — canon divergence, needs explicit approval;
    (c) accept runtime truncation — agy silently drops the last ~42 KB of routing guidance.
    Aggregate risk also stands: routing 66.8 KB + orchestrator ~22.5 KB + AGENTS.md ~17 KB
    vs agy's 20K-token aggregate rules budget.
  - NO code changed for this; the 2 red cap-guard tests remain the standing signal of the open
    question.
- T8 DONE: native review entry checked (RDD on global). The facade-derived candidate = the latest
  work-unit commit, which is docs-only (`odd/tasks/agy-v4-upstream-sync.md` evidence) — trivial
  passive documentation, skipped per the entry rule. The substantive new unit (tripwire test,
  64f3351e) was verified mechanically (RED-proof + full-suite baseline contract) and remains
  available for an explicit native review at the user's discretion. Push: NOT done — user decision.
  Final state: `antigravity-cli-v4` 15 ahead / 0 behind `upstream/main` (5140c5f5); backup branch
  `backup/antigravity-cli-v4-pre-v4-sync-20261002` preserves the pre-sync state (141b9485-era).
