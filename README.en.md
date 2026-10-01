# sdd-kit

[Português](README.md)

A **Spec Driven Development (SDD)** kit for projects driven with Claude Code: the `sdd-delivery` skill, a repository template with per-language CI, and an adoption script. It packages the process that took [cv-craft](https://github.com/fabiodrneles/cv-craft) from prototype to a professional project: **the owner decides, the agent executes**.

```text
discovery → ANALYSIS.md → owner decisions → constitution + specs + ROADMAP
  → one epic per phase → tickets (sub-issues) → one branch and one PR per ticket
  → green CI → owner review → merge → phase closing PR → tag → release
```

> **Status:** under construction. Phase 1 (`v0.1.0`) is tracked in [epic #1](https://github.com/fabiodrneles/sdd-kit/issues/1) and the [ROADMAP](specs/ROADMAP.md).

The specs, issues and pull requests are written in Portuguese; code and commits are in English.

## What is in the kit

| Part | Purpose | Spec |
|---|---|---|
| `sdd-delivery` skill | Teaches the agent the process: analysis, specs, epics, PRs, review, release | [002](specs/002-skill-plugin/spec.md) |
| `template/` | `CLAUDE.md`, session hook, issue and PR templates, `specs/` and CI for Go, Node/TS, Java and Python | [003](specs/003-template/spec.md) |
| Adoption script | Copies the template into a new or existing repository without overwriting anything | [004](specs/004-adoption-script/spec.md) |

## Installing the skill

In Claude Code (from `v0.1.0` on):

```text
/plugin marketplace add fabiodrneles/sdd-kit
/plugin install sdd-delivery@sdd-kit
```

On claude.ai: download `sdd-delivery.zip` from the [latest release](https://github.com/fabiodrneles/sdd-kit/releases) and upload it under *Settings → Capabilities → Skills*.

## Adopting in a repository

Run the script from the root of the target repository (new or existing). Nothing that already exists is overwritten; the report lists what was created and what was skipped.

```text
sh /path/to/sdd-kit/scripts/adopt.sh --lang go .
curl -fsSL https://raw.githubusercontent.com/fabiodrneles/sdd-kit/v0.1.0/scripts/adopt.sh | sh -s -- --lang node .
pwsh -File C:\path\to\sdd-kit\scripts\adopt.ps1 --lang java .
```

Options: `--lang go|node|java|python` (required), `--project`, `--owner`/`--repo` (default: taken from `origin`), `--dry-run`, `--force`.

## License

[MIT](LICENSE)
