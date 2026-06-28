// Package ui builds the Fyne-based bilingual UI for OpenBox.
package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

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
	browseOut   *widget.Button
	compressBtn *widget.Button

	// extract tab
	exSrc      *widget.Entry
	exBrowse   *widget.Button
	exDest     *widget.Entry
	exDestBrw  *widget.Button
	exPw       *widget.Entry
	extractBtn *widget.Button

	// shared
	progress  *widget.ProgressBar
	statusLbl *widget.Label
	logView   *widget.Entry
	langBtn   *widget.Button
	openAfter *widget.Check
	tabs      *container.AppTabs

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

	aboutBtn := widget.NewButton(i18n.T(i18n.About), func() {
		dialog.ShowInformation(i18n.T(i18n.About), i18n.T(i18n.AboutText), w)
	})
	helpBtn := widget.NewButton(i18n.T(i18n.HelpBtn), func() {
		dialog.ShowInformation(i18n.T(i18n.HelpBtn), i18n.T(i18n.HelpText), w)
	})

	toolbar := container.NewHBox(app.langBtn, helpBtn, aboutBtn)

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

	bottom := container.NewVBox(
		container.NewHBox(app.statusLbl, app.progress, layout.NewSpacer(), app.openAfter),
		container.NewMax(app.logView),
	)

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
	a.compressBtn.SetText(i18n.T(i18n.Compress))
	a.extractBtn.SetText(i18n.T(i18n.Extract))
	a.addFile.SetText(i18n.T(i18n.AddFiles))
	a.addFolder.SetText(i18n.T(i18n.AddFolder))
	a.removeSel.SetText(i18n.T(i18n.RemoveSelected))
	a.clearAll.SetText(i18n.T(i18n.ClearAll))
	a.browseOut.SetText(i18n.T(i18n.BrowseDir))
	a.exBrowse.SetText(i18n.T(i18n.Browse))
	a.exDestBrw.SetText(i18n.T(i18n.BrowseDir))
	a.openAfter.SetText(i18n.T(i18n.OpenAfterDone))
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
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			p := reader.URI().Path()
			reader.Close()
			if runtime.GOOS == "windows" {
				p = strings.TrimPrefix(p, "/")
			}
			a.srcItems = append(a.srcItems, p)
			a.srcList.Refresh()
		}, a.win)
	})

	a.addFolder = widget.NewButton(i18n.T(i18n.AddFolder), func() {
		dialog.ShowFolderOpen(func(u fyne.ListableURI, err error) {
			if err != nil || u == nil {
				return
			}
			a.srcItems = append(a.srcItems, u.Path())
			a.srcList.Refresh()
		}, a.win)
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
		[]string{"zip", "tar", "tar.gz"},
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

	a.targetPath = widget.NewEntry()
	a.targetPath.SetPlaceHolder(i18n.T(i18n.TargetLabel))

	a.browseOut = widget.NewButton(i18n.T(i18n.BrowseDir), func() {
		dialog.ShowFolderOpen(func(u fyne.ListableURI, err error) {
			if err != nil || u == nil {
				return
			}
			a.targetPath.SetText(u.Path())
		}, a.win)
	})

	a.compressBtn = widget.NewButtonWithIcon(i18n.T(i18n.Compress), theme.ConfirmIcon(), a.doCompress)
}

func (a *App) compressPanel() fyne.CanvasObject {
	form := container.New(layout.NewFormLayout(),
		widget.NewLabel(i18n.T(i18n.FormatLabel)), a.formatSel,
		widget.NewLabel(i18n.T(i18n.LevelLabel)), a.levelSel,
		widget.NewLabel(i18n.T(i18n.PasswordLabel)), a.pwEntry,
		widget.NewLabel(i18n.T(i18n.TargetLabel)), container.NewBorder(nil, nil, nil, a.browseOut, a.targetPath),
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

func (a *App) buildExtractTab() {
	a.exSrc = widget.NewEntry()
	a.exSrc.SetPlaceHolder(i18n.T(i18n.SourceLabel))
	a.exBrowse = widget.NewButton(i18n.T(i18n.Browse), func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			defer reader.Close()
			a.exSrc.SetText(reader.URI().Path())
		}, a.win)
	})

	a.exDest = widget.NewEntry()
	a.exDest.SetPlaceHolder(i18n.T(i18n.TargetLabel))
	a.exDestBrw = widget.NewButton(i18n.T(i18n.BrowseDir), func() {
		dialog.ShowFolderOpen(func(u fyne.ListableURI, err error) {
			if err != nil || u == nil {
				return
			}
			a.exDest.SetText(u.Path())
		}, a.win)
	})

	a.exPw = widget.NewPasswordEntry()
	a.exPw.SetPlaceHolder(i18n.T(i18n.PasswordLabel))
	a.extractBtn = widget.NewButtonWithIcon(i18n.T(i18n.Extract), theme.DownloadIcon(), a.doExtract)
}

func (a *App) extractPanel() fyne.CanvasObject {
	form := container.New(layout.NewFormLayout(),
		widget.NewLabel(i18n.T(i18n.SourceLabel)), container.NewBorder(nil, nil, nil, a.exBrowse, a.exSrc),
		widget.NewLabel(i18n.T(i18n.TargetLabel)), container.NewBorder(nil, nil, nil, a.exDestBrw, a.exDest),
		widget.NewLabel(i18n.T(i18n.PasswordLabel)), a.exPw,
	)
	return container.NewBorder(nil, a.extractBtn, nil, nil, form)
}

// ---------- actions ----------

func (a *App) setBusy(b bool) {
	a.busy = b
	if b {
		a.compressBtn.Disable()
		a.extractBtn.Disable()
		a.statusLbl.SetText(i18n.T(i18n.StatusWorking))
		a.progress.Show()
		a.progress.SetValue(0)
	} else {
		a.compressBtn.Enable()
		a.extractBtn.Enable()
	}
}

func (a *App) appendLog(s string) {
	ts := time.Now().Format("15:04:05")
	a.logView.SetText(fmt.Sprintf("%s%s\n", a.logView.Text, ts+"  "+s))
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
	go func() {
		defer a.setBusy(false)
		format := formatFromString(a.formatSel.Selected)
		level := levelFromUI(a.levelSel.Selected)
		opts := archiver.Options{
			Format:   format,
			Level:    level,
			Password: a.pwEntry.Text,
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
	a.setBusy(true)
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
				a.appendLog(i18n.T(i18n.StatusDone) + ": " + a.exDest.Text)
			},
		}
		err := archiver.Extract(a.exSrc.Text, a.exDest.Text, opts, prog)
		if err != nil {
			a.statusLbl.SetText(i18n.T(i18n.StatusFailed))
			a.appendLog(i18n.T(i18n.ErrExtractFailed) + err.Error())
			dialog.ShowError(fmt.Errorf("%s%s", i18n.T(i18n.ErrExtractFailed), err.Error()), a.win)
			return
		}
		if a.openAfter.Checked {
			openInOS(a.exDest.Text)
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

func openInOS(path string) {
	switch runtime.GOOS {
	case "darwin":
		_ = execCmd("open", path)
	case "windows":
		_ = execCmd("explorer", path)
	default:
		_ = execCmd("xdg-open", path)
	}
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
