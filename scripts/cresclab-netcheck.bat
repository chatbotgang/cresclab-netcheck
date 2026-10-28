@echo off
setlocal EnableExtensions DisableDelayedExpansion
rem Crescendo Lab network check, cmd version.
rem Collects DNS, TCP/TLS/HTTP, MTU, and traceroute results for Crescendo Lab services
rem into a text file on the Desktop. It only reads information and changes no settings.
rem Every check has its own timeout, and a failed check never stops the later ones.

set "VERSION=dev"
set "SERVICE_HOSTS=maac.cresclab.com caac.cresclab.com api.cresclab.com api.caac.cresclab.com cdn.caac.cresclab.com"
set "CONTROL_HOSTS=www.cloudflare.com www.google.com"
set "TRACE_HOSTS=maac.cresclab.com api.cresclab.com www.cloudflare.com"
rem Fixed Cloudflare edge IPs as of 2026-10-27, so these checks still run when DNS is blocked.
rem 104.18.x serves Crescendo Lab; 104.16.123.96 normally serves www.cloudflare.com.
set "SNI_IPS=104.18.4.188 104.18.5.188 104.16.123.96"
set "SNI_NAMES=maac.cresclab.com www.cloudflare.com"
set "MTU_TARGET=104.18.4.188"
set "MTU_SIZES=1472 1400 1300 1200"
set "REPEAT_URL=https://maac.cresclab.com/cdn-cgi/trace"
set "HTTP_TMP=%TEMP%\cresclab-netcheck-http.tmp"

set "OUTDIR="
for /f "usebackq delims=" %%D in (`powershell -NoProfile -Command "[Environment]::GetFolderPath('Desktop')" 2^>nul`) do set "OUTDIR=%%D"
if not defined OUTDIR set "OUTDIR=%USERPROFILE%\Desktop"
if not exist "%OUTDIR%\" set "OUTDIR=%~dp0."
set "STAMP="
for /f "usebackq delims=" %%T in (`powershell -NoProfile -Command "Get-Date -Format yyyyMMdd-HHmmss" 2^>nul`) do set "STAMP=%%T"
if not defined STAMP set "STAMP=%RANDOM%%RANDOM%"
set "OUT=%OUTDIR%\cresclab-netcheck-%STAMP%.txt"

set "HAS_CURL="
where curl.exe >nul 2>&1 && set "HAS_CURL=1"

echo Crescendo Lab network check
echo This usually takes 3 to 5 minutes, and up to 12 minutes when connections fail.
echo Please keep this window open until it says Done.
echo Some checks may show errors or timeouts; that is expected and is part of the result.
echo.

rem The Desktop can be read-only under some company policies; fall back to the temp folder.
call :write_header
if exist "%OUT%" goto :header_ok
set "OUT=%TEMP%\cresclab-netcheck-%STAMP%.txt"
call :write_header
:header_ok

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

echo Checking public IP ...
call :log "Cloudflare trace: public IP and Cloudflare location"
if defined HAS_CURL (
  >>"%OUT%" 2>&1 curl.exe -sS --ssl-no-revoke --connect-timeout 10 -m 15 https://www.cloudflare.com/cdn-cgi/trace
  >>"%OUT%" 2>&1 curl.exe -sS --ssl-no-revoke --connect-timeout 10 -m 15 https://maac.cresclab.com/cdn-cgi/trace
) else (
  >>"%OUT%" echo Skipped: curl.exe not found
)

rem :probe records hosts whose direct HTTPS request fails, for the verdict.
set "SERVICE_FAILED="
set /a CONTROL_FAILED=0
set /a CONTROL_TOTAL=0
set "PROBE_KIND=service"
for %%H in (%SERVICE_HOSTS%) do (
  echo Checking %%H ...
  call :probe %%H >>"%OUT%" 2>&1
)
set "PROBE_KIND=control"
for %%H in (%CONTROL_HOSTS%) do (
  echo Checking %%H ...
  set /a CONTROL_TOTAL+=1
  call :probe %%H >>"%OUT%" 2>&1
)

echo Comparing domain names on the same Cloudflare IP ...
call :log "SNI comparison: same IP, different domain names"
>>"%OUT%" echo Any HTTP code, including 403, means the network path works. A timeout means it is blocked.
>>"%OUT%" echo If only maac.cresclab.com fails on the same IP, the network filters by domain name.
if defined HAS_CURL (
  for %%I in (%SNI_IPS%) do for %%N in (%SNI_NAMES%) do (
    >>"%OUT%" 2>&1 curl.exe -sS -o NUL --ssl-no-revoke --connect-timeout 8 -m 10 --resolve %%N:443:%%I -w "ip=%%I sni=%%N tcp=%%{time_connect}s tls=%%{time_appconnect}s total=%%{time_total}s http=%%{http_code}\n" https://%%N/cdn-cgi/trace
  )
) else (
  >>"%OUT%" echo Skipped: curl.exe not found
)

echo Checking packet size limits ...
call :log "MTU: ping %MTU_TARGET% with Don't Fragment"
>>"%OUT%" echo 1472 failing alone is normal on PPPoE lines such as HiNet fiber. If 1300 or 1200 fail
>>"%OUT%" echo while plain ping works, large packets are dropped on the path and TLS can stall.
>>"%OUT%" echo If no size gets replies, ICMP is blocked and this check says nothing.
for %%S in (%MTU_SIZES%) do (
  >>"%OUT%" echo --- payload %%S bytes
  >>"%OUT%" 2>&1 ping -f -l %%S -n 2 -w 2000 %MTU_TARGET%
)

echo Repeating requests to maac.cresclab.com 10 times ...
call :log "Repeated HTTPS requests: %REPEAT_URL%"
if defined HAS_CURL (
  for /l %%R in (1,1,10) do (
    >>"%OUT%" 2>&1 curl.exe -sS -o NUL --ssl-no-revoke --connect-timeout 8 -m 8 -w "#%%R ip=%%{remote_ip} tcp=%%{time_connect}s tls=%%{time_appconnect}s total=%%{time_total}s http=%%{http_code}\n" %REPEAT_URL%
  )
) else (
  >>"%OUT%" echo Skipped: curl.exe not found
)

for %%H in (%TRACE_HOSTS%) do (
  echo Tracing route to %%H ...
  call :trace %%H >>"%OUT%" 2>&1
)

if not defined HAS_CURL (
  set "VERDICT=Verdict: not available because curl.exe was not found."
) else if not defined SERVICE_FAILED (
  set "VERDICT=Verdict: OK - every Crescendo Lab host responded over a direct HTTPS connection."
) else if %CONTROL_FAILED% GEQ %CONTROL_TOTAL% (
  set "VERDICT=Verdict: PROBLEM - the control hosts failed too, so this computer has no working internet access or needs a proxy that is not configured."
) else (
  set "VERDICT=Verdict: PROBLEM - direct HTTPS failed for%SERVICE_FAILED%. See the SNI comparison and HTTP port 80 sections."
)

call :log "Finished"
>>"%OUT%" echo %VERDICT%
>>"%OUT%" echo Local time: %DATE% %TIME%
del "%HTTP_TMP%" >nul 2>&1

rem cmd and the tools above write in the console code page, for example Big5 on
rem Traditional Chinese Windows. Re-encode as UTF-8 so the file reads correctly anywhere.
rem The original file is replaced only after the converted copy is complete.
rem The same step puts the verdict at the top in place of the placeholder.
set "NETCHECK_OUT=%OUT%"
set "NETCHECK_VERDICT=%VERDICT%"
powershell -NoProfile -Command "$p=$env:NETCHECK_OUT; $t=[IO.File]::ReadAllText($p,[Console]::OutputEncoding); $t=$t.Replace('Verdict: not available yet; the check did not finish. See the sections below.', $env:NETCHECK_VERDICT); [IO.File]::WriteAllText($p+'.utf8',$t,(New-Object Text.UTF8Encoding $true)); Move-Item -LiteralPath ($p+'.utf8') -Destination $p -Force" >nul 2>&1
if exist "%OUT%.utf8" del "%OUT%.utf8" >nul 2>&1

echo.
echo %VERDICT%
echo.
echo Done. Result file:
echo   %OUT%
echo Please send this file back to Crescendo Lab.
explorer /select,"%OUT%"
echo.
pause
endlocal
exit /b 0


:write_header
> "%OUT%" (
  echo Crescendo Lab network check - cresclab-netcheck.bat %VERSION%
  echo Local time: %DATE% %TIME%
  echo.
  echo Verdict: not available yet; the check did not finish. See the sections below.
  echo.
  echo This file contains network adapter settings from ipconfig /all, proxy settings,
  echo the public IP address seen by Cloudflare, and connection test results for
  echo Crescendo Lab services. The script changes no settings.
) 2>nul
if exist "%OUT%" >>"%OUT%" 2>&1 powershell -NoProfile -Command "'UTC time: ' + (Get-Date).ToUniversalTime().ToString('yyyy-MM-dd HH:mm:ss')"
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
if not defined HAS_CURL goto :probe_powershell
echo --- HTTPS, direct connection via curl
curl.exe -sS -o NUL --ssl-no-revoke --connect-timeout 8 -m 12 -w "ip=%%{remote_ip} dns=%%{time_namelookup}s tcp=%%{time_connect}s tls=%%{time_appconnect}s ttfb=%%{time_starttransfer}s total=%%{time_total}s http=%%{http_code}\n" https://%~1/ || call :https_failed %~1
echo --- HTTP port 80, first lines; a firewall block page shows up here
del "%HTTP_TMP%" >nul 2>&1
curl.exe -sS -i --connect-timeout 8 -m 10 -o "%HTTP_TMP%" http://%~1/
if exist "%HTTP_TMP%" powershell -NoProfile -Command "$t=[IO.File]::ReadAllText($env:HTTP_TMP); if ($t.Length -gt 800) { $t.Substring(0,800) + ' [truncated]' } else { $t }" || type "%HTTP_TMP%"
:probe_powershell
echo --- HTTPS, system proxy settings via PowerShell
powershell -NoProfile -Command "[Net.ServicePointManager]::SecurityProtocol=[Net.SecurityProtocolType]::Tls12; $sw=[Diagnostics.Stopwatch]::StartNew(); try { $r=Invoke-WebRequest -Uri 'https://%~1/' -UseBasicParsing -TimeoutSec 12 -MaximumRedirection 0 -ErrorAction Stop; $s='HTTP ' + [int]$r.StatusCode } catch { if ($_.Exception.Response) { $s='HTTP ' + [int]$_.Exception.Response.StatusCode } else { $s='ERROR: ' + $_.Exception.Message } }; $s + ' elapsed=' + $sw.ElapsedMilliseconds + 'ms'"
exit /b 0


:https_failed
if "%PROBE_KIND%"=="service" (
  set "SERVICE_FAILED=%SERVICE_FAILED% %~1"
) else (
  set /a CONTROL_FAILED+=1
)
exit /b 0


:trace
echo.
echo ==================== traceroute %~1 ====================
echo All "Request timed out" lines usually mean the network blocks ICMP, not that the site is down.
tracert -d -h 20 -w 1000 %~1
exit /b 0
