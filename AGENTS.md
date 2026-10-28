# Agents

Instructions for AI agents working on this project.

## Toolchain

- Use the versions in `mise.toml`. Run `mise run check` before pushing; it
  lints for Windows, macOS, and Linux and runs the race-enabled tests.
- Run `mise run snapshot` after changing `.goreleaser.yaml` or anything that
  affects packaging, and inspect the archives in `dist/`.

## Code

- The checks exist twice: in Go and in `scripts/cresclab-netcheck.bat`. Keep
  the host lists, fixed IP addresses, timeouts, and report section titles in
  sync. CI looks for section titles in both reports.
- Keep `scripts/cresclab-netcheck.bat` ASCII-only with CRLF line endings.
  `.gitattributes` enforces CRLF on checkout; an editor that writes LF can
  still break labels, so check with `file` after editing.
- Platform behavior lives in `platform_windows.go`, `platform_darwin.go`, and
  `platform_linux.go`. Tool flags differ between platforms; for example,
  `ping -W` is milliseconds on macOS and seconds on Linux.
- A check must never stop the program. Give every network operation and
  external command a timeout, record failures in the report, and keep panics
  inside `runCheck`.

## Commits and releases

- Use Conventional Commits. release-please derives versions and the changelog
  from them, so `feat:` and `fix:` messages are user-facing release notes.
- Do not create tags or GitHub releases by hand; merge the release-please pull
  request instead.

## Documentation

README and other documentation files describe the current state of the
project. They are not changelogs.

- When updating a document, re-read the whole file and rewrite the affected
  sections so it reads as a coherent current-state reference.
- Do not write migration notes or "changed X from Y to Z" entries; they belong
  in Git history.
- `docs/customer-guide.zh-TW.md` is sent to customers. Keep it in Traditional
  Chinese (Taiwan), short, and free of internal details.
