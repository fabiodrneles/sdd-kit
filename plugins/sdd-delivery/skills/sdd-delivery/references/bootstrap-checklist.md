# Checklist for applying the process to a new repository

Follow it in order. Items marked **(owner)** are the owner's settings or decisions: the agent suggests,
it does not execute. The `⏸` are stop points.

## 0. Before starting

- [ ] Find the owner's language (see "Language" in SKILL.md): `CLAUDE.md`/`AGENTS.md`, then the
      existing docs; if none, ask once and record it in `CLAUDE.md`.
- [ ] Confirm with the owner: main branch, supported OSes and versions,
      whether the release will be automated.
- [ ] Run `git tag --sort=-v:refname | head`: the phases continue from the last published version.
- [ ] Check what already exists (`specs/`, `CONTRIBUTING.md`, `.github/`, CI, labels, epics)
      to **extend** it, not duplicate it.

## 1. Discovery

- [ ] Read all the code; build it; run the existing lint and tests.
- [ ] Run each documented command and the edge cases (invalid input, existing
      file, closed stdin / no TTY, `--help`, `--version`).
- [ ] Verify each README promise (installation included).
- [ ] `specs/ANALYSIS.md` with findings by severity + evidence, strengths, README,
      improvements by phase and decisions `D1..Dn` with a recommendation.
- [ ] Short message to the owner with the main findings and the decisions.
- [ ] ⏸ **Wait for the answers.** No code commit before them.

## 2. SDD artifacts

- [ ] `specs/constitution.md` with 5–10 verifiable principles (language included).
- [ ] `specs/README.md` (SDD flow, conventions, spec index with status).
- [ ] One spec per area: `specs/NNN-name/spec.md` (FR/NFR, Given/When/Then AC, Out of
      scope, Decisions with the owner's answers).
- [ ] `specs/ROADMAP.md` with phases (no fixed version: go-release-manager computes it at closing) and tasks linked to `FR`/`AC`.

## 2a. Existing repository: green CI first

- [ ] Adopting the kit in existing code usually turns CI red at once (lint and markdownlint of
      the old code and README). The first PR (adoption or discovery) brings **only the minimal
      fixes** for green CI; everything else becomes a finding in `ANALYSIS.md` and a ticket.
- [ ] Dependabot opens PRs right after the adoption: merge each one (owner) with green CI
      against the current `main`, before the phase's tickets; a major bump that breaks the
      build becomes a ticket.

## 3. GitHub

- [ ] Labels: epic, `phase-0..N`, `type:feature|docs|ci|test|chore` or `bug` (GitHub's default label), `P1..P3`
      (created by the first issue that uses them, or explicitly). With the template scripts:
      `épico`, `fase-N`, `tipo:*`.
- [ ] One epic per phase, with the suggested review order.
- [ ] Tickets: create the issue → attach it as a sub-issue of the epic.

## 4. Quality

- [ ] CI with the gates from [quality-gates.md](quality-gates.md): lint, tests on every OS,
      race, coverage, smoke of the real artifact, cross-build, vulnerabilities, release rehearsal,
      docs (markdownlint, links, commands).
- [ ] Equivalent local target (`make ci` or similar) with the same commands and limits.
- [ ] Golden files for generated outputs and a target to rewrite them.

## 5. Practical guide

- [ ] Issue templates (task, bug) with Context / What to do / Acceptance criteria /
      Spec(s); the bug template asks for version, OS, command, minimal reproduction, expected × actual and reminds
      not to publish personal data.
- [ ] PR template starting with `Closes # · Epic # · Spec`.
- [ ] `CONTRIBUTING.md` summarizing the process in practice (flow, branches, commits, PRs,
      review, closing, environment and commands).
- [ ] `CLAUDE.md` with the code map, commands, conventions (language included) and pitfalls.
- [ ] `CODEOWNERS`.
- [ ] (Optional) the repo's own process spec, adapting [process.md](process.md) and
      fixing the local values (labels, commands, phases).

## 6. Release

- [ ] `CHANGELOG.md` in Keep a Changelog with `[Unreleased]`.
- [ ] Release workflow on `v*` tag that calls the full CI before publishing with checksums.

## 7. Ask the owner

- [ ] **(owner)** `main` protection: required CI + Code Owners review.
- [ ] **(owner)** Merge policy: merge commit for phase PRs with stacked ones; squash allowed for the rest.
- [ ] **(owner)** Actions permissions and secrets needed for the release.
