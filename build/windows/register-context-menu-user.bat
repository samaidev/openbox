@echo off
REM OpenBox — portable right-click menu registration (HKCU, no admin needed)
REM
REM Same as register-context-menu.bat but writes to HKCU instead of HKLM,
REM so you don't need Administrator privileges. The entries only show up
REM for the current user.

setlocal
set OPENBOX_EXE=%~dp0openbox.exe

if not exist "%OPENBOX_EXE%" (
    echo ERROR: %OPENBOX_EXE% not found.
    echo Run this script from the folder that contains openbox.exe.
    pause
    exit /b 1
)

echo Registering OpenBox file associations for current user...
echo   openbox.exe = %OPENBOX_EXE%
echo.

REM ---- ProgID ----
reg add "HKCU\Software\Classes\OpenBox.Archive" /ve /d "OpenBox Archive" /f >nul
reg add "HKCU\Software\Classes\OpenBox.Archive\DefaultIcon" /ve /d "%OPENBOX_EXE%,0" /f >nul
reg add "HKCU\Software\Classes\OpenBox.Archive\shell\open\command" /ve /d "\"%OPENBOX_EXE%\" \"%%1\"" /f >nul
reg add "HKCU\Software\Classes\OpenBox.Archive\shell\extract" /ve /d "Extract with OpenBox" /f >nul
reg add "HKCU\Software\Classes\OpenBox.Archive\shell\extract\command" /ve /d "\"%OPENBOX_EXE%\" -x \"%%1\"" /f >nul
reg add "HKCU\Software\Classes\OpenBox.Archive\shell\extractHere" /ve /d "Extract here with OpenBox" /f >nul
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
reg add "HKCU\Software\Classes\*\shell\OpenBoxCompress" /ve /d "Compress with OpenBox" /f >nul
reg add "HKCU\Software\Classes\*\shell\OpenBoxCompress" /v Icon /d "%OPENBOX_EXE%,0" /f >nul
reg add "HKCU\Software\Classes\*\shell\OpenBoxCompress\command" /ve /d "\"%OPENBOX_EXE%\" -c \"%%1\"" /f >nul

reg add "HKCU\Software\Classes\Directory\shell\OpenBoxCompress" /ve /d "Compress with OpenBox" /f >nul
reg add "HKCU\Software\Classes\Directory\shell\OpenBoxCompress" /v Icon /d "%OPENBOX_EXE%,0" /f >nul
reg add "HKCU\Software\Classes\Directory\shell\OpenBoxCompress\command" /ve /d "\"%OPENBOX_EXE%\" -c \"%%1\"" /f >nul

reg add "HKCU\Software\Classes\Directory\Background\shell\OpenBoxCompress" /ve /d "Compress with OpenBox" /f >nul
reg add "HKCU\Software\Classes\Directory\Background\shell\OpenBoxCompress" /v Icon /d "%OPENBOX_EXE%,0" /f >nul
reg add "HKCU\Software\Classes\Directory\Background\shell\OpenBoxCompress\command" /ve /d "\"%OPENBOX_EXE%\" -c \"%%V\"" /f >nul

reg add "HKCU\Software\Classes\Directory\shell\OpenBoxExtract" /ve /d "Extract with OpenBox" /f >nul
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
