# Antigravity CLI Workspace Scope Support (`.agents/plugins/gentle-ai`)

## Overview
Enable Antigravity CLI (`agy`) workspace-level installation and synchronization under `--scope workspace`. Antigravity CLI natively discovers plugins located in `<workspace>/.agents/plugins/<name>/`. Gentle AI will deploy the plugin bundle (rules, subagents, skills, MCP configs) into `<workspace>/.agents/plugins/gentle-ai` without mutating `~/.gemini/config/import_manifest.json` or polluting the user's `HOME`.

## Tasks

- [x] **Task 1: Adapter Workspace Plugin Path & Discovery**
  - Add `WorkspacePluginDir` and `GlobalPluginDir` to `internal/agents/antigravitycli/adapter.go`.
  - Update `PluginDir` to resolve workspace paths when a workspace root is detected (`.agents/` or `.git`), preserving global path defaults.
  - Add unit tests in `internal/agents/antigravitycli/adapter_test.go`.

- [x] **Task 2: Workspace Plugin Deployment & Sync Lifecycle**
  - Update `antigravityCLIPluginImportStep` in `internal/cli/run.go` and `internal/cli/sync.go` to accept `workspaceDir` and `scope`.
  - In `ScopeWorkspace`, deploy to `<workspace>/.agents/plugins/gentle-ai` and finalize orchestrator rule without calling `EnsurePluginImported`.
  - In `ScopeGlobal`, keep existing deployment to `~/.gemini/config/plugins/gentle-ai` and import registration.
  - Add guard tests verifying step planning for both `ScopeGlobal` and `ScopeWorkspace`.

- [x] **Task 3: Scoped Backup Targets, Verification, and Component Paths**
  - Update `componentPathsWithWorkspaceScoped` and `syncBackupTargetsScoped` to properly declare and verify workspace paths under `<workspace>/.agents/plugins/gentle-ai/`.
  - Add end-to-end scope tests in `internal/cli/sync_scope_test.go` and `internal/cli/run_component_paths_test.go`.

- [x] **Task 4: Documentation Update**
  - Update `docs/antigravity-cli/01-foundation.md` and `docs/antigravity-cli/README.md` to document that `--scope workspace` is fully supported via `<workspace>/.agents/plugins/gentle-ai`.
  - Clarify the difference between global discovery (`~/.gemini/config/plugins/`) and project discovery (`.agents/plugins/`).
