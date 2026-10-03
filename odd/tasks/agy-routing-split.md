# Feature: agy-routing-split — deliver the full v4 routing canon to Antigravity CLI within its 24,000-byte per-file rule cap

## Objective
Close the open 24KB design question (option a, evolved): the agentguidance routing render for
Antigravity CLI is 66,834 B on this branch; agy truncates any rule file over 24,000 bytes on line
boundaries (verified: agy 1.2.16 builtin docs `rules.md:29-37`). Deliver the canon 1:1 by splitting
the render into N section-coherent always_on rule files, with a byte-exact union-integrity contract.

## Verified semantics (agy 1.2.16, builtin docs + changelog)
- Per-file: 24,000 bytes (after include expansion); over-cap files are TRUNCATED on line
  boundaries (not dropped). Files must be < 24,000 bytes total (frontmatter included).
- Aggregate: always-on + global rules share a DEDICATED 20,000-token budget; over-budget rules are
  demoted to path+description pointers (model can read files on demand). Demotion of part of the
  canon is ACCEPTED (routing 66.8K + orchestrator 22.1K + AGENTS.md 17.4K exceed 20K tokens);
  the split removes silent truncation, not pointer demotion.
- agy requires `trigger: always_on` frontmatter or it drops rule files.

## Design decisions (binding)
1. **Naming**: part 1 keeps the canonical `rules/gentle-ai-routing.md` (stabilizes bench/uninstall/
   test references); parts 2..N are `gentle-ai-routing-2.md`, `gentle-ai-routing-3.md`, ...
2. **Split unit**: blocks at `\n### ` section boundaries of RenderRouting (H3 sections; the H2
   preamble forms the first block). Greedy in-order packing; a part closes when the next block
   would exceed the body budget (cap − frontmatter − marker overhead). If a single block exceeds
   the body budget, split it at paragraph (`\n\n`) boundaries, then list-item boundaries; if a
   single line is still over budget, return a loud error — never truncate silently.
3. **Determinism**: same render → same parts, byte-exact (the bench idempotency pin depends on it).
4. **Frontmatter + marker**: every part carries the SAME `antigravityCLIRuleFrontmatter` and the
   `<!-- gentle-ai:agent-routing -->` ownership marker (marker must be present in every part so
   ownership-based cleanup detects them all).
5. **Integrity contract**: concatenating part bodies in order (frontmatter+marker stripped) must
   equal the canonical single-render bytes. Enforced by a test.
6. **Stale-part cleanup**: after writing the current part set, the inject step deletes files in
   the rules dir matching `gentle-ai-routing*.md` that contain our ownership marker but are not
   in the written set (handles N shrinking between syncs).
7. **Uninstall**: the retained-ops cleanup removes the canonical file AND all marker-owned parts.
8. **Only agy**: other agents keep single-section delivery in CLAUDE.md/AGENTS.md (no cap).
9. **Cap constant**: lives next to the agy frontmatter in agentguidance with a comment citing the
   agy docs (24,000 per-file; keep files < 24,000).

## Tasks
- [x] T1: RED — restructure the 2 cap-guard tests (inject_test.go) into the split contract:
      part 1 canonical + all parts < 24000 + all parts carry frontmatter+marker; ADD union-integrity
      test (bodies concatenated == canonical render) and stale-part cleanup test. Observe RED.
- [x] T2: GREEN — split implementation in agentguidance inject.go (pack algorithm, part writer,
      stale cleanup). Focused package green.
- [x] T3: uninstall parts cleanup (service.go) + uninstall test update. Package green.
- [x] T4: bench j4500 journey update (first-sync parts existence+frontmatter+marker+per-part cap;
      second-sync byte-idempotency for ALL parts; uninstall removes ALL parts) + `go test ./bench`
      declarations green.
- [x] T5: driven j4500 re-verification with the exact CI command (gentle-ai-bench skill rules).
- [x] T6: full env-scrubbed suite vs the NEW contract: the 2 cap-guard failures must turn green;
      expected known failures reduce to exactly 1 (environmental opencode uninstall mode). Tripwire green.
- [x] T7: work-unit commits + close (feature doc evidence; push remains the user's decision).

## Evidence
- 2026-10-02/03: start. Standing context: rebase complete (antigravity-cli-v4 16 ahead / 0 behind);
  baseline had exactly 3 known failures incl. the 2 cap-guards at 66,834 B.
- T1-T4 DONE (delegated writer, parent-verified). **ROOT-CAUSE DISCOVERY**: the 66,834 B was NOT
  routing alone — the canonical orchestrator render (~47KB) was merged into the routing file for
  every non-Pi runtime (old inject.go: `if agent != model.AgentPi`), while DeployPluginTree ALSO
  deploys `gentle-ai-orchestrator.md` as its own always_on rule (verified on disk). agy received
  a TRUNCATED duplicate orchestrator on top of the intact rule — double delivery + truncation.
- Resolution: new `deliveryAntigravityCLISplit` for agy — routing-only render (17,456 B body)
  → **1 part today** (`gentle-ai-routing.md`, 19,006 B total incl. frontmatter+marker), split
  machinery ready for growth (H3-section packing, paragraph/list-item fallback, loud error instead
  of silent truncation, stale-part cleanup by ownership marker). Orchestrator arrives exactly once
  via the plugin bundle rule (the 1:1 gentle-shell canon distribution, review contract filled by
  ensureAntigravityOrchestratorRule).
- Aggregate budget: routing 19.0K + orchestrator ~22.1K + AGENTS.md ~17.4K ≈ 58.5KB — expected
  WITHIN agy's 20K-token dedicated rules budget (no pointer demotion expected).
- Tests: 4 contract tests in agentguidance (split contract, union integrity vs
  `InjectRemoteAuthorization(RenderRouting())`, stale cleanup, idempotency) + agy excluded from the
  orchestrator-merge set + uninstall `removeAntigravityCLIRoutingParts()` test + j4500 journey pins.
- RED note (honest): the standing red cap-guards (66,834 B) were the failing design signal; the
  restructured contract tests were validated by construction + the driven j4500 run (T5) is the
  end-to-end proof.
- Parent verification: agentguidance green; uninstall green except the KNOWN environmental
  opencode mode test; bench declarations green; tripwire 4/4 green; gofmt clean on all touched files.
- T5 DONE: driven j4500-antigravity-cli-lifecycle-parity COMPLETED with the exact CI commands
  (bench harness build + `go build -trimpath` binary + `--only j4500... --out json`): real
  install → sync idempotency → uninstall all pass with the split parts.
- T6 DONE: full-suite contract verified in two steps. (1) Verifier full run on 28f900cd: exactly 2
  failures — the known environmental opencode mode test PLUS one STALE PIN discovered:
  internal/cli TestInstallAndSyncDeliverOrchestratorOnceForEveryRuntime/antigravity-cli asserted
  the OLD merged delivery (canonical orchestrator render inside the routing file). (2) The pin was
  updated to the ratified contract in ab23402b (agy leaves the merged loop; the IDE keeps it; new
  dedicated routing-only test with ODD protocol + RDD kill-switch + install/sync idempotency),
  after which the FULL internal/cli package passed (398s, all green: tripwire + both ratchets).
  The previously-red cap-guard tests are GREEN — the standing 24KB design signal is CLOSED.
  New standing baseline: exactly 1 known failure (environmental uninstall opencode mode).
- T7 DONE: commits 28f900cd (split + uninstall + j4500) and ab23402b (routing-only contract pin).
  NOT pushed — user decision.
