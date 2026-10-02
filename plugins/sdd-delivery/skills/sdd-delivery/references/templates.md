# Templates

Skeletons to copy and fill in. Replace `<…>`. **Write in the owner's language**: translate the headings
and text, keep the machine-read tokens (`Status:` values, IDs, `ADDED`/`MODIFIED`/`REMOVED`,
`[Unreleased]`, `Closes #N`). In repositories that use the template scripts, use the Portuguese headings
marked *(read by scripts)* in the "Language" section of SKILL.md.

## Contents

- [specs/ANALYSIS.md](#specsanalysismd)
- [specs/constitution.md](#specsconstitutionmd)
- [specs/README.md](#specsreadmemd)
- [specs/NNN-name/spec.md](#specsnnn-namespecmd)
- [Acceptance criteria formats](#acceptance-criteria-formats)
- [specs/ROADMAP.md](#specsroadmapmd)
- [Epic](#epic)
- [Ticket](#ticket)
- [Ticket pull request](#ticket-pull-request)
- [Conflicts comment (review phase)](#conflicts-comment-review-phase)
- [Phase closing PR](#phase-closing-pr)
- [CHANGELOG.md](#changelogmd)
- [Message to the owner at the end of discovery](#message-to-the-owner-at-the-end-of-discovery)

## specs/ANALYSIS.md

````markdown
# Analysis and verification of <project>

> Date: <YYYY-MM-DD> · Base: commit `<sha>` (branch `<main>`)
> Method: full code reading + build, lint and real execution of every command and edge case.

## 1. Executive summary

<2–3 sentences: is it ready for real use? Why?>

- <most serious problem, in user language>
- <…>

<One sentence about the base: rewrite, or fix/consolidate/test?>

## 2. What was verified

| Check | Result |
|---|---|
| `<build command>` | ✅ OK |
| `<documented command>` | ⚠️ <works with a caveat> |
| `<edge case: closed stdin / existing file / invalid input>` | ❌ <what happened> |
| Automated tests | <do they exist? do they pass?> |

## 3. Findings by severity

### Critical (block "actually works")

| # | Finding | Evidence | Spec |
|---|---|---|---|
| C1 | <what> | `<file:line>` or command output | <NNN> |

### High

| # | Finding | Evidence | Spec |
|---|---|---|---|
| H1 | | | |

### Medium

| # | Finding | Spec |
|---|---|---|
| M1 | | |

### Low / hygiene

- <…>

## 4. Strengths (keep)

- <…>

## 5. README assessment

<Each promise verified: does it work? what is missing? does it promise what it does not deliver?>

## 6. Recommended improvements (prioritized)

**Phase 1 — Actually works (P0)**
1. <…> (C1)

**Phase 2 — Reliable (P1)**
1. <…>

**Phase 3 — Professional (P2)**
1. <…>

## 7. Open decisions

| # | Question | Options | Recommendation |
|---|---|---|---|
| D1 | <…> | <a / b / c> | <b, because …> |
````

After the answers, rename section 7 to "Decisions", replace the
"Recommendation" column with "Decision" and record the approval date.

## specs/constitution.md

```markdown
# <project> constitution

Principles every spec and every PR must respect. Changing them requires a spec of its own.

1. **<Short name>.** <Verifiable rule. E.g. "Single source of truth: every format derives from the same model and drops no field silently.">
2. **Fail loud, fail early.** <E.g. errors reported all at once, with the field path and exit code ≠ 0.>
3. **Determinism.** <Same input + same version ⇒ same output.>
4. **Scriptable.** <Works without a TTY; flags for everything; documented exit codes.>
5. **One implementation per behavior.** <No logic duplicated across interfaces.>
6. **Tested.** Every acceptance criterion has an automated test; CI blocks a red merge.
7. **Minimal dependencies.** <Only those justified in a spec.>
8. **Language.** <Specs, issues and PRs in <owner's language>; code and commits in English.>
```

5–10 principles, each verifiable by a test or a review.

## specs/README.md

````markdown
# Specs — <project>

This directory organizes development with **Spec Driven Development (SDD)**: no behavior change
enters the code without a spec that describes it and acceptance criteria that verify it.

## Flow

```text
spec.md (WHAT / WHY) → review → tests from the AC → implementation → status: Done
```

1. **Specify** — `FR-*`, `NFR-*` and `AC-*` (Given/When/Then or EARS).
2. **Resolve decisions** — before implementing.
3. **Test first** — each `AC-*` becomes at least one test.
4. **Implement** — the PR references the IDs (`003 FR-1, AC-2`).
5. **Close** — status updated in the phase's closing PR.

Conventions: MUST/SHOULD/MAY (RFC 2119). Priorities: **P0** (blocks real use), **P1** (reliability), **P2** (polish).

## Documents

| Document | Content |
|---|---|
| [ANALYSIS.md](ANALYSIS.md) | Verification report |
| [constitution.md](constitution.md) | Non-negotiable principles |
| [ROADMAP.md](ROADMAP.md) | Tasks by phase |

## Specs

| ID | Spec | Priority | Status |
|---|---|---|---|
| 001 | [<Area>](001-<area>/spec.md) | P0 | Draft |

Possible statuses: `Draft` → `Approved` → `In Progress` → `Done`.
````

## specs/NNN-name/spec.md

```markdown
# NNN — <Area>

- **Priority:** P0 | P1 | P2
- **Status:** Draft
- **Affected code:** `<paths>`
- **Resolves:** <C1, H3, M2 — IDs from ANALYSIS.md, or #issue>

## Context

<Why this area matters and what is wrong today.>

## Current state (verified)

- <fact observed by running something, with the command or file>

## Functional requirements

- **FR-1** <The system> MUST <verifiable behavior>.
- **FR-2** <…> SHOULD <…>.

## Non-functional requirements

- **NFR-1** <performance, size, portability, accessibility — with a number>.

## Acceptance criteria

- **AC-1** Given <context>, when <action>, then <observable result>.
- **AC-2** Golden test: <sample output> equals `<testdata/file>`.
- **AC-3** WHEN <trigger>, THE SYSTEM SHALL <observable response>.   ← EARS

## Out of scope

- <what will not be done, and why>

## Decisions

- D<n> — <the owner's decision and why>.
- <Revised during implementation: what changed from the original requirement and why.>

## Changes

### Unreleased

- ADDED FR-3 — <what now exists>
- MODIFIED AC-2 — <what changed in the behavior>
- REMOVED FR-1 — <what was removed and why>
```

The "Changes" section is the spec's per-version delta, in the OpenSpec style: each ticket PR adds
its lines under `### Unreleased`, and the closing PR renames it to the version and generates the CHANGELOG
from it. `ADDED` and `MODIFIED` point to IDs that exist in the spec (`sdd-check` checks it). With the
template scripts, the headings are `## Mudanças` and `### Não lançado`, and "Current state (verified)"
is `## Estado atual (verificado)`.

## Acceptance criteria formats

Both formats are accepted; choose by the kind of requirement and keep one per spec when possible.

| Format | Use for | Example |
|---|---|---|
| Given/When/Then | User-visible behavior, scenarios with context | Given a YAML without `name`, when `build` runs, then it exits with code 2 and mentions `name` |
| EARS | System requirements: events, states, errors, options | WHEN the output file already exists, THE SYSTEM SHALL ask before overwriting |

EARS patterns (Easy Approach to Requirements Syntax); in Portuguese, `O SISTEMA DEVE`, `QUANDO`,
`ENQUANTO`, `SE … ENTÃO`, `ONDE`:

| Pattern | Form |
|---|---|
| Ubiquitous | `THE SYSTEM SHALL <response>` |
| Event | `WHEN <trigger>, THE SYSTEM SHALL <response>` |
| State | `WHILE <state>, THE SYSTEM SHALL <response>` |
| Unwanted | `IF <unwanted condition>, THEN THE SYSTEM SHALL <response>` |
| Optional | `WHERE <feature present>, THE SYSTEM SHALL <response>` |

In both, the result must be **observable** by an automated test.

## specs/ROADMAP.md

```markdown
# Roadmap and tasks

Each task references the spec and the acceptance criteria it closes. Suggested order = list order.

## Phase 0 — Decisions (before coding)

- [ ] Answer D1–Dn in [ANALYSIS.md §7](ANALYSIS.md#7-open-decisions) and move the specs to `Approved`.

## Phase 1 — Actually works (P0) → `v0.1.0`

- [ ] **T1** <task> — 007 FR-1..5
- [ ] **T2** <task> — 003 FR-1, AC-1/2

## Phase 2 — Reliable (P1) → `v0.2.0`

- [ ] **Tn** <task> — <IDs>

## Phase 3 — Professional (P2) → `v1.0.0`

- [ ] **Tn** <task> — <IDs>
```

In a repository with published tags, number the phases from the last tag (`git tag --sort=-v:refname`),
not from `v0.1.0`. A task brought forward to an earlier phase: mark it `*(brought forward)*`. An abandoned task: strike it (`~~…~~`) and explain.
The phase heading keeps the `` → `vX.Y.Z` `` form: the scripts and `/sdd-release` read it (with the template
scripts, `` ## Fase N — <nome> → `vX.Y.Z` ``).

## Epic

Title: `Phase N — <name> (vX.Y.Z)` · Labels: epic, `phase-N` (with the template scripts: `épico`, `fase-N`)

```markdown
## Goal

<What this phase delivers, in one or two sentences.> Version: `vX.Y.Z`.

## Tasks (suggested review and merge order)

The tasks are this epic's sub-issues. Suggested order:

1. #<n> <title> — <why first: base for others, unblocks CI…>
2. #<n> <title>

## Definition of done

- All PRs merged with green CI.
- Closing PR merged (specs, ROADMAP, CHANGELOG).
- Tag `vX.Y.Z` published by the owner.

Specs: <NNN, NNN> · Roadmap: <T11–T16>

---
_Generated by [Claude Code](https://claude.ai/code)_
```

## Ticket

Title in Conventional Commits (`feat: add --format all`) · Labels: `phase-N`, `type:<…>`, `P<1-3>`

```markdown
## Context

<The problem and for whom. Source finding: C2 in specs/ANALYSIS.md.>

## What to do

- <item>
- <item>

## Acceptance criteria

- [ ] Given <…>, when <…>, then <…> (automated test).
- [ ] <check in CI / documentation updated>.

## Decision for the review (optional)

<A choice embedded in the ticket that the owner needs to see, with the recommendation.>

**Spec(s):** <NNN FR-x, AC-y> · **Epic:** #<M>

---
_Generated by [Claude Code](https://claude.ai/code)_
```

Call order: create the issue (with labels) → get its `id` → attach it as a sub-issue of the epic.

## Ticket pull request

Branch: `<type>/<number>-<description>` · Title in Conventional Commits.

```markdown
Closes #<N> · Epic #<M> · Spec <NNN>

> Stacked PR on #<NN>.   ← only if stacked

## What changes

<Problem and solution from the user's point of view. Out-of-scope changes that were a
precondition for green CI, explained here.>

## Specs and acceptance criteria

<003 FR-8, AC-6. Spec updated in this PR? Why?>

## How it was tested

- <new/changed tests; each one went through the mutation check: what was broken>
- `<make ci>` green locally on <OS>.
- <what was NOT verified locally (e.g. Windows only in CI)>

## Review notes

- <decisions taken, alternatives, expected conflicts with other PRs of the phase>

## Checklist

- [ ] Full local check passes
- [ ] Tests cover the acceptance criteria
- [ ] Golden files rewritten **and reviewed** (if the outputs changed)
- [ ] Spec updated (if the behavior changed)
- [ ] README/docs updated (if the interface changed)
- [ ] Breaking change flagged with `!` and explained
- [ ] Does not change spec status, ROADMAP checkboxes or CHANGELOG entries

🤖 Generated with [Claude Code](https://claude.com/claude-code)
<session link>
```

If the repository has a `pull_request_template.md`, use it and keep the normative first line.

## Conflicts comment (review phase)

```markdown
### Conflicts with other PRs of the phase

Simulated with `git merge-tree --write-tree origin/<this> origin/<other>` on <date> (commits `<sha>`/`<sha>`):

| With | Files | Resolution |
|---|---|---|
| #<n> | `Makefile`, `.github/workflows/ci.yml` | keep both targets; order: … |
| #<n> | — | no conflict |

Suggested merge order: #a → #b → #c.

---
_Generated by [Claude Code](https://claude.ai/code)_
```

## Phase closing PR

Branch: `chore/<epic>-close-phase-N` · Title: `chore: close phase N (vX.Y.Z)`

```markdown
Closes #<epic> · Epic #<epic> · Spec —

## What changes

Closes Phase N → `vX.Y.Z`.

## Closing checklist

- [ ] All of the phase's PRs merged (list: #a, #b, #c)
- [ ] `specs/README.md`: status updated for each spec touched
- [ ] Each spec's `Status:` header equal to the index
- [ ] "Current state" section of the specs that have one, reflecting what was delivered
- [ ] `specs/ROADMAP.md`: checkboxes of the finished tasks; brought-forward/struck ones explained
- [ ] Specs' "Changes" section: `### Unreleased` → `### vX.Y.Z`
- [ ] `CHANGELOG.md` generated from the "Changes" sections: `[Unreleased]` → `[X.Y.Z] - YYYY-MM-DD`; new empty `[Unreleased]`; comparison links
- [ ] `specs/ANALYSIS.md`: resolved findings marked (if the project keeps that record)
- [ ] Full local check green; CI green
- [ ] Owner's next step: after the merge, `git tag vX.Y.Z && git push origin vX.Y.Z` (or the release workflow)

🤖 Generated with [Claude Code](https://claude.com/claude-code)
```

## CHANGELOG.md

Section names in the owner's language (Portuguese: Adicionado, Alterado, Corrigido, Removido).

```markdown
# Changelog

Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) · Versioning: [SemVer](https://semver.org/).

## [Unreleased]

## [0.1.0] - YYYY-MM-DD

### Added

- <…>

### Changed

### Fixed

### Removed

[Unreleased]: https://github.com/<o>/<r>/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/<o>/<r>/releases/tag/v0.1.0
```

## Message to the owner at the end of discovery

```markdown
Analysis finished and recorded in `specs/ANALYSIS.md` (commit `<sha>`).

**Main findings**
- 🔴 C1 <…>
- 🔴 C2 <…>
- 🟠 H1 <…>

**Decisions I need from you** (my recommendation in parentheses)
- D1 <question> (<recommendation>)
- D2 <…>

Verified: <build, commands X/Y, edge cases>. Not verified: <Windows, release>.
No code was changed; I am waiting for the decisions to start Phase 1.
```
