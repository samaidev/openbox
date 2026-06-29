@echo off
REM OpenBox — portable right-click menu registration (HKCU, no admin needed)
REM
REM Usage:
REM   register-context-menu-user.bat en           English menu (default)
REM   register-context-menu-user.bat zh           Chinese menu
REM   register-context-menu-user.bat bilingual    Bilingual menu (中文 / English)

setlocal

set LANG=%1
if "%LANG%"=="" set LANG=en

set OPENBOX_EXE=%~dp0openbox.exe

if not exist "%OPENBOX_EXE%" (
    echo ERROR: %OPENBOX_EXE% not found.
    echo Run this script from the folder that contains openbox.exe.
    pause
    exit /b 1
)

REM ---- Resolve menu text based on language ----
if /i "%LANG%"=="en" (
    set "COMPRESS_MENU=Compress with OpenBox"
    set "EXTRACT_MENU=Extract with OpenBox"
    set "EXTRACT_HERE_MENU=Extract here with OpenBox"
    set "ARCHIVE_DESC=OpenBox Archive"
)
if /i "%LANG%"=="zh" (
    set "COMPRESS_MENU=用 OpenBox 压缩"
    set "EXTRACT_MENU=用 OpenBox 解压"
    set "EXTRACT_HERE_MENU=用 OpenBox 解压到当前目录"
    set "ARCHIVE_DESC=OpenBox 压缩包"
)
if /i "%LANG%"=="bilingual" (
    set "COMPRESS_MENU=用 OpenBox 压缩 / Compress with OpenBox"
    set "EXTRACT_MENU=用 OpenBox 解压 / Extract with OpenBox"
    set "EXTRACT_HERE_MENU=用 OpenBox 解压到当前目录 / Extract here with OpenBox"
    set "ARCHIVE_DESC=OpenBox 压缩包 / OpenBox Archive"
)

if not defined COMPRESS_MENU (
    echo ERROR: unknown language "%LANG%". Use en, zh, or bilingual.
    pause
    exit /b 1
)

echo Registering OpenBox file associations for current user...
echo   openbox.exe = %OPENBOX_EXE%
echo   language    = %LANG%
echo.

REM ---- ProgID ----
reg add "HKCU\Software\Classes\OpenBox.Archive" /ve /d "%ARCHIVE_DESC%" /f >nul
reg add "HKCU\Software\Classes\OpenBox.Archive\DefaultIcon" /ve /d "%OPENBOX_EXE%,0" /f >nul
reg add "HKCU\Software\Classes\OpenBox.Archive\shell\open\command" /ve /d "\"%OPENBOX_EXE%\" \"%%1\"" /f >nul
reg add "HKCU\Software\Classes\OpenBox.Archive\shell\extract" /ve /d "%EXTRACT_MENU%" /f >nul
reg add "HKCU\Software\Classes\OpenBox.Archive\shell\extract\command" /ve /d "\"%OPENBOX_EXE%\" -x \"%%1\"" /f >nul
reg add "HKCU\Software\Classes\OpenBox.Archive\shell\extractHere" /ve /d "%EXTRACT_HERE_MENU%" /f >nul
reg add "HKCU\Software\Classes\OpenBox.Archive\shell\extractHere" /v Icon /d "%OPENBOX_EXE%,0" /f >nul
reg add "HKCU\Software\Classes\OpenBox.Archive\shell\extractHere\command" /ve /d "\"%OPENBOX_EXE%\" -cli -here x \"%%1\"" /f >nul

REM ---- File associations ----
for %%E in (zip 7z rar iso tar tgz) do (
    reg add "HKCU\Software\Classes\.%%E" /ve /d "OpenBox.Archive" /f >nul
    reg add "HKCU\Software\Classes\.%%E\OpenWithProgids" /v OpenBox.Archive /t REG_SZ /d "" /f >nul
)
reg add "HKCU\Software\Classes\.tar.gz" /ve /d "OpenBox.Archive" /f >nul
reg add "HKCU\Software\Classes\.tar.gz\OpenWithProgids" /v OpenBox.Archive /t REG_SZ /d "" /f >nul

REM ---- Right-click menu ----
reg add "HKCU\Software\Classes\*\shell\OpenBoxCompress" /ve /d "%COMPRESS_MENU%" /f >nul
reg add "HKCU\Software\Classes\*\shell\OpenBoxCompress" /v Icon /d "%OPENBOX_EXE%,0" /f >nul
reg add "HKCU\Software\Classes\*\shell\OpenBoxCompress\command" /ve /d "\"%OPENBOX_EXE%\" -c \"%%1\"" /f >nul

reg add "HKCU\Software\Classes\Directory\shell\OpenBoxCompress" /ve /d "%COMPRESS_MENU%" /f >nul
reg add "HKCU\Software\Classes\Directory\shell\OpenBoxCompress" /v Icon /d "%OPENBOX_EXE%,0" /f >nul
reg add "HKCU\Software\Classes\Directory\shell\OpenBoxCompress\command" /ve /d "\"%OPENBOX_EXE%\" -c \"%%1\"" /f >nul

reg add "HKCU\Software\Classes\Directory\Background\shell\OpenBoxCompress" /ve /d "%COMPRESS_MENU%" /f >nul
reg add "HKCU\Software\Classes\Directory\Background\shell\OpenBoxCompress" /v Icon /d "%OPENBOX_EXE%,0" /f >nul
reg add "HKCU\Software\Classes\Directory\Background\shell\OpenBoxCompress\command" /ve /d "\"%OPENBOX_EXE%\" -c \"%%V\"" /f >nul

reg add "HKCU\Software\Classes\Directory\shell\OpenBoxExtract" /ve /d "%EXTRACT_MENU%" /f >nul
reg add "HKCU\Software\Classes\Directory\shell\OpenBoxExtract" /v Icon /d "%OPENBOX_EXE%,0" /f >nul
reg add "HKCU\Software\Classes\Directory\shell\OpenBoxExtract\command" /ve /d "\"%OPENBOX_EXE%\" -x \"%%1\"" /f >nul

REM ---- App Paths ----
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\App Paths\openbox.exe" /ve /d "%OPENBOX_EXE%" /f >nul

echo.
echo Done. If file associations or right-click menu don't appear immediately:
echo   1. Right-click on the desktop and choose "Refresh", or
echo   2. Open Task Manager and restart "Windows Explorer".
echo.
pause
endlocal
