---
description: Creates the epic of a ROADMAP phase with the tickets as sub-issues
argument-hint: "<phase number>"
---

Use the `sdd-delivery` skill to create the **epic of Phase $ARGUMENTS** of `specs/ROADMAP.md`, in the owner's language.

1. Run `sh scripts/sdd-epic.sh --dry-run $ARGUMENTS` (if it exists) and review the plan. Then run it without `--dry-run`: it creates the epic (labels `épico`, `fase-N`) and one ticket per task as a sub-issue. For a richer ticket body, use `--tickets DIR` with `DIR/Tn.md`.
2. Without the script: create the epic with the goal, suggested review order and definition of done, and one ticket per task (Context, What to do, Acceptance criteria, Spec(s), Epic; labels `phase-N`, `type:*`, `P1-3`). Only then attach each one as a sub-issue.
3. Update the epic's body with the numbers and post the first "Phase status" comment.
