param([Parameter(Mandatory)][ValidateSet('open', 'blocked')][string]$Phase)
$ErrorActionPreference = 'Stop'
$desktop = [Environment]::GetFolderPath('Desktop')
$out = New-Item -ItemType Directory -Force "reports/$Phase"
$required = @('==================== maac.cresclab.com', 'HTTP port 80', 'SNI comparison', 'ip=104.18.4.188', 'MTU: ping', 'Repeated HTTPS requests', 'traceroute maac.cresclab.com')

foreach ($tool in @('cresclab-netcheck.bat', 'dist\cresclab-netcheck.exe')) {
  Get-ChildItem $desktop -Filter 'cresclab-netcheck-*.txt' | Remove-Item
  $sw = [Diagnostics.Stopwatch]::StartNew()
  cmd /c "echo.| $tool" | Out-Host
  Write-Host "### $Phase $tool took $($sw.Elapsed)"
  $files = @(Get-ChildItem $desktop -Filter 'cresclab-netcheck-*.txt')
  if ($files.Count -ne 1) { throw "$tool produced $($files.Count) reports" }
  $f = $files[0]
  $bytes = [IO.File]::ReadAllBytes($f.FullName)
  if ($bytes[0] -ne 0xEF -or $bytes[1] -ne 0xBB -or $bytes[2] -ne 0xBF) { throw "$tool report is not UTF-8 with BOM" }
  $text = [IO.File]::ReadAllText($f.FullName)
  foreach ($r in $required) { if (-not $text.Contains($r)) { throw "$tool report misses '$r'" } }
  if ($text -match 'is not recognized as an internal or external command') { throw "$tool report contains a cmd error" }
  if ($Phase -eq 'blocked' -and $tool -like '*.exe' -and $text -notmatch 'maac\.cresclab\.com: dns=ok tcp=0/') { throw 'exe report does not show maac as blocked' }
  if ($Phase -eq 'blocked' -and $tool -like '*.bat' -and $text -notmatch 'ip=104\.18\.4\.188 sni=maac\.cresclab\.com tcp=0\.0+s') { throw 'bat report does not show maac as blocked' }
  Copy-Item $f.FullName (Join-Path $out ("{0}-{1}" -f (Split-Path $tool -Leaf), $f.Name))
  Write-Host "===== $($f.Name) ($($bytes.Length) bytes)"
  Write-Host $text
}
