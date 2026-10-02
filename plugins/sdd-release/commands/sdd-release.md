---
description: Computes the next version from the commits (go-release-manager) and proposes closing the phase with it
argument-hint: "[pre-release channel, e.g. rc]"
---

Compute this repository's **next version** and propose closing the phase with it. Do not create tags: the tag is the owner's. Talk to the owner in their language.

1. Check that `go-release-manager` is installed (`go-release-manager --version`). If it is not, explain the two ways to install it to the owner and **stop**:
   - `go install github.com/fabiodrneles/go-release-manager@latest`
   - a binary from <https://github.com/fabiodrneles/go-release-manager/releases>
2. On an up-to-date `main`, run:

   ```text
   go-release-manager create --dry-run --output json
   ```

   With a pre-release channel in $ARGUMENTS, add `--pre-release $ARGUMENTS`.
3. Show the owner, in a few lines:
   - the previous version (`previous`), the next one (`next`) and the increment (`increment`);
   - the commits that set the increment (`git log <previous>..HEAD --oneline`, only `feat`, `fix` and breaking ones).
4. Compare it with the phase's version in `specs/ROADMAP.md` (the open phase, `` → `vX.Y.Z` ``):
   - **equal:** go on;
   - **different, or `next` empty:** explain why to the owner. For instance, in a test project `test:` commits do not produce a release under the default rule. Propose one of the ways out and **stop** until they choose:
     - follow the ROADMAP with `--release-as vX.Y.Z` (requires go-release-manager `v1.1.0` or newer);
     - follow the commits and update the ROADMAP;
     - a `.go-releaserc.yml` that makes the project's types produce a release (e.g. `test: minor`).
5. Propose the closing with `/sdd-close <version>` (`sdd-delivery` skill). After the closing PR is merged, the tag is created:
   - by the owner, or by the agent, if the owner delegated it;
   - from the terminal (`go-release-manager create [--release-as vX.Y.Z] [--ref <commit>]`) or by this plugin's `release-tag.yml` workflow (in `templates/`), which accepts `release-as` and `ref`.
