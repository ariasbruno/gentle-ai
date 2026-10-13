# Orchestrator — Verification (lazy-loaded)

Bind this to the parent Antigravity session only. Load it when the Verification rule fires (a high-risk change) or a delegated writer returns; the small path never needs it.

## Verification rule (normative)

5. **Verification rule**: the bounded writer always self-verifies with exact authorized checks. Follow the native risk tier and the table below in any RDD state. High risk always requires independent `gentle-ai-verify` before native review; medium adds it for a small-model writer, and failed assessment is treated as high. Native review never replaces or skips verification and never authorizes delivery.

| Native risk tier | Verification (any RDD state) |
|---|---|
| passive | structural readback by the parent; no separate verifier, no tests |
| medium | writer self-verification stands; a separate `gentle-ai-verify` run is added only when the writer profile is a small model (mini or low effort) |
| high | writer self-verification plus a separate `gentle-ai-verify` run, always |
| unknown / assess failed | treated as high |

The small-model bias raises the tier by one for verification purposes (medium becomes high); an unknown `Receipt-driven development:` line never lowers a tier below `off`. The parent spot check (re-running one reported command before delivery) stays required in every tier. ASSESS takes the writer profile from the runtime-recorded model and effort of the pending mutations for the root; caller `writerModelId`/`writerEffort` are only a fallback when no runtime evidence exists, and a missing model, a `mini` model token (`gemini` is not mini), or `low` effort keeps the conservative small-model bias. When native reports them, ASSESS also projects `reviewDue`, `reviewDueReason`, `candidate.consumed`, and the native continuation verbatim; relay that continuation unchanged and never rebuild it. A native code review is not a substitute for applicable functional checks: tests, builds, and functional verification such as browser checks for UI changes still run when applicable, and review outcomes never authorize delivery. The change's own focused test and suite run inline; a small task needs no separate verifier unless it is high risk.

## Agent escalation (gentle-shell#1494)

When the change touches a Task Size high-risk item and the native tier is passive or medium, the native review assess operation raises the tier to high automatically; the raise binds the verification plan only, not the native review tier. The model never calls assess directly.

## Parallel review protocol (gentle-shell#1731)

1. **Self-review**, required by each writer prompt, in its own session before returning: spec sections by reference (#1713), the request's authorized examples, tests, and typecheck; it fixes and continues (one correction per failing check, a second only if the same check still fails, then `partial`), reporting requirement by requirement. Low and medium risk need nothing else.
2. **Independent verify per unit**, once, after its final self-review (never per worker cycle or retry), in parallel when several finish together, only when that unit is high risk: its Risk line, `assess` over its actual diff or the worker's own `escalate` (high-risk list in Task Size; small-model bias per the tier table), never inferred from the worker's summary alone. Assess per unit via its work-unit commit (`{"baseRef":"<previous>","committedOnly":true}`) or its own isolated worktree. Same-model inline code over several deliveries of one feature: one verify at the feature's end; smaller-model writer code stays per unit.
3. **Seam check**: after parallel units finish, one inline full-suite command (the parent spot check) catches seams between units.
4. **Verify handoff**: the whole feature document (every `S#`, never one task), the baseline commit, and the probe command forms (tests, CLI, typecheck) for probes on isolated state in a fresh `mktemp -d` scratch copy, never just "read-only"; `gentle-ai-verify` holds the checklist. Resolve every unmet item before closing; the writer commits its probes as regression tests.
5. **One correction**: verify blockers get one correction batch that fixes every reported blocker, then one re-verify limited to those blockers, never a new full sweep. Blockers still open after that stop as one **Needs your decision**. Advisories never start a correction. A second correction runs automatically only when the recheck shows the same blocker still failing; a new finding never earns one. A `partial` or `blocked` writer report gets only this correction, never an extra verify run beyond the risk tier's verifier; partial, blocked, unavailable, or exhausted proof left after it becomes that one **Needs your decision**.
