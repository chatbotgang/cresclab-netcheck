# cresclab-netcheck

English · [繁體中文](README.zh-TW.md) · [日本語](README.ja.md) · [ไทย](README.th.md)

Network diagnostics for customers who cannot reach Crescendo Lab services such
as MAAC and CAAC. A customer runs one program on the affected computer and
network, and sends back a text report. The report shows where the connection
fails: DNS, TCP, TLS, HTTP, a firewall that filters by domain name, packet size
limits, or the route.

The tool only reads information and changes no settings.

## Downloads

Download from the
[latest release](https://github.com/chatbotgang/cresclab-netcheck/releases/latest).
The repository is private, so the release page opens only for members of the
chatbotgang organization; download the file and send it to the customer together with the
[customer guide](docs/customer-guide.zh-TW.md).

Each release has an archive per platform, a standalone
`cresclab-netcheck.bat`, and `checksums.txt`. Archive names follow
`cresclab-netcheck_<version>_<os>_<arch>`, so choose by how the name ends.

### Choosing the file

| Computer | File name ends with | Built | Executed in CI |
| --- | --- | --- | --- |
| Windows on Intel or AMD (most PCs) | `_windows_amd64.zip` | ✓ | ✓ open and blocked network |
| Windows on Arm, such as Snapdragon laptops | `_windows_arm64.zip` | ✓ | — |
| Mac with Apple silicon (M series) | `_darwin_arm64.tar.gz` | ✓ | ✓ |
| Mac with an Intel processor | `_darwin_amd64.tar.gz` | ✓ | — |
| Linux on Intel or AMD | `_linux_amd64.tar.gz` | ✓ | ✓ |
| Linux on Arm | `_linux_arm64.tar.gz` | ✓ | — |

`amd64` means a 64-bit Intel or AMD processor, also called x64 or x86-64.
`arm64` means a 64-bit Arm processor.

- **Windows:** press Windows+R, enter `ms-settings:about`, and look at the
  processor details under **Device specifications**: `x64` means
  `windows_amd64`, `ARM` means `windows_arm64`. In PowerShell,
  `$env:PROCESSOR_ARCHITECTURE` prints `AMD64` or `ARM64`. When unsure, take
  `windows_amd64`; Windows 11 on Arm runs x64 programs through emulation.
- **macOS:** choose **Apple menu → About This Mac**. **Chip** followed by an
  Apple M chip means `darwin_arm64`; **Processor** followed by an Intel
  processor means `darwin_amd64`.
- **Linux:** `uname -m` prints `x86_64` for `linux_amd64`, and `aarch64` or
  `arm64` for `linux_arm64`.

The Windows archive also contains `cresclab-netcheck.bat`, a plain-text
version that a customer's IT team can read before running. Prefer it when
company policy blocks unsigned executables; the `.exe` is faster and adds a
summary.

### Opening the program when the system blocks it

The binaries are not code-signed, so Windows and macOS warn the first time a
downloaded copy runs.

**Windows:** when **Windows protected your PC** appears, click **More info**,
then **Run anyway**. Company policy can hide **Run anyway**; in that case use
the `.bat`, or ask the IT team to allow the program.

**macOS 15 and later:**

1. Double-click `cresclab-netcheck`. macOS reports that it cannot verify the
   program; close the message without moving the file to the Trash.
2. Choose **Apple menu → System Settings**, then click **Privacy & Security**
   in the sidebar. You may need to scroll down.
3. Go to **Security** and click **Open**.
4. Click **Open Anyway**. The button is available for about an hour after
   you try to open the program.
5. Enter your login password and click **OK**. macOS then remembers the
   exception, and the program opens with a double-click.

These steps follow Apple's guide
[Open a Mac app from an unknown developer](https://support.apple.com/guide/mac-help/mh40616/mac).
For IT staff, removing the quarantine attribute in Terminal has the same
effect:

```bash
xattr -d com.apple.quarantine cresclab-netcheck
```

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
- **Dependencies:** Dependabot opens weekly pull requests for Go modules and
  GitHub Actions. Update the tool versions in `mise.toml` by hand.
