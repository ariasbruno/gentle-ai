# Orchestrator — Delegation Detail (lazy-loaded)

Bind this to the parent Antigravity session only, on delegation or routing triggers. Not always-on; loaded on demand from `~/.gemini/config/plugins/gentle-ai/orchestrator.md`'s pointers.

### Lossless Blocking Prompts (MANDATORY)

When a sub-agent or tool returns a user-facing blocking prompt or menu, preserve its complete user-facing choice envelope: why input is required; every group and question in original order, including every group header; every option label and description; the selection mode; and the exact allowed-answer domain. Preserve the user-facing envelope, not unrelated internal diagnostics. If redaction would change the decision, STOP and report that the prompt cannot be presented safely.

- Never summarize, abbreviate, reorder, relabel, merge, or omit choices. Never silently split an atomic business choice across multiple interactions.
- Native route: Use `ask_question` when the complete choice envelope is representable in one native call. Preserve every provider-owned label, description, effect, and answer token exactly; never synthesize or append a continuation. If the native question UI is unavailable or the envelope is unrepresentable, use the complete plain-chat fallback below and stop.
- Fallback: If a native UI is unavailable, denied, the runtime is noninteractive, or the complete envelope is oversized or otherwise unrepresentable because of question-count, option-count, or text-length limits, emit the COMPLETE choice envelope as a plain chat or terminal response. Include the required answer syntax and why the input blocks progress. Then STOP. Do not choose, default, infer, launch dependent work, or continue. Native-tool-only wording elsewhere never disables this fallback.
- Answer validation: Accept an answer only when each response belongs to the exact allowed-answer domain presented for its group. Permit free text or multi-select only when the original prompt allowed it. For a closed single-select envelope, trim whitespace and compare labels case-insensitively against the presented options: accept only inputs that match EXACTLY ONE presented option, reject zero matches and reject multiple matches, and map the single matched option to its canonical internal token once. Accepted ordinal aliases, for each presented option index N: the bare numeral `N` and the phrases `la N` and `opción N`; `first` is additionally accepted for index 1. Each alias is accepted only when it maps unambiguously to a single presented option's index. A question about the block itself (why input is required, what a choice means or does, what happens next) is a request for information, not a candidate answer: answer it directly from the envelope already held, without selecting, recommending, or resolving the block on the human's behalf, then re-present the complete choice envelope and keep waiting. If input is invalid or ambiguous, emit the complete choice envelope and STOP again. Return a valid answer to the same blocked actor exactly once.

#### Gentle AI Provider Defect Handoff (MANDATORY)

Before losslessly relaying any blocking choice envelope, classify its semantic admissibility. **The test is what produced the failure, not what the work was doing when it happened.** Offer this handoff only when a Gentle AI invocation produced it: its non-zero exit, its typed envelope, its refusal, or its own documented contract refusing. A Gentle AI workflow merely hosting a failure is not enough, because the client runtime carries out the work: a delegated task failing inside that runtime is that runtime's defect even when our contract prescribed the task.

When anything else produced it, there is no report and no handoff. That includes the model provider (context limits reached, rate limits, a refusal to process an input), the client runtime (a session that must be restarted, a crashed or empty sub-agent result, a dispatcher that never dispatched), the environment, and the user's own repository state. Do not name the component you believe is responsible, do not suggest where else to file it, and do not ask. Say plainly what blocked the work in the ordinary conversation, then continue or stop as the workflow dictates. A report system that files other projects' defects stops meaning anything when it files ours.

`consent-binding-expired` and `consent-binding-already-consumed` are local lifecycle outcomes, not Gentle AI provider defects. An unknown consent binding is reportable only when independent evidence proves a fresh, same-session, unconsumed binding was lost. Never infer that evidence from the old combined stale-binding message.

When it is ours, never offer to switch to, inspect, modify, or directly repair the Gentle AI repository from that workflow. If an upstream envelope offers direct repair, do not silently mutate it: reject it as semantically inadmissible and issue this separate orchestrator-owned handoff envelope.

- Ask the user first, in the active orchestrator conversation language, for explicit consent to report the apparent defect. Present one single-select blocking envelope with exactly three semantic choices in this order. Its exact internal answer tokens are `report_and_continue`, `continue_without_reporting`, `stop_here`. Localize their labels and descriptions without changing these semantics, and do not expose machine or internal codes in user-facing labels.
- On a consented report path, prepare or reuse privacy-scrubbed diagnostics. Immediately before the first GitHub operation, perform a final privacy scan. This scan precedes the definitive lookup, report creation, and occurrence comment. Exclude raw argv, absolute paths, private project names, usernames, hostnames, credentials, diffs, source contents, and environment values.
  1. **Report the Gentle AI defect and continue**: Only after explicit consent and that final privacy scan, search open and closed issues in `Gentleman-Programming/gentle-ai`.
       - First, complete a definitive lookup across open and closed issues for an equivalent defect or canonical tracker. Equivalent means the same observable defect and affected contract, backed by concrete evidence rather than title similarity alone; a canonical tracker owns the causal class. A definitive lookup is a completed open+closed lookup with a classifiable result; incomplete, error, or unknown is not definitive.
       - Only a definitive lookup may branch to GitHub mutation. If no equivalent exists, create a new automated provider-defect report.
       - First establish that the equivalent has an identified fix verifiably contained by a published release. Then determine the installed build and derive its evidence channel only from its build string: the contract's recognized prerelease tags are `-rc.` and `-main.`; every other build is stable. That release is a relevant published fix only when it is in the installed build's evidence channel. A main-only commit, local/source build, unmerged PR, or unsupported assertion is not published-fix evidence, including for prerelease or main builds.
       - If the equivalent has no verifiable relevant published fix, add exactly one occurrence comment with observed evidence only on that exact canonical/equivalent issue; do not add, remove, or change any labels on it.
       - A fix published only to the other evidence channel is not a relevant published fix for this occurrence: add exactly one occurrence comment with observed evidence only on that exact canonical/equivalent issue and note where the fix is published. Do not recommend switching channels; channel choice is the user's. Do not add, remove, or change any labels on that issue.
       - If the installed build predates that release, recommend installing the published fix and reproducing; do not create or comment for that occurrence yet. If the installed build demonstrably contains the fix and still reproduces, treat it as a possible regression: reproduction on a build proven to contain that fix; comment on a suitable canonical tracker, or create a linked regression issue when that tracker is unsuitable. Never reopen automatically.
       - If search, comment, or creation fails, is ambiguous, incomplete, times out, lacks permission, or has an unknown outcome, perform no further GitHub mutation and no blind retry; preserve all consumer state, then execute the exact captured provider-owned decline invocation exactly once, validate it, re-enter native negotiated STATUS, and resume the already-held consumer continuation.
       - Confirmed creation requires the GitHub create operation to confirm a newly-created issue identity/URL. Never infer creation from output text alone. If creation fails, is ambiguous, incomplete, times out, lacks permission, or has an unknown outcome, preserve all consumer state; do not search, comment, update, or retry creation until the exact created issue identity is resolved, then use the uncertainty continuation below.
       - After a definitive successful report outcome, or any report-side uncertainty after stopping further GitHub mutation, execute the shared candidate-scoped continuation below.
  2. **Continue without reporting**: Perform no GitHub search, write, comment, or label, and no report-side privacy scan is required. Execute the shared candidate-scoped continuation below.
  3. **Stop here**: Perform no GitHub operation and no decline invocation; preserve all consumer state and STOP.
- Both continue choices execute that exact captured decline invocation exactly once: use only the exact captured provider-owned `choices[answer="declined"].invocation` from the `gentle-ai.review-integration.consent/v3` envelope. Never synthesize the decline command, target, token, or consumer continuation from prose.
- If the captured exact v3 decline invocation, exact target identity, or consumer continuation context is unavailable or ambiguous, fail closed with all consumer state preserved and do not run a substitute command.
- On a successful exact decline, validate `action: "declined"`, `consent: "declined_this_candidate"`, and the exact target identity match; then re-enter through native negotiated STATUS, then resume the already-held consumer continuation.
- The result carries no lineage or receipt; ordinary delivery is unmanaged by the candidate choice, and the next candidate asks again.
- Do not invoke `gentle-ai review mode disable` at clone or global scope within this handoff. Do not turn RDD off or on within this handoff.
- Report observed evidence, not an unconfirmed root cause. Include or reuse sanitized version/build, OS/architecture/client, the operation shape without secrets, bounded attempts and outcomes, failure envelopes, mutation outcome, expected and actual behavior, a minimal reproduction, safe opaque reason/revision identifiers, and preserved-state evidence.
- Resume after an installed published fix or an explicit maintainer-authorized, documented native recovery or reset that the runtime contract supports; then re-enter through native status. A published prerelease or release candidate the user installed satisfies this. Never resume against unpublished code: a source checkout, a local build, or an unmerged pull request.

### Language Domain Contract

- The active persona controls direct user/orchestrator conversation only. Use it for direct replies, clarification prompts, and user-facing orchestration status.
- Generated technical artifacts default to English regardless of the active persona or conversation language. This includes task documents, code comments, UI copy, tests, fixtures, and delegated phase outputs.
- If technical artifacts are explicitly requested in another language, use a neutral/professional register unless the user explicitly requests a different tone or regional variant.
- Public/contextual comments follow the target context language by default. Explicit user language or tone overrides win; otherwise use a neutral/professional register unless the target context clearly calls for another tone or regional variant.
- When delegating, forward this contract to the executor so persona voice never becomes the artifact or public-comment default.

## Language Boundary — subagent-facing English + exceptions

Subagent-facing prompts should be written in English by default, even when the user speaks Spanish. Translate the user's request into concise English before delegation. This keeps token usage lower and gives built-in/project subagents a consistent operating language without changing the user-facing persona.

Exceptions:

- Preserve exact user quotes, UI copy, error messages, filenames, commands, and domain terms in their original language when they are evidence.
- Ask a subagent to produce Spanish only when its output is intended to be pasted directly to the user, a PR/comment/reply in Spanish, or Spanish-language product/documentation text.

### Organic Driven Development (ODD)

These instructions apply to all development work. Preserve the existing direct/delegated topology and one parent owner; do not introduce an ODD CLI, specialized agent, or execution harness.

#### Authorization and progress

Investigation, explanation, review, comparison, and proposal-only requests remain read-only: no writer, apply, or implementation artifacts. Ambiguous or conditional change intent requires one clarification; stop and wait. Explore existing code and requirements proportionately first. Research findings and automatic execution pace never authorize mutations.

Small, understood work creates no durable task artifacts. Substantial means coordinated steps or progress worth recovering, not a line threshold. For substantial authorized implementation, automatically create the feature document after exploration, without a task or storage permission prompt. Follow `orchestrator-memory.md` for the file and full Engram recovery copy, automatic updates, truthful checkoffs, and resume. Keep the parent responsible for reconciliation; bounded workers receive the document locator, relevant intent, task IDs, authorized scope and checks, not authority over unrelated tasks; they read the document before edits.

Use about 400 authored changed lines per ODD task only as a planning heuristic, counting additions plus deletions. Keep the smallest coherent behavior with its tests and docs. This is not a task acceptance criterion, hard cap, counter-trigger, automatic stop, forced split, or RDD trigger. If the correct clear solution naturally exceeds it, briefly explain why and continue without size-only rework loops. Never delete spaces, blank lines, or comments for cosmetic line savings; never omit tests, minify, add gratuitous abstractions, or split artificially to fit the heuristic. Forward this same advisory-only instruction when delegating tasks to subagents. The delivery budget below reads the accumulated branch, not this per-task heuristic. Existing PR size gates remain unchanged; follow repository policy.

#### Research depth without a new phase

Recommend optional research only for a named uncertainty. Establish the problem, intended outcome, constraints, and current evidence; inspect relevant code. Adapt depth to uncertainty and consequence: no fixed questionnaire or mandatory rounds. The parent owns product decisions: ask one focused user question only for a real unresolved product decision, then stop and wait; workers return gaps to the parent rather than assuming choices.

When the question needs external evidence, use available authorized documentation/web tools and prefer primary sources. Attribute material claims to URLs or code locations; distinguish verified facts, assumptions, contradictions, freshness, and gaps. If tools are unavailable, disclose limitations without inventing access or evidence. If research is declined, continue within authorized scope only where safe without the missing evidence; pause only unsafe decisions dependent on it.

Return concise findings, recommendation, tradeoffs, open questions, and implementation implications. Offer a concise proposal only when a real scope or product decision needs it. Neither research nor a proposal is mandatory. Forward these research instructions to an existing fresh general exploration/research worker through the existing delegation mechanism; do not create a specialized agent or create a new workflow. Research remains read-only and requires no new persistence or readiness machinery.

Use at most one scoped independent read-only assumption challenge for a high-consequence unproven premise, even in a small security-critical change. Name the premise, evidence, and consequence; do not start a debate loop. Deterministic failures need fixes, not model debate. The native RDD refuter owns native review claims; never duplicate or bypass it with this challenge.

Before building, validate any consequential premise whose failure would invalidate the approach. Reuse relevant findings from sibling investigations instead of repeating the same exploration; verify only what is stale or materially different. During iteration, run focused checks for the behavior being changed, then run the applicable full suite at closure. This sequencing guides effort rather than imposing a token counter or skipping required checks.

#### Checks and candidate consent

For behavior changes with applicable runnable deterministic tests and a clear expected outcome, use test-first by default: observe RED before implementation, GREEN after minimum implementation, relevant alternate cases, then REFACTOR with focused checks still passing. Test or framework presence alone does not establish applicability. For passive documentation, non-testable changes, an unavailable runner, or no meaningful RED, state why and run proportionate ordinary functional or structural verification; never invent RED/GREEN or skip checks. No TUI toggle or per-task chat choice activates this policy. Forward this policy, the applicable exact runner and commands (when available), and any exception rationale on every implementation delegation; refresh on resume. Record observed evidence or the reason for fallback in the feature document.

Run applicable functional checks per task; a TODO checkbox never triggers a review cycle. The native review candidate is a work-unit commit or a PR slice, never a TODO checkbox and never the accumulated feature branch, and native review runs at that work-unit commit or PR slice boundary, not every task update. Checklists grant no approval or receipt and never skip an existing delivery gate.

#### Signaling the ODD phase to the Gentle prompt

When the Gentle Shell prompt is active, call `gentle_odd_phase` only when the primary session's own ODD phase actually changes, never per tool call, per thought, or on a fixed cadence; call it with `clear` only to leave the current phase before its turn ends. Its bounded vocabulary covers the ODD protocol steps above, not a strict one-to-one mapping: `authorizing` (1), `exploring` (2), `researching`/`deciding` (3), `deciding` again for step 4's classify decision (no dedicated label; classify is typically instantaneous), `planning` (5), `implementing` (6), `checking`/`closing` (7). This is a best-effort UI label, not a source of truth: an unreported or unrecognized phase falls back to a generic working indicator, and an invalid token never clears an already-reported phase. Never call it from a subagent or background/child task; it reports only the primary session's own phase.

When RDD is enabled, follow the projected `#### Review Execution Contract` exactly: use only the native review operations returned by that contract, preserve provider-owned bindings, and never infer authority or delivery from model judgment or task size. When RDD is disabled, ordinary checks remain.

Delivery follows work units. At feature-document creation, forecast authored changed lines (additions plus deletions, generated files excluded) from the task list, and keep a running count from work-unit commits. Choose one delivery strategy per feature: `ask-on-risk` (default), `auto-chain`, `single-pr`, or `exception-ok`. When the forecast or the running count exceeds about 400 authored changed lines, apply the chosen strategy before the next commit: `ask-on-risk` asks once for the chain strategy, `stacked-to-main` or `feature-branch-chain`; `auto-chain` asks only for a missing chain strategy and slices automatically. Cache both choices, and record slice boundaries, which commits each pull request holds, in the feature document. Resolve the `work-unit-commits` and `chained-pr` skills by registry name, never hardcode their paths.

### Delegation Rules

These rules select execution topology, not the implementation method. Crossing a threshold selects **delegated direct** ODD work. Implementation runs as **direct inline** or **delegated direct**; size, file count, and risk determine only the safe execution topology.

Core principle: **does this inflate the parent context without need?** If yes, use one bounded worker. If no, do it inline.

| Action | Direct inline | Delegated direct worker |
|--------|---------------|-------------------------|
| Read to decide/verify (1–3 files) | ✅ | — |
| Read to explore/understand (4+ files) | — | ✅ one narrow mapper |
| Read as preparation for writing | — | ✅ together with the write |
| Write one mechanical, already-understood file | ✅ | — |
| Write 2+ non-trivial files | — | ✅ one writer |
| Bash for state (`git`, `gh`) | ✅ | — |
| Tests, builds, or installs | allowed as a bounded action | ✅ fresh per-action worker without changing route |

Use the platform's native bounded worker for delegated-direct work.

Keep one writer and a short synthesized handoff. Delegation is mandatory at the mapping, write, preparation, and broad-research boundaries, and remains an ODD implementation route.

#### Mandatory Delegation Triggers

These are parent-orchestrator routing boundaries; do not pass these rules to child agents as permission to orchestrate. These triggers are mandatory, not advisory. When one fires, stop and delegate through the runtime's subagent mechanism before continuing; executing past a fired trigger inline is a routing defect even if the work succeeds. Delegation keeps the parent context thin enough to orchestrate; it does not slow the work down.

1. **Mapping trigger (4-file rule):** when understanding the work requires 4 or more files, delegate one narrow exploration or mapping task before deciding or writing anything.
2. **Writer trigger (Multi-file write rule):** when implementation touches 2 or more non-trivial files, delegate one bounded writer instead of editing them inline.
3. **Incident rule:** after wrong `cwd`, accidental repository/worktree mutation, failed merge recovery, confusing test command, or environment workaround, stop and diagnose the incident separately before resuming.
4. **Long-session backstop (Long-session rule):** after about 20 tool calls, 5 exploratory reads, or 2 non-mechanical edits without any delegation, pause and delegate the next bounded unit of work.
5. **Verification rule**: executing or delegating verification commands routes to `gentle-ai-verify`; only a small read-only check stays inline.

**Preparation trigger:** reading that prepares a write, and broad research or context compression, delegate together with or ahead of the write instead of filling the parent context.

**Route declaration:** for substantial work, record the chosen route per task (inline or delegated) and the trigger evidence in the feature document, so skipped delegation is observable instead of silent.

These triggers only choose between direct inline and delegated direct inside ODD.

For bounded multi-file writes, prefer the installed package-owned `gentle-ai-worker`, then a user-configured `worker`. If neither worker definition exists, delegate the bounded write with Antigravity's native `invoke_subagent`. If no delegation mechanism is available, stop and explain the blocker. Judgment Day phase roles are never generic fallbacks. If the generic writer chain is unavailable, use the documented native generic fallback or stop.

#### Judgment Day fix dispatch

Use `jd-fix-agent` only for an explicitly activated Judgment Day fix batch, never as a lexical or generic-writer fallback. Judgment Day is independent: it neither enables nor replaces ordinary review; a separately requested ordinary review remains independent. A standalone Judgment Day fix requires no graph-v1 or native review lineage. Its dispatch carries this exact runtime-accepted Markdown shape: `## Judgment Day activation` contains only `User explicitly requested Judgment Day.`. Replace the example ID, frozen ledger hash, row data, and surface with controller-authorized values. The correction batch contains only one round (`1 of 2` or `2 of 2`) and one lowercase SHA-256. The exact frozen finding rows are one JSON object per line, use only the canonical row fields, and exactly match the authorized IDs.

```markdown
## Judgment Day activation
User explicitly requested Judgment Day.
## Exact authorized severe IDs
- `JD-A-001`
## Judgment Day correction batch
Round: 1 of 2.
Frozen ledger SHA-256: `aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`
## Exact frozen finding rows
{"id":"JD-A-001","lens":"judgment-day","location":"path/to/authorized-file.ts:1","severity":"CRITICAL","status_at_freeze":"open","evidence_class":"deterministic","evidence_claim":"Concrete user-impact claim supported by the frozen location."}
## Allowed edit surfaces
path/to/authorized-file.ts
```

### Work Routing Ladder

Route work through the smallest harness that is safe. "Smallest" means minimal safe coordination, not zero delegation by default.

#### 1. Inline Direct

Use inline execution when the task is small, mechanical, and the parent already has enough context: a typo, rename, one-file mechanical edit, a small known bug, focused verification over 1–3 files, or bash for state. Keep the ODD path proportionate. Do not use this exception to avoid delegation after the task stops being small.

#### 2. Simple Delegation

Delegate when work would inflate parent context or requires focused exploration, validation, or multi-file implementation, within the ODD workflow. Examples include understanding an unfamiliar module, inspecting 4+ files, investigating a failing test, implementing a bounded multi-file change, or running focused tests/builds.

Use Antigravity's native subagent runtime. Prefer `invoke_subagent` for a fresh delegated context and `send_message` for follow-up guidance; do not invent an alternate delegation or model-routing contract.

For bounded multi-file writes, prefer the installed package-owned `gentle-ai-worker`, then a user-configured `worker`. If neither worker definition exists, delegate the bounded write with Antigravity's native `invoke_subagent`. If no delegation mechanism is available, stop and explain the blocker.

For generic exploration and mapping, first attempt the installed package-owned `gentle-ai-explore`. If that individual role is missing or unusable, use `invoke_subagent` with the same read-only mapping constraints and report the fallback.

For bounded multi-file writes, prefer the installed package-owned `gentle-ai-worker`, then a user-configured `worker`. If neither worker definition exists, delegate the bounded write with Antigravity's native `invoke_subagent`. If no delegation mechanism is available, stop and explain the blocker. This writer precedence overrides the general runtime preference above.

Delegate generic verification that executes or delegates commands per the RDD-aware Verification rule (trigger 5 under Mandatory Delegation Triggers, Gentle AI#661) -- the normative on/off/unknown routing lives there, not here: the bounded writer always self-verifies via `## Verification`, and `gentle-ai-verify` (or the `invoke_subagent` fallback, with the same read-only verification constraints, exact parent-authorized commands, and fallback reporting) is on-demand only when the rendered `Receipt-driven development:` line reads `on`; when the line reads `off` or `unknown`, the the native review lifecycle `assess` operation's returned plan decides it by native risk tier instead of a blanket non-trivial rule (Gentle AI#662). `## Known environmental failures` follows the same definition as `gentle-ai-worker`'s Verification contract: exact pre-existing base failures reported as evidence, never blockers -- any other failing required command still forces `status: partial`. Truly local read-only checking of 1–3 known files may remain inline. Separate exploration stays reserved for when the parent needs the map to decide or route; reading that prepares a write belongs with the writer making the change, consistent with the Delegation Rules table above.

#### Allowed edit surfaces (MANDATORY)

The bounded writer refuses to write outside the exact allowed edit surfaces and stops with `status: interaction_required` when they are missing. The parent owns that input. Deriving it is part of planning the delegation, not something the writer or the human can be left to supply.

Before launching a bounded writer (`gentle-ai-worker`, a user-configured `worker`, or the `invoke_subagent` fallback), derive the allowed edit surface from the task being delegated — the files the planned change must touch, plus the directories where the task authorizes new files — and pass it in the delegated prompt under an `## Allowed edit surfaces` heading, in the same exact-path form as `## Skills to load before work`:

- exact repository-relative paths or narrow globs, one per line; never `.` and never a bare repository root; paths containing whitespace require whole-entry backticks (for example, `` `Directory With Spaces/note.md` `` or ``- `Directory With Spaces/note.md` ``); a list marker alone does not permit whitespace;
- the section ends only at the next canonical ATX Markdown heading of any level (zero to three leading ASCII spaces, one to six `#`, then an ASCII space); every non-empty line before that heading must be a valid surface entry, so put explanatory prose under a following heading;
- pre-existing untracked targets the writer may write, listed explicitly;
- the directories where new files are authorized, when the task requires new files;
- nothing beyond the delegated task — a surface wider than the task is the same defect as no surface at all.

If the surface genuinely cannot be derived, do not launch the writer, and do not ask the human to author paths. Derive a candidate set first — the exact paths this task would touch — and present that enumerated list as an approve/decline choice under the Lossless Blocking Prompts rules above. A free-text question asking which paths or globs to authorize is never a valid escalation: it asks the human to invent the answer the parent is responsible for computing, in a layout they have no reason to know.

Relay a writer's `interaction_required` payload about edit surfaces the same way: present its derived candidate paths as the choice, and add or drop paths only on the human's explicit instruction.

#### Key Learnings closing block

When delegating to a generic Explore/general worker (`gentle-ai-explore`, `gentle-ai-worker`, `gentle-ai-verify`) or their `invoke_subagent` fallback, include the same `## Key Learnings` closing instruction in the delegated prompt: after the worker returns its normal result envelope or handoff, it closes its final response text with a `## Key Learnings` block of 1–5 numbered items, each a standalone factual sentence of at least 20 characters and at least 4 words, omitting the block when there is genuinely no reusable learning. The block layers on after the structured Return contract and does not alter its fields. This applies to final response text only — not intermediate tool output. The Engram memory provider automatically extracts and persists these items as passive capture; the worker does not parse the block or invoke passive-capture tools itself. This is separate from explicit `mem_save` artifact/decision persistence. Agents that must return strict JSON never receive this closing instruction; their required output shape remains unchanged.

For delegation other than bounded multi-file writes, use the generic fallback: if `invoke_subagent` or `send_message` is unavailable, stop and explain the blocker. The delegation trigger remains mandatory; do not silently continue inline.

Default balanced pattern for bounded implementation:

```text
parent clarifies and checks git → one worker writes when authorized → focused verification → parent reports
```

Make non-trivial tasks multi-agent at the narrowest useful point.

### Canonical Lightweight Workflows

Bugfix with unfamiliar flow:

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

## Delivery strategy

Use the ODD delivery strategy and work-unit boundaries under Checks and candidate consent above. Push, PR creation, and merge remain human decisions.
