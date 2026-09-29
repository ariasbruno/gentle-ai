# Antigravity CLI Integration — Phase 5: In-Process Review Transport & Capabilities

This document details the architecture, routing mechanics, interactive TUI model configuration, and verification evidence for **Phase 5: In-Process Review Transport & Capabilities** in the Google Antigravity CLI (`agy`) integration.

---

## 1. Architectural Overview: In-Process Review Transport

In Gentle AI's Receipt-Driven Development (RDD) engine, review lenses (`review-risk`, `review-resilience`, `review-readability`, `review-reliability`) and verification roles (`review-refuter`, `review-validator`) must execute against frozen candidate diffs with guaranteed isolation and deterministic output capture.

### 1.1 Transport Classification

Runtimes in Gentle AI fall into two primary review transport categories:
1. **Host-Mediated Relays**: Used by runtimes like OpenCode (via child process event relay) and Pi (via `gentle-pi` host launcher).
2. **In-Process Capture Subprocesses**: Used by Claude Code, Codex, and now **Google Antigravity CLI** (`reviewImmutableTransportAntigravityCLIInProcess`).

Antigravity CLI executes directly as an in-process subprocess managed by Go:
- Go owns prompt materialization, token budgeting, JSON schema validation, admission, and receipt signing.
- The `AntigravityCLIAdapter` runs a clean, headless `agy` process in an isolated scratch directory.
- Prompt text is passed securely via `stdin` (`--output-format text`).
- Execution is strictly bounded by context cancellation and timeouts.
- Raw final bytes are captured directly from `stdout` without intermediary plugins or proxies.

### 1.2 Minimality Guard & Architectural Isolation

To ensure adapters do not leak domain logic or absorb lifecycle responsibilities, `internal/reviewerprovider/adapter_minimality_guard_test.go` enforces the Go AST boundary:
- The adapter is registered in `reviewerAdapterImplementations` as `AntigravityCLIAdapter`.
- It cannot import provider-owned review semantics (e.g. `internal/reviewtransaction`).
- It cannot resolve lifecycle root material (`--cwd`, `os.Chdir`, `filepath.Abs`).
- It cannot perform looping, parsing, or metric analysis on prompt or result bytes.

---

## 2. Review Role Routing & Configuration

Different review roles require different model capacities and reasoning depths. For example, deep structural vulnerability analysis (`review-risk`) benefits from high-reasoning models (such as `claude-opus-4-6-thinking`), whereas idiomatic linting (`review-readability`) can run efficiently on faster models (such as `gemini-3.8-flash-medium`).

In Antigravity CLI, every model slug in the catalog (`agy models`) already bakes its reasoning tier into the model ID:
- Gemini models: `gemini-3.8-flash-high`, `gemini-3.8-flash-medium`, `gemini-3.8-flash-low`, `gemini-3.1-pro-high`, `gemini-3.1-pro-low`.
- Claude models: `claude-opus-4-6-thinking`, `claude-sonnet-4-6` (these models do not support `--effort`; passing it causes `agy` to error: `--effort is not supported for model "claude-opus-4-6-thinking"`).
- GPT-OSS models: `gpt-oss-120b-medium`.

Passing an explicit `--effort` flag is redundant and causes runtime conflicts (e.g. `--model gemini-3.8-flash-high conflicts with --effort=medium`). Gentle AI intentionally uses pure model slugs for all Antigravity CLI reviewer roles, eliminating `--effort` completely.

### 2.1 Storage & Schema (`review-models.json`)

Antigravity CLI review models are stored per-user at:
```
~/.gemini/antigravity-cli/gentle-ai/review-models.json
```

The configuration schema stores string assignments per role:
```json
{
  "review-risk": "claude-opus-4-6-thinking",
  "review-resilience": "gemini-3.8-flash-medium",
  "review-readability": "gemini-3.8-flash-medium",
  "review-reliability": "gemini-3.1-pro-high",
  "review-refuter": "claude-opus-4-6-thinking",
  "review-validator": "gemini-3.1-pro-high"
}
```

(An object format `{"model": "..."}` is also supported for backward compatibility, with legacy `effort` fields safely ignored).

### 2.2 Resolution Mechanics

In `internal/reviewerprovider/antigravitycli_routing.go`, `ResolveAntigravityCLIReviewRouting(key)` resolves the routing configuration:
1. If `review-models.json` contains the sentinel value `"inherit"` (or `{}`), it resolves to an empty model string `""`.
2. If `review-models.json` is missing or the role key is not assigned, it falls back to empty defaults and emits a diagnostic message on `stderr`.
3. If the routing specifies an invalid or malformed structure, it returns a typed `InvalidReviewRoutingError`.
4. When `adapter.Model` is empty (the `inherit` state), the adapter invokes `agy` without the `--model` flag:
   ```bash
   agy --sandbox --output-format text
   ```
   This ensures the reviewer dynamically inherits the user's active session or global CLI configuration.
5. When `adapter.Model` is non-empty, it formats CLI arguments explicitly:
   ```bash
   agy --sandbox --output-format text --model <model>
   ```

### 2.3 Curated Presets

Gentle AI provides four built-in routing presets for Antigravity review:
- **Default (Inherit)**: Omit `--model` flag across all roles, delegating model choice entirely to the user's active Antigravity CLI session defaults.
- **Recommended**: Optimized balance of precision and speed (`claude-opus-4-6-thinking` for Risk & Refuter, `gemini-3.1-pro-high` for Reliability & Validator, `gemini-3.8-flash-medium` for Resilience & Readability).
- **Performance**: Frontier reasoning models across all roles (`claude-opus-4-6-thinking`, `gemini-3.1-pro-high`, and `claude-sonnet-4-6`).
- **Economy**: Minimal-resource configuration targeting fast turnaround (`gemini-3.8-flash-high`, `gemini-3.8-flash-medium`, and `gemini-3.8-flash-low`).

---

## 3. Interactive TUI Review Model Picker

Phase 5 introduces a full-featured Bubbletea TUI picker in `internal/tui/screens/antigravity_review_model_picker.go`:

### 3.1 Features & User Experience
- **Preset Selection**: Instant one-click selection of Default (Inherit), Recommended, Performance, or Economy profiles.
- **Granular Customization**: Step-by-step role customization flow:
  1. *Role Selection*: Choose any of the 6 canonical review roles.
  2. *Model Selection*: Browse and search recognized Antigravity models (with `(default / inherit)` pinned as the top option) or enter custom models. Selecting a model assigns it immediately and returns to the role list.
- **Validation & Deprecation Badges**: Visually flags unknown or deprecated models with warning tags.
- **Integrated Navigation**: Fully wired into `gentle-ai model-config` (Option 4: "Configure Antigravity CLI review models"), with proper `Esc` return semantics and automatic atomic persistence to `review-models.json`.

---

## 4. Lifecycle & Contract Guarantees

Antigravity CLI is integrated across all RDD capability surfaces:
- **Capability Matrix**: `reviewImmutableRuntimeCapability(model.AgentAntigravityCLI)` advertises `reviewImmutableTransportAntigravityCLIInProcess`.
- **Registered Runtime Inventory**: Verified by `TestRegisteredRuntimeIdentitiesMatchCompiledTransportBoundary` ensuring exact alignment between advertised capabilities and compiled adapters.
- **SDD Contract Resolution**: `ReviewExecutionContractFor(model.AgentAntigravityCLI)` exports the canonical CLI STATUS command and explicit in-process capture instructions, preventing parent orchestrator prompt duplication.

---

## 5. Empirical Verification Evidence

All tests across reviewer provider, CLI commands, SDD contracts, persona injection, and TUI screens execute and pass cleanly:

```bash
# 1. Reviewer Provider Tests
$ go test -v ./internal/reviewerprovider/...
PASS
ok      github.com/gentleman-programming/gentle-ai/v3/internal/reviewerprovider 0.203s

# 2. CLI Review Transport Tests
$ go test -v ./internal/cli/... -run "Review"
PASS
ok      github.com/gentleman-programming/gentle-ai/v3/internal/cli 142.609s

# 3. TUI Model Picker and Screen Tests
$ go test -v ./internal/tui/screens/... -run "Antigravity|ModelConfig"
PASS
ok      github.com/gentleman-programming/gentle-ai/v3/internal/tui/screens 1.952s

$ go test -v ./internal/tui -run "Antigravity|ModelConfig"
PASS
ok      github.com/gentleman-programming/gentle-ai/v3/internal/tui 2.109s

# 4. SDD Contract Resolution Tests
$ go test -v ./internal/components/sdd -run "Antigravity|Flip|InstalledContract"
PASS
ok      github.com/gentleman-programming/gentle-ai/v3/internal/components/sdd 0.227s

# 5. Persona & Orchestrator Injection Tests
$ go test -v ./internal/components/persona -run "TestInjectAntigravityCLIPersonaAndOrchestrator"
PASS
ok      github.com/gentleman-programming/gentle-ai/v3/internal/components/persona 0.043s
```

---

## 6. Associated Work Units & Commits

This phase was implemented and verified across the following atomic conventional commits:
1. `feat(reviewerprovider,cli,sdd): add Antigravity CLI in-process review transport and routing with inherit support` — in-process `agy` review adapter, role routing in `review-models.json`, sentinel `"inherit"` support, and SDD contracts.
2. `feat(tui): add interactive Antigravity CLI review model picker with presets and inherit support` — Bubbletea interactive review model picker, `Default (Inherit)` preset, custom role picker with `(default / inherit)` support, and deprecation flags.
3. `docs(antigravity-cli): document in-process review transport architecture, routing, and TUI picker` — in-process transport classification, AST minimality guard, model resolution mechanics, and test evidence.

