# AGY Gentle statusline

Objective: ship a Gentle Shell-style custom status line for agy (Antigravity CLI), deployed by the gentle-ai plugin and registered in `settings.json` under user-ownership rules.
Branch: `chore/agy-integration-ai-v4.1.1-shell-v4.2.0` (work-unit commits on top; base `5825e03aa`). Delivery: ask-on-risk. Test runner: `go test ./internal/...` focused packages; script behavior via Go exec tests.

## Specs

S1. **Script asset** `internal/assets/antigravitycli/statusline.sh` (bash, `#!/bin/bash`… portable `#!/usr/bin/env bash`, `set -euo pipefail`), embedded via the existing `all:antigravitycli` embed, deployed to `<pluginDir>/statusline.sh` with mode `0755` by `DeployPluginTreeToDir` (same pattern as `hooks/hook.sh`: `WriteFileAtomic` + `os.Chmod`). It reads agy's status-line JSON payload from stdin and prints one line (plus optional `--title` mode, S7).

S2. **Segments** (comfortable density, in order, separated by `⟡`): brand `✿ gentle ai`; state badge from `agent_state` (`● ready`, `◆ thinking`, `⚙ working`, `🔧 tool`, `⏳ initializing`); location = `cwd` with `$HOME` shortened to `~` + `vcs.branch` + `*` when `vcs.dirty`; model = `model.display_name` + `· <execution_mode>` when `execution_mode` is present and non-empty; context = `ctx ▰▰▱▱ 42%` (8 cells, filled `▰` empty `▱`; tone thresholds: accent `<80`, warning `>=80`, error `>=95`; plus `!` marker when `used_percentage >= 90` as the context warning); cost = `$X.YZ` from `cost` (2 decimals when `>=1`, 3 when `<1`, Gentle Shell `formatCost`) + ` sub` suffix when `plan_tier` is non-empty; quota = one window per `quota` bucket: label derived from `reset_in_seconds` (week → `week`, whole days → `Nd`, whole hours → `Nh`, else `Nm`), used = `(1 - remaining_fraction) * 100` clamped 0..100, rendered `<label> ▰▱ NN%` with the same gauge thresholds; jobs = `⧗ N jobs` when `task_count > 0`; subagents = `✦ N subagents` when `subagents` is a non-empty array; sandbox = `⊘ sandbox` when `sandbox.enabled` (`⊘ sandbox·net` when `allow_network` too); vim = `vim.mode` when present. `conversation_title` right-aligned on the same line when it fits.

S3. **Degradation by `terminal_width`** mirroring `lib/shell-bar.ts`: (1) try the full line with right-aligned title; (2) on overflow, compact the location (cwd → last path segment, branch clipped to 15 visible chars with `…`); (3) still overflowing, drop trailing segments from the end (vim, sandbox, subagents, jobs, quota, cost, ctx, model, state — in that order; brand and location survive longest); (4) final `…` truncation to the width. `minimal` density hides the brand and the `ctx`/`subagents`/`sandbox`/`vim` labels' words stay (only brand hides).

S4. **User-owned config** `~/.gentle-ai/statusline.json` (never written by gentle-ai, never inside the plugin dir — sync must not prune or overwrite it). Schema `{"schema":"gentle-ai.statusline/v1","density":"comfortable"|"compact"|"minimal","visibility":{"changes":false,"rdd":true,"agents":true,"usageCost":true,"modelDetails":true,"contextWarning":true}}`. Defaults when absent/malformed: density `comfortable`, all visibility flags true except `changes:false`. `visibility.modelDetails:false` hides the model segment; `usageCost:false` hides ctx+cost+quota; `agents:false` hides jobs+subagents; `rdd:false` hides the RDD segment; `contextWarning:false` hides the `!` marker; `changes:true` enables the changes segment (S5). Unknown keys ignored, malformed file → defaults (script never fails on config).

S5. **RDD segment** (default on, `visibility.rdd:false` hides): `🌹 on` / `_roses: off` → `🌹 off`, unknown → `🌹 ?`, from `gentle-ai review mode status --json --cwd <cwd>` parsing `.status.effective` (exact enum strings pinned from `internal/reviewtransaction/rdd_mode.go` RDDMode constants; worker verifies `on`/`off` literals before writing the parse). Subprocess wrapped in `timeout 0.5` only when `timeout(1)` exists (without it: no subprocess, show cached or `🌹 ?`); result cached at `~/.gentle-ai/statusline-rdd-cache.json` (`{"ts":<epoch>,"mode":"on"}`) with 30s TTL, atomic tmp+mv write; any failure → `🌹 ?` and never blocks the statusline. **Changes segment** (`visibility.changes:true`, default off): `±N +A/−D` from `git status --porcelain` count and `git diff --numstat` sums with the same timeout guard; hidden on failure or non-git.

S6. **jq required**: when `jq` is missing or the payload is unparsable, print exactly `✿ gentle ai ⟡ install jq for the live status line` (single line, no ANSI) and exit 0. Single `jq` pass for the payload (official example's performance pattern), a second pass only for the config file.

S7. **`--title` mode**: same stdin payload, prints `✿ <branch> · <model display name>` (no ANSI; agy strips escapes there anyway), exit 0. Used by the settings `title` block.

S8. **Settings registration** (global scope only; workspace scope never touches `~/.gemini/antigravity-cli/settings.json`): after `DeployPluginTreeToDir`, ensure in settings.json: `"statusLine": {"type":"command","command":"<abs pluginDir>/statusline.sh"}` and `"title": {"type":"command","command":"<abs pluginDir>/statusline.sh --title"}`. Ownership rules per block: if the existing `command` is non-empty and does not point at our plugin script → leave the whole block untouched (user-owned); if `command` is empty/absent or points at our script → we own `type`+`command` only. Never add or modify `enabled`, `padding`, `stack_with_default` (preserve user values verbatim when present). Injection is a dedicated adapter-owned function with its own read-merge-write (the shared `mergeObjects` overlay would overwrite user scalars — do not use it for statusLine). Atomic write, preserve unknown top-level keys, byte-stable formatting for untouched files.

S9. **Uninstall** (`gentle-ai uninstall` antigravity-cli retained operations): remove the `statusLine` and `title` blocks from settings.json only when their `command` points at our plugin script; preserve user-owned blocks untouched. The script file disappears with the plugin dir removal.

S10. **Docs**: `docs/antigravity-cli/README.md` deploy tree gains `statusline.sh`; a statusline section (new `docs/antigravity-cli/` page or subsection of `01-foundation.md`) documents segments, the config file path + schema + defaults, ownership rules, jq requirement, and that `enabled`/`padding`/`stack_with_default` are user-owned knobs (`stack_with_default:true` renders below agy's built-in line). `docs/usage.md` updated if its install-effects table names settings.json effects for antigravity-cli.

S11. **Checks**: test-first (RED→GREEN) for Go behavior: settings ownership (fill empty stub like the real `{"type":"","command":"","enabled":true}`, preserve foreign user block, re-own ours, preserve `enabled:false`/`padding`/`stack_with_default`, workspace scope no-op, atomic/byte-stable), deploy (file written 0755, overwritten on re-deploy), uninstall (removes ours, preserves user). Script behavior via Go `exec` with fixture payloads and isolated `HOME`: full line snapshot, narrow-width degradation, `--title` output, jq-missing fallback (PATH without jq), config-driven hiding. Focused packages green at each task; full affected packages at closure.

## Tasks

- T2 | S1-S3,S4,S6,S7 | delegated worker | script asset + deploy step + script behavior tests | commit: `feat(antigravity-cli): add gentle statusline script and deploy step`
- T3 | S8 | delegated worker | settings statusLine/title injection + ownership rules + install/sync seam + tests | commit: `feat(antigravity-cli): register managed statusline in agy settings`
- T4 | S9 | delegated worker | uninstall cleanup + tests | commit: `feat(antigravity-cli): clean managed statusline on uninstall`
- T5 | S10 | delegated worker | docs (README tree + statusline page/section + usage) | commit: `docs(antigravity-cli): document the gentle statusline`
- T6 | S11 | inline | final verification: full affected `go test` packages + build + gofmt | closure evidence in Log

(T1 = this document + Engram mirror + todo projection, done before first write.)

## Log

L1 (verbatim): "Hay una feature interesante de agy: statusline. Es completamente personalizable por lo qe podemos hacer algo con eso para el tema de gentle ai. Revis que hacen en sus statusline gentle shell, opencode, calude y codex para ver que podemos nosotros integrar en agy"

L2 (verbatim): "revisa toda la personalizacion que tiene gentle shell (directo en el repo) y dime que podemos integhrar en el statusline de agy y que no"

L3 (verbatim): "arranca con eso"

L4: Research closed read-only. agy contract (settings.json `statusLine {type,command,padding,enabled,stack_with_default}`, JSON payload on stdin at every state change, full ANSI, multiline, `/statusline`, separate `title` block) verified against agy 1.3.6 binary: `cost`, `subagents`, `quota`, `conversation_title`, `agent_state`, `terminal_width`, `plan_tier`, `artifact_count`, `exceeds_200k_tokens`, `used_percentage` all present. Payload fields not fully documented upstream: `subagents` shape (count at minimum; labels unverified — script only counts, S2). GS surfaces that do NOT port (recorded, out of scope): clickable usage segment, sidebar card/header rail/below-input float, extension setStatus channel, multi-provider usage table, profiles, job labels, runtime theme painting. Defaults (density comfortable; rdd on; changes off; title injected) are assumptions accepted with "arranca con eso" — flagged in S4/S5; reversible via config/uninstall without code changes.

L5: Feature doc created after exploration. Delivery: ask-on-risk; forecast ~1000-1200 authored lines > 400 → delivery question asked before the first work-unit commit. RRD is on (global): assess after each work-unit commit against the advancing boundary; first boundary = base `5825e03aa`.
