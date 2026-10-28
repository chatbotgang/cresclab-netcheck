---
name: cresclab-netcheck-pr
description: Ship a change in cresclab-netcheck from branch to merged pull request and, when the change is releasable, through the release-please release. Covers the PR title and its effect on the release, pre-push checks, waiting for CI, the squash merge, and verifying the published release. Use for "open a PR", "create PR", "ship this", "merge", "release", "開 PR", "發版".
---

# cresclab-netcheck pull requests and releases

Covers one change from a branch to `main`, and the release that follows when
the change is releasable. Merge and release only when the user has authorized
them.

| Not this skill | Where it lives |
| --- | --- |
| Answering review comments | `vp-pr-comment-resolver` |
| Facts that are already verified, such as CI runner behavior | `docs/verified-facts.md` |
| Code, toolchain, and documentation rules | `AGENTS.md` |

## Repository settings that shape the flow

Checked on 2026-10-28. Recheck with
`gh api repos/chatbotgang/cresclab-netcheck --jq '{allow_squash_merge, allow_merge_commit, allow_rebase_merge, squash_merge_commit_title, squash_merge_commit_message, delete_branch_on_merge}'`
when a merge is refused.

- **Squash merge only.** Rebase and merge commits are disabled.
- **The squash subject is the release note.** release-please reads the
  subject on `main`. GitHub's default subject is the commit subject for a
  single-commit PR and the PR title otherwise, so set it explicitly when
  merging (step 7).
- **No branch protection.** Nothing blocks a merge with failing or pending
  CI; waiting for every job is this skill's rule, not GitHub's.
- **The head branch is deleted on merge.**

## Choosing the PR title

The title is a Conventional Commit subject in American English, written for
the changelog: say what changed for a user of the tool.

| Type | Release effect |
| --- | --- |
| `feat:` | Minor version bump while below 1.0, listed under Features |
| `fix:` | Patch version bump, listed under Bug Fixes |
| `feat!:` or a `BREAKING CHANGE:` footer | Minor bump while below 1.0, major bump after |
| `docs:`, `test:`, `ci:`, `build:`, `refactor:`, `chore:` | No release |

A change that only adds documentation or tests does not reach the release
archives until the next `feat:` or `fix:`. Say so in the PR body when it
matters, for example for README files that ship in the archives.

## Workflow

### 1. Branch

Start from an up-to-date `main` and name the branch `<type>/<short-topic>`,
for example `feat/report-verdict`.

```bash
git switch main && git pull
git switch -c <type>/<short-topic>
```

### 2. Check before pushing

```bash
mise run check
```

Also, depending on what changed:

- `.goreleaser.yaml`, archive contents, or the `.bat`: run
  `mise run snapshot` and inspect the archives in `dist/`.
- `scripts/cresclab-netcheck.bat`: confirm `file` reports ASCII text with
  CRLF line terminators.
- `README.md`: update `README.zh-TW.md`, `README.ja.md`, and `README.th.md`
  in the same PR.
- Behavior or verification that a later change could rely on: add it to
  `docs/verified-facts.md`.

### 3. Push and open a draft

Write the body to a file and pass it with `--body-file`. Use these sections:
**Summary** (the conclusion first), **Acceptance criteria**, **Verification**
(what ran and its result, with links), and **Not covered** (what remains
unverified or out of scope).

```bash
git push -u origin <branch>
gh pr create --draft --base main --title "<type>: <description>" --body-file <path>
```

Turn on the host's PR monitor for the new PR when one is available.

### 4. Wait for CI on the current head

CI runs eight jobs. Wait until every one has finished and passed for the
current head commit; a pending job is not a pass.

- The run takes about 12 minutes. The Windows job dominates it because it runs
  the `.bat` (about 5 minutes) and the `.exe` on an open network and again on
  a blocked one.
- `gh run watch` can exit before the run finishes. Before acting on its
  output, confirm with
  `gh run view <run-id> --json status,conclusion,jobs`.

On a failure, fix it on the branch, push, and wait again.

### 5. Check feedback

Read both PR conversation comments and review comments, and confirm none is
unresolved for the current head:

```bash
gh pr view <number> --json comments,reviews,headRefOid
gh api repos/chatbotgang/cresclab-netcheck/pulls/<number>/comments
```

Route any feedback through `vp-pr-comment-resolver`, then return to step 4.

### 6. Mark ready

```bash
gh pr ready <number>
```

Marking ready is a new event. Read the feedback again before merging.

### 7. Squash merge with an explicit subject

```bash
gh pr merge <number> --squash --subject "<PR title> (#<number>)"
git switch main && git pull
```

### 8. Release, when the merged type releases

After a `feat:` or `fix:` merge, the Release workflow opens or updates a pull
request titled `chore(main): release <version>`.

1. Review its diff: it must change only `CHANGELOG.md` and
   `.release-please-manifest.json`, with the expected version and entries.
2. Its CI ends as `action_required` and tests nothing, because release-please
   opens it with `GITHUB_TOKEN`. Rely on the `main` CI for the same code.
3. Squash merge it: `gh pr merge <number> --squash`.
4. Wait for the Release workflow on `main`; both `release-please` and
   `goreleaser` must succeed.
5. Verify the published release:

   ```bash
   gh release download v<version> -R chatbotgang/cresclab-netcheck -D <empty-dir>
   cd <empty-dir> && shasum -a 256 -c checksums.txt
   grep -a 'set "VERSION=' cresclab-netcheck.bat
   ```

   Expect six archives, `cresclab-netcheck.bat`, and `checksums.txt`; every
   checksum OK; and the `.bat` version equal to the release version.

Do not create tags or releases by hand.

### 9. After a release

The public Google Drive folder that customers download from is updated by
hand. Replacing its files changes what customers receive, so ask the user
before touching it.
