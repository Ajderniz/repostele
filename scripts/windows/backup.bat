@echo off
rem Backs up the database and uploaded images into backup\<timestamp>\.
rem Usage: backup.bat [dest-dir]   (default dest: backup)
rem
rem Needs sqlite3.exe next to this script or on PATH. Safe while the
rem servers run: sqlite3's .backup takes a consistent copy, including what
rem is still in data.db-wal (a plain copy of data.db is not).
rem Restore: see README.md.

setlocal
cd /d "%~dp0"
set BASE=%~1
if "%BASE%"=="" set BASE=backup
rem %DATE% depends on the locale; PowerShell's format does not.
set PS=powershell -nologo -noprofile -command
for /f %%t in ('%PS% "Get-Date -Format yyyyMMdd-HHmmss"') do set STAMP=%%t
set DEST=%BASE%\%STAMP%

if not exist data.db (
  echo no database at %~dp0data.db >&2
  exit /b 1
)
where sqlite3 >nul 2>&1 || (
  echo sqlite3.exe not found >&2
  exit /b 1
)

mkdir "%DEST%" || exit /b 1
sqlite3 data.db ".backup '%DEST:\=/%/data.db'" || exit /b 1
set CHECK=
for /f %%r in ('sqlite3 "%DEST%\data.db" "PRAGMA integrity_check"') do (
  set CHECK=%%r
)
if not "%CHECK%"=="ok" (
  echo backup failed its integrity check: %DEST% >&2
  exit /b 1
)
if exist dyn xcopy dyn "%DEST%\dyn" /e /i /q >nul

echo backup in %DEST%
