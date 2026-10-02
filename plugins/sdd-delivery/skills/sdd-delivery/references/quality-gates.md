# Quality gates, tests and lessons learned

## Contents

- [CI gates](#ci-gates)
- [Equivalent local target](#equivalent-local-target)
- [Tests from the acceptance criteria](#tests-from-the-acceptance-criteria)
- [Mutation check](#mutation-check)
- [Documentation × code consistency tests](#documentation--code-consistency-tests)
- [Adversarial reread of the diff](#adversarial-reread-of-the-diff)
- [Release](#release)
- [Lessons learned (they became checks)](#lessons-learned-they-became-checks)

## CI gates

Run them on every PR and every push to `main`. Adapt the tools to the language; keep the gates.

| Gate | What it guarantees | Example (Go) |
|---|---|---|
| Clean dependencies | manifest without a diff after normalizing | `go mod tidy && git diff --exit-code go.mod go.sum` |
| Formatting + lint | style and static errors | `go vet`, `golangci-lint` |
| Tests on every supported OS | real portability | `ubuntu/macos/windows` matrix, `fail-fast: false` |
| Minimum and latest stable version | does not break whoever uses the declared minimum | `go: [minimum, stable]` matrix |
| Race detector | concurrency | `go test -race` (where supported) |
| Coverage gate | tests do not regress | script that fails below N% (e.g. 80% of internal code) |
| Smoke test of the real artifact | the binary/package works the way the user uses it | script that builds and runs commands checking exit codes and outputs |
| Cross-build | every release target builds | `GOOS × GOARCH` matrix |
| Vulnerabilities | dependencies without known CVEs | `govulncheck`, `npm audit`, `pip-audit` |
| Release rehearsal | the release works without publishing | `goreleaser release --snapshot` + script that checks the artifacts |
| Docs: markdownlint | valid Markdown (generated Markdown outputs included) | `markdownlint-cli2` |
| Docs: links | no broken link | `lychee` |
| Docs: commands | every README example command runs with exit 0 | script that extracts `bash` blocks and runs them |

Workflow good practices: `permissions: contents: read`; `concurrency` cancelling old runs
of the same ref; tools required in CI even when optional locally
(e.g. a `REQUIRE_X=1` variable makes the test fail instead of skipping); pin the tools' versions.

## Equivalent local target

A target (`make ci`, `npm run ci`, `just ci`, `tox`) runs the feasible subset of CI **with the
same commands and limits** (same coverage script, same threshold, same smoke). It runs
before **every** push. Note at the top of the file that it mirrors the workflow.

Typical smoke test (`scripts/smoke.sh`): builds the real artifact in a temporary directory,
defines `check <description> <expected exit> <command…>` and `contains <file> <text>`, runs with
`</dev/null` (no TTY), covers the happy path, each documented exit code, overwriting,
invalid input and the output of each format; accumulates failures and ends with exit ≠ 0 if there is any.

## Tests from the acceptance criteria

- Each `AC-*` becomes at least one automated test; cite the ID in the test's name or comment.
- **Golden files** for generated outputs: exact comparison with `testdata/`; a flag (`-update`)
  or target (`make golden`) rewrites them after an **intentional** change; the goldens' diff is reviewed in the PR.
- Parity test when several outputs derive from the same model (no field dropped).
- CLI tests with exit codes and closed stdin.
- Determinism: injectable dates and randomness (e.g. `SOURCE_DATE_EPOCH`).

## Mutation check

For **each** new test, before the push:

1. Temporarily break the covered code (invert a condition, remove a line, change the value).
2. Run only that test: it **has** to fail, for the expected reason.
3. Undo the break (`git diff` must show only what it should).
4. Record what was broken under "How it was tested".

This has already exposed a page-break test that tested nothing and a `Makefile`
target test that passed with the target broken.

## Documentation × code consistency tests

When the documentation and the code must agree, write a test that **enforces** it:

- Every model/schema field appears in the schema documentation (reflection on the struct × doc).
- Every `make <x>` target cited in README/CONTRIBUTING exists in the `Makefile`.
- Every README example command runs in CI (`scripts/doc-commands.sh`).
- Every example in `examples/` is validated by the program itself.
- `CONTRIBUTING.md` does not contradict the process spec (review in the PR that touches either).

## Adversarial reread of the diff

Before the push, read `git diff origin/<base>...HEAD` as a hostile reviewer:

- Scope: only the ticket? Anything that should be another ticket?
- Forgotten: a new file without `git add`? Spec, README, CHANGELOG (if it is the PR that creates it)?
- Shared status files changed in a ticket PR? (not allowed)
- Secrets, local paths, generated outputs, binaries.
- Claims in the PR description: was each one verified? (e.g. conflicts → simulated with
  `git merge-tree`, not assumed.)
- Do README × specs × code say the same thing?
- Are the specs, issues and PR in the owner's language, and the commits in English?

## Release

- Triggered by a `v*` tag created **by the owner** (or by a release workflow the owner runs or asks for).
- The release workflow **calls the full CI** (`uses: ./.github/workflows/ci.yml`) and only
  then publishes binaries + checksums (e.g. GoReleaser), with `contents: write` only in that job.
- An artifact verification script (count, targets, checksums, included files,
  embedded version) also runs in CI's release rehearsal.
- Changelog grouped by Conventional Commits prefix.

## Lessons learned (they became checks)

| Situation | Check |
|---|---|
| A glob at the root of `.gitignore` (e.g. `cv-craft*`) hid tool configuration files | `git check-ignore -v <file>` for each new config file; `git status` after creating it. `.gitignore` **does not accept end-of-line comments** — comments only on their own line. |
| Conflicts between PRs were described from memory | Simulate: `git merge-tree --write-tree origin/a origin/b`; write only what the simulation showed. |
| The environment's proxy blocked a host (tool download, link check) | Say it was not verified and why; never report success. Let CI verify and check the result. |
| A PDF text extraction library truncated characters above U+00FF, hiding exactly the Unicode bugs | Validate the verification tool on hard cases (accents, non-Latin-1) before trusting it; prefer the reference tool (e.g. `pdftotext`). |
| A Windows runner had one tool of the package but not the other (`pdftotext` without `pdfinfo`) | Detect each external dependency separately; in CI, require them explicitly. |
| Bumping the language version broke the linter built with the previous version | Update the tool in the same PR (precondition for green), explaining it in "What changes". |
| Base of a stacked PR merged and deleted | Check that the PR was retargeted to `main`, that the diff has only the ticket and that CI is green. |
