# Feature: agy-canon-first-routing — slim agy's routing rule to the ODD protocol only

## Objective (user: "hazlo")
Ratified decision: agy becomes canon-first for routing doctrine. The transpiled canon
(orchestrator rule + delegation/memory/skills satellites) is the SINGLE source of ladder,
delegation triggers, TDD policy, RDD boundaries, task-size heuristics, assumption-challenge
discipline, and feature-doc discipline. The dedicated routing rule shrinks to the ONE piece
the canon files do not carry: the mandatory ODD workflow protocol (7 steps + recovery line).

## Verified content map (grep-verified 2026-10-04)
- ODD 7-step protocol: NOT in canon files (lives in gentle-shell extension code + its test
  fixture; Pi receives it as runtime injection). gentle-ai's RenderRouting is agy's only file
  source → the slim rule keeps it.
- Route topology/ladder, delegation triggers, evidence budgets: canon delegation satellite ✓
- TDD policy paragraph: canon orchestrator ✓ (verified in the transpiled rule)
- RDD ownership/boundaries: canon orchestrator section + review contract fill ✓
- ~400-line heuristic, odd/tasks+mirror, assumption challenge, accepted-changes: canon
  satellites ✓ (grep hits in delegation + memory satellites)
- Numbers: CanonicalImplementationRouting = one shared projection for ALL agents, identical to
  the canon's numbers, parity-test-enforced → no per-agent calibration is lost.

## Binding design
1. RenderRouting(agent) branches for AgentAntigravityCLI: emit the protocol-only block
   ("## Implementation Routing" + predefined-workflow paragraph + "### ODD protocol
   (MANDATORY...)" 7 steps + the mem-based resume line). Extract the shared ODD block
   emission so the full render and the slim render use the same source strings.
2. Delivery: single file rules/gentle-ai-routing.md (same canonical path → backup/verify
   contracts and uninstall stay untouched), always_on frontmatter + ownership marker,
   stale-part cleanup KEPT (marker-owned gentle-ai-routing*.md outside the written set —
   cleans any historical parts).
3. The split machinery (deliveryAntigravityCLISplit, splitRoutingForAntigravityCLI,
   splitLargeBlock, splitByListItems, union-integrity test) is REMOVED — no consumer left.
   antigravityCLIRuleCap stays (per-file insurance + test assertion).
4. Always-on budget post-change: AGENTS.md 17.4K + orchestrator 22.5K + slim ~4K ≈ 44KB.
5. Expectation side: no surgery — the file path is unchanged; only its content shrinks.

## Tasks
- [x] T1: RED — reshape agentguidance tests: protocol-only contract (steps present, triggers/
      TDD/RDD text absent, frontmatter+marker, <cap), stale-part cleanup, idempotency.
- [x] T2: GREEN — RenderRouting slim branch + shared ODD emission; replace split delivery with
      the single-file writer; remove dead split code. Focused package green.
- [x] T3: internal/cli orchestrator_install_test update (agy routing = protocol-only) + any
      CLI assertions touching routing content for agy.
- [x] T4: j4500 journey routing pins → single slim-file contract + idempotency + uninstall.
- [x] T5: verification — focused packages + driven j4500 (exact CI commands) + full suite vs
      the standing contract (exactly 1 environmental failure).
- [x] T6: work-unit commit + close. Push remains the user's decision.

## Evidence
- 2026-10-04: start. Verified: ODD protocol absent from canon files (grep 0 in both source and
  bundle satellites); canon covers all other routing doctrine (grep hits documented above);
  Pi's ODD block comes from gentle-shell extension code + fixtures/odd-routing-canonical.md.
