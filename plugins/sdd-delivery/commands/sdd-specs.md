---
description: Creates or updates the constitution, the specs and the ROADMAP from the analysis and the owner's decisions
argument-hint: "[area or idea]"
---

Use the `sdd-delivery` skill to write the spec base, in the owner's language. Additional context: $ARGUMENTS

1. Confirm that the open decisions in `specs/ANALYSIS.md` were answered; if not, list them and **stop**.
2. Write or update `specs/constitution.md`, one spec per area (`specs/NNN-name/spec.md`, with `FR-*`, `NFR-*` and `AC-*` in Given/When/Then or EARS), `specs/README.md` and `specs/ROADMAP.md` by phases.
3. Open a ticket and a PR for this change, like any other.
