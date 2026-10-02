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
- [ ] T2: Pre-rebase baseline — env-scrubbed full `go test ./...` (verifier), record the exact
      known-failure list to compare after the rebase.
- [ ] T3: Backup branch `backup/antigravity-cli-v4-pre-v4-sync-<date>` + fresh `git fetch upstream`.
- [ ] T4: Rebase `antigravity-cli-v4` onto `upstream/main`; resolve the 2 textual conflicts;
      verify the 4 semantic minefields against both sides (no silent drop, no semantic smush).
- [ ] T5: Re-derive capabilitymanifest digests under evidence-budget semantics; fix digest tests.
- [ ] T6: Post-rebase verification — gofmt/vet/build + env-scrubbed full suite vs T2 baseline
      + 3 ratchets (refusal-resolution, deadcode with GOTOOLCHAIN=go.mod, guard-population).
- [ ] T7: Re-measure the 24KB cap question (rendered rule file sizes; recorded 38168 B did not
      reproduce; recomputation gave 22137→22550 B). Report options; the design decision stays open
      for the user.
- [ ] T8: Close — report verified outcome, failed/skipped checks, next step. Push/release
      explicitly deferred to the user.

## Evidence
- 2026-10-02: start. Branch state: 13 ahead / 30 behind upstream/main, working tree clean.
  Hooks verified alive at run.go:905 and sync.go:592; zero tests reference `plugin-import` (gap confirmed).
- T1 DONE: `internal/cli/antigravity_cli_plugin_import_guard_test.go` — table-driven, 4 cases
  (install/sync × agy selected/not). GREEN 4/4 with hooks; loss simulation (both hook blocks
  removed, clean syntax) → RED on both positive cases, negative discrimination cases still
  green; hooks restored via `git checkout`, suite back to 4/4 green. gofmt clean, `go vet ./internal/cli/` ok.
  All go test runs executed under `env -u PI_CODING_AGENT_DIR -u GENTLE_PI_AGENT_HOME`.
  Commit: <pending>
