---
name: gentle-ai-agy-upstream-update
description: "Trigger: AGY upstream update, AGY migration, actualizar AGY, AI/Shell release. Update the existing integration incrementally with pinned inputs."
license: Apache-2.0
metadata:
  author: ariasbruno
  version: "1.0"
---

## Activation Contract

Load for planning or executing an AGY integration update against Gentle AI, Gentle Shell, or AGY runtime changes. Read [the procedure](references/procedure.md) first.

## Hard Rules

- Keep planning read-only. Execute only explicitly authorized migration work; this skill grants no commit, publication, destructive Git, installation, authentication, model-run, or native review consent.
- Pin AI/Shell commits and the AGY validation version; never migrate against moving `main` or a dirty generator source.
- Start one update branch from the last accepted AGY integration: `chore/agy-integration-ai-v{AI}-shell-v{Shell}`. Preserve previous branches and user changes; inspect existing attempts instead of overwriting them.
- Merge pinned AI into that baseline. Never rebuild the fork from upstream or replay its full history for each release.
- Resolve ownership and behavior, not merely conflict markers. Regenerate generated assets; preserve AGY behavior and user-owned data.
- Never equate a clean merge, compilation, capability claim, or generated-file equality with functional compatibility or feature parity.

## Decision Gates

| Condition | Action |
| --- | --- |
| Missing pin, accepted baseline, or ownership | Resolve before mutation. |
| Textual or semantic conflict | Reconcile both intentions; test the affected behavior. |
| Product choice or required capability removed | Ask one focused question; stop. |
| Relevant feature unsupported or unknown | Record evidence and disposition; do not silently mark not applicable. |
| Review exceeds its context budget | Stop for a bounded review strategy over the update; never reconstruct the historical fork. |

## Execution Steps

1. Record pinned inputs, ancestry, baseline checks and execution scope using [the report](assets/migration-report.md).
2. Inventory AI, Shell and AGY changes before filtering; classify impact and feature coverage.
3. For authorized execution, create the update branch, merge pinned AI, resolve impacted seams and regenerate Shell-derived assets reproducibly.
4. Verify affected public behavior, generated drift and final integration; attribute failures against the baseline.
5. Review update work units relative to the accepted AGY baseline under repository policy. Keep official upstream contribution separate.

## Output Contract

Return inputs, branch/base, impact and feature matrices, conflict decisions, observed checks, deferred/unknown support, blockers and closure status. Separate compatibility, coverage and upstream contribution readiness; claim only observed evidence.

## References

- [Procedure](references/procedure.md)
- [Migration report template](assets/migration-report.md)
- [Branch and PR policy](../branch-pr/SKILL.md)
- [Review slicing policy](../chained-pr/SKILL.md)
