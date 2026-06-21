@echo off
setlocal

set "EXE=%~dp0phone-task-client-windows-amd64.exe"
set "DB=%~dp0phone-task-client.db"
set "BASE_URL=http://210.16.170.132:1111/api"
set "TOKEN="
set "MODE=receive"
set "PHONE_SOURCE=txt"
set "INPUT=%~dp0phones.txt"
set "CODE_API="
set "PHONE_API="
set "RESERVE_DEVICES=1"
set "INTERVAL=3s"
set "CREATE_DELAY=0s"
set "TIMEOUT=10s"
set "FAILED_OUTPUT=%~dp0failed.txt"
set "SUCCESS_OUTPUT=%~dp0success.txt"

if not exist "%EXE%" (
  echo executable not found: "%EXE%"
  pause
  exit /b 1
)

"%EXE%" ^
  -db "%DB%" ^
  -base-url "%BASE_URL%" ^
  -token "%TOKEN%" ^
  -mode "%MODE%" ^
  -phone-source "%PHONE_SOURCE%" ^
  -input "%INPUT%" ^
  -phone-api "%PHONE_API%" ^
  -code-api "%CODE_API%" ^
  -reserve-devices "%RESERVE_DEVICES%" ^
  -interval "%INTERVAL%" ^
  -create-delay "%CREATE_DELAY%" ^
  -timeout "%TIMEOUT%" ^
  -failed-output "%FAILED_OUTPUT%" ^
  -success-output "%SUCCESS_OUTPUT%"

pause
