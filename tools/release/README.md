# Release templates

The files a game copies in when it sets up CI/CD (AGENTS.md §5). They are kept
here, not in `.github/workflows/`, so the skeleton's own tags build nothing.

- `release.yml` goes to `.github/workflows/`. When a `v*` tag is pushed it
  runs vet and tests on Ubuntu 22.04 and Windows, checks the sprite sheets and
  that the program icon is the game's own, builds a Windows zip and a Linux
  tarball (each with README and LICENSE), writes `SHA256SUMS.txt`, and
  publishes a GitHub release titled with the tag's message, with that
  version's section of `CHANGELOG.md` as its notes.
- `cliff.toml` and `CHANGELOG.md` go to the root. [git-cliff](https://git-cliff.org)
  adds each version's section to `CHANGELOG.md` from the Conventional Commits
  since the last tag.

`tools/newgame` renames `gameskeleton` in them along with everything else.
