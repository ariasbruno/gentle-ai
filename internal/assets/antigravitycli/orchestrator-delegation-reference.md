# Orchestrator — Delegation Reference (lazy-loaded)

Bind this to the parent Antigravity session only. Read it only when a situation in the delegation card (`orchestrator-delegation.md`) names it; the card holds the triggers and launch rules, and this file holds the reference material behind them.

### Language Domain Contract

- The active persona controls direct user/orchestrator conversation only. Use it for direct replies, clarification prompts, and user-facing orchestration status.
- Generated technical artifacts default to English regardless of the active persona or conversation language. This includes task documents, code comments, UI copy, tests, fixtures, and delegated phase outputs.
- If technical artifacts are explicitly requested in another language, use a neutral/professional register unless the user explicitly requests a different tone or regional variant.
- Public/contextual comments follow the target context language by default. Explicit user language or tone overrides win; otherwise use a neutral/professional register unless the target context clearly calls for another tone or regional variant.
- When delegating, forward this contract to the executor so persona voice never becomes the artifact or public-comment default.

### Subagent-facing English

Subagent-facing prompts are written in English by default, even when the user speaks Spanish; the handoff rule and its exceptions live in the card. This keeps orchestrator output small and gives built-in/project subagents a consistent operating language without changing the user-facing persona or losing exact requirements.

### Delegation Rules

These rules select execution topology, not the implementation method. Implementation runs as **direct inline** or **delegated direct**; the always-on Task Size section and its Mechanisms select the topology, and file count never does.

Core principle: **does this inflate the parent context without need?** If yes, use one bounded worker. If no, do it inline.



| Action | Direct inline | Delegated direct worker |
|--------|---------------|-------------------------|
| Read to decide/verify within the evidence budget (one parallel batch: at most 3 calls, ~10k tokens) | ✅ | — |
| Read to explore/understand beyond the evidence budget | — | ✅ one narrow explorer (handoff of at most ~2k tokens, `path:line` evidence) |
| Read as preparation for writing | — | ✅ together with the write |
| Write a small task (one understood change, any number of files) | ✅ | — |
| Write a large task with no Writer rule reason | ✅ following the logbook | — |
| Write a unit with a Writer rule reason | — | ✅ one bounded writer per unit |
| Bash for state (`git`, `gh`) | ✅ | — |
| Focused test and suite of the change being made | ✅ once each | — |
| Named high-risk item (assess `escalate`), or long suites, builds, or installs of a large task | — | ✅ independent `gentle-ai-verify` |

Use the platform's native bounded worker for delegated-direct work.

Keep each delegated writer bounded, with a short synthesized handoff. Delegation is mandatory only when a mechanism's own trigger fires, and remains an ODD implementation route.

### Work Routing Ladder

Route work through the smallest harness that is safe. "Smallest" means minimal safe coordination, not zero delegation by default.

#### 1. Inline Direct

Use inline execution when the task is small by the always-on Task Size section: read, edit (one understood change may span files), run its focused test and suite once each, or bash for state. Keep the ODD path proportionate. When a mechanism's own trigger fires, turn on only that mechanism, then re-evaluate.

#### 2. Simple Delegation

Delegate when a mechanism's own trigger fires, within the ODD workflow: understanding an unfamiliar module beyond the evidence budget (explore), implementing a unit with a Writer rule reason (writer), or checking a high-risk change (independent verifier).

Use Antigravity's native subagent runtime. Prefer `invoke_subagent` for a fresh delegated context and `send_message` for follow-up guidance; do not invent an alternate delegation or model-routing contract.

For bounded writes, follow the canonical Writer rule under Delegation Triggers.

Waiting on CI, a build, or a server is not delegation: use native `run_command` and its documented completion mechanism. Do not invent Pi background tools, poll, sleep, or delegate an idle waiter; report an unavailable completion mechanism.

For generic mapping, follow the Evidence-budget rule under Mandatory Delegation Triggers.

The canonical Writer rule under Delegation Triggers overrides the general runtime preference above.

Delegate generic verification that executes or delegates commands per the RDD-aware Verification rule (trigger 3 under Delegation Triggers, Gentle AI#661) -- the normative on/off/unknown routing lives in `orchestrator-verification.md`, not here: the bounded writer always self-verifies via `## Verification`, and the the native review lifecycle `assess` operation's returned plan decides whether `gentle-ai-verify` (or the `invoke_subagent` fallback, with the same read-only verification constraints, exact parent-authorized commands, and fallback reporting) also runs, by a named `escalate` item and whatever the rendered `Receipt-driven development:` line reads, instead of a blanket non-trivial rule (Gentle AI#662). `## Known environmental failures` follows the same definition as `gentle-ai-worker`'s Verification contract: exact pre-existing base failures reported as evidence, never blockers -- any other failing required command still forces `status: partial`. The change's own focused test and suite run inline. Separate exploration stays reserved for when the parent needs the map to decide or route; reading that prepares a write belongs with the writer making the change, consistent with the Delegation Rules table above.

For delegation other than a large task's bounded writes, use the generic fallback: if the native subagent mechanism is unavailable, stop the complex work and explain the blocker instead of silently continuing inline. The delegation trigger remains mandatory; the fallback changes the runtime, not the requirement to delegate.

### Canonical Lightweight Workflows

Bugfix with unfamiliar flow (understanding exceeds the evidence budget):

```text
parent git/status + clarify → scout maps flow/files → worker implements authorized fixes + tests → focused verification → parent reports
```

Conflict or dependency-marker cleanup:

```text
parent reproduces/checks conflict → parent or worker resolves inside the active scope → verify markers, package/lock consistency, and repository cleanliness → parent reports
```

After tooling/worktree incident:

```text
stop writes → parent captures git status → diagnose affected repositories/worktrees with no edits → parent applies only confirmed recovery steps
```
