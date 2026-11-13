@echo off
rem Starts the staff and user servers, each in a minimized window.
rem Usage: start.bat   (ports: set STAFF_PORT=8081, USER_PORT=8080)
rem
rem Only staff.exe creates data.db, so staff always starts first. When the
rem database does not exist yet, the user server waits until staff is
rem listening (schema created) before it starts.

setlocal
cd /d "%~dp0"
if not defined STAFF_PORT set STAFF_PORT=8081
if not defined USER_PORT set USER_PORT=8080
if not exist log mkdir log

set FRESH=0
if not exist data.db set FRESH=1

call :start staff %STAFF_PORT% || exit /b 1
if %FRESH%==1 (
  echo no database yet; waiting for staff to create it
  call :wait staff %STAFF_PORT% || exit /b 1
)
call :start user %USER_PORT% || exit /b 1
call :wait user %USER_PORT% || exit /b 1
exit /b 0

:start
tasklist /fi "imagename eq %1.exe" | find /i "%1.exe" >nul && (
  echo %1 already running
  exit /b 0
)
if not exist %1.exe (
  echo missing %~dp0%1.exe >&2
  exit /b 1
)
rem -quiet keeps logs in the log file only; stderr keeps startup failures.
start "repostele %1" /min cmd /c "%1.exe -port %2 -quiet 2>>log\%1.err"
echo %1 started on port %2
exit /b 0

:wait
for /l %%i in (1,1,15) do (
  netstat -an | find ":%2 " | find "LISTENING" >nul && exit /b 0
  tasklist /fi "imagename eq %1.exe" | find /i "%1.exe" >nul || (
    echo %1 exited; see %~dp0log\%1.err >&2
    exit /b 1
  )
  timeout /t 1 /nobreak >nul
)
echo %1 not listening on port %2 after 15s >&2
exit /b 1
