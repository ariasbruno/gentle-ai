# FORK.md — this fork of Gentle AI

This repository is a development fork of [Gentleman-Programming/gentle-ai](https://github.com/Gentleman-Programming/gentle-ai).
Its purpose is native support for **Google Antigravity CLI (`agy`)**: agent adapter, managed plugin installation,
asset transpilation from Gentle Shell, workspace-scope support, and Go-side RDD review transport.

Everything in this file is **fork-only**. None of it is proposed upstream, and all of it is excluded from the
upstream pull request described at the bottom.

## Development branch

`agy/ai-v4.2.0_shell-v4.3.0`

Branch naming scheme: `agy/ai-<gentle-ai version>_shell-<gentle-shell version>`. Each version line supersedes the
previous one and is rebased onto the current upstream `main`. The names of both inputs are in the branch name
because the assets are transpiled from one and the binary is built from the other.

## Installing this fork

The `go.mod` module path is upstream's (`github.com/gentleman-programming/gentle-ai/v4`), so a remote
`go install github.com/ariasbruno/gentle-ai/...@<tag>` does **not** work: the module proxy resolves the declared
path, which is upstream. Install this fork by building it locally:

```bash
git clone -b agy/ai-v4.2.0_shell-v4.3.0 https://github.com/ariasbruno/gentle-ai.git
cd gentle-ai
go install ./cmd/gentle-ai
```

Release binaries built by this fork are published on this repository's Releases page. The Homebrew formula
upload is disabled on purpose (`skip_upload: true`), so a fork release never publishes to the upstream tap.

## Fork-only plumbing

These paths exist only so this fork can build, test, and release itself. Their names are the checklist for
cutting the upstream pull request.

| Path | Why it is fork-only |
| :--- | :--- |
| `FORK.md` | This document. |
| `.github/workflows/fork-ci.yml` | CI for the development branch. Keeps the branch name out of upstream's `ci.yml`. |
| `scripts/release-preflight.sh` | Accepts this fork's repository and pins the branch a fork release may be tagged from. |
| `scripts/require-ci-success.sh` | Resolves `fork-ci.yml` instead of `ci.yml` for this fork's development branch. |
| `scripts/verify-release-assets.sh` | Fork release assets are named differently from upstream's. |
| `internal/releaseprovenancecmd/` | Provenance check adapted to the fork's release flow. |
| `internal/releasepolicy/policy.go`, `.goreleaser.yaml` | `skip_upload: true`: this fork does not publish the Homebrew formula. |
| `.gitignore` (minisign entries) | Keeps local signing keys out of the index. The key files themselves live in `.git/info/exclude`. |

Fork CI runs on pushes to the development branch only. Upstream's workflow still runs on `pull_request` and on
pushes to `main`, so opening a pull request from the development branch is covered by both.

## Cutting the upstream pull request

1. Open an issue upstream describing the integration and what this fork already implements.
2. Only if the issue is accepted, cut a clean branch from `upstream/main`.
3. Build it by taking the integration commits and leaving the fork-only commits behind:
   `git rebase --onto upstream/main <first-fork-commit> agy/ai-v4.2.0_shell-v4.3.0`
4. Verify the result mechanically: `git diff upstream/main..<pr-branch>` must contain none of the paths in the
   table above. If a path from that table appears, a fork-only change was committed together with integration
   work and has to be split.
