@echo off
setlocal EnableExtensions

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
rem Add /32 aliases so those destinations are owned by this Windows host.
powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "$ErrorActionPreference='Stop'; $cfg=Get-NetIPConfiguration ^| Where-Object { $_.IPv4DefaultGateway -ne $null -and $_.NetAdapter.Status -eq 'Up' } ^| Select-Object -First 1; if($null -eq $cfg){throw 'No active IPv4 interface'}; foreach($ip in @('%UPLOAD_IP%','%ENV_IP%')) { if(-not (Get-NetIPAddress -AddressFamily IPv4 -IPAddress $ip -ErrorAction SilentlyContinue)) { New-NetIPAddress -InterfaceIndex $cfg.InterfaceIndex -IPAddress $ip -PrefixLength 32 -SkipAsSource $true ^| Out-Null } }"
if errorlevel 1 (
    echo [ERROR] 无法创建本地数字 IP 别名，未启动中控。
    pause
    exit /b 6
)

set "HOSTS_FILE=%SystemRoot%\System32\drivers\etc\hosts"
findstr /C:"# A_MI_LOCAL_MISERVER" "%HOSTS_FILE%" >nul 2>&1
if errorlevel 1 (
    >>"%HOSTS_FILE%" echo %LOCAL_HOST% %AUTH_HOST% # A_MI_LOCAL_MISERVER
    if errorlevel 1 (
        echo [ERROR] 无法写入 hosts，未启动中控。
        pause
        exit /b 7
    )
)

taskkill /IM miserver-windows-amd64.exe /F >nul 2>&1
echo [INFO] 启动本地 miserver: %LOCAL_HOST%:9999, %UPLOAD_IP%:80, %ENV_IP%:8888
echo [INFO] 环境池转发到: %ENV_UPSTREAM_URL%/internalTool/miEnv
start "A_mi miserver" /b "%MISERVER_EXE%" ^
    -auth-bind-ip %LOCAL_HOST% ^
    -upload-bind-ip %UPLOAD_IP% ^
    -env-bind-ip %ENV_IP% ^
    -auth-port 9999 ^
    -upload-port 80 ^
    -env-port 8888 ^
    -db "%MISERVER_DB%" ^
    -env-upstream-url "%ENV_UPSTREAM_URL%" ^
    -env-upstream-path "/internalTool/miEnv" ^
    -env-internal-key "%ENV_INTERNAL_KEY%" ^
    >>"%MISERVER_LOG%" 2>&1

timeout /t 1 /nobreak >nul
set "MISERVER_PID="
for /f "tokens=2" %%P in ('tasklist /FI "IMAGENAME eq miserver-windows-amd64.exe" /FO LIST ^| findstr /B /C:"PID:"') do set "MISERVER_PID=%%P"
if not defined MISERVER_PID (
    echo [ERROR] miserver 未能启动，请检查 %MISERVER_LOG%
    pause
    exit /b 3
)

if exist "%CLIENT_ROOT%python38\python.exe" (
    "%CLIENT_ROOT%python38\python.exe" "%CLIENT_ROOT%main.py"
) else if exist "%CLIENT_ROOT%Python38\python.exe" (
    "%CLIENT_ROOT%Python38\python.exe" "%CLIENT_ROOT%main.py"
) else (
    echo [ERROR] 找不到 Python38\python.exe
    pause
    exit /b 4
)

set "CLIENT_EXIT=%ERRORLEVEL%"
echo [INFO] 本地模式中控退出，miserver 进程 PID %MISERVER_PID% 仍在运行。
exit /b %CLIENT_EXIT%
