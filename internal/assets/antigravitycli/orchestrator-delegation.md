# Orchestrator — Delegation Detail (lazy-loaded)

Bind this to the parent Antigravity session only, on delegation or routing triggers. Not always-on; loaded on demand from `~/.gemini/config/plugins/gentle-ai/orchestrator.md`'s pointers.

Detail modules, each loaded only when its mechanism fires (small tasks load none):

- `orchestrator-tracking.md` — large-task ODD tracking: authorization and progress, research depth, checks and candidate consent, phase signaling, RDD assess, delivery strategy.
- `orchestrator-verification.md` — the Verification rule, native risk tiers, writer verification contract.
- `orchestrator-writer.md` — allowed edit surfaces and Judgment Day fix dispatch.
- `orchestrator-prompts.md` — explorer web work, lossless prompts and Gentle AI provider defects.

### Language Domain Contract

- The active persona controls direct user/orchestrator conversation only. Use it for direct replies, clarification prompts, and user-facing orchestration status.
- Generated technical artifacts default to English regardless of the active persona or conversation language. This includes task documents, code comments, UI copy, tests, fixtures, and delegated phase outputs.
- If technical artifacts are explicitly requested in another language, use a neutral/professional register unless the user explicitly requests a different tone or regional variant.
- Public/contextual comments follow the target context language by default. Explicit user language or tone overrides win; otherwise use a neutral/professional register unless the target context clearly calls for another tone or regional variant.
- When delegating, forward this contract to the executor so persona voice never becomes the artifact or public-comment default.

## Language Boundary — subagent-facing English + exceptions

Subagent-facing prompts should be written in English by default, even when the user speaks Spanish. Write the handoff instructions in English, but never translate, condense, or paraphrase the user's requirements: pass them by reference to the feature document's `## Specs` (`Spec: odd/tasks/<feature>.md (read until "## Log"). Do T#; S#.`), or include the user's request verbatim when no feature document exists. This keeps orchestrator output small and gives built-in/project subagents a consistent operating language without changing the user-facing persona or losing exact requirements.

Exceptions:

- Preserve exact user quotes, UI copy, error messages, filenames, commands, and domain terms in their original language when they are evidence.
- Ask a subagent to produce Spanish only when its output is intended to be pasted directly to the user, a PR/comment/reply in Spanish, or Spanish-language product/documentation text.

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
| High-risk change, or long suites, builds, or installs of a large task | — | ✅ independent `gentle-ai-verify` |

Use the platform's native bounded worker for delegated-direct work.

Keep each delegated writer bounded, with a short synthesized handoff. Delegation is mandatory only when a mechanism's own trigger fires, and remains an ODD implementation route.

#### Mandatory Delegation Triggers

These are parent-orchestrator routing boundaries; do not pass these rules to child agents as permission to orchestrate. These triggers are mandatory, not advisory. When one fires, stop and delegate through the runtime's subagent mechanism before continuing; executing past a fired trigger inline is a routing defect even if the work succeeds. Delegation keeps the parent context thin enough to orchestrate; it does not slow the work down.

1. **Ask:** an open product or design decision gets one focused user question; stop and wait. When answered, re-evaluate task size.
2. **Mapping trigger (Evidence-budget rule):** read inline only when the evidence fits one parallel batch of at most 3 calls totaling ~10k tokens, using grep and line ranges, never whole large files. When understanding needs more reading or more than ~5 sequential lookups, delegate one scout/explorer that returns a handoff of at most ~2k tokens with `path:line` evidence before deciding or writing anything. Never force delegation for a small targeted question. The parent does not re-read what the handoff covered, except a single spot check. With the handoff, re-evaluate task size; an understood change continues inline.
3. **Verification rule** (Gentle AI#661/#662, RDD-aware): a high-risk change (Task Size) gets an independent `gentle-ai-verify` run after the change's own checks; otherwise whoever made the change runs its focused test and suite inline, small tasks included. The normative on/off/unknown routing is stated once in `orchestrator-verification.md`; reference it, do not restate it.
4. **Track:** a large task gets the feature document, Engram mirror, `todo`, and work-unit commits (`orchestrator-memory.md`).
5. **Writer trigger (Writer rule):** a large task alone never delegates, and file count never fires this trigger. Delegate one bounded writer per unit only for a named reason: (a) parallelism — 2+ independent units with disjoint files, each clearly heavier than starting a subagent; medium tasks included; (b) context — the Context backstop below. Without a reason, the parent works inline, following the logbook (feature document, mirror, `todo`, work-unit commits).
6. **Incident rule:** after wrong `cwd`, accidental repository/worktree mutation, failed merge recovery, confusing test command, or environment workaround, stop and diagnose the incident separately before resuming.
7. **Context backstop:** when the parent context passes ~150k tokens, pause and delegate the next bounded unit of work. Always keep command output bounded in the parent (counts, `--stat`, `tail`).

**Preparation trigger:** when the Writer rule delegates a write, reading that prepares it, and broad research or context compression, delegate together with or ahead of the write instead of filling the parent context. Inline writes read inline.

**Route declaration:** for large work, record the chosen route per task (inline or delegated) and the trigger evidence in the feature document, so skipped delegation is observable instead of silent.

These triggers only choose between direct inline and delegated direct inside ODD.

For a large task's bounded writes, prefer the installed package-owned `gentle-ai-worker`, then a user-configured `worker`. If neither worker definition exists, delegate the bounded write with Antigravity's native `invoke_subagent`. If no delegation mechanism is available, stop and explain the blocker. Judgment Day phase roles are never generic fallbacks. If the generic writer chain is unavailable, use the documented native generic fallback or stop.

### Work Routing Ladder

Route work through the smallest harness that is safe. "Smallest" means minimal safe coordination, not zero delegation by default.

#### 1. Inline Direct

Use inline execution when the task is small by the always-on Task Size section: read, edit (one understood change may span files), run its focused test and suite once each, or bash for state. Keep the ODD path proportionate. When a mechanism's own trigger fires, turn on only that mechanism, then re-evaluate.

#### 2. Simple Delegation

Delegate when a mechanism's own trigger fires, within the ODD workflow: understanding an unfamiliar module beyond the evidence budget (explore), implementing a unit with a Writer rule reason (writer), or checking a high-risk change (independent verifier).

Use Antigravity's native subagent runtime. Prefer `invoke_subagent` for a fresh delegated context and `send_message` for follow-up guidance; do not invent an alternate delegation or model-routing contract.

A child must be able to do the task it is given. Read its installed agent definition and declared tools before launching; never assume the parent's tools transfer. With no qualified role, report the capability gap; a fired trigger is never replaced by inline work.

For bounded writes, follow the canonical Writer rule under Mandatory Delegation Triggers.

Waiting on CI, a build, or a server is not delegation: use native `run_command` and its documented completion mechanism. Do not invent Pi background tools, poll, sleep, or delegate an idle waiter; report an unavailable completion mechanism.

For generic mapping, follow the Evidence-budget rule under Mandatory Delegation Triggers.

The canonical Writer rule under Mandatory Delegation Triggers overrides the general runtime preference above.

Delegate generic verification that executes or delegates commands per the RDD-aware Verification rule (trigger 3 under Mandatory Delegation Triggers, Gentle AI#661) -- the normative on/off/unknown routing lives in `orchestrator-verification.md`, not here: the bounded writer always self-verifies via `## Verification`, and the the native review lifecycle `assess` operation's returned plan decides whether `gentle-ai-verify` (or the `invoke_subagent` fallback, with the same read-only verification constraints, exact parent-authorized commands, and fallback reporting) also runs, by native risk tier and whatever the rendered `Receipt-driven development:` line reads, instead of a blanket non-trivial rule (Gentle AI#662). `## Known environmental failures` follows the same definition as `gentle-ai-worker`'s Verification contract: exact pre-existing base failures reported as evidence, never blockers -- any other failing required command still forces `status: partial`. The change's own focused test and suite run inline. Separate exploration stays reserved for when the parent needs the map to decide or route; reading that prepares a write belongs with the writer making the change, consistent with the Delegation Rules table above.

#### Key Learnings closing block

When delegating to a generic Explore/general worker (`gentle-ai-explore`, `gentle-ai-worker`, `gentle-ai-verify`) or their `invoke_subagent` fallback, include the same `## Key Learnings` closing instruction in the delegated prompt: after the worker returns its normal result envelope or handoff, it closes its final response text with a `## Key Learnings` block of 1–5 numbered items, each a standalone factual sentence of at least 20 characters and at least 4 words, omitting the block when there is genuinely no reusable learning. The block layers on after the structured Return contract and does not alter its fields. This applies to final response text only — not intermediate tool output. The Engram memory provider automatically extracts and persists these items as passive capture; the worker does not parse the block or invoke passive-capture tools itself. This is separate from explicit `mem_save` artifact/decision persistence. Agents that must return strict JSON never receive this closing instruction; their required output shape remains unchanged.

For delegation other than a large task's bounded writes, use the generic fallback: if `invoke_subagent` or `send_message` tools are unavailable, use `invoke_subagent` tool or another available delegation mechanism. The delegation trigger remains mandatory; the fallback changes the runtime, not the requirement to delegate. If no delegation mechanism is available, stop the complex work and explain the blocker instead of silently continuing inline.

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
