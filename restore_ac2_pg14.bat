@echo off
setlocal EnableExtensions EnableDelayedExpansion
REM Restore ac2_for_pg14.sql onto PostgreSQL (target ideally 14).
REM Usage:
REM   restore_ac2_pg14.bat [host] [db-name] [pg-password]
REM Optional:
REM   set PGBIN=C:\Program Files\PostgreSQL\14\bin

set "HOST=%~1"
set "DB=%~2"
set "PASS=%~3"
if "%HOST%"=="" set "HOST=localhost"
if "%DB%"=="" set "DB=ac2"
if not "%PASS%"=="" set "PGPASSWORD=%PASS%"
if "%PGPASSWORD%"=="" set "PGPASSWORD=postgres"

set "DUMP=%~dp0ac2_for_pg14.sql"
if not exist "%DUMP%" (
  echo ERROR: dump not found: %DUMP%
  exit /b 1
)

set "PGBIN_RESOLVED="
if defined PGBIN if exist "%PGBIN%\psql.exe" set "PGBIN_RESOLVED=%PGBIN%"

if not defined PGBIN_RESOLVED (
  for %%V in (14 15 16 17 13 12) do (
    if not defined PGBIN_RESOLVED if exist "C:\Program Files\PostgreSQL\%%V\bin\psql.exe" (
      set "PGBIN_RESOLVED=C:\Program Files\PostgreSQL\%%V\bin"
    )
    if not defined PGBIN_RESOLVED if exist "C:\Program Files (x86)\PostgreSQL\%%V\bin\psql.exe" (
      set "PGBIN_RESOLVED=C:\Program Files (x86)\PostgreSQL\%%V\bin"
    )
  )
)

if not defined PGBIN_RESOLVED (
  where psql >nul 2>nul
  if not errorlevel 1 (
    for /f "delims=" %%I in ('where psql 2^>nul') do (
      if not defined PGBIN_RESOLVED (
        set "PGBIN_RESOLVED=%%~dpI"
        if "!PGBIN_RESOLVED:~-1!"=="\" set "PGBIN_RESOLVED=!PGBIN_RESOLVED:~0,-1!"
      )
    )
  )
)

if not defined PGBIN_RESOLVED (
  echo.
  echo ERROR: psql.exe topilmadi ^(PATH da yo'q^).
  echo PostgreSQL o'rnatilgan bo'lsa, masalan:
  echo   set PGBIN=C:\Program Files\PostgreSQL\14\bin
  echo   restore_ac2_pg14.bat %HOST% %DB%
  exit /b 1
)

echo Using: !PGBIN_RESOLVED!
echo Creating database %DB% on %HOST% ...
"!PGBIN_RESOLVED!\createdb.exe" -h %HOST% -U postgres %DB% 2>nul

echo Restoring...
"!PGBIN_RESOLVED!\psql.exe" -h %HOST% -U postgres -d %DB% -v ON_ERROR_STOP=1 -f "%DUMP%"
if errorlevel 1 (
  echo ERROR: restore failed.
  exit /b 1
)

echo Done.
exit /b 0
