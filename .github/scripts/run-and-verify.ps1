# Runs the packaged .bat and .exe from $Dir and checks their reports. With
# -Phase blocked, also checks that the reports show the blocked Cloudflare IPs.
param(
  [Parameter(Mandatory)][ValidateSet('open', 'blocked')][string]$Phase,
  [Parameter(Mandatory)][string]$Dir
)
$ErrorActionPreference = 'Stop'
$desktop = [Environment]::GetFolderPath('Desktop')
$out = New-Item -ItemType Directory -Force "reports/$Phase"
$required = @('==================== maac.cresclab.com', 'HTTP port 80', 'SNI comparison', 'ip=104.18.4.188', 'MTU: ping', 'Repeated HTTPS requests', 'traceroute maac.cresclab.com')

foreach ($tool in @('cresclab-netcheck.bat', 'cresclab-netcheck.exe')) {
  Get-ChildItem $desktop -Filter 'cresclab-netcheck-*.txt' | Remove-Item
  $sw = [Diagnostics.Stopwatch]::StartNew()
  cmd /c "echo.| `"$(Join-Path $Dir $tool)`"" | Out-Host
  Write-Host "### $Phase $tool took $($sw.Elapsed)"
  $files = @(Get-ChildItem $desktop -Filter 'cresclab-netcheck-*.txt')
  if ($files.Count -ne 1) { throw "$tool produced $($files.Count) reports" }
  $f = $files[0]
  $bytes = [IO.File]::ReadAllBytes($f.FullName)
  if ($bytes[0] -ne 0xEF -or $bytes[1] -ne 0xBB -or $bytes[2] -ne 0xBF) { throw "$tool report is not UTF-8 with BOM" }
  $text = [IO.File]::ReadAllText($f.FullName)
  foreach ($r in $required) { if (-not $text.Contains($r)) { throw "$tool report misses '$r'" } }
  if ($text -match 'is not recognized as an internal or external command') { throw "$tool report contains a cmd error" }
  if ($text -match '- cresclab-netcheck(\.bat)? dev\b') { throw "$tool report has no release version" }
  # The verdict replaces the placeholder near the top of the report.
  $verdictAt = $text.IndexOf('Verdict: ')
  if ($verdictAt -lt 0 -or $verdictAt -gt 400) { throw "$tool report has no verdict near the top" }
  if ($text.Contains('Verdict: not available yet')) { throw "$tool report kept the verdict placeholder" }
  if ($Phase -eq 'open' -and -not $text.Contains('Verdict: OK')) { throw "$tool verdict is not OK on an open network" }
  if ($Phase -eq 'blocked' -and $text -notmatch 'Verdict: PROBLEM - [^\r\n]*maac\.cresclab\.com') { throw "$tool verdict does not name maac as failing" }
  if ($Phase -eq 'blocked' -and $tool -like '*.exe' -and $text -notmatch 'maac\.cresclab\.com: dns=ok tcp4=0/') { throw 'exe report does not show maac as blocked' }
  if ($Phase -eq 'blocked' -and $tool -like '*.bat' -and $text -notmatch 'ip=104\.18\.4\.188 sni=maac\.cresclab\.com tcp=0\.0+s') { throw 'bat report does not show maac as blocked' }
  Copy-Item $f.FullName (Join-Path $out "$tool-$($f.Name)")
  Write-Host "===== $($f.Name) ($($bytes.Length) bytes)"
  Write-Host $text
}
