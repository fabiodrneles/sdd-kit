# Case study: cv-craft with the SDD process

[Português](case-study.md)

[cv-craft](https://github.com/fabiodrneles/cv-craft) is a Go CLI that turns a YAML file into a résumé in PDF, Markdown and plain text. It was the first project driven by the process that became sdd-kit: **the owner decides, the agent executes**. Every number below links to its source.

## Starting point: a prototype

The code was written in October 2025 and sat as a prototype. The discovery phase read the code, ran what the README promised and recorded everything with evidence in [`specs/ANALYSIS.md`](https://github.com/fabiodrneles/cv-craft/blob/main/specs/ANALYSIS.md):

| Severity | Findings | Examples |
|---|---|---|
| Critical | 5 | broken accents in the PDF; experience details dropped from Markdown and text; interactive mode looping forever on closed input; a README `go install` that did not work |
| High | 8 | missing validation, content silently discarded, no tests |
| Medium | 9 | output consistency, documentation, hygiene |

Each finding points to `file:line` or to the command output that showed it, plus the spec that fixes it. No code changed before the owner answered the **6 open decisions (D1–D6)**.

## What the process produced

| Item | cv-craft |
|---|---|
| Specs | 10 ([index](https://github.com/fabiodrneles/cv-craft/blob/main/specs/README.md)), all `Done` |
| Acceptance criteria | 71, each with an automated test |
| Test and benchmark functions | 76, plus 9 golden files for generated output |
| Merged PRs | 24 ([list](https://github.com/fabiodrneles/cv-craft/pulls?q=is%3Apr+is%3Amerged)) |
| CI | Linux, macOS and Windows on the minimum and stable Go versions, with lint, `-race`, a coverage floor, a smoke test and documentation checks |
| Releases | [`v0.2.0` and `v1.0.0`](https://github.com/fabiodrneles/cv-craft/releases), binaries and checksums built only after full CI |

## How the process handled problems

- **No "works on my machine":** the broken install (C5) surfaced only because discovery requires **running** every README command; README command blocks now run in CI.
- **Red CI fixed at the root:** a failure is fixed in the PR that caused it, never by skipping a test or loosening a gate.
- **Small, reviewable PRs:** one ticket, one branch, one PR, in epic order.
- **Cheap resumption:** each phase's state lives in an epic comment and `CLAUDE.md` maps the code, so new sessions resumed without rereading the repository.

## The same process applied to the kit itself

sdd-kit was built the same way ([phase 1 epic](https://github.com/fabiodrneles/sdd-kit/issues/1), [phase 2 epic](https://github.com/fabiodrneles/sdd-kit/issues/21)): 6 specs and 24 acceptance criteria, each cited by a test (checked in CI by `sdd-check --strict`); 18 merged PRs up to [`v0.1.0`](https://github.com/fabiodrneles/sdd-kit/releases/tag/v0.1.0), with CI on Linux, macOS and Windows and an e2e that adopts the template in Go, Node, Java and Python and runs the generated `make ci`. Real problems were caught before merge and recorded in the PRs.

## Takeaways

1. **Evidence-based discovery pays off:** cv-craft's critical findings were invisible to anyone only reading the code.
2. **Explicit decisions prevent rework:** the agent stops, the owner decides, and the decision is recorded in the spec.
3. **AC → test traceability** turns "looks done" into "proven".

To apply it to your repository, see the [README](../README.en.md#starting-a-new-repository).
