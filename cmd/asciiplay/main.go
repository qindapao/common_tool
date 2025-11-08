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
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
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
var soilGray = color.NRGBA{R: 120, G: 120, B: 120, A: 255}

const (
	appTitle            = "Ascii Motion Player"
	appVersion          = "v1.0.1"
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
	frameTexts      map[int]string
	frameImages     map[int]image.Image
	frameDir        string
	frameCount      int
	currentFrame    int
	lastShownFrame  int
	playing         bool
	frameRate       time.Duration
	cacheWindowSize int

	displayArea fyne.CanvasObject

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

	frameMapScroll  *container.Scroll
	frameMapBox     *fyne.Container
	frameMapButtons []*widget.Button

	hideAllControls  bool
	frameMapOnlyMode bool

	frameMapDirty bool

	// 播放循环等待原子锁(0: 不在播放循环 1: 在播放循环)
	inPlayingLock int32

	ghostMode bool

	refreshMode  bool
	lastModTimes map[string]time.Time

	refreshTicker *time.Ticker
	refreshQuit   chan struct{}
}

type FrameDisplay struct {
	widget.BaseWidget
	p       *FramePlayer
	content fyne.CanvasObject
	box     *fyne.Container
}

func NewFrameDisplay(p *FramePlayer) *FrameDisplay {
	d := &FrameDisplay{p: p}
	d.ExtendBaseWidget(d)
	return d
}

func (d *FrameDisplay) CreateRenderer() fyne.WidgetRenderer {
	d.box = container.NewStack()
	if d.content != nil {
		d.box.Add(d.content)
	}
	return widget.NewSimpleRenderer(d.box)
}

func (d *FrameDisplay) SetContent(obj fyne.CanvasObject) {
	d.content = obj
	if d.box != nil {
		d.box.Objects = []fyne.CanvasObject{obj}
		d.box.Refresh()
	}
	d.Refresh()
}

func (d *FrameDisplay) Scrolled(ev *fyne.ScrollEvent) {
	if ev.Scrolled.DY < 0 {
		d.p.nextFrame()
	} else if ev.Scrolled.DY > 0 {
		d.p.prevFrame()
	}
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
		frameTexts:      make(map[int]string),
		frameImages:     make(map[int]image.Image),
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

func (p *FramePlayer) setWindowTitleWithDir(dir string) {
	title := appTitle + " " + appVersion + " - " + filepath.Base(dir)
	if p.ghostMode {
		title += " [GhostMode]"
	}

	if p.refreshMode {
		title += " [RefreshMode]"
	}
	p.window.SetTitle(title)
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

func (p *FramePlayer) playFrames(direction int) {
	// direction = +1 表示正向播放，-1 表示反向播放
	if direction > 0 {
		p.playing = true
		p.reversePlaying = false
		p.updatePlayPauseIcon()
	} else {
		p.reversePlaying = true
		p.playing = false
		p.updateReversePlayIcon()
	}

	go func() {
		atomic.StoreInt32(&p.inPlayingLock, 1)       // 协程开始时置 1
		defer atomic.StoreInt32(&p.inPlayingLock, 0) // 协程退出时置 0

		ticker := time.NewTicker(p.frameRate)
		defer ticker.Stop()

		for (direction > 0 && p.playing) || (direction < 0 && p.reversePlaying) {
			<-ticker.C
			if p.frameCount == 0 {
				continue
			}

			if (direction > 0 && !p.playing) || (direction < 0 && !p.reversePlaying) {
				break
			}

			index := (p.currentFrame + direction + p.frameCount) % p.frameCount
			p.showFrame(index)
			p.currentFrame = index
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
		p.waitForPlaybackExit()
		return
	}

	// 启动反向播放前，确保正向播放停止
	if p.playing {
		p.playing = false
		p.updatePlayPauseIcon()
		p.waitForPlaybackExit()
	}

	go p.playFrames(-1)
}

func (p *FramePlayer) waitForPlaybackExit() {
	for atomic.LoadInt32(&p.inPlayingLock) == 1 {
		time.Sleep(1 * time.Millisecond)
	}
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
			p.setWindowTitleWithDir(d)
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

type FrameInfo struct {
	Number     int    // 提取的帧号
	IsKey      bool   // 是否关键帧
	Annotation string // 关键帧注释（若存在）
	RawName    string // 原始文件名（可选）
}

func (p *FramePlayer) loadFrames(dir string) error {
	log.Printf("load dir:%s", dir)
	p.frameMapDirty = true
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

	sortFrames(frames)

	p.frameFiles = frames
	p.frameCount = len(frames)

	p.lastModTimes = make(map[string]time.Time, len(frames))
	for _, f := range frames {
		if st, err := os.Stat(f); err == nil {
			p.lastModTimes[f] = st.ModTime()
		}
	}

	// 切换目录时清空缓存
	p.clearCache()
	p.frameDir = dir

	return nil
}

func (p *FramePlayer) clearCache() {
	p.frameCache = make(map[int]fyne.CanvasObject)
	p.frameTexts = make(map[int]string)
	p.frameImages = make(map[int]image.Image)
}

// 从文件名末尾提取数字（遇到非数字停止），无数字则返回最大值
func extractFrameInfo(path string) FrameInfo {
	base := filepath.Base(path)
	name := strings.TrimSuffix(base, filepath.Ext(base))

	// 提取末尾数字
	numStr := ""
	i := len(name) - 1
	for ; i >= 0; i-- {
		ch := name[i]
		if ch >= '0' && ch <= '9' {
			numStr = string(ch) + numStr
		} else {
			break
		}
	}

	num := 1 << 30
	if numStr != "" {
		if parsed, err := strconv.Atoi(numStr); err == nil {
			num = parsed
		}
	}

	isKey := false
	annotation := ""

	// 判断关键帧
	if i >= 0 && (name[i] == 'K' || name[i] == 'k') {
		isKey = true

		// 如果 K 前面是 "__"，则提取注释
		if i >= 2 && name[i-2:i] == "__" {
			last := strings.LastIndex(name, "__")
			if last > 0 {
				before := strings.LastIndex(name[:last], "__")
				if before >= 0 {
					annotation = strings.TrimSpace(name[before+2 : last])
				}
			}
		}
	}

	return FrameInfo{
		Number:     num,
		IsKey:      isKey,
		Annotation: annotation,
		RawName:    base,
	}
}
func (p *FramePlayer) updateFrameInfoLabel(index int) {
	fyne.Do(func() {
		p.frameValueLabel.SetText(fmt.Sprintf("%d / %d", index+1, p.frameCount))
	})
}

func (p *FramePlayer) makeTextViewWithStyle(content string, col color.Color, alpha uint8) fyne.CanvasObject {
	lines := strings.Split(content, "\n")
	var items []fyne.CanvasObject
	y := float32(0)

	fontSize := float32(p.fontSize)
	lineHeight := float32(int(fontSize*1.4 + 0.5))

	for _, line := range lines {
		c := col
		// 如果传了 alpha，就用 NRGBA 包装一下
		if nrgba, ok := c.(color.NRGBA); ok {
			nrgba.A = alpha
			c = nrgba
		}
		t := canvas.NewText(line, c)
		t.TextStyle = fyne.TextStyle{Monospace: true}
		t.TextSize = fontSize
		t.Move(fyne.NewPos(0, y))
		items = append(items, t)
		y += lineHeight
	}

	return container.NewWithoutLayout(items...)
}

func renderSVGToObject(path string, want fyne.Size) (fyne.CanvasObject, image.Image) {
	_, err := os.Stat(path)
	if err != nil {
		return widget.NewLabel("SVG 文件不存在或无法访问：" + err.Error()), nil
	}

	svgBytes, err := os.ReadFile(path)
	if err != nil {
		return widget.NewLabel("SVG读取失败：" + err.Error()), nil
	}

	header := string(svgBytes)
	if !strings.Contains(header, "<svg") && !strings.Contains(header, "<?xml") {
		return widget.NewLabel("读取到的文件看起来不是 SVG"), nil
	}

	icon, err := oksvg.ReadIconStream(bytes.NewReader(svgBytes))
	if err != nil {
		return widget.NewLabel("解析SVG失败：" + err.Error()), nil
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

	img.Move(fyne.NewPos(0, 0)) // 显示在左上角
	img.Resize(img.MinSize())   // 确保尺寸被设置
	return container.NewWithoutLayout(img), rgba
}

func (p *FramePlayer) makeImageView(img image.Image, translucency float64) fyne.CanvasObject {
	if img == nil {
		return widget.NewLabel("图像为空")
	}
	pic := canvas.NewImageFromImage(img)
	pic.FillMode = canvas.ImageFillOriginal
	size := fyne.NewSize(float32(img.Bounds().Dx()), float32(img.Bounds().Dy()))
	pic.Resize(size)
	pic.Move(fyne.NewPos(0, 0))
	pic.Translucency = translucency
	return container.NewWithoutLayout(pic)
}

func (p *FramePlayer) checkDirChanged() bool {
	entries, err := os.ReadDir(p.frameDir)
	if err != nil {
		return false
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			ext := strings.ToLower(filepath.Ext(e.Name()))
			if ext == ".txt" || ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".svg" {
				files = append(files, filepath.Join(p.frameDir, e.Name()))
			}
		}
	}

	// 排序规则保持和 loadFrames 一致
	sortFrames(files)

	// 文件数量或名字不同
	if len(files) != len(p.frameFiles) {
		return true
	}
	for i := range files {
		if files[i] != p.frameFiles[i] {
			return true
		}
		st, err := os.Stat(files[i])
		if err != nil {
			return true
		}
		old := p.lastModTimes[files[i]]
		if !st.ModTime().Equal(old) {
			return true
		}
	}
	return false
}

// sortFrames 对帧文件列表进行统一排序：
// 1. 优先按文件名末尾的数字升序；
// 2. 如果数字相同，再按原始名字的字母序升序。
func sortFrames(files []string) {
	sort.Slice(files, func(i, j int) bool {
		fi := extractFrameInfo(files[i])
		fj := extractFrameInfo(files[j])
		if fi.Number != fj.Number {
			return fi.Number < fj.Number
		}
		return strings.ToLower(fi.RawName) < strings.ToLower(fj.RawName)
	})
}

// 中心点触发式缓存刷新 + 显示帧
func (p *FramePlayer) showFrame(index int) {
	// 边界保护
	if p.frameCount == 0 || index < 0 || index >= p.frameCount {
		return
	}
	p.lastShownFrame = index

	half := p.cacheWindowSize / 2
	// 如果当前帧不在缓存，则以当前帧为中心点，缓存前后 N/2 帧（已在缓存的跳过）
	if _, ok := p.frameCache[index]; !ok {
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
					// 保存原始文本到一个 map，方便残影模式调用
					p.frameTexts[j] = string(content)
					obj = p.makeTextViewWithStyle(string(content), theme.Color(theme.ColorNameForeground), uint8(255))
				}
			case ".svg":
				var img image.Image
				obj, img = renderSVGToObject(path, fyne.NewSize(0, 0))
				if img != nil {
					p.frameImages[j] = img
				}
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
						p.frameImages[j] = img
						obj = p.makeImageView(img, 0.0)
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
		co := p.frameCache[index]
		if co == nil {
			co = widget.NewLabel("加载失败")
		}

		// === 残影模式 ===
		if p.ghostMode && index > 0 {
			if prevObj, okPrev := p.frameCache[index-1]; okPrev && prevObj != nil {
				var ghost fyne.CanvasObject
				if prevText, okText := p.frameTexts[index-1]; okText {
					ghost = p.makeTextViewWithStyle(prevText, color.NRGBA{R: 210, G: 180, B: 140, A: 255}, 150)
				} else if prevImg, okImg := p.frameImages[index-1]; okImg {
					ghost = p.makeImageView(prevImg, 0.5)
				}

				if ghost != nil {
					ghostLayer := container.NewStack(ghost, co)
					p.ShowContent(ghostLayer)
				} else {
					p.ShowContent(co)
				}
			} else {
				p.ShowContent(co)
			}
		} else {
			p.ShowContent(co)
		}

		p.updateFrameInfoLabel(index)
	})

	// 清理远离当前帧的缓存帧
	for k := range p.frameCache {
		if circularDistance(k, index, p.frameCount) > half {
			delete(p.frameCache, k)
			delete(p.frameTexts, k)
			delete(p.frameImages, k)
		}
	}

	// 显示新帧的时候滚动条放最顶上。
	// p.scrollableContent.ScrollToTop()
	p.updateFrameMapButtons(index)
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
	if disp, ok := p.displayArea.(*FrameDisplay); ok {
		disp.SetContent(obj)
	}
}

func (p *FramePlayer) togglePlayPause() {
	if p.frameCount == 0 {
		p.updateFrameInfoLabel(0)
		return
	}
	if p.playing {
		p.playing = false
		p.updatePlayPauseIcon()
		p.waitForPlaybackExit()
		return
	}

	// 启动正向播放前，确保反向播放停止
	if p.reversePlaying {
		p.reversePlaying = false
		p.updateReversePlayIcon()

		p.waitForPlaybackExit()
	}

	go p.playFrames(1)
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
		"                Space - Play / Pause (forward)       s     - Stop playback",
		"                r     - Play / Pause (reverse)       o     - Open directory",
		"                → / l - Next frame                   e     - Open Recent Directory",  
		"                .     - Next Key frame               g     - ExportGIF",
		"                ← / h - Previous frame               F12   - Full Screen Switch",  
		"                ,     - Previous Key frame           F4    - Toggle button display",
		"    Mouse Wheel ↑ / ↓ - Previous / Next frame        F6    - Toggle Frame map only mode display", 
		"",
		"                    v - Toggle ghost mode(The afterimage of the previous frame is",
		"                                          superimposed on the current frame)",
		"                   F5 - Toggle refresh mode(Automatically update file changes",
		"                                            Can only be used to manually play frames)",
		"Frame file format:",
		"    Ordinary frame file: my_frame_1.txt (Just end with a number)",
		"    keyframe file: my_frame_K18.txt (The number at the end is preceded by the letter K)",
		"    Keyframe prompt file: my_frame__tips_messages__K19.txt (Keyframe prompt word wrapped in double underline)",
	}
	vbox := container.NewVBox()
	for _, line := range lines {
		t := canvas.NewText(line, theme.Color(theme.ColorNameForeground))
		t.TextStyle = fyne.TextStyle{Monospace: true}
		t.TextSize = 12
		vbox.Add(t)
	}
	return vbox
}
func (p *FramePlayer) SetupUI() fyne.CanvasObject {
	p.displayArea = NewFrameDisplay(p)

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
		tk := time.NewTicker(50 * time.Millisecond)
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
				p.setWindowTitleWithDir(d)
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
		p.stop()
	})

	prevBtn := widget.NewButtonWithIcon("", theme.MediaSkipPreviousIcon(), func() {
		p.prevFrame()
	})
	nextBtn := widget.NewButtonWithIcon("", theme.MediaSkipNextIcon(), func() {
		p.nextFrame()
	})
	settingsBtn := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		p.showSettingsDialog()
	})

	// 新增 About 按钮
	aboutBtn := widget.NewButtonWithIcon("", theme.InfoIcon(), func() {
		// 构建超链接
		repoURL, _ := url.Parse("https://github.com/qindapao/common_tool")

		content := container.NewVBox(
			widget.NewLabel(appTitle+" "+appVersion),
			widget.NewLabel("Author: Qin Qing"),
			widget.NewHyperlink("GitHub Repository", repoURL),
		)

		dialog.ShowCustom("About", "Close", content, p.window)
	})

	// 导出 GIF 按钮
	exportBtn := widget.NewButtonWithIcon("", theme.DocumentSaveIcon(), func() {
		p.showExportGIFDialog()
	})

	// 顶部控制条：仅保留一行按钮 + 速度控制
	topRow := container.NewHBox(
		frameInfo,
		p.playPauseBtn, p.reversePlayBtn, stopBtn, prevBtn, nextBtn,
		unifiedDirBtn, p.recentMenuBtn, settingsBtn, exportBtn, aboutBtn,
		p.layoutSpacer(),
		speedControl,
	)
	p.controlBar = topRow

	p.frameMapBox = container.NewHBox()
	p.frameMapScroll = container.NewHScroll(p.frameMapBox)
	frameMap := container.NewBorder(nil, nil,
		widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
			p.frameMapScroll.ScrollToOffset(fyne.NewPos(p.frameMapScroll.Offset.X-100, 0))
		}),
		widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() {
			p.frameMapScroll.ScrollToOffset(fyne.NewPos(p.frameMapScroll.Offset.X+100, 0))
		}),
		p.frameMapScroll,
	)
	timelineBar := container.NewVBox(timelineSlider, frameMap)

	p.timelineBar = timelineBar

	// 初始可见性：根据配置决定是否隐藏
	if !p.controlsVisible {
		p.controlBar.Hide()
	}

	// 布局：上控制条 + 下内容区 + 进度条
	p.scrollableContent = container.NewScroll(p.displayArea)
	return container.NewBorder(p.controlBar, timelineBar, nil, nil, p.scrollableContent)
}

// stop 播放并回到第一帧
func (p *FramePlayer) stop() {
	// 停止播放协程
	p.playing = false
	p.reversePlaying = false
	// 等待播放循环退出
	p.waitForPlaybackExit()
	// 更新按钮图标
	p.updatePlayPauseIcon()
	p.updateReversePlayIcon()

	if p.frameCount == 0 {
		return
	}

	// 回到第一帧
	p.showFrame(0)
	p.currentFrame = 0
}

// 跳到下一帧
func (p *FramePlayer) nextFrame() {
	if p.frameCount == 0 {
		return
	}
	index := (p.lastShownFrame + 1) % p.frameCount
	p.showFrame(index)
	p.currentFrame = p.lastShownFrame
}

// 跳到上一帧
func (p *FramePlayer) prevFrame() {
	if p.frameCount == 0 {
		return
	}
	index := (p.lastShownFrame - 1 + p.frameCount) % p.frameCount
	p.showFrame(index)
	p.currentFrame = p.lastShownFrame
}

// 通用：从 start 开始按 step（+1 或 -1）搜索下一个关键帧。
// 参数 start 表示从哪个索引开始搜索（传 lastShownFrame）。
func (p *FramePlayer) gotoKeyframeByStep(start, step int) {
	if p.frameCount == 0 {
		return
	}
	if step != 1 && step != -1 {
		return
	}

	n := p.frameCount
	first := (start + step + n) % n

	for offset := range p.frameFiles {
		// idx = (first + offset*step) 环形取模并保证非负
		idx := ((first+offset*step)%n + n) % n
		if extractFrameInfo(p.frameFiles[idx]).IsKey {
			p.showFrame(idx)
			p.lastShownFrame = idx
			p.currentFrame = idx
			return
		}
	}
}

// 向后（下一个）关键帧
func (p *FramePlayer) gotoNextKeyframe() {
	p.gotoKeyframeByStep(p.lastShownFrame, 1)
}

// 向前（上一个）关键帧
func (p *FramePlayer) gotoPrevKeyframe() {
	p.gotoKeyframeByStep(p.lastShownFrame, -1)
}

// 判断指定按钮是否在 HScroll 可见范围内（带边距）
func (p *FramePlayer) isButtonVisible(i int, margin float32) bool {
	if p.frameMapScroll == nil || i < 0 || i >= len(p.frameMapButtons) {
		return true // 视为可见，避免误滚动
	}
	btn := p.frameMapButtons[i]

	// 通过绝对坐标转换为内容坐标，更稳定
	btnAbs := fyne.CurrentApp().Driver().AbsolutePositionForObject(btn)
	contentAbs := fyne.CurrentApp().Driver().AbsolutePositionForObject(p.frameMapBox)
	btnX := btnAbs.X - contentAbs.X
	btnW := btn.Size().Width

	scrollX := p.frameMapScroll.Offset.X
	scrollW := p.frameMapScroll.Size().Width

	leftOk := btnX >= scrollX+margin
	rightOk := (btnX + btnW) <= (scrollX + scrollW - margin)
	return leftOk && rightOk
}

// 将滚动条滚到目标按钮附近（左侧预留 margin 像素）
func (p *FramePlayer) scrollToButton(i int, margin float32) {
	if p.frameMapScroll == nil || i < 0 || i >= len(p.frameMapButtons) {
		return
	}
	btn := p.frameMapButtons[i]

	btnAbs := fyne.CurrentApp().Driver().AbsolutePositionForObject(btn)
	contentAbs := fyne.CurrentApp().Driver().AbsolutePositionForObject(p.frameMapBox)
	btnX := btnAbs.X - contentAbs.X

	targetX := btnX - margin
	targetX = max(targetX, 0)
	fyne.Do(func() {
		p.frameMapScroll.ScrollToOffset(fyne.NewPos(targetX, 0))
	})
}

func (p *FramePlayer) updateFrameMapButtons(index int) {
	if p.frameMapDirty || len(p.frameMapButtons) != p.frameCount {
		fmt.Println("* 初始化 frameMapButtons: 构建帧按钮列表")
		p.frameMapBox.Objects = nil
		p.frameMapButtons = nil
		p.frameMapDirty = false

		for i, file := range p.frameFiles {
			// 这里必须要定义一个新变量，不然闭包捕获的是同一个i值
			// 通常是索引的最后一个
			frameIndex := i
			info := extractFrameInfo(file)
			label := strconv.Itoa(info.Number)

			btn := widget.NewButton(label, func() {
				p.showFrame(frameIndex)
				p.currentFrame = frameIndex
			})

			btn.Importance = widget.LowImportance
			btn.Refresh()

			// 保存按钮本体用于高亮控制
			p.frameMapButtons = append(p.frameMapButtons, btn)

			// 使用封装的样式函数
			styled := p.styledFrameButton(frameIndex, btn)
			p.frameMapBox.Add(styled)
		}

		p.frameMapBox.Refresh()
	}

	p.updateFrameHighlight(index)
	// 然后判断是否需要滚动到当前帧按钮
	const margin float32 = 20
	if !p.isButtonVisible(index, margin) {
		p.scrollToButton(index, margin)
	}
}

func (p *FramePlayer) updateFrameHighlight(index int) {
	// 清除上一帧高亮
	if p.currentFrame >= 0 && p.currentFrame < len(p.frameMapButtons) {
		btn := p.frameMapButtons[p.currentFrame]
		if btn.Importance != widget.LowImportance {
			btn.Importance = widget.LowImportance
			fyne.Do(func() {
				btn.Refresh()
			})
		}
	}

	// 设置当前帧高亮
	if index >= 0 && index < len(p.frameMapButtons) {
		btn := p.frameMapButtons[index]
		if btn.Importance != widget.HighImportance {
			btn.Importance = widget.HighImportance
			fyne.Do(func() {
				btn.Refresh()
			})
		}
	}
}

func (p *FramePlayer) styledFrameButton(i int, btn *widget.Button) fyne.CanvasObject {
	info := extractFrameInfo(p.frameFiles[i])

	// 把“数字 + 注释”合并为按钮文本
	label := strconv.Itoa(info.Number)
	ann := strings.TrimSpace(info.Annotation)
	if ann != "" {
		// 截断注释，避免按钮被撑太宽
		annShort := truncateRunes(ann, 14) // 长度可调 10~16
		label = label + " · " + annShort
	}

	btn.SetText(label)
	btn.Importance = widget.LowImportance
	btn.Refresh()

	// 关键帧：边框包住按钮（你的原逻辑）
	if info.IsKey {
		size := btn.MinSize()
		border := canvas.NewRectangle(color.Transparent)
		border.StrokeColor = soilGray
		border.StrokeWidth = 1.5
		border.FillColor = color.Transparent
		border.CornerRadius = 12
		border.Resize(fyne.NewSize(size.Width+6, size.Height+6))
		return container.NewStack(border, btn)
	}

	return btn
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
			// 这里必须要使用一个新的变量不能直接使用accum
			segPath := accum
			// 因为这里有闭包,所以必须用一个新的变量
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
		p.setWindowTitleWithDir(cur)
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
			p.nextFrame()
		case fyne.KeyLeft, fyne.KeyH:
			p.prevFrame()
		case fyne.KeyS:
			p.stop()
		case fyne.KeyF12:
			p.window.SetFullScreen(!p.window.FullScreen())
		case fyne.KeyF4:
			p.toggleHideAllControls()
		case fyne.KeyF6:
			p.toggleFrameMapOnlyMode()
		case fyne.KeyComma: // ','
			p.gotoPrevKeyframe()
		case fyne.KeyPeriod: // '.'
			p.gotoNextKeyframe()
		case fyne.KeyV:
			p.toggleGhostMode()
		case fyne.KeyF5:
			p.toggleRefreshMode()
		}
	})
}

func (p *FramePlayer) toggleGhostMode() {
	p.ghostMode = !p.ghostMode
	p.setWindowTitleWithDir(p.frameDir)
	p.showFrame(p.currentFrame)
}

func (p *FramePlayer) toggleRefreshMode() {
	p.refreshMode = !p.refreshMode
	p.setWindowTitleWithDir(p.frameDir)

	if p.refreshMode {
		if p.refreshTicker == nil {
			p.refreshTicker = time.NewTicker(100 * time.Millisecond)
			p.refreshQuit = make(chan struct{})
			go func() {
				for {
					select {
					case <-p.refreshTicker.C:
						if !p.playing && !p.reversePlaying {
							if p.checkDirChanged() {
								fyne.Do(func() {
									log.Printf("dir changed!")
									if err := p.loadFrames(p.frameDir); err == nil {
										if p.currentFrame >= p.frameCount {
											p.currentFrame = 0
										}
										p.showFrame(p.currentFrame)
									}
								})
							}
						}
					case <-p.refreshQuit:
						return
					}
				}
			}()
		}
	} else {
		if p.refreshTicker != nil {
			p.refreshTicker.Stop()
			close(p.refreshQuit)
			p.refreshTicker = nil
			p.refreshQuit = nil
		}
	}
}

func (p *FramePlayer) toggleFrameMapOnlyMode() {
	p.frameMapOnlyMode = !p.frameMapOnlyMode
	p.hideAllControls = false // F6 模式不属于“完全隐藏”

	if p.controlBar != nil {
		p.controlBar.Hide()
	}

	if p.timelineBar != nil {
		for i, obj := range p.timelineBar.Objects {
			if i == 1 && p.frameMapOnlyMode {
				obj.Show() // 显示帧地图
			} else {
				obj.Hide()
			}
		}
		p.timelineBar.Show()
		p.timelineBar.Refresh()
	}

	if !p.frameMapOnlyMode {
		// 恢复默认模式（显示所有控件）
		if p.controlBar != nil {
			p.controlBar.Show()
		}
		if p.timelineBar != nil {
			for _, obj := range p.timelineBar.Objects {
				obj.Show()
			}
			p.timelineBar.Show()
			p.timelineBar.Refresh()
		}
	}
}

func (p *FramePlayer) toggleHideAllControls() {
	p.hideAllControls = !p.hideAllControls
	p.frameMapOnlyMode = false // F4 模式不显示帧地图

	if p.controlBar != nil {
		if p.hideAllControls {
			p.controlBar.Hide()
		} else {
			p.controlBar.Show()
		}
	}

	if p.timelineBar != nil {
		if p.hideAllControls {
			p.timelineBar.Hide()
		} else {
			for _, obj := range p.timelineBar.Objects {
				obj.Show()
			}
			p.timelineBar.Show()
		}
		p.timelineBar.Refresh()
	}
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

	// 文本预处理：去 BOM、统一换行、展开 Tab
	if len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		b = b[3:]
	}
	s := string(b)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "\t", "    ")
	lines := strings.Split(s, "\n")

	// 计算最大行宽（逐字符测量）
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

	startX := 0
	startY := 0

	// 绘制每行逐字符
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

	// 左上角对齐绘制
	srcBounds := imgSrc.Bounds()
	targetRect := image.Rect(0, 0, srcBounds.Dx(), srcBounds.Dy())
	draw.Draw(dst, targetRect, imgSrc, srcBounds.Min, draw.Over)
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

	// 在单独 RGBA 渲染，再左上角绘制到 dst
	tmp := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(tmp, tmp.Bounds(), &image.Uniform{color.Transparent}, image.Point{}, draw.Src)
	icon.SetTarget(0, 0, float64(w), float64(h))
	scanner := rasterx.NewScannerGV(w, h, tmp, tmp.Bounds())
	r := rasterx.NewDasher(w, h, scanner)
	icon.Draw(r, 1.0)

	// 左上角绘制
	rect := image.Rectangle{Min: image.Pt(0, 0), Max: image.Pt(w, h)}
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
	w := a.NewWindow(appTitle + " " + appVersion)

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
			player.setWindowTitleWithDir(dir)
			player.currentFrame = 0
			player.showFrame(player.currentFrame)
			player.currentFrame = player.lastShownFrame
		}
	}

	player.BindKeyEvents(w.Canvas())
	player.BindRuneEvents(w.Canvas())

	w.ShowAndRun()
}



