# Verified facts

Behavior that has been checked against real systems. Reuse a fact instead of
verifying it again while its **Recheck when** condition does not apply. Each
fact names its evidence; dates are Asia/Taipei.

Add a fact when a check establishes something a later change would otherwise
need to test again. Remove or update a fact when its recheck condition occurs.

## CI runners

| Fact | Evidence | Recheck when |
| --- | --- | --- |
| GitHub-hosted Windows runners drop outbound ICMP: every `ping` and `tracert` hop times out. CI cannot validate MTU or traceroute content, only that those sections run. | vp-playground run 40518712476 and later Windows runs, 2026-10-27 | GitHub changes runner networking |
| GitHub-hosted Linux runners and the maintainer's macOS network have no IPv6 route; IPv6 dials fail at once with "no route to host". | `www.cloudflare.com` showed 2 of 4 TCP connects failing on IPv6 in PR #1 CI and local runs, 2026-10-28 | GitHub adds IPv6 to runners |
| A Windows Firewall outbound **Block** rule rejects connections immediately (Go: "forbidden by its access permissions"; curl: error 7 after 0 ms) instead of dropping packets. The blocked CI phase therefore checks failure reporting, not timeout duration. | vp-playground run 40522850391, 2026-10-27 | Never by itself; use a different blocking method to measure timeouts |
| `echo.\| cmd /c tool` satisfies the final `pause` or "Press Enter" prompt on a Windows runner. | All Windows CI runs | — |
| `[Environment]::GetFolderPath('Desktop')` returns a writable Desktop on Windows runners, so reports land there. | All Windows CI runs | — |
| Workflow runs on pull requests opened by release-please (`GITHUB_TOKEN`) end as `action_required` and do not test anything. | Release PR #2 run 40616079682, 2026-10-28 | The release workflow uses a different token |
| The repository is public, allows only squash merges, deletes the head branch on merge, and has no branch protection; workflows from a first-time contributor's fork wait for approval. The squash subject defaults to the commit subject for a single-commit PR and to the PR title otherwise (`COMMIT_OR_PR_TITLE`); the body lists the commit messages. | Repository API, 2026-10-28; a rebase merge of PR #6 was refused | Repository settings change |
| When the chatbotgang GitHub Actions budget runs out, jobs in every private repository of the organization end without starting ("an Actions budget is preventing further use") and have no steps or logs. Since this repository became public on 2026-10-28, its CI on standard runners no longer counts against that budget. | PR #7 run 40628129809 and a chatbotgang/cantata run, 2026-10-28; the rerun after going public started at once | Billing or visibility changes |
| One full CI run here takes about 70 minutes of runner time in Linux-equivalent minutes (Windows counts double, macOS ten times), most of it the Windows job. | Job durations of runs from 2026-10-27 to 2026-10-28 | The workflow changes |
| The chatbotgang organization lets GitHub Actions create pull requests (`can_approve_pull_request_reviews: true`). | Repository workflow permissions API, 2026-10-28 | Organization Actions policy changes |

## Network and tools

| Fact | Evidence | Recheck when |
| --- | --- | --- |
| `maac`, `caac`, and `cdn.platform.cresclab.com` resolve to Cloudflare `104.18.4.188` and `104.18.5.188`; `api.cresclab.com` (34.120.125.52), `api.caac.cresclab.com` (34.111.24.246), and `cdn.caac.cresclab.com` (34.8.204.232) are Google Cloud load balancers. | `dig` from Taiwan, 2026-10-27 | DNS changes; then update `main.go` and the `.bat` together |
| The Crescendo Lab Cloudflare IPs also answer TLS and HTTP for `www.cloudflare.com` (200). `104.16.123.96` completes TLS for `maac.cresclab.com` and answers 403. Both make the SNI comparison valid. | `curl --resolve` from macOS and Windows runners, 2026-10-27 | Cloudflare plan or IP changes |
| The `api` and `cdn` hosts do not serve port 80; plain HTTP resets or returns EOF. | Every report since 2026-10-27 | Load balancer configuration changes |
| `http://maac.cresclab.com/` and `http://caac.cresclab.com/` redirect to `*.pages.dev` instead of to their HTTPS hostnames. Unrelated to this tool, reported separately. | Reports from 2026-10-27 and 2026-10-28 | Cloudflare redirect rules change |
| curl prints `-w` output even when the transfer fails (`http=000 tcp=0.000000s`), so the `.bat` SNI and repeat lines stay aligned on failure. | Blocked-network `.bat` report, run 40522850391 | — |
| Go's resolver with `PreferGo` and a custom `Dial` works on Windows; the `.exe` queries 1.1.1.1 and 8.8.8.8 directly there. | Windows `.exe` report, run 40522850391 | — |
| macOS `ping -W` is in milliseconds; Linux iputils `ping -W` is in seconds. On a PPPoE line the router answers `ping -D -s 1472` with "frag needed and DF set (MTU 1492)". | Local macOS runs and `man ping`, 2026-10-28 | — |
| A full `.exe` run takes about 1 minute on an open network; the `.bat` takes about 5 minutes because it runs sequentially and `tracert` waits on ICMP timeouts. | Windows CI runs, 2026-10-27 to 2026-10-28 | Timeouts or the check list change |

## Operating-system prompts

| Fact | Evidence | Recheck when |
| --- | --- | --- |
| The released macOS binaries are only ad-hoc signed by the Go linker (identifier `a.out`); `spctl --assess --type execute` rejects them, so a downloaded copy is blocked by Gatekeeper until the user chooses Open Anyway or removes `com.apple.quarantine`. | `spctl` and `codesign -dv` on the v0.2.0 `darwin_arm64` binary, 2026-10-28 | The binaries are signed with a Developer ID |
| On macOS 15, 26, and 27 the steps are Apple menu → System Settings → Privacy & Security → Security → Open → Open Anyway → login password → OK, and Open Anyway stays available for about an hour. Localized labels: zh-TW 「隱私權與安全性」「打開」「強制打開」「好」; ja 「プライバシーとセキュリティ」「開く」「このまま開く」「OK」; th "ความเป็นส่วนตัวและความปลอดภัย" "เปิด" "ยืนยันจะเปิด". | Apple guide mh40616 in en, zh-tw, ja-jp, and th-th, versions 15.0, 26, and 27, 2026-10-28 | A new macOS release changes the guide |
| Windows SmartScreen labels: en "Windows protected your PC" → More info → Run anyway; zh-TW 「Windows 已保護您的電腦」→「其他資訊」→「仍要執行」; ja 「Windows によって PC が保護されました」→「詳細情報」→「実行」. | [ASUS Taiwan FAQ](https://www.asus.com/tw/support/faq/1053651/), [Buffalo FAQ](https://www.buffalo.jp/support/faq/detail/124145337.html), 2026-10-28 | Windows changes the dialog |

| Windows: `ms-settings:about` opens Settings → System → About, whose **Device specifications** show the processor type; Windows 11 on Arm runs x64 programs through emulation, while Windows 10 on Arm emulates only x86. macOS: About This Mac shows **Chip** on Apple silicon and **Processor** on Intel (zh-TW 「關於這台 Mac」「晶片」「處理器」; ja 「このMacについて」「チップ」「プロセッサ」; th "เกี่ยวกับ Mac เครื่องนี้" "ชิป" "โปรเซสเซอร์"). | Microsoft support "Which version of Windows operating system am I running?" and Learn "How emulation works on Arm"; Apple support 116943 in en, zh-tw, ja-jp, th-th; 2026-10-28 | Vendors change the settings pages |

## Toolchain

| Fact | Evidence | Recheck when |
| --- | --- | --- |
| GoReleaser OSS cannot template archive files (`templated_files` is Pro only), so a `before` hook renders the `.bat` version into `build/`. | GoReleaser docs, `customization/package/archives.md`, v2.19.0 | Moving to GoReleaser Pro |
| release-please proposes 1.0.0 for a first release from manifest `0.0.0` unless `initial-version` is set. | Release PR #2 before and after PR #3, 2026-10-28 | — |
| `go build ./...` in the repository root writes a `cresclab-netcheck` binary there, because the module has a single main package. `.gitignore` excludes it after one was committed by accident in 2ee5181. | Local build, 2026-10-28 | — |
| A file written with LF line endings breaks `.bat` labels; generated or edited `.bat` content must be converted to CRLF and checked with `file`. | Prototype, 2026-10-27 | — |
| golangci-lint refuses to start while another golangci-lint runs on the same machine ("parallel golangci-lint is running"), even in another repository, so `mise run check` can fail because of an unrelated session; rerun it once the other lint finishes. | Local runs, 2026-10-28, while another repository was linting | — |
| An editor or tool may write a literal U+FEFF where Go source should contain the escape `"\xef\xbb\xbf"`; Go rejects it with "illegal byte order mark". | Prototype build, 2026-10-27 | — |

| `gh run watch` can exit before the run finishes; check the run status again before treating its output as final. | PR #4 CI run 40619973173, 2026-10-28 | — |

## Not verified yet

- Big5 (code page 950) output from `tracert` and `nslookup` converts to readable UTF-8 on Traditional Chinese Windows. Only English Windows runners have run the tools.
- How long both tools take when a network silently drops packets instead of rejecting them. The `.bat` estimate of up to 12 minutes comes from its timeouts.
- `windows_arm64`, `darwin_amd64`, and `linux_arm64` binaries run correctly; they are built but not executed.
- The race detector on Windows: CI skips `-race` there on the assumption that it needs cgo.
- The Thai SmartScreen labels "ข้อมูลเพิ่มเติม" and "เรียกใช้ต่อไป"; only a third-party article supports them, so `README.th.md` also shows the English labels.
