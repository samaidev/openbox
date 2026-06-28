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
| 7z     | ❌ | ✅ | via [`bodgit/sevenzip`](https://github.com/bodgit/sevenzip); writer unavailable in the Go ecosystem |
| rar    | ❌ | ✅ | via [`nwaples/rardecode`](https://github.com/nwaples/rardecode); RAR is proprietary so no writer |
| iso    | ❌ | ✅ | via [`kdomanski/iso9660`](https://github.com/kdomanski/iso9660) |

Other features:

- 🌐 中英文一键切换，UI 语言记忆持久化（`fyne.Preferences`）
- 🔒 Path-traversal safe extraction (blocks `../../etc/passwd` style attacks)
- 📊 Progress bar + per-file log
- 🪟 Native look on Windows / macOS / Linux thanks to Fyne
- 🚫 No telemetry, no ads, no paid tiers — forever

## Screenshot

> UI is rendered with Fyne's default theme; toggle `中文 / English` via the toolbar.
> 界面采用 Fyne 默认主题，右上角可一键切换中文 / English。

```
┌─────────────────────────────────────────────────────────────┐
│ [中文]  [Help]  [About]                                      │
├─────────────────────────────────────────────────────────────┤
│ [ Compress ] [ Extract ]                                     │
│                                                              │
│  + Add files   + Add folder   Remove   Clear                 │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ /home/z/Documents/2024-tax.pdf                         │ │
│  │ /home/z/Pictures/vacation/                              │ │
│  └────────────────────────────────────────────────────────┘ │
│  Format: [zip ▾]   Level: [Normal ▾]   Password: [•••••]    │
│  Target: [/home/z/Downloads/openbox_2024.zip]  [Browse…]    │
│                                              [ ✓ Compress ]  │
├─────────────────────────────────────────────────────────────┤
│ Working…   ▓▓▓▓▓▓▓▓░░░░  57%   [✓ Open target when done]   │
│ ┌──────── log ────────────────────────────────────────────┐ │
│ │ 14:22:01  Documents/2024-tax.pdf                         │ │
│ │ 14:22:01  Pictures/vacation/DSC_0001.jpg                │ │
│ └──────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## Build / 构建

### Prerequisites

- Go 1.23+ (earlier 1.21+ may work but is untested)
- CGO toolchain + system GL/X11 headers **on Linux only**:
  - Debian/Ubuntu: `sudo apt install golang gcc libgl1-mesa-dev xorg-dev`
  - Fedora: `sudo dnf install gcc mesa-libGL-devel libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel`
- On Windows/macOS just install a C compiler (MinGW / Xcode CLT).

### Compile

```bash
git clone https://github.com/samaidev/openbox.git
cd openbox
go build -o openbox ./cmd/openbox
./openbox
```

Cross-compile examples:

```bash
# Windows from Linux
GOOS=windows GOARCH=amd64 go build -o openbox.exe ./cmd/openbox

# macOS (note: CGO + cross-compiling to darwin is tricky; build on a Mac when possible)
GOOS=darwin GOARCH=arm64 go build -o openbox ./cmd/openbox
```

### Tests

```bash
go test ./internal/archiver/ -v
```

## Project Layout

```
openbox/
├── cmd/openbox/          # main entry
├── internal/
│   ├── archiver/         # format-agnostic Compress/Extract engine + tests
│   ├── i18n/             # bilingual string table
│   └── ui/               # Fyne GUI
├── go.mod / go.sum
├── LICENSE               # MIT
└── README.md
```

## Roadmap

- [ ] 7z write support (waiting for / will sponsor a Go 7z writer lib)
- [ ] AES-encrypted zip extraction
- [ ] Drag-and-drop file list
- [ ] Per-file size + ratio in the compress list
- [ ] CLI companion (`openbox c|x`) for headless use
- [ ] App icons + signed installers per platform

## Acknowledgements

OpenBox stands on the shoulders of these excellent libraries:

- [`fyne.io/fyne/v2`](https://github.com/fyne-io/fyne) — cross-platform GUI
- [`github.com/bodgit/sevenzip`](https://github.com/bodgit/sevenzip) — 7z reader
- [`github.com/nwaples/rardecode`](https://github.com/nwaples/rardecode) — RAR reader
- [`github.com/kdomanski/iso9660`](https://github.com/kdomanski/iso9660) — ISO reader

## License

MIT — see [LICENSE](LICENSE).

OpenBox is free software: you can use it, modify it, redistribute it, and even
fork it for your own product. The only thing we ask is: **stay free**. Don't
slap a paywall on top of community work.
