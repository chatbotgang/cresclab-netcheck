# cresclab-netcheck

[English](README.md) · 繁體中文 · [日本語](README.ja.md) · [ไทย](README.th.md)

> 本文件翻譯自 [README.md](README.md)，內容如有出入，以英文版為準。

給連不上漸強實驗室服務（例如 MAAC、CAAC）的客戶使用的網路診斷工具。客戶在出問題的那台電腦、那個網路上執行一個程式，再把產生的文字報告傳回來。報告會指出連線卡在哪裡：DNS、TCP、TLS、HTTP、依網域過濾的防火牆、封包大小限制，或是路由。

本工具只讀取資訊，不會變更任何設定。

## 下載

從[最新版本](https://github.com/chatbotgang/cresclab-netcheck/releases/latest)下載。這個頁面是公開的，客戶可以自己下載；請把連結連同[客戶操作說明](docs/customer-guide.zh-TW.md)一起傳給客戶。

每個版本都有各平台的壓縮檔、單獨的 `cresclab-netcheck.bat`，以及 `checksums.txt`。壓縮檔名稱的格式是 `cresclab-netcheck_<版本>_<作業系統>_<架構>`，請依檔名結尾挑選。

### 選擇檔案

| 電腦 | 檔名結尾 | 有建置 | CI 實際執行 |
| --- | --- | --- | --- |
| Intel 或 AMD 處理器的 Windows（大多數電腦） | `_windows_amd64.zip` | ✓ | ✓ 一般網路與封鎖網路 |
| Arm 處理器的 Windows，例如 Snapdragon 筆電 | `_windows_arm64.zip` | ✓ | — |
| Apple 晶片（M 系列）的 Mac | `_darwin_arm64.tar.gz` | ✓ | ✓ |
| Intel 處理器的 Mac | `_darwin_amd64.tar.gz` | ✓ | — |
| Intel 或 AMD 處理器的 Linux | `_linux_amd64.tar.gz` | ✓ | ✓ |
| Arm 處理器的 Linux | `_linux_arm64.tar.gz` | ✓ | — |

`amd64` 指 64 位元的 Intel 或 AMD 處理器，也稱為 x64 或 x86-64；`arm64` 指 64 位元的 Arm 處理器。

- **Windows：** 按 Windows+R，輸入 `ms-settings:about`，查看「**裝置規格**」中的處理器資訊：出現 `x64` 選 `windows_amd64`，出現 `ARM` 選 `windows_arm64`。也可以在 PowerShell 執行 `$env:PROCESSOR_ARCHITECTURE`，結果是 `AMD64` 或 `ARM64`。不確定時選 `windows_amd64`，Windows 11 on Arm 能用模擬方式執行 x64 程式。
- **macOS：** 選擇「**蘋果**」選單 >「**關於這台 Mac**」。顯示「**晶片**」並接著 Apple M 系列晶片，選 `darwin_arm64`；顯示「**處理器**」並接著 Intel 處理器，選 `darwin_amd64`。
- **Linux：** `uname -m` 顯示 `x86_64` 選 `linux_amd64`，顯示 `aarch64` 或 `arm64` 選 `linux_arm64`。

Windows 壓縮檔裡另外附有 `cresclab-netcheck.bat`，這是純文字版本，客戶的 IT 團隊可以先看過內容再執行。公司政策擋下未簽章的執行檔時，請改用它；`.exe` 跑得比較快，而且多了一段摘要。

### 系統擋下程式時如何打開

執行檔沒有程式碼簽章，所以從網路下載的檔案第一次執行時，Windows 和 macOS 都會跳出警告。

**Windows：** 出現「**Windows 已保護您的電腦**」時，按「**其他資訊**」，再按「**仍要執行**」。公司政策可能會隱藏「仍要執行」，這時請改用 `.bat`，或請 IT 團隊放行。

**macOS 15 及以後版本：**

1. 按兩下 `cresclab-netcheck`。macOS 會提示無法驗證這個程式；請關閉提示，不要把檔案丟到垃圾桶。
2. 選擇「**蘋果**」選單 >「**系統設定**」，然後按一下側邊欄中的「**隱私權與安全性**」。可能需要向下捲動。
3. 前往「**安全性**」，然後按一下「**打開**」。
4. 按一下「**強制打開**」。這個按鈕只在嘗試打開程式後約一小時內出現。
5. 輸入登入密碼，然後按一下「**好**」。之後 macOS 會記住這個例外，按兩下就能打開程式。

以上步驟依照 Apple 的說明
[打開來自未知開發者的 Mac App](https://support.apple.com/zh-tw/guide/mac-help/mh40616/mac)。
IT 人員也可以在終端機移除隔離屬性，效果相同：

```bash
xattr -d com.apple.quarantine cresclab-netcheck
```

## 報告內容

報告是 UTF-8 文字檔，存在桌面；桌面無法寫入時，會存在程式旁邊。每一項檢查都有自己的逾時時間，任何一項失敗或卡住都不會中斷其他檢查。`.exe` 每完成一項就寫入檔案，所以中途關閉視窗也會留下已完成的結果。

| 段落 | 能看出什麼 |
| --- | --- |
| Verdict（報告開頭） | 一行結論：漸強的主機全部有回應，或是哪台主機卡在哪個步驟 |
| System、network adapters、DNS、proxy | 本機設定，包括 DNS 伺服器，以及手動設定的 proxy 或 PAC 腳本 |
| Cloudflare trace | 對外 IP，以及為客戶服務的 Cloudflare 節點 |
| 各主機：DNS、ping、TCP、TLS、HTTPS | 每台漸強主機，以及對照用的 `www.cloudflare.com`、`www.google.com`，各自失敗在哪個步驟 |
| HTTP port 80 | 防火牆的封鎖頁，通常會寫出廠牌和被歸到的分類；過長的頁面會被截斷 |
| SNI comparison | 網路是依網域名稱還是依 IP 位址過濾 |
| MTU | 路徑上是否會丟掉大封包 |
| Repeated HTTPS requests | 是否為間歇性失敗 |
| Traceroute | 路由停在哪裡 |
| Summary（僅 `.exe`） | 每台主機一行，列出 `dns`、`tcp4`、`tcp6`、`tls`、`https`、`http80` 的狀態 |

### 判讀報告

1. **Verdict** 最先看。`OK` 表示漸強的主機都透過 HTTPS 回應了，任何 HTTP 狀態碼都算有回應。`PROBLEM` 會點出失敗的主機和步驟。`.exe` 也會把透過系統 proxy 的回應算進去，指出卡在哪個步驟（DNS、TCP connect、TLS 或 HTTPS），並補充 SNI 對照的結果；`.bat` 只判斷直接連線。報告仍顯示「not available yet」，代表程式在完成前就被關掉了。
2. **SNI comparison** 最能快速區分原因：
   - 在 `104.18.4.188` 上只有 `maac.cresclab.com` 失敗，同一個 IP 上的 `www.cloudflare.com` 正常：公司防火牆依網域名稱過濾。請客戶的 IT 團隊放行 `*.cresclab.com`。
   - 兩個名稱在 `104.18.x` 上都失敗，但在 `104.16.123.96` 上正常：某個 IP 位址或路由被擋。
   - 全部失敗，連對照主機也失敗：這台電腦沒有可用的網路，或需要一個尚未設定的 proxy。
3. **HTTPS 計時**中，沒有發生的步驟會標成 `-`。例如 `tcp=30ms tls=-` 表示 TCP 已連上，但 TLS 交握卡住了。
4. **TLS 憑證簽發者**如果不是公開的 CA，而是公司自己的 CA，表示防火牆在攔截 HTTPS。
5. **MTU：** 只有 `1472` 沒有回應，在 HiNet 光世代等 PPPoE 線路上是正常的。一般 ping 正常，但 `1300` 或 `1200` 也沒有回應，表示大封包會被丟掉。
6. **Traceroute 和 ping** 全部逾時，通常是網路擋了 ICMP，不代表網站掛了。請以 TCP 和 HTTPS 的結果為準。

摘要中有兩種結果是預期中的，不需要處理：`api` 和 `cdn` 主機沒有提供 port 80，所以 `http80` 會失敗；`tcp6=0/2(no-route)` 表示網路沒有 IPv6，瀏覽器會改用 IPv4。

[已確認事實](docs/verified-facts.md)（英文）記錄了已經驗證過的行為，例如 CI runner 和 Windows 防火牆的表現，不需要再驗證一次。

## 開發

[mise](https://mise.jdx.dev/) 在 `mise.toml` 固定 Go、golangci-lint 和 GoReleaser 的版本。`GOTOOLCHAIN=local` 讓 Go 不會自行下載其他版本的工具鏈。

```bash
mise install
mise run check     # lint for every target OS, race-enabled tests, config check
mise run snapshot  # build and package every target into dist/
```

`go run .` 會在目前這台電腦上執行檢查。程式最後會等使用者按 Enter，讓按兩下開啟的主控台視窗不會直接關掉。

要檢查的主機和固定的 Cloudflare IP 位址同時寫在 `main.go` 和 `scripts/cresclab-netcheck.bat` 裡。DNS 有變動時，兩邊要一起更新。

## CI 與發版

- **CI** 在每個 pull request 和 push 到 `main` 時執行：lint、在 Linux、macOS、Windows 上跑單元測試、建置所有平台的 snapshot，並在 Linux、macOS、Windows 上實際執行建置出來的程式。在 Windows 上，`.bat` 和 `.exe` 還會在 Windows 防火牆擋住漸強 Cloudflare IP 的網路上再跑一次，確認失敗的流程能跑完並正確回報。
- **發版**採用 [Conventional Commits](https://www.conventionalcommits.org/)。release-please 會在 `main` 上維持一個 release pull request。merge 之後會打 tag、建立 GitHub release，接著在同一個 workflow 中由 GoReleaser 上傳壓縮檔、單獨的 `.bat` 和 `checksums.txt`。release-please 開的 pull request 使用 `GITHUB_TOKEN`，所以不會觸發 CI；這些 PR 只會修改 `CHANGELOG.md` 和版本 manifest。
- **相依套件：** Dependabot 每週為 Go 模組和 GitHub Actions 開 pull request。`mise.toml` 中的工具版本需要手動更新。

## 授權

原始碼採用 [Apache License 2.0](LICENSE) 授權。授權範圍不包含 Crescendo Lab、MAAC、CAAC 等名稱，詳見 [NOTICE](NOTICE)。
