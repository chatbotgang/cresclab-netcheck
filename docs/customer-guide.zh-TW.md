# 漸強實驗室連線檢測：操作說明

為了找出貴公司網路連不上 MAAC / CAAC 的原因，請協助在**連不上的那台電腦、那個網路**上執行檢測程式。

## Windows

1. 從[最新版本](https://github.com/chatbotgang/cresclab-netcheck/releases/latest)下載 `cresclab-netcheck.bat`，並存到桌面。
2. 在檔案上按兩下執行。
   - 如果 Windows 顯示「Windows 已保護您的電腦」，請按「其他資訊」，再按「仍要執行」。
3. 等待黑色視窗跑完並顯示 `Done`，完成後會自動開啟檔案所在的資料夾。通常需要 3～5 分鐘，連線異常時最長約 12 分鐘。
   - 過程中出現錯誤或逾時是正常的，這些也是檢測結果的一部分，請不要關閉視窗。
4. 把桌面上的 `cresclab-netcheck-日期時間.txt` 傳給我們，然後按 Enter 關閉視窗。
   - 如果桌面沒有這個檔案，請看視窗最後顯示的檔案路徑。

如果 `.bat` 被公司政策擋下，可以改用 `windows_amd64.zip` 裡的 `cresclab-netcheck.exe`，操作步驟相同，約 1～2 分鐘完成。

## macOS

1. 從[最新版本](https://github.com/chatbotgang/cresclab-netcheck/releases/latest)下載符合電腦的檔案：Apple 晶片（M 系列）選 `darwin_arm64`，Intel 選 `darwin_amd64`。
2. 按兩下解壓縮，再按兩下裡面的 `cresclab-netcheck`。
   - 如果系統顯示無法驗證而無法打開，請先關閉提示（不要丟到垃圾桶），再到「蘋果」選單 >「系統設定」>「隱私權與安全性」，在「安全性」按「打開」，接著按「強制打開」，輸入登入密碼後按「好」。「強制打開」只在嘗試打開後約一小時內出現。
3. 等待終端機視窗顯示 `Done`，約 1～2 分鐘。
4. 把桌面上的 `cresclab-netcheck-日期時間.txt` 傳給我們。

## 檔案內容

檔案包含：網路卡設定（含內部 IP 與 DNS 伺服器）、Proxy 設定、對外 IP，以及連到漸強服務的測試結果。程式只讀取資訊，**不會變更任何設定**。若貴公司 IT 需要先審閱，`.bat` 可以用記事本打開，內容是純文字指令。
