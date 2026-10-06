---
name: sdd-delivery
description: >-
  End-to-end Spec Driven Development (SDD) delivery process for any repository, with the owner deciding and the agent executing: evidence-based repo audit (specs/ANALYSIS.md and decisions D1..Dn), constitution, one spec per area (FR/NFR/AC), ROADMAP by phases and versions, one epic per phase with tickets as sub-issues, one branch and one PR per ticket, CI with strong gates, red CI fixed at the root cause, review phase, closing PR, CHANGELOG and tag. Writes specs, issues and PRs in the repository owner's language. Use whenever the user asks to analyze, verify, finish, organize or professionalize a repository, even without mentioning SDD. Triggers: "audit this repo", "make this repo production ready", "organize the specs", "write the specs", "create the tickets", "create the epics", "one PR per issue", "ticket per branch", "phase plan", "review phase", "close the phase", "prepare the release", "spec driven development", "SDD", "análise e verificação do repo", "crie as specs", "gere os tickets", "feche a fase"."
---

# SDD Delivery — analysis → specs → tickets → PRs → review → release

The process used to take a repository from prototype to professional project, with
**the owner deciding** and **the agent executing**. It is generic: it depends on no language
or build tool, only on GitHub (issues, sub-issues, Actions) and on a local target
equivalent to CI.

Full normative rules: [references/process.md](references/process.md).
Document and text templates: [references/templates.md](references/templates.md).
CI/test gates and lessons learned: [references/quality-gates.md](references/quality-gates.md).
Checklist for starting in a new repository: [references/bootstrap-checklist.md](references/bootstrap-checklist.md).

## Overview

```text
discovery → ANALYSIS.md → ⏸ owner's decisions → constitution + specs + ROADMAP
  → epic per phase → tickets (sub-issues) → branch per ticket
  → tests + code → local validation → push → PR → green CI
  → ⏸ owner's review → merge (owner) → closing PR → tag (owner) → release
```

`⏸` = **stop point**: the agent stops and waits for the owner. Do not move on alone.

## Language

This skill is written in English, but what it produces is not:

- **Specs, issues, PRs, comments and documentation** are written in the **owner's language**.
- **Commits and code** (identifiers, code comments, branch names) are in **English**.

Find the owner's language in this order and stop at the first hit:

1. An explicit rule in `CLAUDE.md` or `AGENTS.md` (e.g. "Idioma: português", "Language: English").
2. The language `CLAUDE.md` or `AGENTS.md` is written in.
3. The language of the existing `specs/`, `README.md` and recent issues.
4. None of these: **ask the owner once**, then record the answer in `CLAUDE.md`
   (create it if needed) so nobody has to ask again.

The chat follows the language the owner writes in. The templates in
[references/templates.md](references/templates.md) are in English: translate headings and text.
Keep machine-read tokens as they are: `Status:` values (`Draft`, `Approved`, `In Progress`, `Done`),
the IDs (`FR-n`, `NFR-n`, `AC-n`, `Dn`, `Tn`), `MUST`/`SHOULD`/`MAY`, `ADDED`/`MODIFIED`/`REMOVED`,
`[Unreleased]` and the `Closes #N` line.

Portuguese terms, used in Portuguese repositories and by the template scripts (`sdd-epic.sh`,
`sdd-phase-status.sh`, `sdd-check.sh`, `sdd-mark.sh`), which read some of them. In a repository
with those scripts, keep the labels and the headings marked *(read by scripts)* as written here:

| English | Português |
|---|---|
| epic (label) | `épico` *(read by scripts)* |
| phase-N (label) | `fase-N` *(read by scripts)* |
| type:feature, type:test… (labels) | `tipo:feature`, `tipo:teste`… *(written by `sdd-epic.sh`)* |
| "Phase status" comment | "Estado da fase" *(written by scripts)* |
| ROADMAP heading `` ## Phase N — <name> → `vX.Y.Z` ``, "Decisions" section | `` ## Fase N — <nome> → `vX.Y.Z` ``, `## Decisões` *(read by scripts)* |
| spec section "Changes" / "Unreleased" | `## Mudanças` *(read by scripts)* / `### Não lançado` |
| spec section "Current state (verified)" | `## Estado atual (verificado)` *(read by scripts)* |
| Context / Functional requirements / Non-functional requirements | Contexto / Requisitos funcionais / Requisitos não funcionais |
| Acceptance criteria / Out of scope / Decisions | Critérios de aceite / Fora de escopo / Decisões |
| Given / When / Then | Dado / Quando / Então |
| What to do / Spec(s) / Epic | O que fazer / Spec(s) / Épico |
| What changes / How it was tested / Review notes | O que muda / Como foi testado / Notas para a revisão |
| Stacked PR on #NN | PR empilhado sobre #NN |
| Added / Changed / Fixed / Removed (CHANGELOG) | Adicionado / Alterado / Corrigido / Removido |

## Roles (non-negotiable)

- **Owner** decides: open decisions, scope, priorities, approval, **merge, tags,
  releases, repository settings** and any irreversible or externally visible action.
- **Agent** executes: analysis, specs, epics, tickets, code, tests, PRs, keeping CI green.
- The agent **never**: merges (unless the owner explicitly delegates a merge round:
  it covers that round only, follows the epic's order and requires green CI on each head);
  force-pushes someone else's branch; rewrites published history; works outside its ticket's
  branches; creates a tag or release (unless the owner asks for it, e.g. through a release workflow).

## Discovery phase (repository new to the agent)

Goal: understand the real state before changing any line of code.

1. Read **all** the code. Build it. Run **every** documented command and the relevant edge
   cases (invalid input, existing file, closed stdin, no TTY, `--help`).
2. Check every README promise by running it (installation, examples, flags). Run
   `git tag --sort=-v:refname | head` too: with published tags, the ROADMAP continues from the
   last one (after `v0.10.2`, Phase 1 is `v0.11.0`), never restarting at `v0.1.0`.
3. Record everything in `specs/ANALYSIS.md` (template in templates.md):
   executive summary; "what was verified" table (command → result); findings
   **critical / high / medium / low**, each with evidence (`file:line` or command
   output) and the spec that resolves it; strengths to keep; README assessment;
   improvements prioritized by phase; **open decisions D1..Dn**, each with options and
   a recommendation.
4. Always separate what was **verified** (executed) from what is **inferred** (code reading).
5. Write `specs/constitution.md` (5–10 verifiable principles), one spec per area in
   `specs/NNN-name/spec.md`, `specs/README.md` (flow, conventions, index with status) and
   `specs/ROADMAP.md` (phases → versions, tasks `T1..Tn` citing the IDs they close).
   Default phases: **Phase 0** decisions → **Phase 1** "actually works" `v0.1.0` →
   **Phase 2** "reliable" `v0.2.0` → **Phase 3** "professional" `v1.0.0`.
6. Give the owner a **short** list of the main findings (critical first) and the
   decisions D1..Dn with a recommendation, pointing to the files.

**⏸ STOP.** No implementation before the owner answers D1..Dn. Record the answers
in the "Decisions" section of the affected specs and in ANALYSIS.md; specs move from `Draft` to `Approved`.

## Implementation by phase

1. **Epic** per phase (labels epic and `phase-N`; see [Language](#language)), with the suggested review order.
2. **Ticket** per task, as a **native sub-issue** of the epic. Create the issue **first**
   (this creates missing labels) and **then** attach it as a sub-issue — creating it with the
   parent already set fails if any label does not exist yet.
   - Body: **Context / What to do / Acceptance criteria / Spec(s) / Epic**
     (+ optional "Decision for the review").
   - Labels: `phase-N`, `type:feature|docs|ci|test|chore` or `bug` (GitHub's default label), `P1|P2|P3`.
3. **Branch** per ticket: `<type>/<issue-number>-<short-description>`, from `main` or
   stacked on the previous phase's branch not merged yet.
4. **Tests from the acceptance criteria** first; then the code. Each `AC-*` becomes at least
   one test. Generated outputs → golden files.
5. **Commits** in Conventional Commits, in English, imperative (`feat: add --watch mode`);
   a breaking change uses `!` and explains it in the body. Include the attribution lines the tool requires.
6. **Before each push:** run the full local check (e.g. `make ci`) and push only green;
   do the **mutation check** of each new test; reread the diff adversarially
   (scope, forgotten files, secrets, generated outputs, README × specs × code).
7. **PR per ticket.** Title in Conventional Commits. The description starts with
   `Closes #N · Epic #M · Spec NNN` (`Spec —` if none), with **What changes**,
   **How it was tested** and **Review notes**. A stacked PR states, right after the
   `Closes` line: `> Stacked PR on #NN`. End with the tool's attribution footer.
8. **Do not touch shared status files** in a ticket PR (spec status in
   `specs/README.md` and headers, ROADMAP checkboxes, CHANGELOG entries) — that belongs to the
   closing PR. Exceptions: the ticket that creates the file; and the spec's normative content,
   which **changes together with the code** when the implementation diverges (with a "Decisions" entry).
9. Work discovered midway becomes a **new ticket** in the epic (current or future), never a ride-along.
10. A cohesive restructuring that cannot be split without broken states MAY be **one PR for the
    whole phase**, listing the tasks it closes.
11. Subscribe to the activity of each open PR; you own it until it is green.

## Red CI

1. Read the job's exact log (check runs / job logs). Reproduce locally when possible.
2. Find the **root cause**, fix it, run the local check, push.
3. **Forbidden:** calling it a "flake" without evidence; skipping, disabling or weakening tests or
   gates (coverage, lint); an empty commit to "re-run".
4. If the fix is out of scope but is a **precondition** for the PR to turn green (e.g. a linter
   incompatible with the new language version), it goes in the same PR, explained in "What changes".

## Review phase

When all the phase's PRs are open and **green**:

1. Check **pairwise conflicts** between the phase's branches with a real simulation
   (`git merge-tree --write-tree origin/a origin/b`) — do not guess. Document in the affected
   PRs each conflict and how to resolve it, and the suggested merge order.
2. Tell the owner: list of PRs in the epic's order, what was verified and what was not.

**⏸ STOP.** The owner reviews in the epic's order. The agent answers comments; fixes small
requests; for large or design changes, **proposes** in the comment and implements only after
agreement. Merge is the owner's: phase PRs with others stacked on them → **merge commit**
(the stacked ones are retargeted to `main` without rebase); other PRs MAY use squash.
After a base PR is merged, check that the stacked one still has only its ticket and is green.

## Phase closing

1. With all PRs merged, open the **closing PR**: spec status
   (`specs/README.md` + headers + "Current state" section), ROADMAP checkboxes, `CHANGELOG.md`
   (Keep a Changelog: `[Unreleased]` → `[X.Y.Z] - YYYY-MM-DD`). Checklist in templates.md.
2. **⏸ STOP.** The owner merges and creates the tag `vX.Y.Z` (SemVer). The release workflow runs the
   full CI before publishing binaries and checksums.
3. The epic closes when the release is published.

## Communication

- **Short** updates: done / missing / blocking.
- Findings go to the **repository** (specs, issues, PRs); the chat points to them.
- Always say what was **verified** and what was **not** ("tested on Linux; Windows only in CI").
- If the environment blocks something (proxy, network, permission), say so — never report success
  that did not happen.
- A new message from the owner in the middle of a task: handle it and continue the current task.
- Parallel work requested by the owner: launch agents, but **check their output**
  before using or publishing it.
- Write in the owner's language (see [Language](#language)); commits and code in English.

## Resuming and saving usage

The context can end at any time: compaction, a new session or a usage limit. Work so that the **repository** is enough to continue:

- Create the ticket when starting the task and open the PR as soon as it passes the local check. A new request from the owner that does not fit the current task becomes a ticket **right away**.
- Keep a **"Phase status"** comment on the epic, updated at each milestone: PRs and CI, decisions, expected conflicts, next step.
- **When resuming:** read the epic's most recent status comment, the open PRs and issues and `CLAUDE.md`, and continue from the next step.
- **In the repository:** a `CLAUDE.md` (code map, commands, conventions, pitfalls, language) and a session-start hook that installs the CI tools on the web.
- **Saving** (every rule below cut real sessions' usage; apply them from the first message):
  - read excerpts (`sed -n`, `grep -n`) and do not reread, not even after editing a file;
  - batch independent reads and checks in one command;
  - send long output to a file and show only the exit code and the last lines:
    `make ci > /tmp/ci.log 2>&1; echo "exit $?"; tail -n 3 /tmp/ci.log`;
  - validate with a single command (`make ci`), once, before the push;
  - follow CI with `sh scripts/sdd-ci.sh '#PR'` (one line per check, the end of the failing log only),
    not with PR event subscriptions: if the session subscribes to a PR on its own, unsubscribe right away;
  - do not wait for or follow PR events (event-driven engine, spec 015): open the PR (`sh scripts/sdd-pr.sh --no-wait`), save the checkpoint and end the reply; the engine's workflows update the checkpoint on merge, summarize red CI in a PR comment, bring `main` into PRs, open the closing PR and dispatch the release, and you come back only for judgment (code, root cause, a real conflict, review); if you must wait in a shell, run `sh scripts/sdd-wait.sh pr-merged|ci|issue-closed TARGET` in the background, never a hand-written loop;
  - make mechanical edits with a script that fails when the text is missing (e.g. a Python
    `assert old in s` before `replace`), instead of reading the file to edit it;
  - mutation check without rereading: copy the file aside, break it, run the test, copy it back;
  - check tools and network access before starting (a proxy may block a download); try the
    system package manager before giving up;
  - ask GitHub tools for the fields you need (`fields`, `minimal_output`, `perPage`); avoid calls
    whose answer is the whole issue or repository;
  - restart a merged branch from `main` in one command (`git fetch origin main && git checkout -B <branch> origin/main`);
  - subagents only for broad read-only searches, on a small model;
  - "Phase status" only at milestones, a few lines; chat in three lines (done, missing, blocking), details in the PRs.

## Scripts that save steps (spec 009)

When the repository has the template's scripts, call them instead of doing the steps by hand. The output is short: one line per result.

| Step | Script |
|---|---|
| Wait for CI and read only the failures | `sh scripts/sdd-ci.sh [SHA\|#PR\|branch]` |
| Wait with no LLM for a PR merge, CI or an issue to close (background; one line, exit code 0 held, 1 failed, 2 timed out) | `sh scripts/sdd-wait.sh pr-merged\|ci\|issue-closed TARGET` |
| Ship the ticket branch (merge main, `make ci`, push, open/reuse the PR, wait for CI, checkpoint) | `sh scripts/sdd-pr.sh [--spec NNN] [--dry-run]` |
| Check and fix the local environment so `make ci` behaves like CI (even when the session-start hook did not run) | `sh scripts/sdd-doctor.sh [--check]` |
| Prepare the version's closing PR (version computed by go-release-manager, X.Y.Z only forces it; CHANGELOG draft from merged PRs, version bumps from `.sdd-release`); after the merge, tag it | `sh scripts/sdd-release.sh X.Y.Z [--dry-run]`, then `sh scripts/sdd-release.sh --tag X.Y.Z` |
| Record the owner's decisions | `sh scripts/sdd-mark.sh decide D1=a D2=b` |
| Closing status files | `sh scripts/sdd-mark.sh close vX.Y.Z` |
| "Phase status" comment | `sh scripts/sdd-phase-status.sh [--post]` |
| Epic and sub-issues of a phase | `sh scripts/sdd-epic.sh [--dry-run] N` |
| Go release before and after the tag | `sh scripts/sdd-release-check.sh pre\|post vX.Y.Z` |

### Event-driven engine (spec 015)

The template's workflows do these steps with no LLM, so the agent does not wait for them (idempotent; the repository variable `SDD_ENGINE=off` turns them off; "Allow GitHub Actions to create and approve pull requests" must be enabled for the closing PR):

| Event | Workflow | What it does |
|---|---|---|
| Ticket PR merged | `sdd-on-merge.yml` | Updates the epic checkpoint (done: the PR; next: the lowest open sub-issue) |
| CI of a PR finishes | `sdd-ci-summary.yml` | One comment: a line per check and the tail of failed logs |
| `main` changes | `sdd-update-prs.yml` | Merges `main` into open PRs behind it; warns once on real conflicts |
| Last epic ticket closed | `sdd-on-phase-done.yml` | Opens the closing PR (`sdd-release.sh`) |
| Closing PR merged | `sdd-on-release-merge.yml` | Dispatches the release once (`sdd-release.sh --tag`) |

## GitHub operations used

Works with the GitHub MCP or with `gh`:

| Operation | GitHub MCP | `gh` |
|---|---|---|
| Create issue / epic | `issue_write` (create) | `gh issue create` |
| Attach sub-issue | `sub_issue_write` (add; its answer is the whole parent issue, so prefer `gh` when available) | `gh api repos/O/R/issues/EPIC/sub_issues -F sub_issue_id=<id> --jq .number` |
| Create PR | `create_pull_request` (look for the repo's template first) | `gh pr create` |
| Follow PR | `subscribe_pr_activity` | `gh pr checks --watch` |
| Read CI | `pull_request_read` / `get_check_run` / `get_job_logs` | `gh run view --log-failed` |
| Reply to review | `add_reply_to_pull_request_comment` | `gh pr comment` |

Note that `sub_issue_id` is the issue's numeric **id**, not its number `#N`.

## Illustrative example

In cv-craft (a Go CLI, Portuguese-speaking owner): discovery found broken accents in the PDF and
dropped content; D1..D6 were answered; Phase 1 became a single PR (cohesive restructuring); Phase 2
became tickets `chore/4-go-1.26`, `docs/5-contributing`, `ci/6-goreleaser`… in stacked PRs,
each with a green `make ci` before the push. Specs, issues and PRs were written in Portuguese;
commits in English.
