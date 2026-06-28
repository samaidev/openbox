# OpenBox

> 开源、免费、跨平台的中英文双语压缩工具
> Open-source, free, cross-platform bilingual (EN/中文) archiver written in Go.

OpenBox 是一个用 Go + [Fyne](https://fyne.io) 编写的桌面压缩工具，目标是提供一个**完全免费、无广告、不收钱**的替代品，让普通用户不再被各种"会员制"压缩软件骚扰。

OpenBox is a desktop archiver built with Go + [Fyne](https://fyne.io). The goal is simple: a **completely free, ad-free, no-subscription** alternative so normal users don't get nagged by paywalls.

## Features / 功能

| Format | Compress | Extract | Notes |
|--------|:--------:|:-------:|-------|
| zip    | ✅ | ✅ | stdlib `archive/zip`, deflate or store |
| tar    | ✅ | ✅ | stdlib `archive/tar` |
| tar.gz | ✅ | ✅ | stdlib + `compress/gzip`, 5 levels |
| 7z     | ✅* | ✅ | compress via 7-Zip CLI (see below); extract via [`bodgit/sevenzip`](https://github.com/bodgit/sevenzip) |
| rar    | ❌ | ✅ | via [`nwaples/rardecode`](https://github.com/nwaples/rardecode); RAR is proprietary so no writer |
| iso    | ❌ | ✅ | via [`kdomanski/iso9660`](https://github.com/kdomanski/iso9660) |

Other features:

- 🌐 中英文一键切换，UI 语言记忆持久化（`fyne.Preferences`）
- 🔒 Path-traversal safe extraction (blocks `../../etc/passwd` style attacks)
- 📊 Progress bar + per-file log
- 🪟 Native look on Windows / macOS / Linux thanks to Fyne
- 🖱️ **Windows**: file-type associations + right-click "Compress / Extract with OpenBox"
- 🍎 **macOS**: file-type associations via Launch Services, signed .pkg installer with welcome/readme/conclusion screens
- 📦 **Windows installer** built with Inno Setup, ships in 2 languages (en/zh-CN)
- 📦 **macOS installer** built with `pkgbuild` + `productbuild`, ad-hoc signed (optional Developer ID + notarization)
- 🤖 **CI/CD**: every git tag auto-builds Win/Mac/Linux binaries + Windows installer + macOS .pkg, then attaches them to a draft Release
- 🚫 No telemetry, no ads, no paid tiers — forever

### 7z write requires the 7-Zip CLI

7z **extraction** is pure-Go (via `bodgit/sevenzip`) and works out of the box.
7z **compression** shells out to the official 7-Zip CLI because the open-source
Go ecosystem has no stable 7z writer. Install it once and OpenBox picks it up
automatically on next launch:

| Platform | Command |
|----------|---------|
| Windows  | Download from <https://www.7-zip.org/download.html> (the installer adds `7z.exe` to PATH) |
| macOS    | `brew install 7zip` (provides `7zz`) |
| Linux    | `sudo apt install p7zip-full` (Debian/Ubuntu) · `sudo dnf install p7zip` (Fedora) · `sudo pacman -S p7zip` (Arch) |

If you pick "7z" in the format dropdown but no binary is found, OpenBox shows
a clear error with install instructions.

## Download / 下载

Grab the latest release from the [Releases page](https://github.com/samaidev/openbox/releases):

| File | Platform | Notes |
|------|----------|-------|
| `OpenBox-<ver>-windows-amd64-setup.exe` | Windows 10/11 x64 | **Recommended.** Installer with file associations + right-click menu. |
| `OpenBox-<ver>-macos.pkg` | macOS Apple Silicon | **Recommended.** Installer with welcome/readme/license screens + automatic file association registration. |
| `openbox-windows-amd64.zip` | Windows 10/11 x64 | Portable — just unzip and run `openbox.exe`. No associations. |
| `openbox-macos-arm64.tar.gz` | macOS Apple Silicon | Portable. Untar, then `xattr -d com.apple.quarantine openbox` before first run. |
| `openbox-macos-amd64.tar.gz` | macOS Intel | Same as above. |
| `openbox-linux-amd64.tar.gz` | Linux x64 | Requires glibc 2.31+ and OpenGL/EGL runtime. |

## Windows Installation / Windows 安装

### Recommended: use the installer (推荐)

1. Download `OpenBox-<ver>-windows-amd64-setup.exe` from Releases.
2. Double-click to run. The installer is **bilingual** — pick English or 简体中文 on the first screen.
3. On the "Additional tasks" page you'll see:
   - ☐ Create a desktop icon
   - ☑ Associate `.zip .rar .7z .tar .tgz .tar.gz .iso` with OpenBox
4. Click **Install**. After install, Explorer will:
   - Show the OpenBox icon next to every supported archive
   - Open those archives with OpenBox when double-clicked
   - Show **"Compress with OpenBox"** when right-clicking any file or folder
   - Show **"Extract with OpenBox"** when right-clicking a `.zip/.rar/.7z/...`

### Portable (绿色版)

If you don't want installer / associations, just unzip `openbox-windows-amd64.zip`
to any folder and run `openbox.exe` directly. No registry changes, no admin
rights needed.

### Build the installer locally / 本地构建安装包

```powershell
# Prereqs: Go 1.23+ and Inno Setup 6 (https://jrsoftware.org/isdl.php)
git clone https://github.com/samaidev/openbox.git
cd openbox
pwsh build\windows\build.ps1
# Output: dist\OpenBox-0.1.0-windows-amd64-setup.exe
```

The PowerShell script:

1. Cross-checks for Inno Setup at standard paths or `%PATH%`.
2. Builds `openbox.exe` with `-H windowsgui` (no console window on launch) and embeds the app icon + version info via `cmd/openbox/resource_windows_amd64.syso`.
3. Invokes `ISCC.exe` on `build/windows/openbox.iss` to produce the installer.

### What the installer registers / 安装包注册了什么

The Inno Setup script (`build/windows/openbox.iss`) writes the following
registry entries under `HKEY_CURRENT_USER\Software\Classes` (per-user, no admin
needed at runtime even though the installer itself asks for admin to write
into `Program Files`):

| Registry key | Effect |
|---|---|
| `Software\Classes\OpenBox.Archive\shell\open\command` | Defines the ProgID: `openbox.exe "%1"` |
| `Software\Classes\.zip → OpenBox.Archive` (and `.rar / .7z / .tar / .tgz / .tar.gz / .iso`) | Double-click opens with OpenBox |
| `Software\Classes\*\shell\OpenBoxCompress` | Right-click any file → "Compress with OpenBox" |
| `Software\Classes\Directory\shell\OpenBoxCompress` | Right-click any folder → "Compress with OpenBox" |
| `Software\Classes\Directory\Background\shell\OpenBoxCompress` | Right-click inside a folder's empty space → "Compress with OpenBox" |
| `Software\Classes\Directory\shell\OpenBoxExtract` | Right-click any folder → "Extract with OpenBox" |
| `Software\Microsoft\Windows\CurrentVersion\App Paths\openbox.exe` | Win+R → `openbox` launches the GUI |

All entries carry `Flags: uninsdeletekey` so uninstalling OpenBox cleans them
up. The installer also calls `SHChangeNotify(SHCNE_ASSOCCHANGED)` so Explorer
picks up the changes immediately — no logoff required.

## macOS Installation / macOS 安装

### Recommended: use the .pkg installer (推荐)

1. Download `OpenBox-<ver>-macos.pkg` from Releases.
2. Double-click to open it in **Installer.app**.
3. Click through the welcome → readme → license → install screens.
4. Enter your password when prompted (the installer copies `OpenBox.app` to `/Applications` and runs `lsregister` to register file associations).
5. After install, double-clicking any `.zip / .tar.gz / .7z / .rar / .iso` file opens it in OpenBox. Right-click → "Open With" also lists OpenBox.

**Apple Silicon note:** the .pkg is ad-hoc signed, which is enough for personal use. On first launch macOS Gatekeeper may still prompt — right-click the app → "Open" → "Open" to whitelist it. For broad public distribution you'd want to sign with a Developer ID and notarize; see "Build locally" below for the env vars to enable that in CI.

### Portable / 绿色版

If you don't want an installer, grab `openbox-macos-arm64.tar.gz` (Apple Silicon) or `openbox-macos-amd64.tar.gz` (Intel), untar, and run:

```bash
tar xzf openbox-macos-arm64.tar.gz
xattr -d com.apple.quarantine openbox   # strip the Gatekeeper warning
./openbox
```

No file associations are registered in portable mode — double-clicking a `.zip` will still use Archive Utility.

### Build the .pkg locally / 本地构建

```bash
git clone https://github.com/samaidev/openbox.git
cd openbox
VERSION=0.1.0 bash build/macos/build.sh
# Output: dist/OpenBox-0.1.0-macos.pkg
```

The build script:

1. `go build`s the darwin binary (current architecture).
2. Assembles `OpenBox.app/Contents/{MacOS,Resources}` with `Info.plist`, `icon.icns` (generated via `sips` + `iconutil`), `LICENSE`, `README.md`.
3. Ad-hoc codesigns the bundle (required for Apple Silicon to run at all).
4. `pkgbuild`s the .app into a component package.
5. `productbuild` wraps it with the welcome/readme/license/conclusion screens from `build/macos/resources/`.
6. Optionally `productsign`s with `APPLE_DEVELOPER_ID_INSTALLER` and notarizes via `NOTARY_PROFILE` if those env vars are set.

### What the .pkg registers / 安装包注册了什么

The installer writes:

| Path | Effect |
|------|--------|
| `/Applications/OpenBox.app` | The app itself |
| `~/Library/Preferences/io.github.samaidev.openbox.plist` | User prefs (language, last-used paths) — created on first launch, not by installer |
| Launch Services registration (via `lsregister -f` in postinstall) | Tells Finder that `.zip / .tar / .tar.gz / .7z / .rar / .iso` should open with OpenBox |

The `Info.plist` declares `CFBundleDocumentTypes` for all supported archive types. Standard UTIs (`public.zip-archive`, `public.tar-archive`, `public.iso-image`) are system-defined; for `.7z` and `.rar` the Info.plist also declares `UTImportedTypeDeclarations` so macOS knows about those extensions.

To uninstall: drag `OpenBox.app` from `/Applications` to Trash. File associations become inert but harmless; macOS cleans them up on its own schedule.

## CLI / 命令行

The same `openbox.exe` understands a few CLI flags. These are what the
Windows shell integration uses under the hood, but you can call them directly:

```bash
openbox                       # launch GUI
openbox <archive>             # launch GUI with Extract tab pre-filled
openbox -c <file|dir>...      # launch GUI with Compress tab pre-filled
openbox -x <archive>          # alias for plain <archive>
openbox -version              # print version
```

## Build / 构建

### Prerequisites

- Go 1.23+ (earlier 1.21+ may work but is untested)
- CGO toolchain + system GL/X11 headers **on Linux only**:
  - Debian/Ubuntu: `sudo apt install golang gcc libgl1-mesa-dev xorg-dev`
  - Fedora: `sudo dnf install gcc mesa-libGL-devel libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel`
- On Windows: install [TDM-GCC](https://jmeubank.github.io/tdm-gcc/) or MSYS2 mingw-w64.
- On macOS: just install Xcode Command Line Tools (`xcode-select --install`).

### Compile

```bash
git clone https://github.com/samaidev/openbox.git
cd openbox
go build -o openbox ./cmd/openbox
./openbox
```

### Tests

```bash
go test ./internal/... -v
```

## Project Layout

```
openbox/
├── cmd/openbox/                          # main entry + Windows .syso resource
├── internal/
│   ├── archiver/                         # Compress/Extract engine + tests
│   │   ├── archiver.go                   #   core: zip/tar/tar.gz + 7z/rar/iso extract
│   │   ├── sevenzip_write.go             #   7z write via 7-Zip CLI subprocess
│   │   ├── gzip.go                       #   gzip helpers
│   │   └── archiver_test.go              #   round-trip tests for every format
│   ├── assets/                           # embedded icon (PNG + SVG)
│   ├── i18n/                             # bilingual string table
│   └── ui/                               # Fyne GUI
├── build/
│   ├── icons/                            # icon.ico (multi-size, Windows installer)
│   ├── windows/
│   │   ├── openbox.iss                   # Inno Setup script
│   │   ├── openbox.manifest              # DPI + common-controls manifest
│   │   ├── version.rc                    # VERSIONINFO resource
│   │   └── build.ps1                     # local Windows build helper
│   └── macos/
│       ├── Info.plist                    # app bundle metadata + file associations
│       ├── Distribution.xml              # productbuild distribution
│       ├── build.sh                      # local macOS build helper
│       ├── scripts/postinstall           # lsregister call
│       └── resources/                    # welcome/readme/license/conclusion HTML
├── scripts/
│   └── gen_icon.py                       # regenerate icon.svg/png/ico
├── .github/workflows/
│   ├── ci.yml                            # tests on every push (incl. 7z write tests)
│   └── release.yml                       # Win/Mac/Linux + Win installer + macOS .pkg
├── go.mod / go.sum
├── LICENSE                               # MIT
└── README.md
```

## CI / CD

Two workflows live under `.github/workflows/`:

- **`ci.yml`** — runs on every push / PR to `main`. Installs `p7zip-full` so the 7z write tests actually run, then does `go vet`, `go test`, and a Linux smoke build.
- **`release.yml`** — runs when you push a tag like `v0.1.0`. Builds:
  - Windows x64 portable `.zip`
  - macOS arm64 + amd64 portable `.tar.gz`
  - Linux x64 portable `.tar.gz`
  - Windows Inno Setup `.exe` installer
  - macOS `.pkg` installer (built on `macos-14`, ad-hoc signed; set `APPLE_DEVELOPER_ID_INSTALLER` + `NOTARY_PROFILE` secrets to enable full signing + notarization)
  
  Everything gets attached to a **draft GitHub Release** with auto-generated release notes. To publish, edit the draft and click "Publish release".

To cut a new release:

```bash
git tag v0.2.0
git push origin v0.2.0
# wait ~12 min for the workflow, then review the draft release
```

## Roadmap

- [x] App icon + Windows installer
- [x] Windows file-type associations + right-click menu
- [x] macOS .pkg installer with file associations
- [x] 7z compression (via 7-Zip CLI subprocess)
- [x] CI/CD with auto-release on tag push
- [ ] AES-encrypted zip extraction
- [ ] Drag-and-drop file list
- [ ] Per-file size + ratio in the compress list
- [ ] CLI companion (`openbox c|x`) for headless use without GUI launch
- [ ] Linux .deb / .rpm / Flatpak

## Acknowledgements

OpenBox stands on the shoulders of these excellent libraries:

- [`fyne.io/fyne/v2`](https://github.com/fyne-io/fyne) — cross-platform GUI
- [`github.com/bodgit/sevenzip`](https://github.com/bodgit/sevenzip) — 7z reader
- [`github.com/nwaples/rardecode`](https://github.com/nwaples/rardecode) — RAR reader
- [`github.com/kdomanski/iso9660`](https://github.com/kdomanski/iso9660) — ISO reader
- [`github.com/akavel/rsrc`](https://github.com/akavel/rsrc) — Windows `.syso` resource compiler
- [Inno Setup](https://jrsoftware.org/isinfo.php) — Windows installer framework

## License

MIT — see [LICENSE](LICENSE).

OpenBox is free software: you can use it, modify it, redistribute it, and even
fork it for your own product. The only thing we ask is: **stay free**. Don't
slap a paywall on top of community work.
