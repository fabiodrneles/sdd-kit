---
description: Opens the phase closing PR (spec status, ROADMAP and CHANGELOG) for the owner to create the tag
argument-hint: "<version, e.g. v0.2.0>"
---

Use the `sdd-delivery` skill to **close the phase** of version $ARGUMENTS. Write the PR in the owner's language.

1. Confirm that all of the epic's ticket PRs were merged; if any is missing, list them and **stop**.
2. Open the closing PR. Use `sh scripts/sdd-mark.sh close $ARGUMENTS` (if it exists): it checks the ROADMAP boxes, moves the specs to `Done` or `In Progress` and opens `[X.Y.Z] - YYYY-MM-DD` in the CHANGELOG, with no risk of emptying files. Review the warnings and the CHANGELOG text. Run `sh scripts/sdd-check.sh --strict` if the phase requires full traceability.
3. In Go projects with GoReleaser, run `sh scripts/sdd-release-check.sh pre $ARGUMENTS` before asking for the tag: it simulates the release in a throwaway clone.
4. **Stop**: merge and tag are the owner's. After the tag, `sh scripts/sdd-release-check.sh post $ARGUMENTS` checks the published release.
