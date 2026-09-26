# Feature: antigravity-odd-rebase-and-native-integration

## Objective

Port and natively integrate Gentle AI with Google Antigravity CLI (`agy`) onto official `upstream/main` (commit `6c7f162f`), completely adopting Organic Driven Development (ODD), eliminating legacy SDD residue, ensuring fail-open hook resilience to prevent tool lockouts, and verifying all Go test suites and live Antigravity CLI execution.

## Context and Upstream Changes

- `upstream/main` purged `.engram/chunks/` from history, causing a forced update.
- PR #4967 retired SDD and OpenSpec in favor of ODD, removing over 120,000 lines.
- Antigravity CLI plugin adopts native ODD primitives: `gentle-ai-explore`, `gentle-ai-worker`, `gentle-ai-verify`, 4R review subagents (`review-risk`, `review-readability`, `review-reliability`, `review-resilience`), and Judgment Day agents.
- Rules must remain strictly within the 24,000-byte per-file loading cap.
- `gentle-ai hook` must be compiled into the binary with fail-open safety (`{}`) to prevent blocking tool calls.

## Scope

- Target base: `upstream/main` (`6c7f162f4e596dc70ce6b1a1e9680277587bd7b9`)
- Feature branch: `feat/antigravity-cli-odd-integration`
- Backup reference: `backup/antigravity-cli-integration-pre-odd-rebase-20260925` (`26fa26a7`)
- Core packages: `internal/model/`, `internal/catalog/`, `internal/agents/antigravitycli/`, `internal/reviewerprovider/`, `internal/cli/`, `internal/tui/screens/`, `scripts/transpile-antigravitycli/`, `bench/`, `docs/antigravity-cli/`

## Tasks

- [x] Task 1: Foundation, Catalog & Agent Adapter
  - Target: `internal/model/types.go`, `internal/catalog/catalog.go`, `internal/agents/antigravitycli/adapter.go`, `internal/agents/antigravitycli/adapter_test.go`, component wiring (`internal/components/permissions/`, `internal/components/engram/`).
  - Evidence: `go test -v ./internal/model/... ./internal/catalog/... ./internal/agents/... ./internal/agents/antigravitycli/... ./internal/components/permissions/... ./internal/components/engram/...` PASS.
  - Commit: `eaccaf69` (`feat(antigravitycli): add agent adapter and catalog registration`)

- [x] Task 2: Reviewer Provider (RDD In-Process) & TUI
  - Target: `internal/reviewerprovider/antigravitycli_adapter.go`, `antigravitycli_routing.go`, unit tests, and TUI picker in `internal/tui/screens/antigravity_review_model_picker.go`.
  - Evidence: `go test -v ./internal/reviewerprovider/... ./internal/tui/screens/... ./internal/tui/...` PASS.
  - Commit: `7abb02d2` (`feat(reviewerprovider): add antigravity-cli in-process reviewer and model picker`)

- [x] Task 3: Resilient Hook Implementation
  - Target: `internal/cli/antigravity_hook.go` with strict fail-open safety (`{}` return on missing/unexpected events, exit 0) and unit tests in `internal/cli/antigravity_hook_test.go`.
  - Evidence: `go test -v ./internal/cli -run TestAntigravityHook` PASS (13 suites/subtests) and `go test -v ./internal/app -run TestRenderedSurfaceNamesOnlyItsOwnRuntime` PASS.
  - Commit: `e6079a99` (`feat(cli): add fail-open antigravity-cli hook command`)

- [x] Task 4: Asset Transpiler & ODD Assets
  - Target: `scripts/transpile-antigravitycli/` updated to drop SDD and generate ODD-only assets (agents, 4R chains, rules under 24KB limit, MCP config).
  - Evidence: `go test -v ./scripts/transpile-antigravitycli/...` PASS (8/8), `go test -v ./internal/assets -run TestAntigravityCLI_` PASS (11/11), and rule size verified (orchestrator.md 7,307 bytes < 24KB).
  - Commit: `bfd609b1` (`feat(assets): transpile odd-only assets and rules for antigravity-cli`)

- [x] Task 5: Benchmarks, Docs, Full Test Suite & Installation
  - Target: `bench/journeys_antigravity_cli.go`, `docs/antigravity-cli/`, run full test suite across touched packages, compile binary `go install ./cmd/gentle-ai`, and run `gentle-ai install --agent antigravity-cli`.
  - Evidence: `go test ./...` PASS on bench, binary compiled (`3.0.0-...-f8c777f8`), live `gentle-ai install --agent antigravity-cli` passed 15/15 checks, and `gentle-ai hook` verified.
  - Commit: `f8c777f8` (`feat(bench): add antigravity-cli journey benchmarks and docs`)

- [x] Task 6: Rule Truncation Fix & ODD Plugin Tree Deployment
  - Target: `internal/agents/antigravitycli/adapter.go`, `internal/cli/run.go`, `internal/cli/sync.go`, `internal/components/agentguidance/inject.go`. Split routing into `rules/gentle-ai-routing.md` (< 24 KB), deployed ODD subagents (`DeployPluginTree`) and pruned legacy `sdd-*` residues.
  - Evidence: `go test` PASS across modified packages. Installed files: `AGENTS.md` (17,378 bytes) and `gentle-ai-routing.md` (18,238 bytes) strictly under 24KB limit. `gentle-ai doctor` and live hook execution verified.
  - Commit: `2697dbcc` (`fix(antigravitycli): deploy odd plugin assets and separate routing rule file`)

## Rollback Boundary

- Backup branch: `backup/antigravity-cli-integration-pre-odd-rebase-20260925` (points to `26fa26a7`).

