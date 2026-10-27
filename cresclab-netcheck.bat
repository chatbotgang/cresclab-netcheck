@echo off
setlocal EnableExtensions DisableDelayedExpansion
rem Crescendo Lab network check, cmd version.
rem Collects DNS, TCP/TLS/HTTP, and traceroute results for Crescendo Lab services
rem into a text file on the Desktop. It only reads information and changes no settings.

set "VERSION=2026-10-27"
set "SERVICE_HOSTS=maac.cresclab.com caac.cresclab.com api.cresclab.com api.caac.cresclab.com cdn.caac.cresclab.com"
set "CONTROL_HOSTS=www.cloudflare.com www.google.com"
set "TRACE_HOSTS=maac.cresclab.com api.cresclab.com www.cloudflare.com"

set "OUTDIR="
for /f "usebackq delims=" %%D in (`powershell -NoProfile -Command "[Environment]::GetFolderPath('Desktop')" 2^>nul`) do set "OUTDIR=%%D"
if not defined OUTDIR set "OUTDIR=%USERPROFILE%\Desktop"
if not exist "%OUTDIR%\" set "OUTDIR=%~dp0."
set "STAMP="
for /f "usebackq delims=" %%T in (`powershell -NoProfile -Command "Get-Date -Format yyyyMMdd-HHmmss" 2^>nul`) do set "STAMP=%%T"
if not defined STAMP set "STAMP=%RANDOM%"
set "OUT=%OUTDIR%\cresclab-netcheck-%STAMP%.txt"

set "HAS_CURL="
where curl.exe >nul 2>&1 && set "HAS_CURL=1"

echo Crescendo Lab network check
echo This takes about 3 minutes. Please keep this window open.
echo.

> "%OUT%" (
  echo Crescendo Lab network check - cresclab-netcheck.bat %VERSION%
  echo Local time: %DATE% %TIME%
  echo.
  echo This file contains network adapter settings from ipconfig /all, proxy settings,
  echo the public IP address seen by Cloudflare, and connection test results for
  echo Crescendo Lab services. The script changes no settings.
)
>>"%OUT%" 2>&1 powershell -NoProfile -Command "'UTC time: ' + (Get-Date).ToUniversalTime().ToString('yyyy-MM-dd HH:mm:ss')"

echo Collecting system and network settings ...
call :log "System"
>>"%OUT%" 2>&1 ver
call :log "Network adapters: ipconfig /all"
>>"%OUT%" 2>&1 ipconfig /all
call :log "Proxy settings"
>>"%OUT%" 2>&1 netsh winhttp show proxy
>>"%OUT%" 2>&1 reg query "HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings" /v ProxyEnable
>>"%OUT%" 2>&1 reg query "HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings" /v ProxyServer
>>"%OUT%" 2>&1 reg query "HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings" /v AutoConfigURL
>>"%OUT%" 2>&1 powershell -NoProfile -Command "'System proxy for https://maac.cresclab.com/: ' + [Net.WebRequest]::GetSystemWebProxy().GetProxy([Uri]'https://maac.cresclab.com/').AbsoluteUri + '  (same URL means direct)'"

call :log "Cloudflare trace: public IP and Cloudflare location"
if defined HAS_CURL (
  >>"%OUT%" 2>&1 curl.exe -sS --ssl-no-revoke -m 15 https://www.cloudflare.com/cdn-cgi/trace
  >>"%OUT%" 2>&1 curl.exe -sS --ssl-no-revoke -m 15 https://maac.cresclab.com/cdn-cgi/trace
) else (
  >>"%OUT%" echo curl.exe not found
)

for %%H in (%SERVICE_HOSTS% %CONTROL_HOSTS%) do (
  echo Checking %%H ...
  call :probe %%H >>"%OUT%" 2>&1
)

for %%H in (%TRACE_HOSTS%) do (
  echo Tracing route to %%H ...
  call :trace %%H >>"%OUT%" 2>&1
)

call :log "Finished"
>>"%OUT%" echo Local time: %DATE% %TIME%

rem cmd and the tools above write in the console code page, for example Big5 on
rem Traditional Chinese Windows. Re-encode as UTF-8 so the file reads correctly anywhere.
set "NETCHECK_OUT=%OUT%"
powershell -NoProfile -Command "$p=$env:NETCHECK_OUT; $t=[IO.File]::ReadAllText($p,[Console]::OutputEncoding); [IO.File]::WriteAllText($p,$t,(New-Object Text.UTF8Encoding $true))" >nul 2>&1

echo.
echo Done. Result file:
echo   %OUT%
echo Please send this file back to Crescendo Lab.
explorer /select,"%OUT%"
echo.
pause
endlocal
exit /b 0


:log
echo.>>"%OUT%"
echo ==================== %~1 ====================>>"%OUT%"
exit /b 0


:probe
echo.
echo ==================== %~1 ====================
echo --- DNS, system resolver
nslookup -timeout=5 -retry=1 %~1
echo --- DNS, 1.1.1.1
nslookup -timeout=5 -retry=1 %~1 1.1.1.1
echo --- ping
ping -n 4 -w 2000 %~1
if defined HAS_CURL (
  echo --- HTTPS, direct connection via curl
  curl.exe -sS -o NUL --ssl-no-revoke --connect-timeout 10 -m 20 -w "ip=%%{remote_ip} dns=%%{time_namelookup}s tcp=%%{time_connect}s tls=%%{time_appconnect}s ttfb=%%{time_starttransfer}s total=%%{time_total}s http=%%{http_code}\n" https://%~1/
)
echo --- HTTPS, system proxy settings via PowerShell
powershell -NoProfile -Command "[Net.ServicePointManager]::SecurityProtocol=[Net.SecurityProtocolType]::Tls12; $sw=[Diagnostics.Stopwatch]::StartNew(); try { $r=Invoke-WebRequest -Uri 'https://%~1/' -UseBasicParsing -TimeoutSec 20 -MaximumRedirection 0 -ErrorAction Stop; $s='HTTP ' + [int]$r.StatusCode } catch { if ($_.Exception.Response) { $s='HTTP ' + [int]$_.Exception.Response.StatusCode } else { $s='ERROR: ' + $_.Exception.Message } }; $s + ' elapsed=' + $sw.ElapsedMilliseconds + 'ms'"
exit /b 0


:trace
echo.
echo ==================== traceroute %~1 ====================
tracert -d -h 20 -w 1000 %~1
exit /b 0
