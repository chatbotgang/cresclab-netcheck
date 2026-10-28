# cresclab-netcheck

Network diagnostics for customers who cannot reach Crescendo Lab services such
as MAAC and CAAC. A customer runs one program on the affected computer and
network, and sends back a text report. The report shows where the connection
fails: DNS, TCP, TLS, HTTP, a firewall that filters by domain name, packet size
limits, or the route.

The tool only reads information and changes no settings.

## Downloads

- **Customers:** share the public Google Drive folder
  [Crescendo Lab Network Check](https://drive.google.com/drive/folders/1qz-rjwxXQvT64_adtLXW_M_-na3qmn4e?usp=sharing)
  together with the [customer guide](docs/customer-guide.zh-TW.md).
- **Every release:** [GitHub Releases](https://github.com/chatbotgang/cresclab-netcheck/releases)
  has an archive per platform, a standalone `cresclab-netcheck.bat`, and
  `checksums.txt`.

| Platform | Archive | Built | Executed in CI |
| --- | --- | --- | --- |
| Windows x64 | `windows_amd64.zip` | ✓ | ✓ open and blocked network |
| Windows ARM64 | `windows_arm64.zip` | ✓ | — |
| macOS Apple Silicon | `darwin_arm64.tar.gz` | ✓ | ✓ |
| macOS Intel | `darwin_amd64.tar.gz` | ✓ | — |
| Linux x64 | `linux_amd64.tar.gz` | ✓ | ✓ |
| Linux ARM64 | `linux_arm64.tar.gz` | ✓ | — |

The Windows archive also contains `cresclab-netcheck.bat`, a plain-text
version that a customer's IT team can read before running. Prefer it when
company policy blocks unsigned executables; the `.exe` is faster and adds a
summary.

The binaries are not code-signed. Windows SmartScreen asks the user to choose
**More info → Run anyway**. On macOS, if the system refuses to open the
binary, allow it in **System Settings → Privacy & Security → Open Anyway**.

## What the report contains

The report is a UTF-8 text file saved to the Desktop, or next to the program
when the Desktop is not writable. Every check has its own timeout, and a
failed or hung check never stops the others. The `.exe` writes each result as
soon as it is ready, so closing the window early still leaves partial results.

| Section | What it tells you |
| --- | --- |
| Verdict (top of the report) | One line: every Crescendo Lab host answered, or which host failed at which step |
| System, network adapters, DNS, proxy | Local configuration, including DNS servers and any manual proxy or PAC script |
| Cloudflare trace | Public IP address and the Cloudflare location serving the customer |
| Per host: DNS, ping, TCP, TLS, HTTPS | Which step fails for each Crescendo Lab host and for the control hosts `www.cloudflare.com` and `www.google.com` |
| HTTP port 80 | A firewall block page, which usually names the vendor and the category; long pages are truncated |
| SNI comparison | Whether the network filters by domain name or by IP address |
| MTU | Whether large packets are dropped on the path |
| Repeated HTTPS requests | Whether failures are intermittent |
| Traceroute | Where the route stops |
| Summary (`.exe` only) | One line per host with `dns`, `tcp4`, `tcp6`, `tls`, `https`, and `http80` status |

### Reading a report

1. **Verdict** comes first. `OK` means every Crescendo Lab host answered over
   HTTPS; any HTTP status counts as an answer. `PROBLEM` names the failing
   host and step. The `.exe` also counts an answer through the system proxy,
   names the step (DNS, TCP connect, TLS, or HTTPS), and adds what the SNI
   comparison shows; the `.bat` judges the direct connection only. A report
   that still says "not available yet" was closed before it finished.
2. **SNI comparison** is the fastest discriminator:
   - Only `maac.cresclab.com` fails on `104.18.4.188` while
     `www.cloudflare.com` works on the same IP: the company firewall filters
     by domain name. Ask the customer's IT team to allow `*.cresclab.com`.
   - Both names fail on `104.18.x` but work on `104.16.123.96`: an IP address
     or route is blocked.
   - Everything fails, including the control hosts: the computer has no
     working internet access or needs a proxy that is not configured.
3. **HTTPS timing** marks a step that never happened with `-`. For example,
   `tcp=30ms tls=-` means TCP connected and the TLS handshake stalled.
4. **TLS issuer** other than a public CA, such as the company's own CA, means
   a firewall intercepts HTTPS.
5. **MTU:** a missing reply for `1472` alone is normal on PPPoE lines such as
   HiNet fiber. Missing replies for `1300` or `1200` while plain ping works
   mean large packets are dropped.
6. **Traceroute and ping** showing only timeouts usually mean the network
   blocks ICMP, not that the site is down. Rely on the TCP and HTTPS results
   instead.

Two summary results are expected and need no action: `http80` fails on the
`api` and `cdn` hosts because they do not serve port 80, and
`tcp6=0/2(no-route)` means the network has no IPv6, which browsers handle by
using IPv4.

[Verified facts](docs/verified-facts.md) records behavior that has already
been checked, such as how CI runners and Windows Firewall behave, so it does
not need to be verified again.

## Development

[mise](https://mise.jdx.dev/) pins Go, golangci-lint, and GoReleaser in
`mise.toml`. `GOTOOLCHAIN=local` keeps Go from downloading another toolchain.

```bash
mise install
mise run check     # lint for every target OS, race-enabled tests, config check
mise run snapshot  # build and package every target into dist/
```

`go run .` runs the checks on the current machine. The program waits for
Enter at the end so that a double-clicked console window stays open.

The checked hosts and the fixed Cloudflare IP addresses live in both `main.go`
and `scripts/cresclab-netcheck.bat`. Keep the two in sync when DNS changes.

## CI and releases

- **CI** runs on every pull request and push to `main`: lint, unit tests on
  Linux, macOS, and Windows, a snapshot build of every target, and the built
  binary on Linux, macOS, and Windows. On Windows, both the `.bat` and the
  `.exe` also run on a network where Windows Firewall blocks the Crescendo Lab
  Cloudflare IPs, to check that the failure path completes and is reported.
- **Releases** follow [Conventional Commits](https://www.conventionalcommits.org/).
  release-please keeps a release pull request open on `main`. Merging it tags
  the version and creates the GitHub release, and GoReleaser then uploads the
  archives, the standalone `.bat`, and `checksums.txt` in the same workflow.
  Pull requests opened by release-please use `GITHUB_TOKEN`, so CI does not
  run on them; they only change `CHANGELOG.md` and the version manifest.
- After a release, replace the files in the Google Drive folder with the new
  `cresclab-netcheck.bat` and the `.exe` from `windows_amd64.zip`.
- **Dependencies:** Dependabot opens weekly pull requests for Go modules and
  GitHub Actions. Update the tool versions in `mise.toml` by hand.
