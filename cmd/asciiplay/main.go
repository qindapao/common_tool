package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// 下面这一行并不是注释哦，是编译阶段告诉编译器去把这个文件嵌入进来
// 一定要使用32x32的图标，用256x256的反而显示得非常不清晰
//
//go:embed assets/icon_32.png
var iconPNG []byte

const (
	appTitle            = "Ascii Motion Player"
	defaultFrameRateMs  = 500
	defaultFontSize     = 12
	maxRecentDirs       = 15
	maxRecentDirNameLen = 16
	configFileName      = ".ascii_motion_player_config.json"

	minFrameRateMs   = 50
	maxFrameRateMs   = 2000
	frameRateStep    = 25
	cacheSizeMax     = 10000
	cacheSizeMin     = 5
	cacheSizeDefault = 50

	reverseSVG = `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
  <path fill="#666666" d="M20 5L8 12L20 19Z"/>
</svg>`
)

// 配置结构体
type PlayerConfig struct {
	RecentDirs      []string `json:"recentDirs"`
	FontSize        int      `json:"fontSize"`
	FontPath        string   `json:"fontPath"`
	FrameRateMs     int      `json:"frameRateMs"`
	CacheWindowSize int      `json:"cacheWindowSize"`
	ControlsVisible bool     `json:"controlsVisible"`
}

// 自定义主题：在 TextStyle.Monospace 为 true 时，返回用户选择的等宽字体资源
type customTheme struct {
	base     fyne.Theme
	monoFont fyne.Resource
}

func (t *customTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	return t.base.Color(n, v)
}
func (t *customTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return t.base.Icon(n)
}
func (t *customTheme) Font(s fyne.TextStyle) fyne.Resource {
	if s.Monospace && t.monoFont != nil {
		return t.monoFont
	}
	return t.base.Font(s)
}
func (t *customTheme) Size(n fyne.ThemeSizeName) float32 {
	return t.base.Size(n)
}

type FramePlayer struct {
	frameFiles      []string
	frameCount      int
	currentFrame    int
	lastShownFrame  int
	playing         bool
	frameRate       time.Duration
	cacheWindowSize int

	displayArea       *fyne.Container
	frameValueLabel   *widget.Label
	speedValueLabel   *widget.Label
	playPauseBtn      *widget.Button
	reversePlaying    bool
	reversePlayBtn    *widget.Button
	recentMenuBtn     *widget.Button
	window            fyne.Window
	reverseIcon       fyne.Resource
	controlsVisible   bool // 是否显示控制条
	controlBar        *fyne.Container
	timelineBar       *fyne.Container
	scrollableContent *container.Scroll

	// 字体设置
	fontSize    int
	fontPath    string
	app         fyne.App
	activeTheme *customTheme

	// 最近打开目录
	recentDirs []string
	recentBar  *fyne.Container // 保留以兼容旧逻辑

	// 配置路径
	configPath string

	// 缓存：索引 -> 渲染好的 CanvasObject
	frameCache map[int]fyne.CanvasObject
}

func NewFramePlayer(a fyne.App, w fyne.Window, configPath string) *FramePlayer {
	// 基于当前主题创建可覆盖等宽字体的主题
	base := a.Settings().Theme()
	if base == nil {
		base = theme.DefaultTheme()
	}
	ct := &customTheme{base: base, monoFont: nil}
	a.Settings().SetTheme(ct)

	p := &FramePlayer{
		frameRate:       defaultFrameRateMs * time.Millisecond,
		window:          w,
		fontSize:        defaultFontSize,
		app:             a,
		activeTheme:     ct,
		recentDirs:      []string{},
		configPath:      configPath,
		frameCache:      make(map[int]fyne.CanvasObject),
		cacheWindowSize: cacheSizeDefault,
		controlsVisible: true, // 默认显示控制条
	}

	p.reverseIcon = fyne.NewStaticResource("play_reverse.svg", []byte(reverseSVG))

	// 启动时加载配置（字号、字体、历史目录）
	p.loadConfig()

	// 如果配置中有字体路径，尝试加载
	if p.fontPath != "" {
		if data, err := os.ReadFile(p.fontPath); err == nil {
			res := fyne.NewStaticResource(filepath.Base(p.fontPath), data)
			p.activeTheme.monoFont = res
			p.app.Settings().SetTheme(p.activeTheme)
		}
	}

	return p
}
func (p *FramePlayer) addRecentDir(path string) {
	// 去重并移到最前
	for i, dir := range p.recentDirs {
		if dir == path {
			copy(p.recentDirs[1:i+1], p.recentDirs[0:i])
			p.recentDirs[0] = path
			p.refreshRecentDirsUI()
			p.saveConfig()
			return
		}
	}
	// 插入最前，限制数量
	p.recentDirs = append([]string{path}, p.recentDirs...)
	if len(p.recentDirs) > maxRecentDirs {
		p.recentDirs = p.recentDirs[:maxRecentDirs]
	}
	p.refreshRecentDirsUI()
	p.saveConfig()
}

func (p *FramePlayer) playFramesReverse() {
	p.reversePlaying = true
	p.updateReversePlayIcon()

	go func() {
		for p.reversePlaying {
			if p.frameCount == 0 {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			index := (p.lastShownFrame - 1 + p.frameCount) % p.frameCount
			p.showFrame(index)
			p.currentFrame = index

			// 每次都读取最新的播放速率
			time.Sleep(p.frameRate)
		}
	}()
}

func (p *FramePlayer) toggleReversePlayPause() {
	if p.frameCount == 0 {
		p.updateFrameInfoLabel(0)
		return
	}
	if p.reversePlaying {
		p.reversePlaying = false
		p.updateReversePlayIcon()
		return
	}

	// 启动反向播放前，确保正向播放停止
	if p.playing {
		p.playing = false
		p.updatePlayPauseIcon()
	}

	go p.playFramesReverse()
}

func (p *FramePlayer) updateReversePlayIcon() {
	fyne.Do(func() {
		if p.reversePlaying {
			p.reversePlayBtn.SetIcon(theme.MediaPauseIcon())
		} else {
			if p.reverseIcon != nil {
				p.reversePlayBtn.SetIcon(p.reverseIcon)
			} else {
				p.reversePlayBtn.SetIcon(theme.NavigateBackIcon()) // 兜底方案
			}
		}
		p.reversePlayBtn.Refresh() // 刷新按钮显示
	})
}

// rune 安全截断：超过 limit 个 rune，则截断并追加 "..."
func truncateRunes(s string, limit int) string {
	r := []rune(s)
	if len(r) <= limit {
		return s
	}
	return string(r[:limit]) + "..."
}

func (p *FramePlayer) refreshRecentDirsUI() {
	// 此方法保留，避免影响 addRecentDir 的调用；但 recentBar 不再显示在 UI 中
	if p.recentBar == nil {
		return
	}
	p.recentBar.Objects = nil
	if len(p.recentDirs) == 0 {
		p.recentBar.Add(widget.NewLabel(""))
		p.recentBar.Refresh()
		return
	}
	p.recentBar.Add(widget.NewLabel("最近打开："))

	for _, dir := range p.recentDirs {
		d := dir // 捕获
		name := filepath.Base(d)
		name = truncateRunes(name, maxRecentDirNameLen)

		btn := widget.NewButtonWithIcon(name, theme.FolderOpenIcon(), func() {
			if err := p.loadFrames(d); err != nil {
				return
			}
			p.window.SetTitle(appTitle + " - " + filepath.Base(d))
			p.addRecentDir(d)
			p.currentFrame = 0
			if p.frameCount > 0 {
				p.showFrame(p.currentFrame)
				p.currentFrame = p.lastShownFrame
			} else {
				fyne.Do(func() {
					p.ShowContent(widget.NewLabel("No valid frames found in directory"))
				})
			}
		})
		btn.Importance = widget.LowImportance
		p.recentBar.Add(btn)
	}

	p.recentBar.Refresh()
}

func (p *FramePlayer) loadFrames(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var frames []string
	for _, entry := range entries {
		if !entry.IsDir() {
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext == ".txt" || ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".svg" {
				frames = append(frames, filepath.Join(dir, entry.Name()))
			}
		}
	}

	// 数字序排序：从文件名末尾提取连续数字并排序；无数字排最后
	sort.Slice(frames, func(i, j int) bool {
		ni := extractNumber(frames[i])
		nj := extractNumber(frames[j])
		if ni != nj {
			return ni < nj
		}
		// 次级：不区分大小写的文件名，保证稳定性
		bi := strings.ToLower(filepath.Base(frames[i]))
		bj := strings.ToLower(filepath.Base(frames[j]))
		return bi < bj
	})

	p.frameFiles = frames
	p.frameCount = len(frames)

	// 切换目录时清空缓存
	p.frameCache = make(map[int]fyne.CanvasObject)

	return nil
}

// 从文件名末尾提取数字（遇到非数字停止），无数字则返回最大值
func extractNumber(path string) int {
	base := filepath.Base(path)
	name := strings.TrimSuffix(base, filepath.Ext(base))

	numStr := ""
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] >= '0' && name[i] <= '9' {
			numStr = string(name[i]) + numStr
		} else {
			if numStr != "" {
				break // 已经开始提取数字，遇到非数字就结束
			}
		}
	}

	if numStr == "" {
		return 1 << 30 // 没有数字，排在最后
	}

	var num int
	if _, err := fmt.Sscanf(numStr, "%d", &num); err != nil {
		return 1 << 30 // 解析失败，排在最后
	}
	return num
}
func (p *FramePlayer) updateFrameInfoLabel(index int) {
	fyne.Do(func() {
		p.frameValueLabel.SetText(fmt.Sprintf("%d / %d", index+1, p.frameCount))
	})
}

// :TODO: 文本超出范围的区域无法出现滚动条，但其实也不需要滚动条。
func (p *FramePlayer) makeTextView(content string) fyne.CanvasObject {
	lines := strings.Split(content, "\n")
	var items []fyne.CanvasObject
	y := float32(0)
	for _, line := range lines {
		t := canvas.NewText(line, theme.Color(theme.ColorNameForeground))
		t.TextStyle = fyne.TextStyle{Monospace: true}
		t.TextSize = float32(p.fontSize)
		t.Move(fyne.NewPos(0, y))
		items = append(items, t)
		y += t.TextSize // 累加高度以紧密排列
	}
	return container.NewWithoutLayout(items...)
}

func renderSVGToObject(path string, want fyne.Size) fyne.CanvasObject {
	_, err := os.Stat(path)
	if err != nil {
		return widget.NewLabel("SVG 文件不存在或无法访问：" + err.Error())
	}

	svgBytes, err := os.ReadFile(path)
	if err != nil {
		return widget.NewLabel("SVG读取失败：" + err.Error())
	}

	header := string(svgBytes)
	if !strings.Contains(header, "<svg") && !strings.Contains(header, "<?xml") {
		return widget.NewLabel("读取到的文件看起来不是 SVG")
	}

	icon, err := oksvg.ReadIconStream(bytes.NewReader(svgBytes))
	if err != nil {
		return widget.NewLabel("解析SVG失败：" + err.Error())
	}

	// 决定渲染尺寸：优先 want，否则尝试 viewBox，再兜底 200x200
	var twf, thf float64
	if want.Width > 0 && want.Height > 0 {
		twf = float64(want.Width)
		thf = float64(want.Height)
	} else {
		vb := icon.ViewBox
		if vb.W > 0 && vb.H > 0 {
			twf = vb.W
			thf = vb.H
		} else {
			twf, thf = 200, 200
		}
	}

	const minSz = 24
	const maxSz = 2048
	if twf < minSz {
		twf = minSz
	}
	if thf < minSz {
		thf = minSz
	}
	if twf > maxSz {
		twf = maxSz
	}
	if thf > maxSz {
		thf = maxSz
	}

	tw := int(twf + 0.5)
	th := int(thf + 0.5)

	icon.SetTarget(0, 0, twf, thf)

	rgba := image.NewRGBA(image.Rect(0, 0, tw, th))
	draw.Draw(rgba, rgba.Bounds(), &image.Uniform{color.Transparent}, image.Point{}, draw.Src)

	scanner := rasterx.NewScannerGV(tw, th, rgba, rgba.Bounds())
	raster := rasterx.NewDasher(tw, th, scanner)
	icon.Draw(raster, 1.0)

	img := canvas.NewImageFromImage(rgba)
	img.FillMode = canvas.ImageFillContain

	if want.Width > 0 && want.Height > 0 {
		img.SetMinSize(want)
	} else {
		img.SetMinSize(fyne.NewSize(float32(tw), float32(th)))
	}

	return container.NewCenter(img)
}

// 中心点触发式缓存刷新 + 显示帧
func (p *FramePlayer) showFrame(index int) {
	// 边界保护
	if p.frameCount == 0 || index < 0 || index >= p.frameCount {
		return
	}
	p.lastShownFrame = index

	// 如果当前帧不在缓存，则以当前帧为中心点，缓存前后 N/2 帧（已在缓存的跳过）
	if _, ok := p.frameCache[index]; !ok {
		// log.Printf("[Cache] Triggered by index: %d", index)
		half := p.cacheWindowSize / 2
		for delta := -half; delta <= half; delta++ {
			j := (index + delta) % p.frameCount
			if j < 0 {
				j += p.frameCount
			}
			if _, exists := p.frameCache[j]; exists {
				continue
			}
			// 加载并渲染第 j 帧
			path := p.frameFiles[j]
			ext := strings.ToLower(filepath.Ext(path))

			var obj fyne.CanvasObject
			switch ext {
			case ".txt":
				content, err := os.ReadFile(path)
				if err != nil {
					obj = widget.NewLabel("读取失败：" + err.Error())
				} else {
					obj = p.makeTextView(string(content))
				}
			case ".svg":
				obj = renderSVGToObject(path, fyne.NewSize(0, 0))
			case ".png", ".jpg", ".jpeg":
				f, err := os.Open(path)
				if err != nil {
					obj = widget.NewLabel("图像读取失败：" + err.Error())
				} else {
					defer func(f *os.File) { _ = f.Close() }(f)
					var img image.Image
					if ext == ".png" {
						img, err = png.Decode(f)
					} else {
						img, err = jpeg.Decode(f)
					}
					if err != nil {
						obj = widget.NewLabel("图像解码失败：" + err.Error())
					} else {
						pic := canvas.NewImageFromImage(img)
						pic.FillMode = canvas.ImageFillOriginal
						obj = container.NewCenter(pic)
					}
				}
			default:
				obj = widget.NewLabel("不支持的格式：" + ext)
			}
			p.frameCache[j] = obj
		}
	}

	// 显示当前帧（无论是否刚刚加载）
	fyne.Do(func() {
		p.displayArea.RemoveAll()
		co := p.frameCache[index]
		// 极端情况下（IO/解码失败）保护
		if co == nil {
			co = widget.NewLabel("加载失败")
		}
		p.displayArea.Add(co)
		p.updateFrameInfoLabel(index)
		p.displayArea.Refresh()
	})

	// 清理远离当前帧的缓存帧
	for k := range p.frameCache {
		if circularDistance(k, index, p.frameCount) > p.cacheWindowSize {
			delete(p.frameCache, k)
		}
	}

	// 显示新帧的时候滚动条放最顶上。
	p.scrollableContent.ScrollToTop()

	// log.Printf("[Cache] Total cached frames: %d", len(p.frameCache))
}

func circularDistance(a, b, total int) int {
	d := abs(a - b)
	return min(d, total-d)
}
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (p *FramePlayer) ShowContent(obj fyne.CanvasObject) {
	p.displayArea.RemoveAll()
	p.displayArea.Add(obj)
	p.displayArea.Refresh()
}

func (p *FramePlayer) playFrames() {
	p.playing = true
	p.updatePlayPauseIcon()

	go func() {
		for p.playing {
			if p.frameCount == 0 {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			index := (p.lastShownFrame + 1) % p.frameCount
			p.showFrame(index)
			p.currentFrame = index

			// Ticker 会自动补偿时间差值，帧率更加稳定。
			// Ticker 适合高帧率的场景，当前情况下我们用Sleep足够
			// 每次都读取最新的播放速率
			time.Sleep(p.frameRate)
		}
	}()
}

func (p *FramePlayer) togglePlayPause() {
	if p.frameCount == 0 {
		p.updateFrameInfoLabel(0)
		return
	}
	if p.playing {
		p.playing = false
		p.updatePlayPauseIcon()
		return
	}

	// 启动正向播放前，确保反向播放停止
	if p.reversePlaying {
		p.reversePlaying = false
		p.updateReversePlayIcon()
	}

	go p.playFrames()
}

func (p *FramePlayer) updatePlayPauseIcon() {
	fyne.Do(func() {
		if p.playing {
			p.playPauseBtn.SetIcon(theme.MediaPauseIcon())
		} else {
			p.playPauseBtn.SetIcon(theme.MediaPlayIcon())
		}
	})
}

func (p *FramePlayer) layoutSpacer() fyne.CanvasObject {
	return container.NewStack(widget.NewLabel(""))
}

// 设置弹窗：将字体相关设置集中在这里
func (p *FramePlayer) showSettingsDialog() {
	// 字号设置
	fontSizeLabel := widget.NewLabel(fmt.Sprintf("Font size: %d", p.fontSize))
	fontSizeSlider := widget.NewSlider(6, 64)
	fontSizeSlider.Step = 1
	fontSizeSlider.Value = float64(p.fontSize)
	fontSizeSlider.OnChanged = func(v float64) {
		p.fontSize = int(v)
		fontSizeLabel.SetText(fmt.Sprintf("Font size: %d", p.fontSize))
		p.saveConfig()
		p.frameCache = make(map[int]fyne.CanvasObject)
		if p.frameCount > 0 {
			p.showFrame(p.lastShownFrame)
		}
	}
	fontSizeSliderWrap := container.NewGridWrap(fyne.NewSize(280, 32), fontSizeSlider)

	// 缓存窗口大小设置（滑块 + 输入框）
	cacheLabel := widget.NewLabel(fmt.Sprintf("Cache size: %d", p.cacheWindowSize))
	cacheSlider := widget.NewSlider(cacheSizeMin, cacheSizeMax)
	cacheSlider.Step = 1
	cacheSlider.Value = float64(p.cacheWindowSize)

	cacheEntry := widget.NewEntry()
	cacheEntry.SetText(fmt.Sprintf("%d", p.cacheWindowSize))
	cacheEntry.Validator = func(s string) error {
		v, err := strconv.Atoi(s)
		if err != nil || v < 5 {
			return fmt.Errorf("请输入 ≥ 5 的整数")
		}
		return nil
	}

	// 滑块变化时更新 Entry 和 Label
	cacheSlider.OnChanged = func(v float64) {
		p.cacheWindowSize = int(v)
		cacheLabel.SetText(fmt.Sprintf("Cache size: %d", p.cacheWindowSize))
		cacheEntry.SetText(fmt.Sprintf("%d", p.cacheWindowSize))
		p.saveConfig()
		p.frameCache = make(map[int]fyne.CanvasObject)
		if p.frameCount > 0 {
			p.showFrame(p.lastShownFrame)
		}
	}

	// Entry变化时更新滑块和 Label
	cacheEntry.OnSubmitted = func(s string) {
		v, err := strconv.Atoi(s)
		if err != nil || v < 5 {
			dialog.ShowError(fmt.Errorf("请输入 ≥ 5 的整数"), p.window)
			return
		}
		p.cacheWindowSize = v
		cacheSlider.Value = float64(v)
		cacheSlider.Refresh()
		cacheLabel.SetText(fmt.Sprintf("Cache size: %d", v))
		p.saveConfig()
		p.frameCache = make(map[int]fyne.CanvasObject)
		if p.frameCount > 0 {
			p.showFrame(p.lastShownFrame)
		}
	}

	cacheSliderWrap := container.NewGridWrap(fyne.NewSize(280, 32), cacheSlider)
	cacheEntryWrap := container.NewGridWrap(fyne.NewSize(80, 32), cacheEntry)

	// 字体选择
	fontName := "默认"
	if p.fontPath != "" {
		fontName = filepath.Base(p.fontPath)
	}
	fontPathLabel := widget.NewLabel("Font: " + fontName)

	sysFontBtn := widget.NewButton("Select font", func() {
		dir := getSystemFontDir()
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			defer func() {
				if err := reader.Close(); err != nil {
					log.Printf("failed to close reader: %v", err)
				}
			}()

			data, rErr := io.ReadAll(reader)
			if rErr != nil {
				fmt.Println("字体读取失败：", rErr)
				return
			}
			path := reader.URI().Path()
			name := filepath.Base(path)
			res := fyne.NewStaticResource(name, data)
			p.activeTheme.monoFont = res
			p.fontPath = path
			p.app.Settings().SetTheme(p.activeTheme)
			p.saveConfig()
			fontPathLabel.SetText("Font: " + filepath.Base(p.fontPath))
			p.frameCache = make(map[int]fyne.CanvasObject)
			if p.frameCount > 0 {
				p.showFrame(p.lastShownFrame)
			}
		}, p.window)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".ttf", ".otf"}))
		if dir != "" {
			if l, err := storage.ListerForURI(storage.NewFileURI(dir)); err == nil {
				fd.SetLocation(l)
			}
		}
		fd.Show()
	})

	// 设置内容布局
	content := container.NewVBox(
		widget.NewLabelWithStyle("Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		container.NewHBox(widget.NewLabel("Font size:"), fontSizeSliderWrap, fontSizeLabel),
		container.NewHBox(widget.NewLabel("Cache size:"), cacheSliderWrap, cacheEntryWrap, cacheLabel),
		container.NewHBox(sysFontBtn, fontPathLabel),
		widget.NewSeparator(),
		makeShortcutInfo(),
	)

	d := dialog.NewCustom("Settings", "Close", content, p.window)
	d.Show()
}

func makeShortcutInfo() fyne.CanvasObject {
	lines := []string{
		"Keyboard Shortcuts:",
		"  Space — Play / Pause (forward)",
		"  r     — Play / Pause (reverse)",
		"  → / l — Next frame",
		"  ← / h — Previous frame",
		"  s     — Stop playback",
		"  o     — Open directory",
		"  e     — Open Recent Directory",
		"  g     — Export GIF",
		"  F12   — Full Screen Switch",
		"  F4    — Toggle button display",
	}
	vbox := container.NewVBox()
	for _, line := range lines {
		t := canvas.NewText(line, theme.Color(theme.ColorNameForeground))
		t.TextStyle = fyne.TextStyle{Monospace: true}
		t.TextSize = 14
		vbox.Add(t)
	}
	return vbox
}
func (p *FramePlayer) SetupUI() fyne.CanvasObject {
	p.displayArea = container.NewVBox()

	// TimeLine
	timelineSlider := widget.NewSlider(0, 1)
	timelineSlider.Step = 1
	timelineSlider.Value = 0
	timelineSlider.OnChanged = func(v float64) {
		if p.frameCount <= 0 {
			return
		}
		idx := int(v + 0.5)
		idx = max(0, idx)
		if idx >= p.frameCount {
			idx = p.frameCount - 1
		}
		p.showFrame(idx)
		p.currentFrame = idx
	}

	go func() {
		tk := time.NewTicker(150 * time.Millisecond)
		defer tk.Stop()
		for range tk.C {
			fyne.Do(func() {
				maxV := float64(max(1, p.frameCount-1))
				if timelineSlider.Max != maxV {
					timelineSlider.Max = maxV
					timelineSlider.Refresh()
				}
				if int(timelineSlider.Value+0.5) != p.lastShownFrame {
					timelineSlider.Value = float64(p.lastShownFrame)
					timelineSlider.Refresh()
				}
			})
		}
	}()

	// Frame info
	frameTitle := widget.NewLabel("Current frame:")
	p.frameValueLabel = widget.NewLabel("0 / 0")
	p.frameValueLabel.Alignment = fyne.TextAlignLeading
	frameValueFixed := container.NewGridWrap(fyne.NewSize(120, 32), p.frameValueLabel)
	frameInfo := container.NewHBox(frameTitle, frameValueFixed)

	// Speed control
	speedTitle := widget.NewLabel("Speed:")
	p.speedValueLabel = widget.NewLabel(fmt.Sprintf("%dms", int(p.frameRate.Milliseconds())))
	p.speedValueLabel.Alignment = fyne.TextAlignLeading
	speedValueFixed := container.NewGridWrap(fyne.NewSize(60, 32), p.speedValueLabel)
	speedSlider := widget.NewSlider(minFrameRateMs, maxFrameRateMs)
	speedSlider.Step = frameRateStep
	speedSlider.Value = float64(p.frameRate.Milliseconds())
	speedSlider.OnChanged = func(v float64) {
		p.frameRate = time.Duration(v) * time.Millisecond
		p.speedValueLabel.SetText(fmt.Sprintf("%dms", int(v)))
		p.updateFrameInfoLabel(p.lastShownFrame)
		p.saveConfig()
	}
	speedSliderWrap := container.NewGridWrap(fyne.NewSize(300, 32), speedSlider)
	speedControl := container.NewHBox(speedTitle, speedValueFixed, speedSliderWrap)

	// 选择目录按钮
	unifiedDirBtn := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		start := ""
		if len(p.recentDirs) > 0 {
			start = p.recentDirs[0]
		} else if home, err := os.UserHomeDir(); err == nil {
			start = home
		}
		if start == "" {
			start, _ = os.Getwd()
		}
		// 回到上层目录
		start = filepath.Dir(start)
		p.showDirSelector(start)
	})

	// 最近打开下拉菜单
	p.recentMenuBtn = widget.NewButtonWithIcon("", theme.MenuDropDownIcon(), func() {
		if len(p.recentDirs) == 0 {
			dialog.ShowInformation("最近打开", "暂无历史记录", p.window)
			return
		}
		items := make([]*fyne.MenuItem, 0, len(p.recentDirs))
		for _, dir := range p.recentDirs {
			d := dir
			name := filepath.Base(d)
			name = truncateRunes(name, maxRecentDirNameLen)
			items = append(items, fyne.NewMenuItem(name, func() {
				if err := p.loadFrames(d); err != nil {
					return
				}
				p.window.SetTitle(appTitle + " - " + filepath.Base(d))
				p.addRecentDir(d)
				p.currentFrame = 0
				if p.frameCount > 0 {
					p.showFrame(p.currentFrame)
					p.currentFrame = p.lastShownFrame
				} else {
					fyne.Do(func() {
						p.ShowContent(widget.NewLabel("No valid frames found in directory"))
					})
				}
			}))
		}
		menu := fyne.NewMenu("最近打开", items...)
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(p.recentMenuBtn)
		btnSize := p.recentMenuBtn.Size()
		anchor := fyne.NewPos(pos.X, pos.Y+btnSize.Height+theme.Padding())
		widget.ShowPopUpMenuAtPosition(menu, p.window.Canvas(), anchor)
	})

	// 播放控制按钮
	p.playPauseBtn = widget.NewButtonWithIcon("", theme.MediaPlayIcon(), func() {
		p.togglePlayPause()
	})
	p.reversePlayBtn = widget.NewButtonWithIcon("", p.reverseIcon, func() {
		p.toggleReversePlayPause()
	})
	stopBtn := widget.NewButtonWithIcon("", theme.MediaStopIcon(), func() {
		p.playing = false
		p.reversePlaying = false
		p.updatePlayPauseIcon()
		p.updateReversePlayIcon()

		if p.frameCount == 0 {
			return
		}
		p.currentFrame = 0
		p.showFrame(p.currentFrame)
		p.currentFrame = p.lastShownFrame
	})

	prevBtn := widget.NewButtonWithIcon("", theme.MediaSkipPreviousIcon(), func() {
		if p.frameCount == 0 {
			return
		}
		index := (p.lastShownFrame - 1 + p.frameCount) % p.frameCount
		p.showFrame(index)
		p.currentFrame = p.lastShownFrame
	})
	nextBtn := widget.NewButtonWithIcon("", theme.MediaSkipNextIcon(), func() {
		if p.frameCount == 0 {
			return
		}
		index := (p.lastShownFrame + 1) % p.frameCount
		p.showFrame(index)
		p.currentFrame = p.lastShownFrame
	})
	settingsBtn := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		p.showSettingsDialog()
	})

	// 导出 GIF 按钮
	exportBtn := widget.NewButtonWithIcon("", theme.DocumentSaveIcon(), func() {
		p.showExportGIFDialog()
	})

	// 顶部控制条：仅保留一行按钮 + 速度控制
	topRow := container.NewHBox(
		frameInfo,
		p.playPauseBtn, p.reversePlayBtn, stopBtn, prevBtn, nextBtn,
		unifiedDirBtn, p.recentMenuBtn, settingsBtn, exportBtn,
		p.layoutSpacer(),
		speedControl,
	)
	p.controlBar = topRow

	timelineBar := container.NewVBox(timelineSlider)
	p.timelineBar = timelineBar

	// 初始可见性：根据配置决定是否隐藏
	if !p.controlsVisible {
		p.controlBar.Hide()
	}

	// 布局：上控制条 + 下内容区 + 进度条
	p.scrollableContent = container.NewScroll(p.displayArea)
	return container.NewBorder(p.controlBar, timelineBar, nil, nil, p.scrollableContent)
}

// 获取系统字体目录（首选常见路径，若存在则使用）
func getSystemFontDir() string {
	var candidates []string
	switch runtime.GOOS {
	case "windows":
		userFontDir := filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Windows", "Fonts")
		if st, err := os.Stat(userFontDir); err == nil && st.IsDir() {
			return userFontDir
		}
		return `C:\Windows\Fonts`
	case "darwin":
		candidates = []string{
			`/System/Library/Fonts`,
			`/Library/Fonts`,
			filepath.Join(os.Getenv("HOME"), "Library", "Fonts"),
		}
	case "linux":
		candidates = []string{
			`/usr/share/fonts`,
			`/usr/local/share/fonts`,
			filepath.Join(os.Getenv("HOME"), ".fonts"),
		}
	default:
		return ""
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c
		}
	}
	return ""
}

// 配置加载
func (p *FramePlayer) loadConfig() {
	data, err := os.ReadFile(p.configPath)
	if err != nil {
		return // 文件不存在或读取失败，跳过
	}
	var cfg PlayerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Println("配置解析失败：", err)
		return
	}

	p.recentDirs = cfg.RecentDirs
	p.fontSize = cfg.FontSize
	p.fontPath = cfg.FontPath
	p.frameRate = time.Duration(cfg.FrameRateMs) * time.Millisecond
	p.cacheWindowSize = cfg.CacheWindowSize
	p.controlsVisible = cfg.ControlsVisible
}

// 配置保存
func (p *FramePlayer) saveConfig() {
	cfg := PlayerConfig{
		RecentDirs:      p.recentDirs,
		FontSize:        p.fontSize,
		FontPath:        p.fontPath,
		FrameRateMs:     int(p.frameRate.Milliseconds()),
		CacheWindowSize: p.cacheWindowSize,
		ControlsVisible: p.controlsVisible,
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		fmt.Println("配置序列化失败：", err)
		return
	}
	if err := os.WriteFile(p.configPath, data, 0644); err != nil {
		fmt.Println("配置保存失败：", err)
	}
}

// 统一目录选择器：面包屑导航 + 过滤 + Windows 盘符切换（修复崩溃）
func (p *FramePlayer) showDirSelector(start string) {
	cur := filepath.Clean(start)
	win := fyne.CurrentApp().NewWindow("Select Dir")
	win.Resize(fyne.NewSize(640, 520))

	search := widget.NewEntry()
	search.SetPlaceHolder("Enter keywords to filter subdirectories at the current level...")

	var subdirs []string
	var filtered []string

	readSubdirs := func(dir string) []string {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil
		}
		dirs := make([]string, 0, len(entries))
		for _, e := range entries {
			if e.IsDir() {
				dirs = append(dirs, e.Name())
			}
		}
		sort.Strings(dirs)
		return dirs
	}

	applyFilter := func() {
		kw := strings.ToLower(strings.TrimSpace(search.Text))
		if kw == "" {
			filtered = slices.Clone(subdirs)
			return
		}
		filtered = filtered[:0]
		for _, name := range subdirs {
			if strings.Contains(strings.ToLower(name), kw) {
				filtered = append(filtered, name)
			}
		}
	}

	list := widget.NewList(
		func() int { return len(filtered) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i int, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(filtered[i])
		},
	)

	pathBar := container.NewHBox()
	pathScroll := container.NewHScroll(pathBar)
	pathScroll.SetMinSize(fyne.NewSize(0, 32))

	var refreshForDir func()

	makePathBar := func() (*fyne.Container, *widget.Button) {
		bar := container.NewHBox()
		var lastBtn *widget.Button

		clean := filepath.Clean(cur)
		vol := filepath.VolumeName(clean)
		rest := clean[len(vol):]

		segments := []string{}
		if rest != "" {
			rest = strings.TrimPrefix(rest, string(os.PathSeparator))
			if rest != "" {
				segments = strings.Split(rest, string(os.PathSeparator))
			}
		}

		rootLabel := "/"
		rootPath := string(os.PathSeparator)
		if vol != "" {
			rootLabel = vol + `\`
			rootPath = vol + `\`
		}
		rootBtn := widget.NewButton(rootLabel, func() {
			cur = rootPath
			refreshForDir()
		})
		rootBtn.Importance = widget.LowImportance
		bar.Add(rootBtn)
		lastBtn = rootBtn

		accum := rootPath
		for _, seg := range segments {
			// bar.Add(widget.NewLabel("/"))
			bar.Add(widget.NewLabel(""))
			accum = filepath.Join(accum, seg)
			segPath := accum
			btn := widget.NewButton(seg, func() {
				cur = segPath
				refreshForDir()
			})
			btn.Importance = widget.LowImportance
			bar.Add(btn)
			lastBtn = btn
		}
		return bar, lastBtn
	}

	var driveSelect *widget.Select
	if runtime.GOOS == "windows" {
		drives := listWindowsDrives()
		driveSelect = widget.NewSelect(drives, nil)
		vol := filepath.VolumeName(cur)
		if vol != "" {
			driveSelect.Selected = vol + `\`
		} else if len(drives) > 0 {
			driveSelect.Selected = drives[0]
			cur = drives[0]
		}
	}

	selectBtn := widget.NewButtonWithIcon("Select Current Directory", theme.ConfirmIcon(), func() {
		if err := p.loadFrames(cur); err != nil {
			dialog.ShowError(err, win)
			return
		}
		p.window.SetTitle(appTitle + " - " + filepath.Base(cur))
		p.addRecentDir(cur)
		p.saveConfig()
		p.currentFrame = 0
		if p.frameCount > 0 {
			p.showFrame(p.currentFrame)
			p.currentFrame = p.lastShownFrame
		} else {
			fyne.Do(func() {
				p.ShowContent(widget.NewLabel("No valid frames found in directory"))
			})
		}
		win.Close()
	})

	list.OnSelected = func(i int) {
		if i < 0 || i >= len(filtered) {
			return
		}
		cur = filepath.Join(cur, filtered[i])
		search.SetText("")
		refreshForDir()
	}

	search.OnChanged = func(_ string) {
		applyFilter()
		list.Refresh()
	}

	refreshForDir = func() {
		subdirs = readSubdirs(cur)
		applyFilter()
		list.UnselectAll()
		list.Refresh()

		newBar, _ := makePathBar()
		pathBar.Objects = newBar.Objects
		pathBar.Refresh()

		if driveSelect != nil {
			vol := filepath.VolumeName(cur)
			if vol != "" {
				want := vol + `\`
				if driveSelect.Selected != want {
					driveSelect.Selected = want
					driveSelect.Refresh()
				}
			}
		}

		// 滚动到最右边
		fyne.Do(func() {
			pathScroll.ScrollToOffset(fyne.NewPos(pathBar.Size().Width, 0))
		})
	}

	pathBar.Refresh()

	if driveSelect != nil {
		driveSelect.OnChanged = func(selected string) {
			if selected == "" {
				return
			}
			cur = filepath.Clean(selected)
			search.SetText("")
			refreshForDir()
		}
	}

	refreshForDir()

	header := container.NewHBox(selectBtn)
	top := container.NewVBox(pathScroll, search, header)
	if driveSelect != nil {
		top = container.NewVBox(driveSelect, pathScroll, search, header)
	}

	win.SetContent(container.NewBorder(top, nil, nil, nil, list))
	win.Show()
}
func listWindowsDrives() []string {
	drives := []string{}
	for _, letter := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
		drive := fmt.Sprintf("%c:\\", letter)
		if fi, err := os.Stat(drive); err == nil && fi.IsDir() {
			drives = append(drives, drive)
		}
	}
	return drives
}

func (p *FramePlayer) BindKeyEvents(canvas fyne.Canvas) {
	canvas.SetOnTypedKey(func(k *fyne.KeyEvent) {
		switch k.Name {
		case fyne.KeySpace:
			p.togglePlayPause()
		case fyne.KeyR:
			p.toggleReversePlayPause()
		case fyne.KeyRight, fyne.KeyL:
			if p.frameCount > 0 {
				index := (p.lastShownFrame + 1) % p.frameCount
				p.showFrame(index)
				p.currentFrame = p.lastShownFrame
			}
		case fyne.KeyLeft, fyne.KeyH:
			if p.frameCount > 0 {
				index := (p.lastShownFrame - 1 + p.frameCount) % p.frameCount
				p.showFrame(index)
				p.currentFrame = p.lastShownFrame
			}
		case fyne.KeyS:
			p.playing = false
			p.reversePlaying = false
			p.updatePlayPauseIcon()
			p.updateReversePlayIcon()
			if p.frameCount > 0 {
				p.currentFrame = 0
				p.showFrame(p.currentFrame)
				p.currentFrame = p.lastShownFrame
			}
		case fyne.KeyF12:
			p.window.SetFullScreen(!p.window.FullScreen())
		case fyne.KeyF4:
			p.toggleControlsVisible()
		}
	})
}

func (p *FramePlayer) toggleControlsVisible() {
	p.controlsVisible = !p.controlsVisible

	if p.controlBar != nil {
		if p.controlsVisible {
			p.controlBar.Show()
		} else {
			p.controlBar.Hide()
		}
		p.controlBar.Refresh()
	}
	if p.timelineBar != nil {
		if p.controlsVisible {
			p.timelineBar.Show()
		} else {
			p.timelineBar.Hide()
		}
		p.timelineBar.Refresh()
	}
	if p.displayArea != nil {
		p.displayArea.Refresh()
	}
	p.saveConfig()
}

func (p *FramePlayer) BindRuneEvents(canvas fyne.Canvas) {
	canvas.SetOnTypedRune(func(r rune) {
		switch r {
		case 'o':
			start := ""
			if len(p.recentDirs) > 0 {
				start = p.recentDirs[0]
			} else if home, err := os.UserHomeDir(); err == nil {
				start = home
			}
			if start == "" {
				start, _ = os.Getwd()
			}
			start = filepath.Dir(start)
			p.showDirSelector(start)
		case 'e':
			if p.recentMenuBtn != nil && p.recentMenuBtn.OnTapped != nil {
				p.recentMenuBtn.OnTapped()
			}
		case 'g':
			p.showExportGIFDialog()
		}
	})
}

func (p *FramePlayer) showExportGIFDialog() {
	fd := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil || writer == nil {
			return
		}
		defer func() { _ = writer.Close() }()

		path := writer.URI().Path()
		if !strings.HasSuffix(strings.ToLower(path), ".gif") {
			path += ".gif"
		}

		// 自定义进度弹窗
		bar := widget.NewProgressBarInfinite()
		msg := widget.NewLabel("Exporting in progress, please wait...")
		content := container.NewVBox(msg, bar)
		prog := dialog.NewCustomWithoutButtons("Export GIF", content, p.window)

		// 显示进度条（主线程）
		fyne.Do(func() {
			bar.Start()
			prog.Show()
		})

		// 后台导出任务
		go func() {
			err := p.ExportGIF(path)

			// 所有 UI 操作必须在主线程中执行
			fyne.Do(func() {
				bar.Stop()
				prog.Hide()

				if err != nil {
					dialog.ShowError(err, p.window)
				} else {
					p.app.SendNotification(&fyne.Notification{
						Title:   "Export completed",
						Content: "GIF has been exported: " + path,
					})
					// dialog.ShowInformation("Export GIF", "Export completed: \n"+path, p.window)
				}
			})
		}()
	}, p.window)

	fd.SetFileName("output.gif")
	fd.Show()
}

// ========== 导出 GIF：严格基于文件名渲染，不使用缓存 ==========

// ExportGIF 根据当前目录中的帧文件（按已排序顺序），将所有帧导出为 GIF。
// 要求：
// - 仅使用文件名，不依赖 UI 缓存（避免不完整）。
// - 文本帧使用用户定义的等宽字体及字号渲染，尽可能与显示一致。
// - 播放频率（delay）与当前设置一致，单位为 1/100 秒。
func (p *FramePlayer) ExportGIF(outPath string) error {
	if p.frameCount == 0 {
		return fmt.Errorf("没有可导出的帧")
	}

	// 预加载字体（文本帧使用）
	face, lineHeightPx, err := p.prepareFontFace()
	if err != nil {
		return fmt.Errorf("加载字体失败：%w", err)
	}
	defer func() {
		// opentype.NewFace 返回的是 font.Face，不需要额外 Close
		_ = face
	}()

	// 第一次遍历：计算每一帧的自然尺寸，得到最大尺寸（GIF 所有帧统一尺寸）
	naturalSizes := make([]image.Rectangle, p.frameCount)
	maxW, maxH := 1, 1

	for i, path := range p.frameFiles {
		w, h, err := p.measureFrame(path, face, lineHeightPx)
		if err != nil {
			// 出错也不要中断，避免影响整体导出
			w, h = 200, 200
		}
		naturalSizes[i] = image.Rect(0, 0, w, h)
		if w > maxW {
			maxW = w
		}
		if h > maxH {
			maxH = h
		}
	}
	if maxW < 1 {
		maxW = 1
	}
	if maxH < 1 {
		maxH = 1
	}

	// 第二次遍历：按统一尺寸渲染每帧到 RGBA，再转 Paletted，写入 GIF
	outGif := &gif.GIF{
		Image:     make([]*image.Paletted, 0, p.frameCount),
		Delay:     make([]int, 0, p.frameCount),
		LoopCount: 0, // 0 表示循环播放
	}

	bgCol, fgCol := p.exportColors()
	delay := int(p.frameRate.Milliseconds() / 10)
	delay = max(1, delay)

	for _, path := range p.frameFiles {
		rgba := image.NewRGBA(image.Rect(0, 0, maxW, maxH))
		draw.Draw(rgba, rgba.Bounds(), &image.Uniform{bgCol}, image.Point{}, draw.Src)

		if err := p.renderFrameIntoRGBA(path, face, lineHeightPx, rgba, fgCol); err != nil {
			// 出错时，写错误提示文本
			p.drawErrorText(rgba, fmt.Sprintf("Frame error: %v", err), fgCol, face, lineHeightPx)
		}

		// 转为调色板图（GIF 需要）
		paletted := imageToPaletted(rgba)
		outGif.Image = append(outGif.Image, paletted)
		outGif.Delay = append(outGif.Delay, delay)
	}

	// 写出
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	if err := gif.EncodeAll(f, outGif); err != nil {
		return err
	}
	return nil
}

// 选择导出使用的前景/背景色，尽量贴合当前主题
func (p *FramePlayer) exportColors() (bg color.Color, fg color.Color) {
	th := p.app.Settings().Theme()
	variant := p.app.Settings().ThemeVariant()
	bg = th.Color(theme.ColorNameBackground, variant)
	fg = th.Color(theme.ColorNameForeground, variant)

	// 背景透明会导致 GIF 不一致，这里确保有色背景
	if n := color.NRGBAModel.Convert(bg).(color.NRGBA); n.A == 0 {
		bg = color.White
	}
	return bg, fg
}

func getLineHeight(face font.Face, fontSize int) int {
	met := face.Metrics()
	lineH := (met.Ascent + met.Descent).Ceil()

	// 如果字体度量异常或行高太小，就用估算值兜底
	if lineH <= 0 || lineH < int(float64(fontSize)*0.9) {
		lineH = int(float64(fontSize) * 1.35)
	}
	return lineH
}

// 载入字体：优先用户自定义字体（ttf/otf），否则退回 basicfont
func (p *FramePlayer) prepareFontFace() (font.Face, int, error) {
	if p.fontPath != "" {
		data, err := os.ReadFile(p.fontPath)
		if err == nil {
			ft, err := opentype.Parse(data)
			if err == nil {
				adjustedSize := adjustFontSizeForExport(p.fontSize, ft)
				face, err := opentype.NewFace(ft, &opentype.FaceOptions{
					Size:    float64(adjustedSize),
					DPI:     96,
					Hinting: font.HintingFull,
				})
				if err == nil {
					lineH := getLineHeight(face, adjustedSize)
					return face, lineH, nil
				}
			}
		}
	}
	// fallback 到 basicfont，仅当用户字体加载失败
	face := basicfont.Face7x13
	lineH := 13
	return face, lineH, nil
}

// 测量一帧的自然尺寸（不缩放）：文本按字体；位图按原尺寸；SVG按 viewBox 或默认
func (p *FramePlayer) measureFrame(path string, face font.Face, lineH int) (int, int, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".txt":
		b, err := os.ReadFile(path)
		if err != nil {
			return 200, 200, err
		}
		lines := strings.Split(string(b), "\n")
		maxAdvance := fixed.I(0)
		d := &font.Drawer{Face: face}
		for _, ln := range lines {
			adv := d.MeasureString(ln)
			if adv > maxAdvance {
				maxAdvance = adv
			}
		}
		w := (maxAdvance.Ceil())
		h := (len(lines) * lineH)
		if w < 1 {
			w = 1
		}
		if h < 1 {
			h = 1
		}
		return w, h, nil
	case ".png", ".jpg", ".jpeg":
		f, err := os.Open(path)
		if err != nil {
			return 200, 200, err
		}
		defer func() { _ = f.Close() }()
		var img image.Image
		if ext == ".png" {
			img, err = png.Decode(f)
		} else {
			img, err = jpeg.Decode(f)
		}
		if err != nil {
			return 200, 200, err
		}
		b := img.Bounds()
		return b.Dx(), b.Dy(), nil
	case ".svg":
		data, err := os.ReadFile(path)
		if err != nil {
			return 200, 200, err
		}
		icon, err := oksvg.ReadIconStream(bytes.NewReader(data))
		if err != nil {
			return 200, 200, err
		}
		vb := icon.ViewBox
		w, h := int(vb.W+0.5), int(vb.H+0.5)
		if w <= 0 || h <= 0 {
			w, h = 200, 200
		}
		return w, h, nil
	default:
		return 200, 200, fmt.Errorf("不支持的格式：%s", ext)
	}
}

// 将指定帧渲染到统一尺寸的 RGBA 上（内容不拉伸，居中）
func (p *FramePlayer) renderFrameIntoRGBA(path string, face font.Face, lineH int, canvasRGBA *image.RGBA, fg color.Color) error {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".txt":
		return p.drawTextIntoRGBA(path, face, lineH, canvasRGBA, fg)
	case ".png", ".jpg", ".jpeg":
		return p.drawImageIntoRGBA(path, canvasRGBA)
	case ".svg":
		return p.drawSVGIntoRGBA(path, canvasRGBA)
	default:
		return fmt.Errorf("不支持的格式：%s", ext)
	}
}

func (p *FramePlayer) drawTextIntoRGBA(path string, face font.Face, lineH int, dst *image.RGBA, fg color.Color) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// \U0001f527 文本预处理：去 BOM、统一换行、展开 Tab
	if len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		b = b[3:]
	}
	s := string(b)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "\t", "    ")
	lines := strings.Split(s, "\n")

	// \U0001f527 计算最大行宽（逐字符测量）
	maxWidth := 0
	for _, ln := range lines {
		width := 0
		for _, r := range ln {
			adv, ok := face.GlyphAdvance(r)
			if !ok || adv == 0 {
				adv = fixed.I(p.fontSize) // 兜底宽度
			}
			width += adv.Ceil()
		}
		if width > maxWidth {
			maxWidth = width
		}
	}
	textW := maxWidth
	textH := len(lines) * lineH

	// \U0001f527 居中起点
	dstW, dstH := dst.Bounds().Dx(), dst.Bounds().Dy()
	startX := (dstW - textW) / 2
	startY := (dstH - textH) / 2

	// \U0001f527 绘制每行逐字符
	d := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(fg),
		Face: face,
	}
	y := startY + face.Metrics().Ascent.Ceil()
	for _, ln := range lines {
		x := startX
		for _, r := range ln {
			d.Dot = fixed.P(x, y)
			d.DrawString(string(r))
			adv, ok := face.GlyphAdvance(r)
			if !ok || adv == 0 {
				adv = fixed.I(p.fontSize)
			}
			x += adv.Ceil()
		}
		y += lineH
	}
	return nil
}

func (p *FramePlayer) drawImageIntoRGBA(path string, dst *image.RGBA) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	ext := strings.ToLower(filepath.Ext(path))
	var imgSrc image.Image
	if ext == ".png" {
		imgSrc, err = png.Decode(f)
	} else {
		imgSrc, err = jpeg.Decode(f)
	}
	if err != nil {
		return err
	}
	b := imgSrc.Bounds()
	dw, dh := dst.Bounds().Dx(), dst.Bounds().Dy()
	off := image.Pt((dw-b.Dx())/2, (dh-b.Dy())/2)
	r := image.Rectangle{Min: off, Max: off.Add(b.Size())}
	draw.Draw(dst, r, imgSrc, b.Min, draw.Over)
	return nil
}

func (p *FramePlayer) drawSVGIntoRGBA(path string, dst *image.RGBA) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	icon, err := oksvg.ReadIconStream(bytes.NewReader(data))
	if err != nil {
		return err
	}
	vb := icon.ViewBox
	w, h := int(vb.W+0.5), int(vb.H+0.5)
	if w <= 0 || h <= 0 {
		w, h = 200, 200
	}
	// 在单独 RGBA 渲染，再居中绘制到 dst
	tmp := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(tmp, tmp.Bounds(), &image.Uniform{color.Transparent}, image.Point{}, draw.Src)
	icon.SetTarget(0, 0, float64(w), float64(h))
	scanner := rasterx.NewScannerGV(w, h, tmp, tmp.Bounds())
	r := rasterx.NewDasher(w, h, scanner)
	icon.Draw(r, 1.0)

	dw, dh := dst.Bounds().Dx(), dst.Bounds().Dy()
	off := image.Pt((dw-w)/2, (dh-h)/2)
	rect := image.Rectangle{Min: off, Max: off.Add(tmp.Bounds().Size())}
	draw.Draw(dst, rect, tmp, image.Point{}, draw.Over)
	return nil
}

func (p *FramePlayer) drawErrorText(dst *image.RGBA, msg string, fg color.Color, face font.Face, lineH int) {
	d := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(fg),
		Face: face,
	}
	width := 0
	for _, r := range msg {
		adv, ok := face.GlyphAdvance(r)
		if !ok || adv == 0 {
			adv = fixed.I(p.fontSize)
		}
		width += adv.Ceil()
	}
	x := (dst.Bounds().Dx() - width) / 2
	y := (dst.Bounds().Dy()-lineH)/2 + face.Metrics().Ascent.Ceil()

	for _, r := range msg {
		d.Dot = fixed.P(x, y)
		d.DrawString(string(r))
		adv, ok := face.GlyphAdvance(r)
		if !ok || adv == 0 {
			adv = fixed.I(p.fontSize)
		}
		x += adv.Ceil()
	}
}

func adjustFontSizeForExport(size int, ft *opentype.Font) int {
	// 检查当前字号是否对齐
	if isDoubleWidthAligned(ft, size) {
		// fmt.Printf("当前字号 %d 已对齐，无需调整\n", size)
		return size
	}

	// 尝试上下浮动 ±3 的字号
	for offset := 1; offset <= 3; offset++ {
		if isDoubleWidthAligned(ft, size-offset) {
			// fmt.Printf("字号 %d 未对齐，调整为 %d（向下修正）\n", size, size-offset)
			return size - offset
		}
		if isDoubleWidthAligned(ft, size+offset) {
			// fmt.Printf("字号 %d 未对齐，调整为 %d（向上修正）\n", size, size+offset)
			return size + offset
		}
	}

	// 都不满足，返回原始字号
	// fmt.Printf("字号 %d 无法对齐，使用原始字号\n", size)
	return size
}

// 检查双宽字符是否是英文字符宽度的两倍
func isDoubleWidthAligned(ft *opentype.Font, fontSize int) bool {
	face, err := opentype.NewFace(ft, &opentype.FaceOptions{
		Size:    float64(fontSize),
		DPI:     96,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return false
	}
	defer func() {
		_ = face // opentype.NewFace 返回 font.Face，无需关闭
	}()

	advanceA, okA := face.GlyphAdvance('A')
	advanceDouble, okDouble := face.GlyphAdvance('你')

	if !okA || !okDouble {
		return false // 无法测量字符宽度，返回不对齐
	}

	return advanceDouble.Round() == 2*advanceA.Round()
}

// 将 RGBA 转换为调色板图像（GIF 需要）
func imageToPaletted(src image.Image) *image.Paletted {
	// 构建 web 安全调色板（216 色）
	palette := make(color.Palette, 0, 216)
	for _, r := range []uint8{0, 51, 102, 153, 204, 255} {
		for _, g := range []uint8{0, 51, 102, 153, 204, 255} {
			for _, b := range []uint8{0, 51, 102, 153, 204, 255} {
				palette = append(palette, color.RGBA{R: r, G: g, B: b, A: 255})
			}
		}
	}

	b := src.Bounds()
	dst := image.NewPaletted(b, palette)

	// 使用 Floyd-Steinberg 抖动
	draw.FloydSteinberg.Draw(dst, b, src, image.Point{})
	return dst
}

func main() {
	// 配置文件路径：用户主目录下
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("无法获取用户主目录：", err)
		return
	}
	configPath := filepath.Join(home, configFileName)

	a := app.NewWithID("com.example.asciiplay")
	w := a.NewWindow(appTitle)

	res := fyne.NewStaticResource("icon.png", iconPNG)
	w.SetIcon(res)

	w.Resize(fyne.NewSize(900, 620))

	player := NewFramePlayer(a, w, configPath)
	layout := player.SetupUI()
	w.SetContent(layout)

	// 启动时自动打开最近一次目录并应用标题、显示首帧
	if len(player.recentDirs) > 0 {
		dir := player.recentDirs[0]
		if err := player.loadFrames(dir); err == nil && player.frameCount > 0 {
			player.window.SetTitle(appTitle + " - " + filepath.Base(dir))
			player.currentFrame = 0
			player.showFrame(player.currentFrame)
			player.currentFrame = player.lastShownFrame
		}
	}

	player.BindKeyEvents(w.Canvas())
	player.BindRuneEvents(w.Canvas())

	w.ShowAndRun()
}
