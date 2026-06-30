// Package ui builds the Fyne-based bilingual UI for OpenBox.
package ui

import (
        "fmt"
        "net/url"
        "os"
        "path/filepath"
        "runtime"
        "sort"
        "strconv"
        "strings"
        "time"

        "fyne.io/fyne/v2"
        "fyne.io/fyne/v2/container"
        "fyne.io/fyne/v2/dialog"
        "fyne.io/fyne/v2/layout"
        "fyne.io/fyne/v2/storage"
        "fyne.io/fyne/v2/theme"
        "fyne.io/fyne/v2/widget"

        "fyne.io/fyne/v2/driver/desktop"

        "github.com/samaidev/openbox/internal/archiver"
        "github.com/samaidev/openbox/internal/i18n"
)

// App holds all UI state.
type App struct {
        win fyne.Window
        app fyne.App

        // compress tab
        srcList     *widget.List
        srcItems    []string
        selectedIdx int // -1 when nothing selected
        addFile     *widget.Button
        addFolder   *widget.Button
        removeSel   *widget.Button
        clearAll    *widget.Button
        formatSel   *widget.Select
        levelSel    *widget.Select
        pwEntry     *widget.Entry
        targetPath  *widget.Entry
        volEntry    *widget.Entry
        browseOut   *widget.Button
        compressBtn *widget.Button

        // extract tab
        exSrc          *widget.Entry
        exBrowse       *widget.Button
        exDest         *widget.Entry
        exDestBrw      *widget.Button
        exPw           *widget.Entry
        extractBtn     *widget.Button
        extractHereBtn *widget.Button
        extractSelBtn  *widget.Button
        // archive contents browser — file-browser style (like Windows Explorer).
        // contentAll is the full entry list from archiver.List.
        // contentItems is the currently-visible subset: only the DIRECT
        // children of currentPath, optionally filtered by the Find query,
        // sorted by sortField/sortAscending.
        contentList      *widget.List
        contentItems     []archiver.Entry
        contentAll       []archiver.Entry
        contentFilter    string
        contentLbl       *widget.Label
        contentSelIdx    int // -1 when nothing selected
        headerRow        *fyne.Container
        currentPath      string // folder path we're viewing ("" = root, "subfolder/" = inside subfolder)
        pathLbl          *widget.Label // breadcrumb showing currentPath
        upBtn            *widget.Button // navigate to parent folder
        sortField        string // "name", "size", "type", "modified"
        sortAscending    bool
        // WinRAR-style toolbar buttons (Extract tab)
        findBtn          *widget.Button
        addBtn           *widget.Button
        deleteBtn        *widget.Button
        findBar          *fyne.Container
        findEntry        *widget.Entry
        findCloseBtn     *widget.Button

        // shared
        progress  *widget.ProgressBar
        statusLbl *widget.Label
        logView   *widget.Entry
        langBtn   *widget.Button
        aboutBtn  *widget.Button
        helpBtn   *widget.Button
        openAfter *widget.Check
        tabs      *container.AppTabs
        // form labels (need updating on language switch)
        srcLbl        *widget.Label // "Source" / "源文件"
        tgtLbl        *widget.Label // "Target" / "目标位置"
        pwLbl         *widget.Label // "Password (optional)" / "密码（可选）"
        exSrcLbl      *widget.Label
        exTgtLbl      *widget.Label
        exPwLbl       *widget.Label
        fmtLbl        *widget.Label // "Format" / "格式"
        lvlLbl        *widget.Label // "Level" / "压缩级别"
        volLbl        *widget.Label // "Split volume" / "分卷大小"
        contentsHdrLbl *widget.Label // "Archive contents" / "压缩包内容"
        // background-mode UI
        backgroundBtn  *widget.Button
        cancelBtn      *widget.Button
        backgroundMode bool   // true when user clicked 'Background' while a task was running
        lastTaskDesc   string // human-readable description of the current/last task

        // state
        busy bool
}

const prefLang = "ui.lang"

// Tab identifies which tab to focus on startup.
type Tab int

const (
        TabCompress Tab = iota
        TabExtract
)

// InitialState carries CLI-provided inputs that pre-fill the UI on launch.
// It's used by the Windows file-association and shell-context-menu flows:
// double-clicking a .zip launches OpenBox with Tab=TabExtract and Archive set,
// right-clicking a folder and picking "Compress with OpenBox" launches with
// Tab=TabCompress and Sources set.
type InitialState struct {
        Tab     Tab
        Sources []string // pre-filled Compress sources
        Archive string   // pre-filled Extract source
}

// New constructs the application window with no pre-filled inputs.
func New(a fyne.App) fyne.Window {
        return NewWithState(a, nil)
}

// NewWithState constructs the application window, optionally pre-filling the
// Compress or Extract tab from CLI arguments.
func NewWithState(a fyne.App, init *InitialState) fyne.Window {
        w := a.NewWindow(i18n.T(i18n.AppTitle))
        w.Resize(fyne.NewSize(920, 640))

        app := &App{win: w, app: a, selectedIdx: -1}

        saved := a.Preferences().StringWithFallback(prefLang, "en")
        if saved == "zh" {
                i18n.Set(i18n.SimplifiedChinese)
        }

        app.langBtn = widget.NewButton(i18n.T(i18n.LanguageToggle), func() {
                i18n.Toggle()
                if i18n.Get() == i18n.SimplifiedChinese {
                        a.Preferences().SetString(prefLang, "zh")
                } else {
                        a.Preferences().SetString(prefLang, "en")
                }
                app.rebuild()
        })

        app.aboutBtn = widget.NewButton(i18n.T(i18n.About), func() {
                app.showAboutDialog()
        })
        app.helpBtn = widget.NewButton(i18n.T(i18n.HelpBtn), func() {
                dialog.ShowInformation(i18n.T(i18n.HelpBtn), i18n.T(i18n.HelpText), w)
        })

        toolbar := container.NewHBox(app.langBtn, app.helpBtn, app.aboutBtn)

        app.buildCompressTab()
        app.buildExtractTab()

        app.tabs = container.NewAppTabs(
                container.NewTabItem(i18n.T(i18n.TabCompress), app.compressPanel()),
                container.NewTabItem(i18n.T(i18n.TabExtract), app.extractPanel()),
        )

        app.progress = widget.NewProgressBar()
        app.progress.Hide()
        app.statusLbl = widget.NewLabel(i18n.T(i18n.StatusIdle))
        app.logView = widget.NewMultiLineEntry()
        app.logView.SetMinRowsVisible(6)
        app.logView.Disable()
        app.openAfter = widget.NewCheck(i18n.T(i18n.OpenAfterDone), nil)

        // Background + Cancel buttons — only visible while a task is running.
        app.backgroundBtn = widget.NewButtonWithIcon(i18n.T(i18n.Background), theme.ViewRestoreIcon(), app.doBackground)
        app.cancelBtn = widget.NewButtonWithIcon(i18n.T(i18n.CancelTask), theme.CancelIcon(), app.doCancel)
        app.backgroundBtn.Hide()
        app.cancelBtn.Hide()

        bottom := container.NewVBox(
                container.NewHBox(app.statusLbl, app.progress, layout.NewSpacer(),
                        app.backgroundBtn, app.cancelBtn, app.openAfter),
                container.NewMax(app.logView),
        )

        // Drag-and-drop support: when the user drags files from Windows
        // Explorer (or macOS Finder / a Linux file manager) onto the OpenBox
        // window, we add them to the current tab:
        //   - Compress tab: add to the source list.
        //   - Extract tab: if an archive is open, add the dropped files to it;
        //     if no archive is open and the dropped file IS an archive, open it.
        w.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {
                app.onDropped(pos, uris)
        })

        w.SetContent(container.NewBorder(toolbar, bottom, nil, nil, app.tabs))

        // Apply CLI-provided initial state.
        if init != nil {
                app.applyInitialState(init)
        }

        w.SetCloseIntercept(func() {
                if app.busy {
                        dialog.ShowConfirm(i18n.T(i18n.AppTitle), i18n.T(i18n.ConfirmQuit), func(ok bool) {
                                if ok {
                                        w.Close()
                                }
                        }, w)
                } else {
                        w.Close()
                }
        })

        return w
}

// rebuild refreshes widget captions after a language change.
func (a *App) rebuild() {
        a.win.SetTitle(i18n.T(i18n.AppTitle))
        a.langBtn.SetText(i18n.T(i18n.LanguageToggle))
        a.aboutBtn.SetText(i18n.T(i18n.About))
        a.helpBtn.SetText(i18n.T(i18n.HelpBtn))
        a.compressBtn.SetText(i18n.T(i18n.Compress))
        a.extractBtn.SetText(i18n.T(i18n.Extract))
        a.extractHereBtn.SetText(i18n.T(i18n.ExtractHere))
        a.extractSelBtn.SetText(i18n.T(i18n.ExtractSelected))
        a.backgroundBtn.SetText(i18n.T(i18n.Background))
        a.cancelBtn.SetText(i18n.T(i18n.CancelTask))
        a.addFile.SetText(i18n.T(i18n.AddFiles))
        a.addFolder.SetText(i18n.T(i18n.AddFolder))
        a.removeSel.SetText(i18n.T(i18n.RemoveSelected))
        a.clearAll.SetText(i18n.T(i18n.ClearAll))
        a.browseOut.SetText(i18n.T(i18n.BrowseDir))
        a.exBrowse.SetText(i18n.T(i18n.Browse))
        a.exDestBrw.SetText(i18n.T(i18n.BrowseDir))
        a.openAfter.SetText(i18n.T(i18n.OpenAfterDone))
        // WinRAR-style toolbar buttons (Extract tab)
        a.findBtn.SetText(i18n.T(i18n.Find))
        a.addBtn.SetText(i18n.T(i18n.AddToArchive))
        a.deleteBtn.SetText(i18n.T(i18n.DeleteEntry))
        a.findCloseBtn.SetText(i18n.T(i18n.CloseFind))
        a.findEntry.SetPlaceHolder(i18n.T(i18n.FindPlaceholder))
        // Refresh the column-header labels so they follow the active language.
        a.refreshHeader()
        // Up button label follows language too.
        if a.upBtn != nil {
                a.upBtn.SetText(i18n.T(i18n.UpBtn))
        }
        if !a.busy {
                a.statusLbl.SetText(i18n.T(i18n.StatusIdle))
        }
        a.levelSel.Options = []string{
                i18n.T(i18n.LevelStore), i18n.T(i18n.LevelFastest), i18n.T(i18n.LevelFast),
                i18n.T(i18n.LevelNormal), i18n.T(i18n.LevelMax),
        }
        a.levelSel.SetSelected(i18n.T(i18n.LevelNormal))
        if len(a.tabs.Items) >= 2 {
                a.tabs.Items[0].Text = i18n.T(i18n.TabCompress)
                a.tabs.Items[1].Text = i18n.T(i18n.TabExtract)
                a.tabs.Refresh()
        }
        // Update form labels.
        if a.fmtLbl != nil {
                a.fmtLbl.SetText(i18n.T(i18n.FormatLabel))
        }
        if a.lvlLbl != nil {
                a.lvlLbl.SetText(i18n.T(i18n.LevelLabel))
        }
        if a.pwLbl != nil {
                a.pwLbl.SetText(i18n.T(i18n.PasswordLabel))
        }
        if a.volLbl != nil {
                a.volLbl.SetText(i18n.T(i18n.VolumeSizeLabel))
        }
        if a.tgtLbl != nil {
                a.tgtLbl.SetText(i18n.T(i18n.TargetLabel))
        }
        if a.srcLbl != nil {
                a.srcLbl.SetText(i18n.T(i18n.SourceLabel))
        }
        if a.exSrcLbl != nil {
                a.exSrcLbl.SetText(i18n.T(i18n.SourceLabel))
        }
        if a.exTgtLbl != nil {
                a.exTgtLbl.SetText(i18n.T(i18n.TargetLabel))
        }
        if a.exPwLbl != nil {
                a.exPwLbl.SetText(i18n.T(i18n.PasswordLabel))
        }
        if a.contentsHdrLbl != nil {
                a.contentsHdrLbl.SetText(i18n.T(i18n.ArchiveContents))
        }
        // Rebuild the header row (localized labels + sort indicator) and
        // refresh the list so type strings pick up the new language.
        if a.headerRow != nil {
                newHeader := a.buildHeaderRow()
                a.headerRow.Objects = newHeader.Objects
                a.headerRow.Refresh()
        }
        a.contentList.Refresh()
        // Re-count entries in the current language.
        a.refreshEntryCount()
}

// refreshHeader rebuilds the column-header row with the current language
// and sort indicator.
func (a *App) refreshHeader() {
        if a.headerRow == nil {
                return
        }
        newHeader := a.buildHeaderRow()
        a.headerRow.Objects = newHeader.Objects
        a.headerRow.Refresh()
}

// ---------- Compress tab ----------

func (a *App) buildCompressTab() {
        a.srcList = widget.NewList(
                func() int { return len(a.srcItems) },
                func() fyne.CanvasObject { return widget.NewLabel("") },
                func(i widget.ListItemID, o fyne.CanvasObject) {
                        o.(*widget.Label).SetText(a.srcItems[i])
                },
        )
        a.srcList.OnSelected = func(id widget.ListItemID) {
                a.selectedIdx = id
        }
        a.srcList.OnUnselected = func(id widget.ListItemID) {
                a.selectedIdx = -1
        }

        a.addFile = widget.NewButton(i18n.T(i18n.AddFiles), func() {
                p := a.nativeFilePicker()
                if p == "" {
                        return
                }
                a.srcItems = append(a.srcItems, p)
                a.srcList.Refresh()
        })

        a.addFolder = widget.NewButton(i18n.T(i18n.AddFolder), func() {
                p := a.nativeFolderPicker()
                if p == "" {
                        return
                }
                a.srcItems = append(a.srcItems, p)
                a.srcList.Refresh()
        })

        a.removeSel = widget.NewButton(i18n.T(i18n.RemoveSelected), func() {
                if a.selectedIdx < 0 || a.selectedIdx >= len(a.srcItems) {
                        return
                }
                a.srcItems = append(a.srcItems[:a.selectedIdx], a.srcItems[a.selectedIdx+1:]...)
                a.selectedIdx = -1
                a.srcList.UnselectAll()
                a.srcList.Refresh()
        })

        a.clearAll = widget.NewButton(i18n.T(i18n.ClearAll), func() {
                a.srcItems = nil
                a.selectedIdx = -1
                a.srcList.Refresh()
        })

        a.formatSel = widget.NewSelect(
                []string{"zip", "tar", "tar.gz", "7z"},
                func(s string) {},
        )
        a.formatSel.SetSelected("zip")

        a.levelSel = widget.NewSelect(
                []string{i18n.T(i18n.LevelStore), i18n.T(i18n.LevelFastest), i18n.T(i18n.LevelFast), i18n.T(i18n.LevelNormal), i18n.T(i18n.LevelMax)},
                func(s string) {},
        )
        a.levelSel.SetSelected(i18n.T(i18n.LevelNormal))

        a.pwEntry = widget.NewPasswordEntry()
        a.pwEntry.SetPlaceHolder(i18n.T(i18n.PasswordLabel))

        a.volEntry = widget.NewEntry()
        a.volEntry.SetPlaceHolder(i18n.T(i18n.VolumeSizeHint))

        a.targetPath = widget.NewEntry()
        a.targetPath.SetPlaceHolder(i18n.T(i18n.TargetLabel))

        a.browseOut = widget.NewButton(i18n.T(i18n.BrowseDir), func() {
                p := a.nativeFolderPicker()
                if p == "" {
                        return
                }
                a.targetPath.SetText(p)
        })

        a.compressBtn = widget.NewButtonWithIcon(i18n.T(i18n.Compress), theme.ConfirmIcon(), a.doCompress)
}

func (a *App) compressPanel() fyne.CanvasObject {
        a.fmtLbl = widget.NewLabel(i18n.T(i18n.FormatLabel))
        a.lvlLbl = widget.NewLabel(i18n.T(i18n.LevelLabel))
        a.pwLbl = widget.NewLabel(i18n.T(i18n.PasswordLabel))
        a.volLbl = widget.NewLabel(i18n.T(i18n.VolumeSizeLabel))
        a.tgtLbl = widget.NewLabel(i18n.T(i18n.TargetLabel))
        form := container.New(layout.NewFormLayout(),
                a.fmtLbl, a.formatSel,
                a.lvlLbl, a.levelSel,
                a.pwLbl, a.pwEntry,
                a.volLbl, a.volEntry,
                a.tgtLbl, container.NewBorder(nil, nil, nil, a.browseOut, a.targetPath),
        )
        actions := container.NewHBox(a.addFile, a.addFolder, a.removeSel, a.clearAll)
        return container.NewBorder(nil, container.NewVBox(form, a.compressBtn), nil, nil,
                container.NewVBox(
                        actions,
                        container.NewMax(a.srcList),
                ),
        )
}

// ---------- Extract tab ----------

// columnWidths are the pixel widths for the four columns of the archive
// contents table. They are shared between the header row and the table
// itself so the columns visually line up.
var columnWidths = []float32{380, 90, 90, 150}

const (
        colName = iota
        colSize
        colType
        colModified
)

func (a *App) buildExtractTab() {
        a.exSrc = widget.NewEntry()
        a.exSrc.SetPlaceHolder(i18n.T(i18n.SourceLabel))
        // When the user picks an archive (or types a path and presses Enter),
        // we automatically list its contents in the panel below.
        a.exSrc.OnChanged = func(s string) {
                if s == "" {
                        a.contentAll = nil
                        a.contentItems = nil
                        a.contentList.Refresh()
                        a.contentLbl.SetText(i18n.T(i18n.NoArchiveSelected))
                        a.deleteBtn.Disable()
                        return
                }
                if _, err := os.Stat(s); err != nil {
                        a.contentAll = nil
                        a.contentItems = nil
                        a.contentList.Refresh()
                        a.contentLbl.SetText(i18n.T(i18n.NoArchiveSelected))
                        a.deleteBtn.Disable()
                        return
                }
                a.refreshContentList(s)
        }
        a.exBrowse = widget.NewButton(i18n.T(i18n.Browse), func() {
                p := a.nativeFilePicker()
                if p == "" {
                        return
                }
                a.exSrc.SetText(p)
        })

        a.exDest = widget.NewEntry()
        a.exDest.SetPlaceHolder(i18n.T(i18n.TargetLabel))
        a.exDestBrw = widget.NewButton(i18n.T(i18n.BrowseDir), func() {
                p := a.nativeFolderPicker()
                if p == "" {
                        return
                }
                a.exDest.SetText(p)
        })

        a.exPw = widget.NewPasswordEntry()
        a.exPw.SetPlaceHolder(i18n.T(i18n.PasswordLabel))
        a.extractBtn = widget.NewButtonWithIcon(i18n.T(i18n.Extract), theme.DownloadIcon(), a.doExtract)

        // "Extract here" — extracts into the archive's parent directory,
        // into a subfolder named after the archive (so files don't get
        // dumped into the parent dir directly). Matches the behaviour
        // of 7-Zip / WinRAR's "Extract to <name>\" right-click entry.
        a.extractHereBtn = widget.NewButtonWithIcon(i18n.T(i18n.ExtractHere), theme.MoveDownIcon(), a.doExtractHere)

        // "Extract selected" — only the entry highlighted in the content list.
        a.extractSelBtn = widget.NewButtonWithIcon(i18n.T(i18n.ExtractSelected), theme.FileIcon(), a.doExtractSelected)
        a.extractSelBtn.Disable()

        // ---------- WinRAR-style toolbar buttons ----------
        a.findBtn = widget.NewButtonWithIcon(i18n.T(i18n.Find), theme.SearchIcon(), a.toggleFindBar)
        a.addBtn = widget.NewButtonWithIcon(i18n.T(i18n.AddToArchive), theme.ContentAddIcon(), a.doAdd)
        a.deleteBtn = widget.NewButtonWithIcon(i18n.T(i18n.DeleteEntry), theme.DeleteIcon(), a.doDelete)
        a.deleteBtn.Disable()

        // Find bar — a small entry above the table that filters contentItems
        // in real time. Hidden by default; toggled by the Find button.
        a.findEntry = widget.NewEntry()
        a.findEntry.SetPlaceHolder(i18n.T(i18n.FindPlaceholder))
        a.findEntry.OnChanged = func(s string) {
                a.contentFilter = s
                a.applyContentFilter()
        }
        a.findCloseBtn = widget.NewButton(i18n.T(i18n.CloseFind), func() {
                a.findEntry.SetText("")
                a.contentFilter = ""
                a.applyContentFilter()
                a.findBar.Hide()
        })
        a.findBar = container.NewBorder(nil, nil, widget.NewLabel(i18n.T(i18n.Find)), a.findCloseBtn, a.findEntry)
        a.findBar.Hide()

        // ---------- Archive contents table ----------
        a.contentAll = []archiver.Entry{}
        a.contentItems = []archiver.Entry{}
        a.contentLbl = widget.NewLabel(i18n.T(i18n.NoArchiveSelected))
        a.contentLbl.TextStyle = fyne.TextStyle{Italic: true}

        // Content list: widget.List handles tap routing and selection
        // highlight natively. OnSelected fires on EVERY click (unlike
        // widget.Table which skips already-selected cells), so double-click
        // detection via timing works reliably.
        a.contentList = widget.NewList(
                func() int { return len(a.contentItems) },
                func() fyne.CanvasObject {
                        // Template row: a tappableRow wrapping a 4-column
                        // fixedColumnsLayout. The tappableRow handles
                        // single/double/right-click via Fyne's native
                        // Tappable / DoubleTappable / SecondaryTappable
                        // interfaces.
                        inner := container.New(&fixedColumnsLayout{widths: columnWidths},
                                container.NewHBox(
                                        widget.NewIcon(theme.FileIcon()),
                                        widget.NewLabel(""),
                                ),
                                widget.NewLabel(""),
                                widget.NewLabel(""),
                                widget.NewLabel(""),
                        )
                        // Wrap in tappableRow with a placeholder entry;
                        // the real entry is set in the update callback.
                        return newTappableRow(a, archiver.Entry{}, -1, inner)
                },
                func(i widget.ListItemID, o fyne.CanvasObject) {
                        if i < 0 || i >= len(a.contentItems) {
                                return
                        }
                        e := a.contentItems[i]
                        tr := o.(*tappableRow)
                        // Update the row's entry + index for click handling.
                        tr.entry = e
                        tr.index = i
                        // Update the visual content.
                        inner := tr.child.(*fyne.Container)
                        col0 := inner.Objects[0].(*fyne.Container)
                        icon := col0.Objects[0].(*widget.Icon)
                        nameLbl := col0.Objects[1].(*widget.Label)
                        sizeLbl := inner.Objects[1].(*widget.Label)
                        typeLbl := inner.Objects[2].(*widget.Label)
                        modLbl := inner.Objects[3].(*widget.Label)

                        if e.IsDir {
                                icon.SetResource(theme.FolderIcon())
                        } else {
                                icon.SetResource(theme.FileIcon())
                        }
                        nameLbl.SetText(basenameOf(e.Name))
                        nameLbl.Truncation = fyne.TextTruncateEllipsis

                        if e.IsDir {
                                sizeLbl.SetText("")
                        } else {
                                sizeLbl.SetText(formatSize(e.Size))
                        }

                        if e.IsDir {
                                typeLbl.SetText(i18n.T(i18n.TypeFolder))
                        } else if e.IsSymlink {
                                typeLbl.SetText(i18n.T(i18n.TypeSymlink))
                        } else {
                                typeLbl.SetText(fileTypeOf(e.Name))
                        }

                        if e.ModTime.IsZero() {
                                modLbl.SetText("")
                        } else {
                                modLbl.SetText(e.ModTime.Format("2006-01-02 15:04"))
                        }
                },
        )

        // OnSelected is kept for visual selection sync, but double-click
        // is now handled by tappableRow.DoubleTapped() (Fyne's native
        // DoubleTappable interface), which is much more reliable than
        // timing-based detection.
        a.contentList.OnSelected = func(id widget.ListItemID) {
                if id < 0 || id >= len(a.contentItems) {
                        return
                }
                e := a.contentItems[id]
                a.contentSelIdx = id
                if e.IsDir {
                        a.extractSelBtn.Disable()
                } else {
                        a.extractSelBtn.Enable()
                }
                if !e.IsDir && a.archiveFormat() == archiver.FormatZip {
                        a.deleteBtn.Enable()
                } else {
                        a.deleteBtn.Disable()
                }
        }
        a.contentList.OnUnselected = func(id widget.ListItemID) {
                // Don't clear — tappableRow handles selection.
        }

        // Build the header row with clickable cells for sorting.
        // Set sort state BEFORE building the header so the ▲ indicator
        // appears next to "Name" on the initial render.
        a.sortField = "name"
        a.sortAscending = true
        a.headerRow = a.buildHeaderRow()
}


func (a *App) extractPanel() fyne.CanvasObject {
        a.exSrcLbl = widget.NewLabel(i18n.T(i18n.SourceLabel))
        a.exTgtLbl = widget.NewLabel(i18n.T(i18n.TargetLabel))
        a.exPwLbl = widget.NewLabel(i18n.T(i18n.PasswordLabel))
        form := container.New(layout.NewFormLayout(),
                a.exSrcLbl, container.NewBorder(nil, nil, nil, a.exBrowse, a.exSrc),
                a.exTgtLbl, container.NewBorder(nil, nil, nil, a.exDestBrw, a.exDest),
                a.exPwLbl, a.exPw,
        )

        // Header above the content list: "Archive contents" label on the
        // left, WinRAR-style toolbar (Find / Add / Delete / Extract selected)
        // on the right.
        a.contentsHdrLbl = widget.NewLabel(i18n.T(i18n.ArchiveContents))
        contentHeader := container.NewHBox(
                a.contentsHdrLbl,
                a.contentLbl,
                layout.NewSpacer(),
                a.findBtn,
                a.addBtn,
                a.deleteBtn,
                a.extractSelBtn,
        )

        // Breadcrumb bar: [Up] [current path]. Shows where the user is
        // inside the archive and lets them navigate back to the parent.
        a.pathLbl = widget.NewLabel("/")
        a.pathLbl.TextStyle = fyne.TextStyle{Bold: true}
        a.upBtn = widget.NewButtonWithIcon(i18n.T(i18n.UpBtn), theme.MoveUpIcon(), func() {
                a.navigateUp()
        })
        a.upBtn.Disable()
        breadcrumb := container.NewHBox(a.upBtn, a.pathLbl)

        // The content area: find bar (toggled) + breadcrumb + toolbar +
        // header row + table.
        contentArea := container.NewBorder(
                container.NewVBox(a.findBar, breadcrumb, contentHeader, a.headerRow),
                nil, nil, nil,
                a.contentList,
        )

        // Bottom action row: Extract + Extract here
        actions := container.NewHBox(
                a.extractBtn,
                a.extractHereBtn,
        )

        return container.NewBorder(form, actions, nil, nil, contentArea)
}


// refreshContentList loads the archive's file table in a goroutine and
// updates contentAll/contentItems when done. currentPath is
// reset to root ("") so the user starts at the top level of the new
// archive.
func (a *App) refreshContentList(src string) {
        a.contentLbl.SetText(i18n.T(i18n.Loading))
        a.contentAll = nil
        a.contentItems = nil
        a.currentPath = ""
        a.updateBreadcrumb()
        a.contentList.Refresh()
        go func() {
                opts := archiver.Options{Password: a.exPw.Text}
                entries, err := archiver.List(src, opts)
                if err != nil {
                        a.contentLbl.SetText(i18n.T(i18n.ListFailed) + err.Error())
                        return
                }
                a.contentAll = entries
                a.applyContentFilter()
                a.refreshEntryCount()
        }()
}

// refreshEntryCount updates the contentLbl to show the total entry count
// in the current language ("N entries" / "N 项").
func (a *App) refreshEntryCount() {
        if a.contentLbl == nil {
                return
        }
        n := len(a.contentAll)
        unit := "entries"
        if i18n.Get() == i18n.SimplifiedChinese {
                unit = "项"
        }
        a.contentLbl.SetText(fmt.Sprintf("%d %s", n, unit))
}

// applyContentFilter recomputes contentItems from contentAll based on the
// current contentFilter string AND the current folder (currentPath).
//
// Visibility rules:
//   - When no Find filter is active: show only the DIRECT children of
//     currentPath (files and subfolders at the current level, not
//     recursive — like Windows Explorer).
//   - When a Find filter IS active: search the ENTIRE archive for matches
//     (ignoring currentPath), so the user can find files inside any
//     subfolder. The Find bar is the "search the whole archive" mode.
func (a *App) applyContentFilter() {
        visible := make([]archiver.Entry, 0, len(a.contentAll))
        hasFilter := a.contentFilter != ""
        needle := strings.ToLower(a.contentFilter)
        for _, e := range a.contentAll {
                if hasFilter {
                        // Search mode: show all entries whose full path
                        // contains the filter text.
                        if !strings.Contains(strings.ToLower(e.Name), needle) {
                                continue
                        }
                        visible = append(visible, e)
                } else {
                        // Browse mode: show only direct children of currentPath.
                        if !isDirectChild(e.Name, a.currentPath) {
                                continue
                        }
                        visible = append(visible, e)
                }
        }
        a.contentItems = visible
        a.sortContentItems()
        a.contentSelIdx = -1
        a.extractSelBtn.Disable()
        a.deleteBtn.Disable()
        a.contentList.Refresh()
}

// sortContentItems sorts contentItems in-place by sortField/sortAscending.
// Folders always sort before files within the same sort field (like Windows
// Explorer's default behavior).
func (a *App) sortContentItems() {
        field := a.sortField
        if field == "" {
                field = "name"
        }
        asc := a.sortAscending
        sort.SliceStable(a.contentItems, func(i, j int) bool {
                a_e, b_e := a.contentItems[i], a.contentItems[j]
                // Folders first.
                if a_e.IsDir != b_e.IsDir {
                        return a_e.IsDir
                }
                var less bool
                switch field {
                case "size":
                        less = a_e.Size < b_e.Size
                case "type":
                        less = fileTypeOf(a_e.Name) < fileTypeOf(b_e.Name)
                case "modified":
                        less = a_e.ModTime.Before(b_e.ModTime)
                default: // "name"
                        less = strings.ToLower(basenameOf(a_e.Name)) < strings.ToLower(basenameOf(b_e.Name))
                }
                if !asc {
                        return !less
                }
                return less
        })
}

// toggleSort flips the sort direction for the given field, or switches to
// it if a different field was active. Then re-sorts and re-renders.
func (a *App) toggleSort(field string) {
        if a.sortField == field {
                a.sortAscending = !a.sortAscending
        } else {
                a.sortField = field
                a.sortAscending = true
        }
        a.sortContentItems()
        a.contentList.Refresh()
        a.refreshHeader()
}

// buildHeaderRow creates a header row with clickable sort cells.
func (a *App) buildHeaderRow() *fyne.Container {
        // Each field has a sort key and a localized label. We build the
        // label now (language may change between calls).
        type hdrField struct {
                field string
                label string
        }
        fields := []hdrField{
                {"name", i18n.T(i18n.FileName)},
                {"size", i18n.T(i18n.Size)},
                {"type", i18n.T(i18n.Type)},
                {"modified", i18n.T(i18n.Modified)},
        }
        objs := make([]fyne.CanvasObject, 4)
        for i, f := range fields {
                text := f.label
                if a.sortField == f.field {
                        if a.sortAscending {
                                text += " ▲"
                        } else {
                                text += " ▼"
                        }
                }
                objs[i] = newHeaderCell(a, f.field, text)
        }
        return container.New(&fixedColumnsLayout{widths: columnWidths}, objs...)
}

// fileDialogStartDir returns a fyne.ListableURI for the directory the
// file dialog should start in. We use:
//   - the parent directory of the current archive (if one is open)
//   - the parent directory of the first compress source (if on Compress tab)
//   - the user's home directory as fallback
// This avoids the Fyne file dialog showing an empty listing when it
// can't enumerate the default location.
func (a *App) fileDialogStartDir() fyne.ListableURI {
        var dir string
        if a.exSrc != nil && a.exSrc.Text != "" {
                dir = filepath.Dir(a.exSrc.Text)
        } else if len(a.srcItems) > 0 {
                dir = filepath.Dir(a.srcItems[0])
        } else if home, err := os.UserHomeDir(); err == nil {
                dir = home
        }
        if dir == "" {
                return nil
        }
        // Convert the local path to a file:// URI, then to a ListableURI.
        uri := storage.NewFileURI(dir)
        lu, err := storage.ListerForURI(uri)
        if err != nil {
                return nil
        }
        return lu
}

// nativeFilePicker launches the Windows native file open dialog via
// PowerShell and returns the selected file path, or "" if cancelled.
// This replaces Fyne's built-in file dialog which crashes on Windows
// when scrolling/navigating folders (known bug #4260).
func (a *App) nativeFilePicker() string {
        if runtime.GOOS != "windows" {
                // Fallback: use Fyne dialog on non-Windows.
                ch := make(chan string, 1)
                dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
                        if err != nil || reader == nil {
                                ch <- ""
                                return
                        }
                        p := reader.URI().Path()
                        reader.Close()
                        ch <- p
                }, a.win)
                return <-ch
        }
        // Windows: use PowerShell + System.Windows.Forms.OpenFileDialog.
        startDir := ""
        if a.exSrc != nil && a.exSrc.Text != "" {
                startDir = filepath.Dir(a.exSrc.Text)
        } else if home, err := os.UserHomeDir(); err == nil {
                startDir = home
        }
        return runPowerShellFilePicker(startDir, false)
}

// nativeFolderPicker launches the Windows native folder browser dialog
// via PowerShell and returns the selected folder path, or "" if cancelled.
func (a *App) nativeFolderPicker() string {
        if runtime.GOOS != "windows" {
                ch := make(chan string, 1)
                dialog.ShowFolderOpen(func(u fyne.ListableURI, err error) {
                        if err != nil || u == nil {
                                ch <- ""
                                return
                        }
                        ch <- u.Path()
                }, a.win)
                return <-ch
        }
        startDir := ""
        if a.exSrc != nil && a.exSrc.Text != "" {
                startDir = filepath.Dir(a.exSrc.Text)
        } else if home, err := os.UserHomeDir(); err == nil {
                startDir = home
        }
        return runPowerShellFilePicker(startDir, true)
}

// runPowerShellFilePicker runs a PowerShell script that shows the native
// Windows file/folder picker and prints the selected path to stdout.
// If folder=true, shows FolderBrowserDialog; otherwise OpenFileDialog.
func runPowerShellFilePicker(startDir string, folder bool) string {
        var script string
        if folder {
                script = `Add-Type -AssemblyName System.Windows.Forms
$dlg = New-Object System.Windows.Forms.FolderBrowserDialog
$dlg.Description = "Select folder"
if ("__STARTDIR__" -ne "") { $dlg.SelectedPath = "__STARTDIR__" }
$dlg.ShowNewFolderButton = $true
$dlg.ShowDialog() | Out-Null
if ($dlg.SelectedPath) { Write-Output $dlg.SelectedPath }`
        } else {
                script = `Add-Type -AssemblyName System.Windows.Forms
$dlg = New-Object System.Windows.Forms.OpenFileDialog
$dlg.Title = "Select file"
$dlg.Multiselect = $false
if ("__STARTDIR__" -ne "") { $dlg.InitialDirectory = "__STARTDIR__" }
$dlg.ShowDialog() | Out-Null
if ($dlg.FileName) { Write-Output $dlg.FileName }`
        }
        // Substitute the start directory, escaping backslashes for PowerShell.
        safeDir := strings.ReplaceAll(startDir, `\`, `\\`)
        script = strings.ReplaceAll(script, "__STARTDIR__", safeDir)

        out, err := execHidden("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
        if err != nil {
                return ""
        }
        return strings.TrimSpace(string(out))
}

// rowAtY returns the contentItems index for the given Y position
// relative to the list, or -1 if none. Used by rightClickList to find
// which row was right-clicked.
func (a *App) rowAtY(y float32) int {
        if len(a.contentItems) == 0 {
                return -1
        }
        // widget.List uses the item template's MinSize height for each row.
        // We approximate by using 35px per row (icon + label height + padding).
        rowHeight := float32(35)
        idx := int(y / rowHeight)
        if idx < 0 || idx >= len(a.contentItems) {
                return -1
        }
        return idx
}

// openEntry handles double-click: folders are navigated into, files are
// extracted to a temp dir and opened with the OS default application.
func (a *App) openEntry(e archiver.Entry) {
        if e.IsDir {
                a.navigateTo(e.Name)
                return
        }
        // File: extract to temp, open with default app.
        a.openFileFromArchive(e)
}

// openFileFromArchive extracts a single entry to a temp directory and
// opens it with the OS default application (e.g. Notepad for .txt).
func (a *App) openFileFromArchive(e archiver.Entry) {
        if a.exSrc.Text == "" {
                a.appendLog("open: no archive selected")
                return
        }
        a.appendLog("opening: " + e.Name)
        go func() {
                tmp, err := os.MkdirTemp("", "openbox-open-*")
                if err != nil {
                        a.appendLog("open: temp dir failed: " + err.Error())
                        return
                }
                opts := archiver.Options{Password: a.exPw.Text}
                if err := archiver.ExtractEntry(a.exSrc.Text, e.Name, tmp, opts); err != nil {
                        a.appendLog("open: extract failed: " + err.Error())
                        os.RemoveAll(tmp)
                        return
                }
                fullPath := filepath.Join(tmp, filepath.FromSlash(e.Name))
                // Verify the file actually exists before trying to open it.
                if _, err := os.Stat(fullPath); err != nil {
                        a.appendLog("open: extracted file not found: " + fullPath + " (" + err.Error() + ")")
                        os.RemoveAll(tmp)
                        return
                }
                a.appendLog("open: launching " + fullPath)
                openInOS(fullPath)
                // Note: we deliberately do NOT clean up tmp here, because the
                // launched app (e.g. Notepad) needs the file to stay on disk.
                // The OS will clean up the temp dir on reboot.
        }()
}

// showRowContextMenu displays a right-click popup menu for the given entry.
func (a *App) showRowContextMenu(e archiver.Entry, pos fyne.Position) {
        var items []*fyne.MenuItem
        // Open (enter folder or open file)
        items = append(items, fyne.NewMenuItem(i18n.T(i18n.OpenEntry), func() {
                a.openEntry(e)
        }))
        if !e.IsDir {
                // Extract selected
                items = append(items, fyne.NewMenuItem(i18n.T(i18n.ExtractSelected), func() {
                        a.contentSelIdx = a.findContentIndex(e.Name)
                        a.doExtractSelected()
                }))
        }
        // Delete (zip only)
        if a.archiveFormat() == archiver.FormatZip {
                items = append(items, fyne.NewMenuItem(i18n.T(i18n.DeleteEntry), func() {
                        a.contentSelIdx = a.findContentIndex(e.Name)
                        a.doDelete()
                }))
        }
        menu := widget.NewPopUpMenu(fyne.NewMenu("", items...), a.win.Canvas())
        menu.ShowAtPosition(pos)
}

// findContentIndex returns the index in contentItems of the entry with the
// given name, or -1 if not found.
func (a *App) findContentIndex(name string) int {
        for i, e := range a.contentItems {
                if e.Name == name {
                        return i
                }
        }
        return -1
}

// navigateTo sets currentPath to the given folder path and refreshes the
// view. folderName is the full in-archive path of the folder (e.g.
// "subfolder/" or "subfolder/deep/").
func (a *App) navigateTo(folderName string) {
        a.appendLog("navigate to: " + folderName)
        a.currentPath = folderName
        a.updateBreadcrumb()
        a.applyContentFilter()
}

// navigateUp moves to the parent of the current folder. At root, does nothing.
func (a *App) navigateUp() {
        if a.currentPath == "" {
                return
        }
        a.currentPath = parentPath(a.currentPath)
        a.updateBreadcrumb()
        a.applyContentFilter()
}

// updateBreadcrumb refreshes the path label and the Up button's enabled
// state based on currentPath.
func (a *App) updateBreadcrumb() {
        if a.pathLbl == nil {
                return
        }
        if a.currentPath == "" {
                a.pathLbl.SetText("/")
                a.upBtn.Disable()
        } else {
                a.pathLbl.SetText("/" + a.currentPath)
                a.upBtn.Enable()
        }
}

// parentPath returns the parent folder path of the given entry name.
//   parentPath("subfolder/test.txt") → "subfolder/"
//   parentPath("subfolder/")         → ""
//   parentPath("test.txt")           → ""
//   parentPath("subfolder/deep/x")   → "subfolder/deep/"
func parentPath(name string) string {
        n := strings.TrimSuffix(name, "/")
        if i := strings.LastIndex(n, "/"); i >= 0 {
                return n[:i+1]
        }
        return ""
}

// isDirectChild reports whether `name` is a direct child of `dir`.
//   isDirectChild("subfolder/", "")            → true  (subfolder is a direct child of root)
//   isDirectChild("subfolder/test.txt", "subfolder/") → true
//   isDirectChild("subfolder/deep/", "subfolder/")    → true
//   isDirectChild("subfolder/deep/x", "subfolder/")   → false (it's a grandchild)
//   isDirectChild("test.txt", "")                     → true
//   isDirectChild("test.txt", "subfolder/")           → false
func isDirectChild(name, dir string) bool {
        // The entry must be inside `dir`.
        if !strings.HasPrefix(name, dir) {
                return false
        }
        rest := name[len(dir):]
        if rest == "" {
                return false
        }
        // `rest` must not contain any '/' (otherwise it's a deeper descendant).
        // Exception: if the entry IS a directory, its name ends with '/'.
        // We strip the trailing '/' before checking.
        rest = strings.TrimSuffix(rest, "/")
        if rest == "" {
                return false
        }
        return !strings.Contains(rest, "/")
}

// basenameOf returns the last path segment of an entry name.
//   basenameOf("subfolder/test.txt") → "test.txt"
//   basenameOf("subfolder/")         → "subfolder"
//   basenameOf("test.txt")           → "test.txt"
func basenameOf(name string) string {
        n := strings.TrimSuffix(name, "/")
        if i := strings.LastIndex(n, "/"); i >= 0 {
                return n[i+1:]
        }
        return n
}

// archiveFormat returns the detected format of the current archive, or
// FormatAuto if no archive is selected.
func (a *App) archiveFormat() archiver.Format {
        if a.exSrc == nil || a.exSrc.Text == "" {
                return archiver.FormatAuto
        }
        return archiver.Detect(a.exSrc.Text)
}

// toggleFindBar shows or hides the Find filter bar above the content table.
func (a *App) toggleFindBar() {
        if a.findBar.Visible() {
                a.findBar.Hide()
                a.findEntry.SetText("")
                a.contentFilter = ""
                a.applyContentFilter()
        } else {
                a.findBar.Show()
                a.win.Canvas().Focus(a.findEntry)
        }
}


// formatSize turns a byte count into a human-readable string.
func formatSize(n int64) string {
        const (
                KB = 1024
                MB = KB * 1024
                GB = MB * 1024
        )
        switch {
        case n >= GB:
                return fmt.Sprintf("%.2f GB", float64(n)/float64(GB))
        case n >= MB:
                return fmt.Sprintf("%.2f MB", float64(n)/float64(MB))
        case n >= KB:
                return fmt.Sprintf("%.1f KB", float64(n)/float64(KB))
        }
        return fmt.Sprintf("%d B", n)
}

// ---------- actions ----------

func (a *App) setBusy(b bool) {
        a.busy = b
        if b {
                a.compressBtn.Disable()
                a.extractBtn.Disable()
                a.extractHereBtn.Disable()
                a.extractSelBtn.Disable()
                a.deleteBtn.Disable()
                a.addBtn.Disable()
                a.findBtn.Disable()
                a.statusLbl.SetText(i18n.T(i18n.StatusWorking))
                a.progress.Show()
                a.progress.SetValue(0)
                a.backgroundBtn.Show()
                a.cancelBtn.Show()
                // Update window title to show working state.
                a.win.SetTitle(i18n.T(i18n.AppTitle) + " — " + i18n.T(i18n.StatusWorking))
        } else {
                a.compressBtn.Enable()
                a.extractBtn.Enable()
                a.extractHereBtn.Enable()
                a.addBtn.Enable()
                a.findBtn.Enable()
                // extractSelBtn / deleteBtn are re-enabled conditionally based
                // on the current row selection.
                if a.contentSelIdx >= 0 && a.contentSelIdx < len(a.contentItems) {
                        e := a.contentItems[a.contentSelIdx]
                        if !e.IsDir {
                                a.extractSelBtn.Enable()
                        }
                        if !e.IsDir && a.archiveFormat() == archiver.FormatZip {
                                a.deleteBtn.Enable()
                        }
                }
                a.progress.Hide()
                a.backgroundBtn.Hide()
                a.cancelBtn.Hide()
                a.win.SetTitle(i18n.T(i18n.AppTitle))
                // If we were running in background and just finished, surface
                // a desktop notification so the user knows even if the window
                // is minimised.
                if a.backgroundMode {
                        a.backgroundMode = false
                        a.showDesktopNotification()
                        // Bring the window back to front so the user sees the result.
                        a.win.Show()
                        a.win.RequestFocus()
                }
        }
}

func (a *App) appendLog(s string) {
        ts := time.Now().Format("15:04:05")
        a.logView.SetText(fmt.Sprintf("%s%s\n", a.logView.Text, ts+"  "+s))
}

// doBackground minimises the window so the user can do other work while
// the task runs in the background. The task continues in its goroutine;
// when it finishes, setBusy(false) will surface a desktop notification
// and bring the window back to front.
func (a *App) doBackground() {
        a.backgroundMode = true
        a.win.Hide()
        a.appendLog("--- " + i18n.T(i18n.Background) + " ---")
}

// doCancel is a placeholder for cancellation — true cancel requires
// plumbing a context through archiver.Compress/Extract, which is a
// bigger refactor. For now we just bring the window back so the user
// can see the in-progress state and wait, or close the app to abort.
func (a *App) doCancel() {
        if a.backgroundMode {
                a.backgroundMode = false
                a.win.Show()
                a.win.RequestFocus()
                return
        }
        // Not in background mode — ask the user to confirm quitting,
        // since we can't safely kill the goroutine mid-archive.
        dialog.ShowConfirm(i18n.T(i18n.AppTitle), i18n.T(i18n.ConfirmQuit),
                func(ok bool) {
                        if ok {
                                a.win.Close()
                        }
                }, a.win)
}

// showDesktopNotification fires a desktop notification when a background
// task finishes. Uses Fyne's app.SendNotification(), which on Windows
// shows a toast notification and on macOS uses Notification Center.
func (a *App) showDesktopNotification() {
        title := i18n.T(i18n.TaskComplete)
        body := i18n.Tf(i18n.TaskCompleteBody, map[string]string{"task": a.lastTaskDesc})
        n := fyne.NewNotification(title, body)
        a.app.SendNotification(n)
}

// onDropped handles external file drag-and-drop onto the OpenBox window.
// Fyne delivers all dropped files at once as a slice of fyne.URI. We
// extract each URI's local path and route it to the active tab.
func (a *App) onDropped(pos fyne.Position, uris []fyne.URI) {
        if len(uris) == 0 {
                return
        }

        // Determine the active tab.
        tabIndex := 0
        if a.tabs != nil && len(a.tabs.Items) > 0 {
                tabIndex = a.tabs.SelectedIndex()
        }

        switch tabIndex {
        case 0: // Compress tab
                for _, uri := range uris {
                        p := uri.Path()
                        if runtime.GOOS == "windows" {
                                p = strings.TrimPrefix(p, "/")
                        }
                        if p == "" {
                                continue
                        }
                        a.srcItems = append(a.srcItems, p)
                        a.appendLog("added: " + p)
                }
                a.srcList.Refresh()

        case 1: // Extract tab
                // If no archive is open, check if the first dropped file
                // IS an archive; if so, open it.
                if a.exSrc.Text == "" {
                        for _, uri := range uris {
                                p := uri.Path()
                                if runtime.GOOS == "windows" {
                                        p = strings.TrimPrefix(p, "/")
                                }
                                f := archiver.Detect(p)
                                if f != archiver.FormatAuto {
                                        a.exSrc.SetText(p)
                                        return
                                }
                        }
                }
                // Otherwise, add the dropped files to the currently open
                // archive (zip only).
                if a.archiveFormat() == archiver.FormatZip {
                        paths := make([]string, 0, len(uris))
                        for _, uri := range uris {
                                p := uri.Path()
                                if runtime.GOOS == "windows" {
                                        p = strings.TrimPrefix(p, "/")
                                }
                                if p != "" {
                                        paths = append(paths, p)
                                }
                        }
                        if len(paths) > 0 {
                                a.runAdd(paths)
                        }
                } else {
                        dialog.ShowError(fmt.Errorf("%s", i18n.T(i18n.ErrAddNotSupported)), a.win)
                }
        }
}

// showAboutDialog opens a custom About dialog with clickable hyperlinks
// to the OpenBox GitHub project and to SamAI.cc (the developer's site).
// We can't use dialog.ShowInformation() because that only accepts a plain
// string body — no widgets — so we build a custom dialog with widget.Hyperlink
// entries.
func (a *App) showAboutDialog() {
        // Split the localized AboutText into the pre-link intro and the
        // post-link tagline, with the two hyperlinks in between.
        // We build the layout manually so the links are real widgets.
        isZh := i18n.Get() == i18n.SimplifiedChinese
        var intro, tagline string
        if isZh {
                intro = "OpenBox — 开源跨平台压缩工具\n基于 MIT 协议发布"
                tagline = "由 SamAI Studio 开发——为每个人打造实用的 AI 工具。"
        } else {
                intro = "OpenBox — open-source cross-platform archiver.\nLicensed under MIT."
                tagline = "Developed by SamAI Studio — building practical AI tools for everyone."
        }

        introLbl := widget.NewLabel(intro)
        introLbl.Wrapping = fyne.TextWrapWord

        // Project hyperlink.
        projURL, _ := url.Parse("https://github.com/samaidev/openbox")
        projLink := widget.NewHyperlink("github.com/samaidev/openbox", projURL)
        var projCaption *widget.Label
        if isZh {
                projCaption = widget.NewLabel("项目地址：")
        } else {
                projCaption = widget.NewLabel("Project:")
        }
        projRow := container.NewHBox(projCaption, projLink)

        // SamAI hyperlink.
        samaiURL, _ := url.Parse("https://samai.cc")
        samaiLink := widget.NewHyperlink("SamAI.cc", samaiURL)
        var samaiCaption *widget.Label
        if isZh {
                samaiCaption = widget.NewLabel("开发者：")
        } else {
                samaiCaption = widget.NewLabel("Developer:")
        }
        samaiRow := container.NewHBox(samaiCaption, samaiLink)

        taglineLbl := widget.NewLabel(tagline)
        taglineLbl.Wrapping = fyne.TextWrapWord
        taglineLbl.TextStyle = fyne.TextStyle{Italic: true}

        content := container.NewVBox(
                introLbl,
                widget.NewLabel(""),
                projRow,
                samaiRow,
                widget.NewLabel(""),
                taglineLbl,
        )
        d := dialog.NewCustom(i18n.T(i18n.About), i18n.T(i18n.Ok), content, a.win)
        d.Show()
}

func (a *App) doCompress() {
        if len(a.srcItems) == 0 {
                dialog.ShowError(fmt.Errorf("%s", i18n.T(i18n.ErrNoSource)), a.win)
                return
        }
        if a.targetPath.Text == "" {
                dialog.ShowError(fmt.Errorf("%s", i18n.T(i18n.ErrNoTarget)), a.win)
                return
        }
        dest := a.targetPath.Text
        if fi, err := os.Stat(dest); err == nil && fi.IsDir() {
                dest = filepath.Join(dest, "openbox_"+time.Now().Format("20060102_150405")+"."+a.formatSel.Selected)
        }
        if _, err := os.Stat(dest); err == nil {
                dialog.ShowConfirm(i18n.T(i18n.AppTitle), i18n.T(i18n.OverwriteQuestion), func(ok bool) {
                        if ok {
                                a.runCompress(dest)
                        }
                }, a.win)
                return
        }
        a.runCompress(dest)
}

func (a *App) runCompress(dest string) {
        a.setBusy(true)
        a.lastTaskDesc = i18n.Tf(i18n.CompressingTitle, map[string]string{"name": filepath.Base(dest)})
        go func() {
                defer a.setBusy(false)
                format := formatFromString(a.formatSel.Selected)
                level := levelFromUI(a.levelSel.Selected)
                opts := archiver.Options{
                        Format:   format,
                        Level:    level,
                        Password: a.pwEntry.Text,
                }
                // Parse volume size from the UI field, if non-empty.
                if v := strings.TrimSpace(a.volEntry.Text); v != "" {
                        vb, err := parseVolumeSizeUI(v)
                        if err != nil {
                                a.statusLbl.SetText(i18n.T(i18n.StatusFailed))
                                a.appendLog("invalid volume size: " + err.Error())
                                dialog.ShowError(fmt.Errorf("invalid volume size: %v", err), a.win)
                                return
                        }
                        if vb < 1024 {
                                a.statusLbl.SetText(i18n.T(i18n.StatusFailed))
                                a.appendLog("volume size must be at least 1k")
                                dialog.ShowError(fmt.Errorf("volume size must be at least 1k (1024 bytes)"), a.win)
                                return
                        }
                        opts.VolumeSize = vb
                }
                prog := &archiver.Progress{
                        OnAdvance: func(done, total int, current string) {
                                if total > 0 {
                                        a.progress.SetValue(float64(done) / float64(total))
                                }
                                a.statusLbl.SetText(current)
                                if current != "" {
                                        a.appendLog(current)
                                }
                        },
                        OnDone: func() {
                                a.progress.SetValue(1)
                                a.statusLbl.SetText(i18n.T(i18n.StatusDone))
                                a.appendLog(i18n.T(i18n.StatusDone) + ": " + dest)
                        },
                }
                err := archiver.Compress(a.srcItems, dest, opts, prog)
                if err != nil {
                        a.statusLbl.SetText(i18n.T(i18n.StatusFailed))
                        a.appendLog(i18n.T(i18n.ErrCompressFailed) + err.Error())
                        dialog.ShowError(fmt.Errorf("%s%s", i18n.T(i18n.ErrCompressFailed), err.Error()), a.win)
                        return
                }
                if a.openAfter.Checked {
                        openInOS(filepath.Dir(dest))
                }
        }()
}

func (a *App) doExtract() {
        if a.exSrc.Text == "" {
                dialog.ShowError(fmt.Errorf("%s", i18n.T(i18n.ErrNoSource)), a.win)
                return
        }
        if a.exDest.Text == "" {
                dialog.ShowError(fmt.Errorf("%s", i18n.T(i18n.ErrNoTarget)), a.win)
                return
        }
        a.runExtract(a.exSrc.Text, a.exDest.Text, "")
}

// doExtractHere extracts into a subfolder of the archive's parent directory,
// named after the archive. e.g. /path/to/foo.zip → /path/to/foo/
//
// This is the behaviour the user wants when right-clicking an archive and
// picking "Extract here" — they expect a folder, not a pile of files dumped
// next to the archive.
func (a *App) doExtractHere() {
        if a.exSrc.Text == "" {
                dialog.ShowError(fmt.Errorf("%s", i18n.T(i18n.ErrNoSource)), a.win)
                return
        }
        src := a.exSrc.Text
        dir := filepath.Dir(src)
        base := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
        // .tar.gz has compound ext; strip twice.
        if strings.HasSuffix(base, ".tar") {
                base = strings.TrimSuffix(base, ".tar")
        }
        dest := filepath.Join(dir, base)
        a.exDest.SetText(dest)
        a.runExtract(src, dest, "")
}

// doExtractSelected extracts only the entry currently highlighted in
// content list, preserving its in-archive path.
func (a *App) doExtractSelected() {
        if a.exSrc.Text == "" {
                dialog.ShowError(fmt.Errorf("%s", i18n.T(i18n.ErrNoSource)), a.win)
                return
        }
        if a.exDest.Text == "" {
                dialog.ShowError(fmt.Errorf("%s", i18n.T(i18n.ErrNoTarget)), a.win)
                return
        }
        id := a.contentSelIdx
        if id < 0 || id >= len(a.contentItems) {
                return
        }
        entry := a.contentItems[id]
        if entry.IsDir {
                return
        }
        a.runExtract(a.exSrc.Text, a.exDest.Text, entry.Name)
}

// runExtract is the shared worker for all three extract actions.
// If entryName != "", only that entry is extracted (via ExtractEntry);
// otherwise the whole archive is extracted (via Extract).
func (a *App) runExtract(src, dest, entryName string) {
        a.setBusy(true)
        a.lastTaskDesc = i18n.Tf(i18n.ExtractingTitle, map[string]string{"name": filepath.Base(src)})
        go func() {
                defer a.setBusy(false)
                opts := archiver.Options{Password: a.exPw.Text}
                prog := &archiver.Progress{
                        OnAdvance: func(done, total int, current string) {
                                if total > 0 {
                                        a.progress.SetValue(float64(done) / float64(total))
                                }
                                a.statusLbl.SetText(current)
                                if current != "" {
                                        a.appendLog(current)
                                }
                        },
                        OnDone: func() {
                                a.progress.SetValue(1)
                                a.statusLbl.SetText(i18n.T(i18n.StatusDone))
                                a.appendLog(i18n.T(i18n.StatusDone) + ": " + dest)
                        },
                }
                var err error
                if entryName != "" {
                        // single-entry extraction
                        a.statusLbl.SetText(entryName)
                        a.appendLog(entryName)
                        err = archiver.ExtractEntry(src, entryName, dest, opts)
                        if err == nil {
                                a.progress.SetValue(1)
                                a.statusLbl.SetText(i18n.T(i18n.StatusDone))
                                a.appendLog(i18n.T(i18n.StatusDone) + ": " + filepath.Join(dest, entryName))
                        }
                } else {
                        err = archiver.Extract(src, dest, opts, prog)
                }
                if err != nil {
                        a.statusLbl.SetText(i18n.T(i18n.StatusFailed))
                        a.appendLog(i18n.T(i18n.ErrExtractFailed) + err.Error())
                        dialog.ShowError(fmt.Errorf("%s%s", i18n.T(i18n.ErrExtractFailed), err.Error()), a.win)
                        return
                }
                if a.openAfter.Checked {
                        openInOS(dest)
                }
        }()
}

// ---------- helpers ----------

func formatFromString(s string) archiver.Format {
        switch s {
        case "zip":
                return archiver.FormatZip
        case "tar":
                return archiver.FormatTar
        case "tar.gz":
                return archiver.FormatTarGz
        case "7z":
                return archiver.Format7z
        }
        return archiver.FormatAuto
}

func levelFromUI(s string) archiver.Level {
        switch s {
        case i18n.T(i18n.LevelStore):
                return archiver.LevelStore
        case i18n.T(i18n.LevelFastest):
                return archiver.LevelFastest
        case i18n.T(i18n.LevelFast):
                return archiver.LevelFast
        case i18n.T(i18n.LevelMax):
                return archiver.LevelMaximum
        }
        return archiver.LevelNormal
}

// openInOS opens a file or folder with the OS default application.
// On Windows we use 'rundll32 url.dll,FileProtocolHandler <path>' which
// is the most reliable way to launch a file with its default app (used by
// the popular skratchdot/open-golang library). The 'cmd /c start' approach
// has issues with nested paths and paths containing spaces.
func openInOS(path string) {
        switch runtime.GOOS {
        case "darwin":
                _ = execCmd("open", path)
        case "windows":
                sysRoot := os.Getenv("SYSTEMROOT")
                rundll32 := filepath.Join(sysRoot, "System32", "rundll32.exe")
                _ = execCmd(rundll32, "url.dll,FileProtocolHandler", path)
        default:
                _ = execCmd("xdg-open", path)
        }
}

// parseVolumeSizeUI parses a human-friendly volume size string (e.g.
// "100m", "1g", "500k") into bytes. Empty string returns 0.
func parseVolumeSizeUI(s string) (int64, error) {
        s = strings.TrimSpace(s)
        if s == "" {
                return 0, nil
        }
        low := strings.ToLower(s)
        multiplier := int64(1)
        numStr := low
        // Order matters: check 2-char suffixes before 1-char.
        for _, suffix := range []string{"gib", "mib", "kib", "gb", "mb", "kb", "g", "m", "k"} {
                if strings.HasSuffix(low, suffix) {
                        switch suffix[0] {
                        case 'g':
                                multiplier = 1024 * 1024 * 1024
                        case 'm':
                                multiplier = 1024 * 1024
                        case 'k':
                                multiplier = 1024
                        }
                        numStr = low[:len(low)-len(suffix)]
                        break
                }
        }
        numStr = strings.TrimSpace(numStr)
        n, err := strconv.ParseInt(numStr, 10, 64)
        if err != nil {
                return 0, fmt.Errorf("invalid number %q: %w", numStr, err)
        }
        return n * multiplier, nil
}

// applyInitialState pre-fills the Compress or Extract tab from CLI args.
// It's invoked once after the window content is built.
func (a *App) applyInitialState(init *InitialState) {
        if init == nil {
                return
        }
        switch init.Tab {
        case TabCompress:
                for _, s := range init.Sources {
                        if s != "" {
                                a.srcItems = append(a.srcItems, s)
                        }
                }
                a.srcList.Refresh()
                // Default the target path to the parent dir of the first source.
                if len(a.srcItems) > 0 && a.targetPath.Text == "" {
                        a.targetPath.SetText(filepath.Dir(a.srcItems[0]))
                }
                a.tabs.SelectIndex(int(TabCompress))
        case TabExtract:
                if init.Archive != "" {
                        a.exSrc.SetText(init.Archive)
                        // Default the target dir to the archive's parent + base name.
                        dir := filepath.Dir(init.Archive)
                        base := strings.TrimSuffix(filepath.Base(init.Archive), filepath.Ext(init.Archive))
                        if base != "" && dir != "" {
                                a.exDest.SetText(filepath.Join(dir, base))
                        } else if dir != "" {
                                a.exDest.SetText(dir)
                        }
                }
                a.tabs.SelectIndex(int(TabExtract))
        }
}

// ---------- WinRAR-style Add / Delete / Find helpers ----------

// doDelete removes the currently selected entry from the archive.
// Only .zip is supported because we need read+write access via archive/zip.
// For other formats we show a clear error so the user understands why.
func (a *App) doDelete() {
        if a.exSrc.Text == "" {
                dialog.ShowError(fmt.Errorf("%s", i18n.T(i18n.ErrNoSource)), a.win)
                return
        }
        if a.archiveFormat() != archiver.FormatZip {
                dialog.ShowError(fmt.Errorf("%s", i18n.T(i18n.ErrDeleteNotSupported)), a.win)
                return
        }
        id := a.contentSelIdx
        if id < 0 || id >= len(a.contentItems) {
                return
        }
        entry := a.contentItems[id]
        dialog.ShowConfirm(i18n.T(i18n.DeleteEntry), i18n.T(i18n.ConfirmDeleteEntry), func(ok bool) {
                if !ok {
                        return
                }
                a.setBusy(true)
                a.lastTaskDesc = i18n.Tf(i18n.DeleteSuccess, map[string]string{"name": entry.Name})
                go func() {
                        defer a.setBusy(false)
                        err := archiver.DeleteEntries(a.exSrc.Text, []string{entry.Name}, archiver.Options{})
                        if err != nil {
                                a.statusLbl.SetText(i18n.T(i18n.StatusFailed))
                                a.appendLog(i18n.T(i18n.ErrExtractFailed) + err.Error())
                                dialog.ShowError(fmt.Errorf("%s: %v", i18n.T(i18n.DeleteEntry), err), a.win)
                                return
                        }
                        a.statusLbl.SetText(i18n.T(i18n.StatusDone))
                        a.appendLog(i18n.Tf(i18n.DeleteSuccess, map[string]string{"name": entry.Name}))
                        a.refreshContentList(a.exSrc.Text)
                }()
        }, a.win)
}

// doAdd opens a file picker and appends the selected files/folders to the
// current archive. Only .zip is supported; for other formats the user gets
// a clear error message.
func (a *App) doAdd() {
        if a.exSrc.Text == "" {
                dialog.ShowError(fmt.Errorf("%s", i18n.T(i18n.ErrNoSource)), a.win)
                return
        }
        if a.archiveFormat() != archiver.FormatZip {
                dialog.ShowError(fmt.Errorf("%s", i18n.T(i18n.ErrAddNotSupported)), a.win)
                return
        }
        p := a.nativeFilePicker()
        if p == "" {
                return
        }
        a.runAdd([]string{p})
}

// runAdd appends the given sources to the current archive in a goroutine.
func (a *App) runAdd(sources []string) {
        a.setBusy(true)
        a.lastTaskDesc = i18n.T(i18n.AddToArchive)
        go func() {
                defer a.setBusy(false)
                err := archiver.AddToArchive(a.exSrc.Text, sources, archiver.Options{})
                if err != nil {
                        a.statusLbl.SetText(i18n.T(i18n.StatusFailed))
                        a.appendLog(i18n.T(i18n.ErrCompressFailed) + err.Error())
                        dialog.ShowError(fmt.Errorf("%s: %v", i18n.T(i18n.AddToArchive), err), a.win)
                        return
                }
                a.statusLbl.SetText(i18n.T(i18n.StatusDone))
                a.appendLog(i18n.Tf(i18n.AddSuccess, map[string]string{"n": fmt.Sprintf("%d", len(sources))}))
                a.refreshContentList(a.exSrc.Text)
        }()
}

// ---------- layout + display helpers ----------

// fixedColumnsLayout is a custom Fyne layout that lays out its objects
// left-to-right with the given pixel widths. Used by the table's header
// row so the column titles visually line up with the table cells below.
type fixedColumnsLayout struct {
        widths []float32
}

func (l *fixedColumnsLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
        pos := fyne.NewPos(0, 0)
        for i, obj := range objects {
                if i >= len(l.widths) {
                        break
                }
                w := l.widths[i]
                obj.Resize(fyne.NewSize(w, size.Height))
                obj.Move(pos)
                pos = pos.Add(fyne.NewPos(w, 0))
        }
}

func (l *fixedColumnsLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
        totalW := float32(0)
        for _, w := range l.widths {
                totalW += w
        }
        return fyne.NewSize(totalW, 28)
}

// boldLabel returns a Label with the bold text style. Used for the
// column-header row of the archive contents table.
func boldLabel(text string) *widget.Label {
        l := widget.NewLabel(text)
        l.TextStyle = fyne.TextStyle{Bold: true}
        return l
}

// fileTypeOf returns a short human-readable type description based on the
// file extension, mirroring what WinRAR shows in its "Type" column.
// Unknown extensions fall back to the localised "File" string.
func fileTypeOf(name string) string {
        ext := strings.ToLower(filepath.Ext(name))
        switch ext {
        case "":
                return i18n.T(i18n.TypeFile)
        case ".txt", ".log", ".md":
                return "Text"
        case ".zip":
                return "ZIP archive"
        case ".7z":
                return "7-Zip archive"
        case ".rar":
                return "RAR archive"
        case ".tar", ".tgz", ".gz":
                return "Tarball"
        case ".iso":
                return "ISO image"
        case ".pdf":
                return "PDF document"
        case ".doc", ".docx":
                return "Word document"
        case ".xls", ".xlsx":
                return "Excel spreadsheet"
        case ".ppt", ".pptx":
                return "PowerPoint"
        case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".tiff":
                return "Image"
        case ".mp3", ".wav", ".flac", ".aac", ".ogg":
                return "Audio"
        case ".mp4", ".mkv", ".avi", ".mov", ".webm":
                return "Video"
        case ".exe", ".msi":
                return "Application"
        case ".dll":
                return "DLL"
        case ".go", ".c", ".cpp", ".h", ".py", ".js", ".ts", ".java", ".rs", ".rb", ".php":
                return "Source code"
        case ".json", ".xml", ".yaml", ".yml", ".toml", ".ini", ".cfg":
                return "Config / data"
        case ".html", ".htm", ".css":
                return "Web"
        case ".sh", ".bat", ".ps1":
                return "Script"
        case ".zip.001", ".7z.001":
                return "Split archive"
        }
        return ext[1:] + " file"
}

// ---------- tappableRow: a row that handles single + double click ----------

// tappableRow wraps a fyne.CanvasObject (the row content) and implements
// fyne.Tappable and fyne.DoubleTappable. Fyne's input router natively
// detects double-clicks and calls DoubleTapped(), which is far more
// reliable than timing-based detection in widget.List.OnSelected (which
// may not fire on the second click of a double-click).
type tappableRow struct {
        widget.BaseWidget
        app   *App
        entry archiver.Entry
        index int
        child fyne.CanvasObject
}

func newTappableRow(app *App, entry archiver.Entry, index int, child fyne.CanvasObject) *tappableRow {
        r := &tappableRow{app: app, entry: entry, index: index, child: child}
        r.ExtendBaseWidget(r)
        return r
}

// Tapped handles single clicks: select the row in the underlying list.
func (r *tappableRow) Tapped(pe *fyne.PointEvent) {
        r.app.contentSelIdx = r.index
        // Select in the list for visual highlight.
        r.app.contentList.Select(r.index)
        e := r.entry
        if e.IsDir {
                r.app.extractSelBtn.Disable()
        } else {
                r.app.extractSelBtn.Enable()
        }
        if !e.IsDir && r.app.archiveFormat() == archiver.FormatZip {
                r.app.deleteBtn.Enable()
        } else {
                r.app.deleteBtn.Disable()
        }
}

// DoubleTapped handles double-clicks: open folder (navigate into it) or
// open file (extract to temp + launch with default app).
func (r *tappableRow) DoubleTapped(pe *fyne.PointEvent) {
        r.app.appendLog("double-click: " + r.entry.Name)
        if r.entry.IsDir {
                r.app.navigateTo(r.entry.Name)
        } else {
                r.app.openFileFromArchive(r.entry)
        }
}

// TappedSecondary handles right-clicks: show context menu.
func (r *tappableRow) TappedSecondary(pe *fyne.PointEvent) {
        r.app.contentSelIdx = r.index
        r.app.contentList.Select(r.index)
        r.app.showRowContextMenu(r.entry, pe.AbsolutePosition)
}

// CreateRenderer renders the wrapped child widget.
func (r *tappableRow) CreateRenderer() fyne.WidgetRenderer {
        return widget.NewSimpleRenderer(r.child)
}

// ---------- rightClickList: wrapper that adds right-click + double-click ----------

// rightClickList wraps a fyne.CanvasObject (typically a widget.List) and
// implements desktop.Mouseable to capture BOTH left and right mouse
// events. This gives us reliable double-click detection (independent of
// widget.List's internal OnSelected behavior) AND right-click context
// menus.
type rightClickList struct {
        widget.BaseWidget
        app         *App
        child       fyne.CanvasObject
        lastDownTime time.Time
        lastDownRow  int
}

func newRightClickList(app *App, child fyne.CanvasObject) *rightClickList {
        r := &rightClickList{app: app, child: child, lastDownRow: -1}
        r.ExtendBaseWidget(r)
        return r
}

// MouseDown captures all mouse button presses. We handle:
//   - Right-click: show context menu.
//   - Left-click: detect double-click via 500ms timing on the same row.
//     This is MORE reliable than widget.List.OnSelected because Fyne's
//     List may not re-fire OnSelected when clicking an already-selected
//     row, but MouseDown fires on every physical click.
func (r *rightClickList) MouseDown(me *desktop.MouseEvent) {
        // Find which row was clicked.
        idx := r.app.rowAtY(me.Position.Y)
        if idx < 0 || idx >= len(r.app.contentItems) {
                return
        }
        e := r.app.contentItems[idx]

        if me.Button == desktop.MouseButtonSecondary {
                // Right-click: select the row and show context menu.
                r.app.contentSelIdx = idx
                r.app.contentList.Select(idx)
                r.app.showRowContextMenu(e, me.AbsolutePosition)
                return
        }

        if me.Button == desktop.MouseButtonPrimary {
                // Left-click: detect double-click.
                now := time.Now()
                isDouble := idx == r.lastDownRow &&
                        !r.lastDownTime.IsZero() &&
                        now.Sub(r.lastDownTime) < 500*time.Millisecond
                r.lastDownTime = now
                r.lastDownRow = idx

                if isDouble {
                        // Double-click: open folder or file.
                        r.app.appendLog("double-click (mouse): " + e.Name)
                        // Also select it so the highlight shows.
                        r.app.contentSelIdx = idx
                        r.app.contentList.Select(idx)
                        if e.IsDir {
                                r.app.navigateTo(e.Name)
                        } else {
                                r.app.openFileFromArchive(e)
                        }
                }
                // Single-click selection is handled by widget.List.OnSelected,
                // so we don't need to do anything here for single clicks.
        }
}

// MouseUp is required by desktop.Mouseable but we don't need it.
func (r *rightClickList) MouseUp(me *desktop.MouseEvent) {}

// CreateRenderer renders the wrapped child widget.
func (r *rightClickList) CreateRenderer() fyne.WidgetRenderer {
        return widget.NewSimpleRenderer(r.child)
}

// ---------- headerCell: clickable sort header ----------

// headerCell is a column header label that toggles sort direction when
// clicked. It implements fyne.Tappable.
type headerCell struct {
        widget.BaseWidget
        app   *App
        field string
        text  string
}

func newHeaderCell(app *App, field, text string) *headerCell {
        h := &headerCell{app: app, field: field, text: text}
        h.ExtendBaseWidget(h)
        return h
}

func (h *headerCell) Tapped(pe *fyne.PointEvent) {
        h.app.toggleSort(h.field)
}

func (h *headerCell) CreateRenderer() fyne.WidgetRenderer {
        lbl := widget.NewLabel(h.text)
        lbl.TextStyle = fyne.TextStyle{Bold: true}
        return widget.NewSimpleRenderer(lbl)
}
