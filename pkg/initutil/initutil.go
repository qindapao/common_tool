package initutil

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"common_tool/pkg/logutil"
	"common_tool/pkg/toolutil/textutil"
)

// :TODO: 可能还有别的，结构体和构造函数都需要扩展
type WebConfig struct {
	WebFilePath   string   // Web配置文件路径
	DelayFilePath string   // 延时配置文件路径
	MesWebKeys    []string // MesWeb解析的关键字
	MesWeb        []string // MES Web 地址列表
	TsdWebkeys    []string // TSDWeb解析的关键字
	TsdWeb        []string // TSD Web 地址列表
	// 总延时时间(单位秒)
	AllDelayTime int64
	// 每次的延时间隔(单位秒)
	EachDelayTime int64
}

// :TODO: 其它的内容待添加
type Config struct {
	RootDir string
	PlatDir string
	// web地址相关
	WebConfig WebConfig
}

func NewWebConfig() WebConfig {
	return WebConfig{
		MesWebKeys: []string{"MES_WEBSERVER1", "MES_WEBSERVER2"},
		TsdWebkeys: []string{"TSD_BARCODE_WEBSER1", "TSD_BARCODE_WEBSER2"},
	}
}

var (
	globalConfig Config
	once         sync.Once
)

// 寻找根目录
func findRootDir() string {
	// 获取执行路径()
	// execPath, err := os.Executable()
	// 这里应该获取当前工作路径
	execPath, err := os.Getwd()
	if err != nil {
		logutil.Error("获取当前工作目录失败: %w", err)
		return ""
	}

	// 统一转换路径格式（适用于 Windows 和 Unix）
	execPath = filepath.FromSlash(execPath)
	execDir := filepath.Dir(execPath)
	execDir = filepath.Clean(execDir)

	// :TODO: 待验证
	if execDir == "/root" || execDir == "/tmp" || execDir == "/data/dft" {
		logutil.Debug("basePath: %s", execDir)
		return execDir
	}

	logutil.Debug("execPath: %s execDir: %s", execPath, execDir)

	// 查找 "TestPlat" 之前的目录
	parts := strings.Split(execDir, string(filepath.Separator))

	logutil.Debug("parts: %v", parts)

	var basePath string
	for i, part := range parts {
		logutil.Debug("i:%d part: %s", i, part)
		if part == "TestPlat" && i > 0 {
			// 自动适应系统分隔符
			basePath = strings.Join(parts[:i], string(os.PathSeparator))
			break
		}
	}

	logutil.Debug("basePath: %s", basePath)

	if basePath == "" {
		logutil.Error("未找到 根 目录")
		return ""
	}

	return basePath
}

// InitSystem 初始化系统（跨平台路径处理）
func InitSystem(logFileName string, logLevel logutil.LogLevel) {
	once.Do(func() {
		// 初始化日志
		logutil.InitLogger(logFileName, logLevel)

		basePath := findRootDir()

		if basePath == "" {
			os.Exit(1) // 直接退出程序
		}

		// 赋值到全局结构体
		globalConfig = Config{
			RootDir:   basePath,
			PlatDir:   filepath.Join(basePath, "TestPlat"),
			WebConfig: NewWebConfig(),
		}
		globalConfig.WebConfig.WebFilePath = filepath.Join(
			basePath, "TestPlat", "common", "WebSerList.ini")
		globalConfig.WebConfig.DelayFilePath = filepath.Join(
			basePath, "TestPlat", "common", "netcapacity.ini")

		// 截取MesWeb TsdWeb 字段
		parseWebConfig(globalConfig.WebConfig.WebFilePath)

		// 获取延时参数
		globalConfig.WebConfig.EachDelayTime,
			globalConfig.WebConfig.AllDelayTime = parseDelayConfig(
			globalConfig.WebConfig.DelayFilePath)

		// 漂亮打印完整的结构体
		logutil.Info("globalConfig struct:\n%v", globalConfig)
	})
}

func extractWebValues(content string, keys []string) []string {
	var out []string
	for _, key := range keys {
		val := textutil.NewBuilderByLines(content).
			Filter(func(s string) bool {
				return strings.HasPrefix(s, key+"=")
			}).
			SplitSep(key + "=").Index(1).
			SplitSep("=").Index(0).
			First()
		out = append(out, val)
	}
	return out
}

func extractIntConfig(content, key string, defaultVal int) int {
	val := textutil.NewBuilder(content).
		RegexGroup(fmt.Sprintf(`(?m)^%s=([^;\r\n]+)`, regexp.QuoteMeta(key)), 1).
		First()

	if v, err := strconv.Atoi(val); err == nil {
		return v
	}
	return defaultVal
}

// parseWebConfig 解析 Web 配置文件
func parseWebConfig(filePath string) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		logutil.Error("无法打开 Web 配置文件 %s", filePath)
		return
	}

	globalConfig.WebConfig.MesWeb = extractWebValues(string(content), globalConfig.WebConfig.MesWebKeys)
	globalConfig.WebConfig.TsdWeb = extractWebValues(string(content), globalConfig.WebConfig.TsdWebkeys)
}

func parseDelayConfig(filePath string) (int64, int64) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		logutil.Error("无法打开 延时 配置文件 %s", filePath)
		return 480, 480 * 60
	}

	loop := extractIntConfig(string(content), "loop", 480)
	interval := extractIntConfig(string(content), "loopinterval", 60)

	return int64(interval), int64(interval * loop)
}

// GetConfig 获取全局配置
func GetConfig() Config {
	return globalConfig
}