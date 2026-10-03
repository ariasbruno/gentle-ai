# Feature: agy-bundle-provenance-skills — Gap 5 (anti-drift) + Gap 2 (bundle-owned skills) of the 1:1 plan

## Objective
Two ratified deliverables of the agy-vs-gentle-shell 1:1 adaptation:
- **Gap 5 — freeze with an alarm**: the transpiler records its exact gentle-shell source ref
  (provenance) and gains a `--check` drift gate (re-transpile from the pin, byte-diff vs the
  committed bundle) plus a CI job (blocking drift gate + advisory freshness signal vs gentle-shell HEAD).
- **Gap 2 — single-source skills**: the transpiler consumes `gentle-shell/skills/*` and emits them
  into the plugin bundle; the plugin bundle becomes the ONE owner of agy's skills (the install-time
  skills component stops delivering catalog skills for Antigravity CLI), matching what the gentle-pi
  package ships to Pi. The 14/15-skill internal catalog keeps feeding claude/codex/opencode untouched.

## Verified facts (recon)
- Transpiler: scripts/transpile-antigravitycli/main.go — flags --source-dir/--source-repo/--output-dir/--strict;
  defaultRemoteRepo = https://github.com/Gentleman-Programming/gentle-shell; defaultOutputDir = internal/assets/antigravitycli;
  consumes assets/{agents,orchestrator.md,satellites,support,chains}; NO provenance, NO --check, NO skills today.
- DeployPluginTree copies the embedded bundle subtree wholesale (os.CopyFS after a refuse-non-regular
  walk) — a new bundle skills/ dir deploys automatically; adapter has a legacy per-name skills prune.
- internal/assets/assets_test.go pins the CATALOG skills (skills/*), not the antigravitycli bundle.
- Local gentle-shell checkout: /home/bruno/dev/gentle-shell (main @ 7a27c1c0).

## Binding design
1. **Provenance is DETERMINISTIC**: PROVENANCE.json = source repo URL + resolved commit sha (+ source
   tree hash if cheap). NO timestamps or machine paths — the --check gate byte-compares regenerated
   output, so any nondeterminism breaks the gate.
2. **--check contract**: read the committed PROVENANCE.json; resolve the source AT the pinned ref
   (local --source-dir allowed only if its HEAD equals the pin, else clone the pinned ref); transpile
   to a temp dir; byte-diff every emitted file (and file SET) against the committed bundle; exit 0
   identical / exit 1 with a per-file drift report.
3. **CI**: blocking job/step running `go run ./scripts/transpile-antigravitycli --check`; plus an
   ADVISORY freshness step comparing the pin vs gentle-shell remote HEAD (warn, never fail).
4. **Skills pipeline (Gap 2)**: consume ALL of gentle-shell/skills (skill dirs + _shared fragments +
   the `gentle-ai` harness skill), run the existing transformation pipeline (translatePiTools,
   stripDynamicInjections, path mutations), emit to <output>/skills/<name>/SKILL.md (+ extra files).
5. **Single-source delivery**: after the bundle carries skills, the skills component (internal/
   components/skills) SKIPS Antigravity CLI (bundle owns plugin skills/); DeployPluginTree prunes
   plugin skills/ entries that are not in the current bundle (stale selected skills from the old
   flow); the capability manifest Skills claim for agy stays true (the agent has skills — delivered
   by the bundle).
6. Regen runs use --source-dir /home/bruno/dev/gentle-shell (HEAD 7a27c1c0) and the diff is reviewed
   by the parent before committing.

## Tasks
- [x] T0: local hygiene — 11 legacy sdd-* skills moved out of ~/.agents/skills to
      ~/.agents/skills/.trash-sdd-legacy-20261003 (rm blocked by safety policy; mv is reversible).
- [ ] T1 (Gap 5, worker A): provenance emission + --check gate + CI job + tests; regen to land
      PROVENANCE.json with a clean (expected no-op besides provenance) bundle diff.
- [ ] T2 (Gap 2, worker B): skills consumption in the transpiler + regen (new bundle skills/) +
      DeployPluginTree prune + skills component agy skip + tests + journey check.
- [ ] T3: verification — focused packages + driven j4500 (exact CI commands) + full suite vs the
      standing contract (exactly 1 known environmental failure).
- [ ] T4: work-unit commits (one per gap) + close. Push remains the user's decision.

## Evidence
- 2026-10-03: start. Recon verified the transpiler surface, CopyFS deployment, and catalog-vs-bundle
  test boundaries. Item 4 (user request) done: 11 sdd-* skills moved to trash.
