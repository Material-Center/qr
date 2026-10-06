@echo off
chcp 936 >nul
setlocal EnableExtensions

rem Emergency cleanup for the local MI service. Run as Administrator.
set "HOSTS_FILE=%SystemRoot%\System32\drivers\etc\hosts"
set "UPLOAD_IP=120.77.84.13"
set "ENV_IP=39.108.96.33"

net session >nul 2>&1
if errorlevel 1 (
    echo [ERROR] 请右键选择“以管理员身份运行”。
    pause
    exit /b 5
)

taskkill /IM miserver-windows-amd64.exe /F >nul 2>&1
attrib -R "%HOSTS_FILE%" >nul 2>&1

powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "$ErrorActionPreference='Stop'; foreach($ip in @('%UPLOAD_IP%','%ENV_IP%')) { Get-NetIPAddress -AddressFamily IPv4 -IPAddress $ip -ErrorAction SilentlyContinue | Remove-NetIPAddress -Confirm:$false -ErrorAction SilentlyContinue }; $path='%HOSTS_FILE%'; if(Test-Path -LiteralPath $path){ $raw=[System.IO.File]::ReadAllText($path); $clean=[regex]::Replace($raw,'(?m)^[^\r\n]*# A_MI_LOCAL_MISERVER[^\r\n]*(?:\r?\n|$)',''); [System.IO.File]::WriteAllText($path,$clean,[System.Text.Encoding]::ASCII) }"
if errorlevel 1 (
    echo [ERROR] 清理网络接管失败，请检查 hosts 权限或安全软件拦截。
    pause
    exit /b 7
)

ipconfig /flushdns >nul
echo [INFO] miserver、hosts 和本地 IP 接管配置已清理。
pause
exit /b 0
