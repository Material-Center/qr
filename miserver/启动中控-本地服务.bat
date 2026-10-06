@echo off
chcp 936 >nul
setlocal EnableExtensions EnableDelayedExpansion

rem A_mi 3.0-2 local MI service launcher.
rem Keep the original 启动中控.bat unchanged. Run this file for local mode.

set "CLIENT_ROOT=%~dp0"
set "MISERVER_EXE=%CLIENT_ROOT%miserver-windows-amd64.exe"
set "MISERVER_DB=%CLIENT_ROOT%miserver.db"
set "MISERVER_LOG=%CLIENT_ROOT%miserver.log"
set "AUTH_HOST=py.j8nda.xyz"
set "LOCAL_HOST=127.0.0.2"
set "UPLOAD_IP=120.77.84.13"
set "ENV_IP=39.108.96.33"
set "ENV_UPSTREAM_URL=http://210.16.170.132:1111/api"
set "ENV_INTERNAL_KEY=cd5d1c1b4bd95fcb561d2a3f2b5407de82b9088d8b1f4eb3ebce9c28331ef42f"
set "HOSTS_FILE=%SystemRoot%\System32\drivers\etc\hosts"
set "CLIENT_EXIT=0"

if not exist "%MISERVER_EXE%" (
    echo [ERROR] 找不到 %MISERVER_EXE%
    echo 请把 miserver-windows-amd64.exe 复制到当前客户端目录。
    pause
    exit /b 2
)

net session >nul 2>&1
if errorlevel 1 (
    echo [ERROR] 请右键选择“以管理员身份运行”。本地 80 端口、hosts 和数字 IP 接管都需要管理员权限。
    pause
    exit /b 5
)

rem The compiled client uses literal public IPs for upload and env traffic.
rem Add /32 aliases only to the Windows Loopback interface; never modify the
rem physical/VPN adapter that carries the machine's default internet route.
powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "$ErrorActionPreference='Stop'; foreach($ip in @('%UPLOAD_IP%','%ENV_IP%')) { Get-NetIPAddress -AddressFamily IPv4 -IPAddress $ip -ErrorAction SilentlyContinue | Remove-NetIPAddress -Confirm:$false -ErrorAction SilentlyContinue }; $loopback=Get-NetIPInterface -AddressFamily IPv4 | Where-Object { $_.InterfaceAlias -match 'Loopback|回环|环回' } | Sort-Object InterfaceMetric | Select-Object -First 1; if($null -eq $loopback){throw 'No IPv4 loopback interface'}; foreach($ip in @('%UPLOAD_IP%','%ENV_IP%')) { New-NetIPAddress -InterfaceIndex $loopback.ifIndex -IPAddress $ip -PrefixLength 32 -SkipAsSource $true -PolicyStore ActiveStore | Out-Null }"
if errorlevel 1 (
    echo [ERROR] 无法创建本地数字 IP 别名，未启动中控。
    set "CLIENT_EXIT=6"
    goto cleanup
)

rem Write hosts through PowerShell instead of cmd redirection. This handles
rem hosts files with non-ANSI encoding and reports permission failures clearly.
attrib -R "%HOSTS_FILE%" >nul 2>&1
powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "$ErrorActionPreference='Stop'; $path='%HOSTS_FILE%'; if(-not (Test-Path -LiteralPath $path)){throw 'hosts file not found'}; $raw=[System.IO.File]::ReadAllText($path); if($raw -notmatch '(?m)# A_MI_LOCAL_MISERVER'){ $line='%LOCAL_HOST% %AUTH_HOST% # A_MI_LOCAL_MISERVER' + [Environment]::NewLine; [System.IO.File]::AppendAllText($path,$line,[System.Text.Encoding]::ASCII) }"
if errorlevel 1 (
    echo [ERROR] 无法写入 hosts，未启动中控。请确认已管理员运行，并检查安全软件是否拦截 hosts 修改。
    set "CLIENT_EXIT=7"
    goto cleanup
)

taskkill /IM miserver-windows-amd64.exe /F >nul 2>&1
echo [INFO] 启动本地 miserver: %LOCAL_HOST%:9999, %UPLOAD_IP%:80, %ENV_IP%:8888
echo [INFO] 环境池转发到: %ENV_UPSTREAM_URL%/internalTool/miEnv
echo [INFO] miserver 文件: "%MISERVER_EXE%"
rem The empty title is required by START when the executable path is quoted.
start "" /b "%MISERVER_EXE%" ^
    -auth-bind-ip %LOCAL_HOST% ^
    -upload-bind-ip %UPLOAD_IP% ^
    -env-bind-ip %ENV_IP% ^
    -auth-port 9999 ^
    -upload-port 80 ^
    -env-port 8888 ^
    -db "%MISERVER_DB%" ^
    -seed python3806250511 ^
    -iv 0625051106250511 ^
    -response-seed-prefix python38x64 ^
    -env-upstream-url "%ENV_UPSTREAM_URL%" ^
    -env-upstream-path "/internalTool/miEnv" ^
    -env-internal-key "%ENV_INTERNAL_KEY%" ^
    >>"%MISERVER_LOG%" 2>&1

timeout /t 1 /nobreak >nul
set "MISERVER_PID="
for /f "tokens=2" %%P in ('tasklist /FI "IMAGENAME eq miserver-windows-amd64.exe" /FO LIST ^| findstr /B /C:"PID:"') do set "MISERVER_PID=%%P"
if not defined MISERVER_PID (
    echo [ERROR] miserver 未能启动，请检查 %MISERVER_LOG%
    set "CLIENT_EXIT=3"
    goto cleanup
)

if exist "%CLIENT_ROOT%python38\python.exe" (
    "%CLIENT_ROOT%python38\python.exe" "%CLIENT_ROOT%main.py"
) else if exist "%CLIENT_ROOT%Python38\python.exe" (
    "%CLIENT_ROOT%Python38\python.exe" "%CLIENT_ROOT%main.py"
) else (
    echo [ERROR] 找不到 Python38\python.exe
    set "CLIENT_EXIT=4"
    goto cleanup
)

set "CLIENT_EXIT=%ERRORLEVEL%"
goto cleanup

:cleanup
echo [INFO] 正在退出本地模式，清理 miserver、IP 别名和 hosts 接管配置。
if defined MISERVER_PID (
    taskkill /PID !MISERVER_PID! /F >nul 2>&1
    set "MISERVER_PID="
)

rem These two /32 addresses are reserved by this local-mode launcher.
powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "$ErrorActionPreference='SilentlyContinue'; foreach($ip in @('%UPLOAD_IP%','%ENV_IP%')) { Get-NetIPAddress -AddressFamily IPv4 -IPAddress $ip | Remove-NetIPAddress -Confirm:$false }" >nul 2>&1

rem Remove only the hosts line written by this launcher.
powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "$ErrorActionPreference='Stop'; $path='%HOSTS_FILE%'; if(Test-Path -LiteralPath $path){ $raw=[System.IO.File]::ReadAllText($path); $clean=[regex]::Replace($raw,'(?m)^[^\r\n]*# A_MI_LOCAL_MISERVER[^\r\n]*(?:\r?\n|$)',''); [System.IO.File]::WriteAllText($path,$clean,[System.Text.Encoding]::ASCII) }" >nul 2>&1

echo [INFO] 本地网络接管已恢复。
if not "!CLIENT_EXIT!"=="0" (
    echo [ERROR] 本地模式启动或客户端运行失败，退出码 !CLIENT_EXIT!。
    echo [ERROR] 请查看日志: !MISERVER_LOG!
    pause
)
exit /b !CLIENT_EXIT!
