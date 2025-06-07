package initutil

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"web_tool/pkg/logutil"
	"web_tool/pkg/toolutil"
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
		TsdWebkeys: []string{"TSDCLOUD_WEBSER1", "TSDCLOUD_WEBSER2"},
	}
}

var (
	globalConfig Config
	once         sync.Once
)

// 寻找根目录
func findRootDir() string {
	// 获取执行路径
	execPath, err := os.Executable()
	if err != nil {
		logutil.Error("无法获取执行路径: %s", err)
		return ""
	}

	// 统一转换路径格式（适用于 Windows 和 Unix）
	execPath = filepath.FromSlash(execPath)
	execDir := filepath.Dir(execPath)

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
func InitSystem(logFileName string, logLevel int) {
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

// parseWebConfig 解析 Web 配置文件
func parseWebConfig(filePath string) {
	lines, err := toolutil.ReadFileToLines(filePath)
	if err != nil {
		logutil.Error("无法打开 Web 配置文件 %s", filePath)
		return
	}

	// :TODO: 如果后面要解析的多可以提取出来
	for _, key := range globalConfig.WebConfig.MesWebKeys {
		// Grep返回的内容追加到mesWeb,如果为空那么什么都不发生
		globalConfig.WebConfig.MesWeb = append(
			globalConfig.WebConfig.MesWeb, toolutil.MapStrings(
				toolutil.Grep(lines, key, false, false), func(s string) string {
					return toolutil.NewStringProcessor(s).Split(
						key+"=", 1).Split("=", 0).Res()
				})...)
	}

	for _, key := range globalConfig.WebConfig.TsdWebkeys {
		// Grep返回的内容追加到TsdWeb,如果为空那么什么都不发生
		globalConfig.WebConfig.TsdWeb = append(
			globalConfig.WebConfig.TsdWeb, toolutil.MapStrings(
				toolutil.Grep(lines, key, false, false), func(s string) string {
					return toolutil.NewStringProcessor(s).Split(
						key+"=", 1).Split("=", 0).Res()
				})...)
	}
}

func parseDelayConfig(filePath string) (int64, int64) {
	loop := "480"
	loopInterval := "60"
	lines, err := toolutil.ReadFileToLines(filePath)
	if err != nil {
		logutil.Error("无法打开 延时 配置文件 %s", filePath)
		// :TODO: 这里的魔鬼数字要处理下
		return 480, 480 * 60
	}
	var loopList, loopInterList []string
	loopList = toolutil.Grep(lines, "loop=", false, false)
	loopInterList = toolutil.Grep(lines, "loopinterval=", false, false)

	if len(loopList) > 0 {
		re := regexp.MustCompile(`loop=([^;]+)`)
		matchLoop := re.FindStringSubmatch(loopList[0])
		if len(matchLoop) > 1 {
			loop = matchLoop[1]
		}
	}

	if len(loopInterList) > 0 {
		re := regexp.MustCompile(`loopinterval=([^;]+)`)
		matchLoop := re.FindStringSubmatch(loopInterList[0])
		if len(matchLoop) > 1 {
			loopInterval = matchLoop[1]
		}
	}

	loopInt, _ := strconv.Atoi(loop)
	loopIntervalInt, _ := strconv.Atoi(loopInterval)

	return int64(loopIntervalInt), int64(loopIntervalInt * loopInt)
}

// GetConfig 获取全局配置
func GetConfig() Config {
	return globalConfig
}
