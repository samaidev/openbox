; OpenBox — Inno Setup script
; ----------------------------------------------------------------------------
; Builds the Windows installer with:
;   * 64-bit install to {autopf}\OpenBox
;   * Start Menu + Desktop shortcuts (Desktop optional via component)
;   * File type associations for .zip / .rar / .7z / .tar / .tgz / .tar.gz / .iso
;   * Right-click "Compress with OpenBox" on files and folders
;   * Right-click "Extract with OpenBox" on the supported archive types
;   * Clean uninstall (associations + context menu entries removed)
;
; Build:
;   iscc build\windows\openbox.iss
;
; The installer is published as a release artifact by .github/workflows/release.yml.
; ----------------------------------------------------------------------------

#define MyAppName        "OpenBox"
#define MyAppPublisher   "samaidev"
#define MyAppURL         "https://github.com/samaidev/openbox"
#define MyAppExeName     "openbox.exe"
#define MyAppVersion     "0.3.0"
#define MyAppVersionFull "0.3.0.0"

[Setup]
; NOTE: The value of AppId uniquely identifies this application.
; Do not use the same AppId value in installers for other applications.
AppId={{8B7E4F2A-3D9C-4E1A-9F5B-6C2D8E1A7B3F}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppVerName={#MyAppName} {#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}/issues
AppUpdatesURL={#MyAppURL}/releases
AppCopyright=Copyright (c) 2026 {#MyAppPublisher} (MIT)
LicenseFile=..\..\LICENSE
DefaultDirName={autopf}\{#MyAppName}
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=yes
OutputDir=..\..\dist
OutputBaseFilename=OpenBox-{#MyAppVersion}-windows-amd64-setup
SetupIconFile=..\icons\icon.ico
Compression=lzma2/ultra64
SolidCompression=yes
WizardStyle=modern
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
PrivilegesRequired=admin
UninstallDisplayIcon={app}\{#MyAppExeName}
UninstallDisplayName={#MyAppName}
; Show the language picker on first install (defaults to user OS language).
; ChineseSimplified.isl is NOT bundled with Inno Setup 6.5+ — download it
; from https://raw.githubusercontent.com/jrsoftware/issrc/main/Files/Languages/ChineseSimplified.isl
; into the Inno Setup Languages folder before building this .iss.
ShowLanguageDialog=yes
LanguageDetectionMethod=uilanguage

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"
Name: "chinesesimplified"; MessagesFile: "compiler:Languages\ChineseSimplified.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked
Name: "associate"; Description: "{cm:AssocFileExtension,OpenBox,.zip .rar .7z .tar .tgz .tar.gz .iso}"; GroupDescription: "File associations:"

[Files]
; openbox.exe is expected to be in dist\openbox-windows-amd64\openbox.exe
; (produced by `go build` + staged by build-installer.bat / CI workflow).
; Paths in [Files] are relative to the .iss file location (build/windows/),
; so ..\..\ resolves to the repo root.
Source: "..\..\dist\openbox-windows-amd64\openbox.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\LICENSE"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\README.md"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"
Name: "{group}\{cm:UninstallProgram,{#MyAppName}}"; Filename: "{uninstallexe}"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon

; ----------------------------------------------------------------------------
; Run once after install so associations + right-click entries take effect
; without requiring the user to log off.
; ----------------------------------------------------------------------------
[Run]
Filename: "{app}\{#MyAppExeName}"; Description: "{cm:LaunchProgram,{#MyAppName}}"; Flags: nowait postinstall skipifsilent

; ----------------------------------------------------------------------------
; Registry: file associations.
;   *.ext -> ProgID "OpenBox.Archive" -> open with openbox.exe "%1"
;   OpenWithProgids also adds OpenBox to the "Open with" submenu.
;
; We list each extension explicitly rather than via a #define macro because
; Inno Setup's preprocessor doesn't let you emit multiple [Registry] entries
; from a single macro call (the `Root:` keyword is parsed in the data layer,
; not the preprocessor layer).
; ----------------------------------------------------------------------------
[Registry]
; Define the ProgID once.
Root: HKA; Subkey: "Software\Classes\OpenBox.Archive"; ValueType: string; ValueName: ""; ValueData: "OpenBox Archive"; Flags: uninsdeletekey
Root: HKA; Subkey: "Software\Classes\OpenBox.Archive\DefaultIcon"; ValueType: string; ValueName: ""; ValueData: "{app}\{#MyAppExeName},0"; Flags: uninsdeletekey
Root: HKA; Subkey: "Software\Classes\OpenBox.Archive\shell\open\command"; ValueType: string; ValueName: ""; ValueData: """{app}\{#MyAppExeName}"" ""%1"""; Flags: uninsdeletekey
Root: HKA; Subkey: "Software\Classes\OpenBox.Archive\shell\extract"; ValueType: string; ValueName: ""; ValueData: "Extract with OpenBox"; Flags: uninsdeletekey
Root: HKA; Subkey: "Software\Classes\OpenBox.Archive\shell\extract\command"; ValueType: string; ValueName: ""; ValueData: """{app}\{#MyAppExeName}"" -x ""%1"""; Flags: uninsdeletekey
; "Extract here" — extracts silently to <archive-parent-dir>\<archive-basename>\
; without launching the GUI. Matches the 7-Zip / WinRAR right-click entry
; that most users expect.
Root: HKA; Subkey: "Software\Classes\OpenBox.Archive\shell\extractHere"; ValueType: string; ValueName: ""; ValueData: "Extract here with OpenBox"; Flags: uninsdeletekey
Root: HKA; Subkey: "Software\Classes\OpenBox.Archive\shell\extractHere"; ValueType: string; ValueName: "Icon"; ValueData: "{app}\{#MyAppExeName},0"; Flags: uninsdeletekey
Root: HKA; Subkey: "Software\Classes\OpenBox.Archive\shell\extractHere\command"; ValueType: string; ValueName: ""; ValueData: """{app}\{#MyAppExeName}"" -cli -here x ""%1"""; Flags: uninsdeletekey

; .zip
Root: HKA; Subkey: "Software\Classes\.zip"; ValueType: string; ValueName: ""; ValueData: "OpenBox.Archive"; Flags: uninsdeletekey
Root: HKA; Subkey: "Software\Classes\.zip\OpenWithProgids"; ValueType: string; ValueName: "OpenBox.Archive"; ValueData: ""; Flags: uninsdeletekey
; .rar
Root: HKA; Subkey: "Software\Classes\.rar"; ValueType: string; ValueName: ""; ValueData: "OpenBox.Archive"; Flags: uninsdeletekey
Root: HKA; Subkey: "Software\Classes\.rar\OpenWithProgids"; ValueType: string; ValueName: "OpenBox.Archive"; ValueData: ""; Flags: uninsdeletekey
; .7z
Root: HKA; Subkey: "Software\Classes\.7z"; ValueType: string; ValueName: ""; ValueData: "OpenBox.Archive"; Flags: uninsdeletekey
Root: HKA; Subkey: "Software\Classes\.7z\OpenWithProgids"; ValueType: string; ValueName: "OpenBox.Archive"; ValueData: ""; Flags: uninsdeletekey
; .tar
Root: HKA; Subkey: "Software\Classes\.tar"; ValueType: string; ValueName: ""; ValueData: "OpenBox.Archive"; Flags: uninsdeletekey
Root: HKA; Subkey: "Software\Classes\.tar\OpenWithProgids"; ValueType: string; ValueName: "OpenBox.Archive"; ValueData: ""; Flags: uninsdeletekey
; .tgz
Root: HKA; Subkey: "Software\Classes\.tgz"; ValueType: string; ValueName: ""; ValueData: "OpenBox.Archive"; Flags: uninsdeletekey
Root: HKA; Subkey: "Software\Classes\.tgz\OpenWithProgids"; ValueType: string; ValueName: "OpenBox.Archive"; ValueData: ""; Flags: uninsdeletekey
; .tar.gz (compound extension; registered as a whole)
Root: HKA; Subkey: "Software\Classes\.tar.gz"; ValueType: string; ValueName: ""; ValueData: "OpenBox.Archive"; Flags: uninsdeletekey
Root: HKA; Subkey: "Software\Classes\.tar.gz\OpenWithProgids"; ValueType: string; ValueName: "OpenBox.Archive"; ValueData: ""; Flags: uninsdeletekey
; .iso
Root: HKA; Subkey: "Software\Classes\.iso"; ValueType: string; ValueName: ""; ValueData: "OpenBox.Archive"; Flags: uninsdeletekey
Root: HKA; Subkey: "Software\Classes\.iso\OpenWithProgids"; ValueType: string; ValueName: "OpenBox.Archive"; ValueData: ""; Flags: uninsdeletekey

; ----------------------------------------------------------------------------
; Right-click "Compress with OpenBox" on ANY file or folder.
; Uses HKCR\*\shell and HKCR\Directory\shell\... so the entry appears on:
;   * files (right-click in Explorer or in a folder background)
;   * directories (right-click on a folder)
;   * directory background (right-click on empty space inside a folder)
; ----------------------------------------------------------------------------
Root: HKA; Subkey: "Software\Classes\*\shell\OpenBoxCompress"; ValueType: string; ValueName: ""; ValueData: "Compress with OpenBox"; Flags: uninsdeletekey
Root: HKA; Subkey: "Software\Classes\*\shell\OpenBoxCompress"; ValueType: string; ValueName: "Icon"; ValueData: "{app}\{#MyAppExeName},0"; Flags: uninsdeletekey
Root: HKA; Subkey: "Software\Classes\*\shell\OpenBoxCompress\command"; ValueType: string; ValueName: ""; ValueData: """{app}\{#MyAppExeName}"" -c ""%1"""; Flags: uninsdeletekey

Root: HKA; Subkey: "Software\Classes\Directory\shell\OpenBoxCompress"; ValueType: string; ValueName: ""; ValueData: "Compress with OpenBox"; Flags: uninsdeletekey
Root: HKA; Subkey: "Software\Classes\Directory\shell\OpenBoxCompress"; ValueType: string; ValueName: "Icon"; ValueData: "{app}\{#MyAppExeName},0"; Flags: uninsdeletekey
Root: HKA; Subkey: "Software\Classes\Directory\shell\OpenBoxCompress\command"; ValueType: string; ValueName: ""; ValueData: """{app}\{#MyAppExeName}"" -c ""%1"""; Flags: uninsdeletekey

Root: HKA; Subkey: "Software\Classes\Directory\Background\shell\OpenBoxCompress"; ValueType: string; ValueName: ""; ValueData: "Compress with OpenBox"; Flags: uninsdeletekey
Root: HKA; Subkey: "Software\Classes\Directory\Background\shell\OpenBoxCompress"; ValueType: string; ValueName: "Icon"; ValueData: "{app}\{#MyAppExeName},0"; Flags: uninsdeletekey
Root: HKA; Subkey: "Software\Classes\Directory\Background\shell\OpenBoxCompress\command"; ValueType: string; ValueName: ""; ValueData: """{app}\{#MyAppExeName}"" -c ""%V"""; Flags: uninsdeletekey

; Also add "Extract with OpenBox" on Directory entries (e.g. right-click a .zip
; that's also a folder? rare, but cheap to add).
Root: HKA; Subkey: "Software\Classes\Directory\shell\OpenBoxExtract"; ValueType: string; ValueName: ""; ValueData: "Extract with OpenBox"; Flags: uninsdeletekey
Root: HKA; Subkey: "Software\Classes\Directory\shell\OpenBoxExtract"; ValueType: string; ValueName: "Icon"; ValueData: "{app}\{#MyAppExeName},0"; Flags: uninsdeletekey
Root: HKA; Subkey: "Software\Classes\Directory\shell\OpenBoxExtract\command"; ValueType: string; ValueName: ""; ValueData: """{app}\{#MyAppExeName}"" -x ""%1"""; Flags: uninsdeletekey

; ----------------------------------------------------------------------------
; App Paths so users can `Win+R` -> `openbox` and launch the GUI.
; ----------------------------------------------------------------------------
Root: HKA; Subkey: "Software\Microsoft\Windows\CurrentVersion\App Paths\{#MyAppExeName}"; ValueType: string; ValueName: ""; ValueData: "{app}\{#MyAppExeName}"; Flags: uninsdeletekey
Root: HKA; Subkey: "Software\Microsoft\Windows\CurrentVersion\App Paths\{#MyAppExeName}"; ValueType: string; ValueName: "Path"; ValueData: "{app}"; Flags: uninsdeletekey

; ----------------------------------------------------------------------------
; Notify the shell that file associations + context menu have changed.
;
; We declare SHChangeNotify as an external function from shell32.dll so we
; can call it directly. The previous approach of using rundll32 to invoke
; SHChangeNotify did NOT work — rundll32 can only call functions that take
; string arguments, but SHChangeNotify takes 4 numeric parameters.
; ----------------------------------------------------------------------------
[Code]
const
  SHCNE_ASSOCCHANGED = $08000000;
  SHCNF_IDLIST = $0000;

procedure SHChangeNotify(wEventId: LongWord; uFlags: Cardinal; dwItem1: LongWord; dwItem2: LongWord);
external 'SHChangeNotify@shell32.dll stdcall';

procedure NotifyShellOfAssocChange;
begin
  SHChangeNotify(SHCNE_ASSOCCHANGED, SHCNF_IDLIST, 0, 0);
end;

procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep = ssPostInstall then
    NotifyShellOfAssocChange;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  if CurUninstallStep = usPostUninstall then
    NotifyShellOfAssocChange;
end;
