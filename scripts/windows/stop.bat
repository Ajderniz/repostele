@echo off
rem Stops the servers started by start.bat.
rem Usage: stop.bat
rem
rem Forced, like a kill on Linux: the servers have no shutdown step, and
rem SQLite in WAL mode loses nothing that was committed.

setlocal
call :stop user
call :stop staff
exit /b 0

:stop
tasklist /fi "imagename eq %1.exe" | find /i "%1.exe" >nul || (
  echo %1 not running
  exit /b 0
)
taskkill /f /im %1.exe >nul && echo %1 stopped
exit /b 0
