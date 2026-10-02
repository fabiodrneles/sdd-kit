---
description: Posts the "Phase status" comment on the open epic (PRs, CI, decisions, next step)
---

Use the `sdd-delivery` skill to update the **phase status**, in the owner's language.

1. Run `sh scripts/sdd-phase-status.sh` (if it exists): it builds the comment with the ticket → PR → CI table, the pending decisions and the next step. Review it, add the expected conflicts and post it with `--post` (or `--next "text"` to adjust the next step). Without the script, build the same comment by hand.
2. The comment is titled "Phase status — YYYY-MM-DD" ("Estado da fase" in Portuguese).
3. Reply in the chat with a three-line summary: done, missing, blocking.
