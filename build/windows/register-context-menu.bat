@echo off
REM OpenBox — portable right-click menu registration
REM
REM Use this if you downloaded the portable .zip build instead of the
REM installer, or if you need to re-register the right-click entries
REM after Windows reset file associations.
REM
REM Run as Administrator. Writes to HKLM\Software\Classes so the entries
REM appear for all users. Use register-context-menu-user.bat for HKCU
REM (current user only, no admin needed).

setlocal
set OPENBOX_EXE=%~dp0openbox.exe

if not exist "%OPENBOX_EXE%" (
    echo ERROR: %OPENBOX_EXE% not found.
    echo Run this script from the folder that contains openbox.exe.
    pause
    exit /b 1
)

net session >nul 2>&1
if %errorlevel% neq 0 (
    echo ERROR: This script needs Administrator privileges.
    echo Right-click the .bat file and choose "Run as administrator".
    pause
    exit /b 1
)

echo Registering OpenBox file associations and right-click menu...
echo   openbox.exe = %OPENBOX_EXE%
echo.

REM ---- ProgID ----
reg add "HKLM\Software\Classes\OpenBox.Archive" /ve /d "OpenBox Archive" /f >nul
reg add "HKLM\Software\Classes\OpenBox.Archive\DefaultIcon" /ve /d "%OPENBOX_EXE%,0" /f >nul
reg add "HKLM\Software\Classes\OpenBox.Archive\shell\open\command" /ve /d "\"%OPENBOX_EXE%\" \"%%1\"" /f >nul
reg add "HKLM\Software\Classes\OpenBox.Archive\shell\extract" /ve /d "Extract with OpenBox" /f >nul
reg add "HKLM\Software\Classes\OpenBox.Archive\shell\extract\command" /ve /d "\"%OPENBOX_EXE%\" -x \"%%1\"" /f >nul
reg add "HKLM\Software\Classes\OpenBox.Archive\shell\extractHere" /ve /d "Extract here with OpenBox" /f >nul
reg add "HKLM\Software\Classes\OpenBox.Archive\shell\extractHere" /v Icon /d "%OPENBOX_EXE%,0" /f >nul
reg add "HKLM\Software\Classes\OpenBox.Archive\shell\extractHere\command" /ve /d "\"%OPENBOX_EXE%\" -cli -here x \"%%1\"" /f >nul

REM ---- File associations: .zip .7z .rar .iso .tar .tar.gz .tgz ----
for %%E in (zip 7z rar iso tar tgz) do (
    reg add "HKLM\Software\Classes\.%%E" /ve /d "OpenBox.Archive" /f >nul
    reg add "HKLM\Software\Classes\.%%E\OpenWithProgids" /v OpenBox.Archive /t REG_SZ /d "" /f >nul
)
REM .tar.gz (compound extension)
reg add "HKLM\Software\Classes\.tar.gz" /ve /d "OpenBox.Archive" /f >nul
reg add "HKLM\Software\Classes\.tar.gz\OpenWithProgids" /v OpenBox.Archive /t REG_SZ /d "" /f >nul

REM ---- Right-click "Compress with OpenBox" on files, folders, folder backgrounds ----
reg add "HKLM\Software\Classes\*\shell\OpenBoxCompress" /ve /d "Compress with OpenBox" /f >nul
reg add "HKLM\Software\Classes\*\shell\OpenBoxCompress" /v Icon /d "%OPENBOX_EXE%,0" /f >nul
reg add "HKLM\Software\Classes\*\shell\OpenBoxCompress\command" /ve /d "\"%OPENBOX_EXE%\" -c \"%%1\"" /f >nul

reg add "HKLM\Software\Classes\Directory\shell\OpenBoxCompress" /ve /d "Compress with OpenBox" /f >nul
reg add "HKLM\Software\Classes\Directory\shell\OpenBoxCompress" /v Icon /d "%OPENBOX_EXE%,0" /f >nul
reg add "HKLM\Software\Classes\Directory\shell\OpenBoxCompress\command" /ve /d "\"%OPENBOX_EXE%\" -c \"%%1\"" /f >nul

reg add "HKLM\Software\Classes\Directory\Background\shell\OpenBoxCompress" /ve /d "Compress with OpenBox" /f >nul
reg add "HKLM\Software\Classes\Directory\Background\shell\OpenBoxCompress" /v Icon /d "%OPENBOX_EXE%,0" /f >nul
reg add "HKLM\Software\Classes\Directory\Background\shell\OpenBoxCompress\command" /ve /d "\"%OPENBOX_EXE%\" -c \"%%V\"" /f >nul

REM ---- Right-click "Extract with OpenBox" on directories ----
reg add "HKLM\Software\Classes\Directory\shell\OpenBoxExtract" /ve /d "Extract with OpenBox" /f >nul
reg add "HKLM\Software\Classes\Directory\shell\OpenBoxExtract" /v Icon /d "%OPENBOX_EXE%,0" /f >nul
reg add "HKLM\Software\Classes\Directory\shell\OpenBoxExtract\command" /ve /d "\"%OPENBOX_EXE%\" -x \"%%1\"" /f >nul

REM ---- App Paths ----
reg add "HKLM\Software\Microsoft\Windows\CurrentVersion\App Paths\openbox.exe" /ve /d "%OPENBOX_EXE%" /f >nul
reg add "HKLM\Software\Microsoft\Windows\CurrentVersion\App Paths\openbox.exe" /v Path /d "%~dp0" /f >nul

REM ---- Notify shell ----
REM We can't easily call SHChangeNotify from cmd.exe; tell the user to
REM log off / log on if associations don't refresh immediately.
echo.
echo Done. If file associations or right-click menu don't appear immediately:
echo   1. Right-click on the desktop and choose "Refresh", or
echo   2. Log off and log back on, or
echo   3. Open Task Manager and restart "Windows Explorer".
echo.
pause
endlocal
