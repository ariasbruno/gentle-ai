# Feature: Antigravity CLI re-land on v4 base (local, no push)

## Objective

Re-land the Antigravity CLI integration on the v4-era upstream base as a clean,
responsibility-divided commit series, working entirely LOCAL on branch
`antigravity-cli-v4` (derived from upstream/main). `antigravity-cli-integration`
and origin stay untouched; when upstream publishes the official v4 and the user
approves, the commits move onto `antigravity-cli-integration` and only then push.

## Why

User directive: freeze the released integration (v3.8.1) until official v4;
v4 work = new clean commit strategy (no fix-on-fix history); no pushes while
upstream keeps moving. Upstream main is already the /v4 module (a9f1390d) —
the re-land absorbs the module migration plus the 137-commit gap.

## Canon-divergence approvals (granted 2026-09-29 with evidence)

- always_on rule-file frontmatter (agy rule loader mandates it; official agy
  doc: "Only always_on rules are loaded unconditionally")
- MCP-bridge tool mapping in the transpiler (Pi wraps the CLI via
  gentle-shell/extensions/codegraph-tools.ts; agy registers the MCP server
  directly, whose single tool is codegraph_explore)
- settings.json command allow-list (agy's permission surface has no Pi
  equivalent; removes prompt friction for the rule-driven commands)

## Method

1. Branch `antigravity-cli-v4` at upstream/main tip (fresh fetch).
2. Net fork contribution = `git diff 6c7f162f..antigravity-cli-integration`
   (merge-base = old upstream fork point).
3. Apply with `git apply -3`; resolve conflicts (known hot spots: model/types.go
   [Conductor agent + v4 imports], communitytool codegraph contract [Conductor
   entry], cli/sync.go [workspace-scope fixes], engram/inject.go [antigravity
   registration fixes — upstream's; we take theirs], transpiler imports, baselines).
4. Commit in 9 responsibility slices, tests with each.
5. Re-transpile assets from current gentle-shell (183 commits; 1-line asset drift).
6. Full verification battery. No push. No sync of the user's installed binary
   (j4500's sandbox is the verification surface).

## Commit slices (planned)

1. feat(model,catalog): register AgentAntigravityCLI + adapter base
2. feat(permissions,mcp): security overlay, allow entries, plugin mcp config
3. feat(scripts): deterministic asset transpiler + invariants
4. feat(assets,agents): transpiled bundle + DeployPluginTree + native import
5. feat(cli): fail-open lifecycle hooks + uninstall
6. feat(agentguidance,filemerge): always-on rule injection + contract fill
7. feat(communitytool): codegraph wiring (reconciled + reconciler)
8. feat(reviewerprovider,tui): in-process review transport + routing + picker
9. test(bench) + docs: j4500 + documentation set

## Constraints

- No push. No release. User's installed 3.8.1 binary untouched.
- Everything Pi-derived flows through the transpiler; the three approved
  divergences are the only hand-written agy-specific semantics.
- Upstream's newer shared code (engram antigravity fixes, sync workspace-scope,
  filemerge file-mode) is TAKEN AS IS where we have no local stake.

## Checks

- go build ./... ; focused suites (filemerge, agentguidance, communitytool,
  permissions, uninstall, antigravitycli, cli-antigravity tests)
- go run ./scripts/transpile-antigravitycli --strict (zero drift)
- bench: build + vet + test + j4500 driven
- Ratchets: refusal-resolution, deadcode (GOTOOLCHAIN pinned), guard population
- gofmtcheck

## Tasks

- [ ] T1: branch base + fork net-diff inventory
- [ ] T2: apply contribution over v4 base, resolve conflicts
- [ ] T3: commit in 9 responsibility slices
- [ ] T4: re-transpile assets from updated gentle-shell
- [ ] T5: full verification battery
- [ ] T6: final report (no push)

## Progress

(2026-09-29) Started after user's "procede con la actualizacion" following the
strategy rules (freeze until official v4; local commits only; new clean history).
