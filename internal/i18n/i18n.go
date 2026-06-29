// Package i18n provides bilingual (English / Simplified Chinese) strings for OpenBox.
package i18n

import "sync/atomic"

// Lang is the current UI language.
type Lang int32

const (
	English Lang = iota
	SimplifiedChinese
)

var current atomic.Int32 // stores Lang

// Set switches the active language. Safe for concurrent use.
func Set(l Lang) { current.Store(int32(l)) }

// Get returns the active language.
func Get() Lang { return Lang(current.Load()) }

// Toggle flips between English and SimplifiedChinese and returns the new value.
func Toggle() Lang {
	old := current.Load()
	next := English
	if Lang(old) == English {
		next = SimplifiedChinese
	}
	current.Store(int32(next))
	return next
}

// key identifies a translatable string.
type key int

// Enumerate every translatable string. Keep order stable so that the
// JSON-style table below stays readable.
const (
	AppTitle key = iota
	LanguageToggle
	TabCompress
	TabExtract
	SourceLabel
	TargetLabel
	PasswordLabel
	Browse
	BrowseDir
	Compress
	Extract
	FormatLabel
	LevelLabel
	LevelStore
	LevelFastest
	LevelFast
	LevelNormal
	LevelMax
	StatusIdle
	StatusWorking
	StatusDone
	StatusFailed
	StatusCancelled
	LogTitle
	SelectFiles
	SelectFolder
	SelectArchive
	ConfirmOverwrite
	ErrNoSource
	ErrNoTarget
	ErrUnsupportedFormat
	ErrCompressFailed
	ErrExtractFailed
	ConfirmQuit
	About
	AboutText
	Ok
	Cancel
	Yes
	No
	OverwriteQuestion
	FileName
	Size
	Compressed
	Type
	Modified
	AddFiles
	AddFolder
	RemoveSelected
	ClearAll
	OpenAfterDone
	HelpBtn
	HelpText
	CompressionProgress
	ExtractionProgress
	FileCount
	TotalSize
	EstimatedRemaining
	ArchiveContents
	ExtractHere
	ExtractHereFolder
	ExtractSelected
	Loading
	NoArchiveSelected
	ListFailed
	ExtractHereShort
	Background
	CancelTask
	TaskComplete
	TaskCompleteBody
	TaskFailedBody
	CompressingTitle
	ExtractingTitle
	VolumeSizeLabel
	VolumeSizeHint
)

// table maps a key -> [English, Chinese].
var table = map[key][2]string{
	AppTitle:             {"OpenBox", "OpenBox 压缩工具"},
	LanguageToggle:       {"中文", "English"},
	TabCompress:          {"Compress", "压缩"},
	TabExtract:           {"Extract", "解压"},
	SourceLabel:          {"Source", "源文件"},
	TargetLabel:          {"Target", "目标位置"},
	PasswordLabel:        {"Password (optional)", "密码（可选）"},
	Browse:               {"Browse…", "浏览…"},
	BrowseDir:            {"Choose folder…", "选择目录…"},
	Compress:             {"Compress", "开始压缩"},
	Extract:              {"Extract", "开始解压"},
	FormatLabel:          {"Format", "格式"},
	LevelLabel:           {"Level", "压缩级别"},
	LevelStore:           {"Store", "仅存储"},
	LevelFastest:         {"Fastest", "最快"},
	LevelFast:            {"Fast", "快速"},
	LevelNormal:          {"Normal", "标准"},
	LevelMax:             {"Max", "最大"},
	StatusIdle:           {"Idle", "空闲"},
	StatusWorking:        {"Working…", "处理中…"},
	StatusDone:           {"Done", "完成"},
	StatusFailed:         {"Failed", "失败"},
	StatusCancelled:      {"Cancelled", "已取消"},
	LogTitle:             {"Log", "日志"},
	SelectFiles:          {"Select files to add", "选择要添加的文件"},
	SelectFolder:         {"Select a folder", "选择一个目录"},
	SelectArchive:        {"Select an archive", "选择一个压缩包"},
	ConfirmOverwrite:     {"File exists. Overwrite?", "文件已存在，是否覆盖？"},
	ErrNoSource:          {"No source specified.", "未选择源文件。"},
	ErrNoTarget:          {"No target specified.", "未选择目标位置。"},
	ErrUnsupportedFormat: {"Unsupported format.", "不支持的格式。"},
	ErrCompressFailed:    {"Compression failed: ", "压缩失败："},
	ErrExtractFailed:     {"Extraction failed: ", "解压失败："},
	ConfirmQuit:          {"A task is still running. Quit anyway?", "任务进行中，确定退出吗？"},
	About:                {"About", "关于"},
	AboutText: {"OpenBox — open-source cross-platform archiver.\nLicensed under MIT.\nhttps://github.com/samaidev/openbox",
		"OpenBox — 开源跨平台压缩工具\n基于 MIT 协议发布\nhttps://github.com/samaidev/openbox"},
	Ok:                {"OK", "确定"},
	Cancel:            {"Cancel", "取消"},
	Yes:               {"Yes", "是"},
	No:                {"No", "否"},
	OverwriteQuestion: {"Overwrite existing file?", "是否覆盖已存在的文件？"},
	FileName:          {"Name", "名称"},
	Size:              {"Size", "大小"},
	Compressed:        {"Compressed", "压缩后"},
	Type:              {"Type", "类型"},
	Modified:          {"Modified", "修改时间"},
	AddFiles:          {"Add files…", "添加文件…"},
	AddFolder:         {"Add folder…", "添加目录…"},
	RemoveSelected:    {"Remove", "移除"},
	ClearAll:          {"Clear", "清空"},
	OpenAfterDone:     {"Open target when done", "完成后打开目标"},
	HelpBtn:           {"Help", "帮助"},
	HelpText: {"1) Pick files/folders on the Compress tab and choose a format.\n2) Switch to Extract tab and pick an archive to unpack.\n3) Toggle 中文/English anytime via the toolbar.\nSupported formats: zip, tar, tar.gz, 7z, rar (extract-only), iso (extract-only).",
		"1）在“压缩”页选择文件或目录并选择格式。\n2）在“解压”页选择压缩包即可解压。\n3）右上角可随时切换中文/English。\n支持格式：zip、tar、tar.gz、7z、rar（仅解压）、iso（仅解压）。"},
	CompressionProgress: {"Compressing {n}/{total}", "正在压缩 {n}/{total}"},
	ExtractionProgress:  {"Extracting {n}/{total}", "正在解压 {n}/{total}"},
	FileCount:           {"Files: {n}", "文件数：{n}"},
	TotalSize:           {"Total: {size}", "总计：{size}"},
	EstimatedRemaining:  {"ETA: {eta}", "剩余：{eta}"},
	ArchiveContents:     {"Archive contents", "压缩包内容"},
	ExtractHere:         {"Extract here", "解压到当前目录"},
	ExtractHereFolder:   {"Extract to \"{folder}\"", "解压到“{folder}”"},
	ExtractSelected:     {"Extract selected", "解压选中项"},
	Loading:             {"Loading…", "加载中…"},
	NoArchiveSelected:   {"No archive selected. Pick a file above.", "未选择压缩包，请在上方选择文件。"},
	ListFailed:          {"Failed to list archive: ", "读取压缩包内容失败："},
	ExtractHereShort:    {"Here", "当前目录"},
	Background:          {"Background", "后台运行"},
	CancelTask:          {"Cancel", "取消"},
	TaskComplete:        {"Task complete", "任务完成"},
	TaskCompleteBody:    {"{task} finished successfully.", "{task} 已完成。"},
	TaskFailedBody:      {"{task} failed: {error}", "{task} 失败：{error}"},
	CompressingTitle:    {"Compressing {name}", "正在压缩 {name}"},
	ExtractingTitle:     {"Extracting {name}", "正在解压 {name}"},
	VolumeSizeLabel:     {"Split volume", "分卷大小"},
	VolumeSizeHint:      {"e.g. 100m / 1g / 500k (blank = no split)", "如 100m / 1g / 500k（留空=不分卷）"},
}

// T returns the string for the current language.
func T(k key) string {
	row, ok := table[k]
	if !ok {
		return ""
	}
	return row[Get()]
}

// Tf returns the string with {n}, {total}, {size}, {eta} placeholders replaced.
// Only literal "{key}" tokens are substituted; unknown tokens are left as-is.
func Tf(k key, vars map[string]string) string {
	s := T(k)
	for k, v := range vars {
		for {
			i := indexOf(s, "{"+k+"}")
			if i < 0 {
				break
			}
			s = s[:i] + v + s[i+len(k)+2:]
		}
	}
	return s
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// Reset forces the language back to English (used by tests).
func Reset() { Set(English) }
