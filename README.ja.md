# cresclab-netcheck

[English](README.md) · [繁體中文](README.zh-TW.md) · 日本語 · [ไทย](README.th.md)

> この文書は [README.md](README.md) の翻訳です。内容に相違がある場合は英語版を正とします。

MAAC や CAAC など Crescendo Lab のサービスに接続できないお客様向けのネットワーク診断ツールです。問題が起きているパソコンとネットワークでプログラムを 1 つ実行し、作成されたテキストレポートを送り返してもらいます。レポートから、接続がどこで失敗しているか（DNS、TCP、TLS、HTTP、ドメイン名で通信を制限するファイアウォール、パケットサイズの制限、経路）がわかります。

このツールは情報を読み取るだけで、設定は一切変更しません。

## ダウンロード

- **お客様向け:** 公開されている Google Drive フォルダー
  [Crescendo Lab Network Check](https://drive.google.com/drive/folders/1qz-rjwxXQvT64_adtLXW_M_-na3qmn4e?usp=sharing)
  を共有し、[お客様向けガイド](docs/customer-guide.zh-TW.md)（繁体字中国語）を添えてください。
- **各リリース:** [GitHub Releases](https://github.com/chatbotgang/cresclab-netcheck/releases)
  に、プラットフォームごとのアーカイブ、単体の `cresclab-netcheck.bat`、`checksums.txt` があります。

| プラットフォーム | アーカイブ | ビルド | CI での実行 |
| --- | --- | --- | --- |
| Windows x64 | `windows_amd64.zip` | ✓ | ✓ 通常のネットワークと遮断されたネットワーク |
| Windows ARM64 | `windows_arm64.zip` | ✓ | — |
| macOS Apple シリコン | `darwin_arm64.tar.gz` | ✓ | ✓ |
| macOS Intel | `darwin_amd64.tar.gz` | ✓ | — |
| Linux x64 | `linux_amd64.tar.gz` | ✓ | ✓ |
| Linux ARM64 | `linux_arm64.tar.gz` | ✓ | — |

Windows のアーカイブには `cresclab-netcheck.bat` も入っています。これはプレーンテキスト版で、お客様の IT チームが実行前に中身を確認できます。会社のポリシーで署名のない実行ファイルがブロックされる場合はこちらを使ってください。`.exe` のほうが速く、サマリーも出力されます。

### システムにブロックされたときの開き方

実行ファイルにはコード署名がないため、ダウンロードしたものを初めて実行すると Windows と macOS が警告を表示します。

**Windows:** 「**Windows によって PC が保護されました**」と表示されたら、「**詳細情報**」をクリックし、「**実行**」をクリックします。会社のポリシーで「実行」が表示されない場合は、`.bat` を使うか、IT チームに許可を依頼してください。

**macOS 15 以降:**

1. `cresclab-netcheck` をダブルクリックします。macOS がこのプログラムを検証できないと表示したら、ファイルをゴミ箱に入れずにメッセージを閉じます。
2. アップルメニュー ＞「**システム設定**」と選択し、サイドバーで「**プライバシーとセキュリティ**」をクリックします。下にスクロールする必要がある場合があります。
3. 「**セキュリティ**」に移動して、「**開く**」をクリックします。
4. 「**このまま開く**」をクリックします。このボタンは、プログラムを開こうとしてから約 1 時間だけ表示されます。
5. ログインパスワードを入力して、「**OK**」をクリックします。以降は macOS が例外として記憶し、ダブルクリックで開けるようになります。

この手順は Apple のガイド
[開発元が不明なMacアプリを開く](https://support.apple.com/ja-jp/guide/mac-help/mh40616/mac)
に従っています。IT 担当者は、ターミナルで隔離属性を削除しても同じ効果が得られます。

```bash
xattr -d com.apple.quarantine cresclab-netcheck
```

## レポートの内容

レポートは UTF-8 のテキストファイルで、デスクトップに保存されます。デスクトップに書き込めない場合はプログラムと同じ場所に保存されます。各チェックにはそれぞれタイムアウトがあり、失敗したりハングしたりしても他のチェックは止まりません。`.exe` はチェックが終わるたびに結果を書き込むため、途中でウィンドウを閉じてもそれまでの結果が残ります。

| セクション | わかること |
| --- | --- |
| Verdict（レポートの先頭） | 1 行の結論: Crescendo Lab のホストがすべて応答したか、どのホストがどの段階で失敗したか |
| System、network adapters、DNS、proxy | ローカルの設定。DNS サーバー、手動設定の proxy や PAC スクリプトを含みます |
| Cloudflare trace | グローバル IP アドレスと、お客様に応答している Cloudflare の拠点 |
| ホストごと: DNS、ping、TCP、TLS、HTTPS | Crescendo Lab の各ホストと、比較用の `www.cloudflare.com`、`www.google.com` がどの段階で失敗するか |
| HTTP port 80 | ファイアウォールのブロックページ。通常、製品名と分類が書かれています。長いページは切り詰められます |
| SNI comparison | ネットワークがドメイン名で制限しているか、IP アドレスで制限しているか |
| MTU | 経路上で大きなパケットが破棄されているか |
| Repeated HTTPS requests | 失敗が断続的かどうか |
| Traceroute | 経路がどこで止まっているか |
| Summary（`.exe` のみ） | ホストごとに 1 行で、`dns`、`tcp4`、`tcp6`、`tls`、`https`、`http80` の状態 |

### レポートの読み方

1. 最初に **Verdict** を見ます。`OK` は Crescendo Lab のホストがすべて HTTPS で応答したことを示します。HTTP ステータスコードであれば何でも応答とみなします。`PROBLEM` は失敗したホストと段階を示します。`.exe` はシステムの proxy 経由の応答も応答とみなし、失敗した段階（DNS、TCP connect、TLS、HTTPS）と SNI 比較の結果も示します。`.bat` は直接接続だけを判定します。レポートに「not available yet」が残っている場合は、完了前にプログラムが閉じられています。
2. **SNI comparison** で原因をすばやく切り分けられます。
   - `104.18.4.188` で `maac.cresclab.com` だけが失敗し、同じ IP の `www.cloudflare.com` は成功する: 会社のファイアウォールがドメイン名で制限しています。お客様の IT チームに `*.cresclab.com` の許可を依頼してください。
   - `104.18.x` では両方の名前が失敗し、`104.16.123.96` では成功する: IP アドレスまたは経路がブロックされています。
   - 比較用のホストも含めてすべて失敗する: このパソコンからインターネットに接続できないか、設定されていない proxy が必要です。
3. **HTTPS のタイミング**では、発生しなかった段階が `-` で表示されます。たとえば `tcp=30ms tls=-` は、TCP は接続できたものの TLS ハンドシェイクで止まったことを示します。
4. **TLS 証明書の発行者**が公開 CA ではなく会社独自の CA の場合、ファイアウォールが HTTPS を傍受しています。
5. **MTU:** `1472` だけ応答がないのは、HiNet 光回線などの PPPoE 回線では正常です。通常の ping は通るのに `1300` や `1200` も応答がない場合は、大きなパケットが破棄されています。
6. **Traceroute と ping** がすべてタイムアウトする場合、通常はネットワークが ICMP をブロックしているだけで、サイトが停止しているわけではありません。TCP と HTTPS の結果で判断してください。

サマリーのうち次の 2 つは想定どおりの結果で、対応は不要です。`api` と `cdn` のホストはポート 80 を提供していないため `http80` が失敗します。`tcp6=0/2(no-route)` はネットワークに IPv6 がないことを示し、ブラウザーは IPv4 を使います。

[確認済みの事実](docs/verified-facts.md)（英語）には、CI ランナーや Windows ファイアウォールの挙動など、すでに検証済みの事項をまとめています。再度検証する必要はありません。

## 開発

[mise](https://mise.jdx.dev/) が `mise.toml` で Go、golangci-lint、GoReleaser のバージョンを固定しています。`GOTOOLCHAIN=local` により、Go が別のツールチェーンを自動でダウンロードすることはありません。

```bash
mise install
mise run check     # lint for every target OS, race-enabled tests, config check
mise run snapshot  # build and package every target into dist/
```

`go run .` で、いま使っているマシン上でチェックを実行します。ダブルクリックで開いたコンソールウィンドウがすぐ閉じないよう、プログラムは最後に Enter の入力を待ちます。

チェック対象のホストと固定の Cloudflare IP アドレスは、`main.go` と `scripts/cresclab-netcheck.bat` の両方に書かれています。DNS が変わったときは両方を合わせて更新してください。

## CI とリリース

- **CI** は pull request ごとと `main` への push ごとに実行されます。lint、Linux・macOS・Windows でのユニットテスト、全プラットフォームのスナップショットビルド、ビルドしたプログラムの Linux・macOS・Windows での実行を行います。Windows では、`.bat` と `.exe` を Windows ファイアウォールで Crescendo Lab の Cloudflare IP を遮断したネットワークでも実行し、失敗時の流れが最後まで進んで正しく報告されることを確認します。
- **リリース**は [Conventional Commits](https://www.conventionalcommits.org/) に従います。release-please が `main` にリリース用の pull request を開いたままにします。これをマージするとタグと GitHub リリースが作成され、同じ workflow で GoReleaser がアーカイブ、単体の `.bat`、`checksums.txt` をアップロードします。release-please が開く pull request は `GITHUB_TOKEN` を使うため CI は実行されません。変更されるのは `CHANGELOG.md` とバージョンの manifest だけです。
- リリース後は、Google Drive フォルダーのファイルを新しい `cresclab-netcheck.bat` と `windows_amd64.zip` 内の `.exe` に差し替えてください。
- **依存関係:** Dependabot が毎週、Go モジュールと GitHub Actions の pull request を開きます。`mise.toml` のツールのバージョンは手動で更新してください。
