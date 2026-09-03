# Contributing

## Pull requests

- Target branch: `develop`
- **Squash merge only** — one commit on `develop` per PR (GitHub merge method: Squash and merge)
- PR title follows [Conventional Commits](https://www.conventionalcommits.org/) in English (e.g. `feat(ci): add release workflow`)
- Delete the head branch after merge (enabled on the repository)

## Releases

Tag `develop` with a semver tag (`0.1.0`, `1.2.3`, …) to trigger the release workflow and publish binaries on GitHub Releases.
