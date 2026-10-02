---
description: SDD discovery phase — analyzes and verifies the repository and proposes the open decisions
argument-hint: "[optional focus]"
---

Use the `sdd-delivery` skill and run the **discovery phase** in this repository. Additional focus, if any: $ARGUMENTS

1. Find the owner's language (`CLAUDE.md`/`AGENTS.md`, then the existing docs; if none, ask once) and write the documents in it.
2. Read the code and run what the README promises (installation, commands, tests), plus edge cases.
3. Record in `specs/ANALYSIS.md`: what was verified (command → result), findings by severity with evidence (`file:line` or output) and open decisions D1..Dn, each with options and a recommendation.
4. Separate what was **verified** from what is **inferred**.
5. Give a short summary with the decisions and **stop**: nothing is implemented before the owner answers.
