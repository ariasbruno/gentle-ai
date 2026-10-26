# Feature: Antigravity CLI Delegation Triggers and Review Rule Split

## Objective

Separate the Review Execution Contract from `rules/gentle-ai-orchestrator.md` into its own `rules/gentle-ai-review.md` rule file, freeing ~17 KB of headroom in the orchestrator rule to inline the 7 Delegation Triggers (specifically the Evidence-Budget Mapping Trigger) and fix subagent invocation instructions, ensuring Antigravity CLI proactively delegates complex research, implementation, and verification tasks to subagents.

## Specs

- S1: User reported: "Sabes que me llama la atencion, cada vez que hago alguna tarea en gentle shell, este delega a subagentes, tanto worker como verifier, o explorer, y demas. pero aca no. Yo se que me vas a decir que no es necesario, pero quiero que lo investigues porque los resultados que me das son muy vagos, poco proactivos e incompletos."
- S2: Separate the 17.4 KB Review Execution Contract out of `rules/gentle-ai-orchestrator.md` into `rules/gentle-ai-review.md` with `trigger: model_decision` frontmatter, keeping both files well under the 24,000-byte cap.
- S3: In `rules/gentle-ai-orchestrator.md`, inline the 7 Delegation Triggers (specifically Mapping trigger: >3 file reads or >5 lookups forces `gentle-ai-explore`), point `#### Review Execution Contract` to `rules/gentle-ai-review.md`, add `gentle-ai-review.md` to `## Required Phase Loading`, and update subagent invocation instructions to use registered `TypeName` (`gentle-ai-explore`, `gentle-ai-worker`, `gentle-ai-verify`) instead of `agents/<name>/agent.md` or `define_subagent`.
- S4: Update `ensureAntigravityOrchestratorRuleForScope` and `ensureAntigravityOrchestratorRule` in `internal/cli/antigravity_orchestrator_rule.go` to deploy and finalize both `gentle-ai-orchestrator.md` and `gentle-ai-review.md`, guaranteeing idempotency and determinism.
- S5: Update uninstall operations in `internal/components/uninstall/service.go` (`retainedAntigravityCLIPluginOperations`) to remove `rules/gentle-ai-review.md`.
- S6: Update tests across `internal/cli/`, `internal/components/uninstall/`, and `bench/` to verify both rule files deploy, remain under 24,000 bytes, and uninstall cleanly.

## Tasks

- [x] T1: Update `scripts/transpile-antigravitycli/orchestrator-core.md` and `internal/assets/antigravitycli/orchestrator.md` with inlined Delegation Triggers, corrected `TypeName` invocation, reference to `gentle-ai-review.md`, and RDD phase loading entry (S2, S3) [commit 50eaf8409].
- [x] T2: Update `internal/cli/antigravity_orchestrator_rule.go` to generate `rules/gentle-ai-review.md` with `trigger: model_decision` and finalize `rules/gentle-ai-orchestrator.md` with review pointer and `trigger: always_on` (S2, S4) [commit 50eaf8409].
- [x] T3: Update uninstall cleanup in `internal/components/uninstall/service.go` and tests in `internal/components/uninstall/` (S5) [commit 50eaf8409].
- [x] T4: Update tests in `internal/cli/antigravity_orchestrator_rule_test.go` and `bench/journeys_antigravity_cli.go` to assert both rules exist, are under 24,000 bytes, and verify idempotent sync (S6) [commit 50eaf8409].
- [x] T5: Sync and verify deployed plugin rules and run test suite (S1-S6) [commit 50eaf8409].

## Log

- L1: User requested investigation into why Antigravity CLI does not delegate to subagents like Gentle Shell does. Analysis identified the 24 KB rule cap catch-22, inlined brevity contract, and subagent invocation syntax. User chose Option A (split review contract to `gentle-ai-review.md`, inline delegation triggers in orchestrator).
- L2: Completed T1-T5 in commit 50eaf8409. Split Review Execution Contract to `rules/gentle-ai-review.md` (17,502 bytes) with `trigger: model_decision`. Inlined 7 Delegation Triggers and updated subagent invocation in `rules/gentle-ai-orchestrator.md` (10,141 bytes) with `trigger: always_on`. Verified idempotency and clean uninstall. Full test suite passing.
