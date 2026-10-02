---
description: Takes the next ticket of the open epic and carries it to a PR with green CI
argument-hint: "[ticket number]"
---

Use the `sdd-delivery` skill to work on the **next ticket**. Specific ticket, if given: $ARGUMENTS

1. Read the most recent "Phase status" comment of the open epic and pick the next ticket in the epic's order.
2. Create the branch `<type>/<number>-<description>`, write the tests from the acceptance criteria (citing `NNN AC-n`) and then the code. Commits and code in English.
3. Run the full local check (`make ci`), do the mutation check of the new tests and reread the diff.
4. Open the PR in the owner's language (`Closes #N · Epic #M · Spec NNN`), follow CI with `sh scripts/sdd-ci.sh "#PR"` (one line per check and only the end of the failing logs) until it is green, and update the "Phase status".
