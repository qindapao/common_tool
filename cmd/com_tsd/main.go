package main

// _ 是确保需要的包被导入，执行里面的init注册函数
import (
	"fmt"
	"os"

	"common_tool/pkg/initutil"
	"common_tool/pkg/logutil"
	"common_tool/pkg/toolutil"

	_ "common_tool/pkg/tsdinfoget"
	"common_tool/pkg/webbase"
)

const TOOL_VERSION = "1.0.0+20250604"

// 打印版本号
func printVersion() {
	fmt.Println("工具版本: ", TOOL_VERSION)
}

// :TODO: 可以打印一个漂亮的字符画
// :TODO: 还有举例的内容
func printHelpInfo() {
	// 先打印工具帮助的固定内容
	helpText := fmt.Sprintf(`
工具作用: com_tsd 工具用于获取tsd系统上的数据
工具版本: %s
依赖文件: 
    平台目录下的 ${root}/TestPlat/common/WebSerList.ini
    平台目录下的 ${root}/TestPlat/common/netcapacity.ini

    注：${root}为平台的根目录
参数:
    -l
    	指定日志文件的路径，如果是 stdout 表示日志打印到标准输出
    -d
    	设置日志级别，如果没有设置，默认级别为WARN
    	可以设置的值
    		DEBUG INFO WARN ERROR
    -v
        打印版本信息后退出
    -h
        打印本帮助信息后退出
    -a
        后面跟具体的动作，比如：GetSoftInfoExt，此选项必须包含
        每个动作自己的参数独立，可查看每个动作的说明
    -o
        输出的结果json文件的输出路径，如果只有一个文件名，那么保存到当前目录
        下
每个动作单独的参数和返回值说明:`, TOOL_VERSION)
	fmt.Println(helpText)
	for _, desc := range webbase.HelpRegistry {
		fmt.Printf("    %s: %s\n", desc[0], desc[1])
	}
}

// 解析命令行参数
// :TODO: 子模块中的参数如何校验？
func parseArgs() (map[string]string, error) {
	// 裸选项不需要内容的
	nullOptions := map[string]string{
		"-v": "", "-h": "",
	}

	// 存储解析的参数
	argsMap := make(map[string]string)

	// 解析 `os.Args`
	// :TODO: 这里的逻辑是乱的，后面要改正
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]

		// 检查是否是有效选项
		if toolutil.HasAnyKey(nullOptions, arg) {
			argsMap[arg] = ""
		} else {
			if i+1 < len(os.Args) {
				argsMap[arg] = os.Args[i+1]
				i++
			} else {
				return nil, fmt.Errorf("错误: 选项 %s 需要一个值", arg)
			}
		}
	}

	return argsMap, nil
}

func main() {
	// fmt.Println("main exec")
	defer func() {
		if err := logutil.CloseLogger(); err != nil {
			fmt.Printf("Logger close error: %v\n", err)
		}
	}()

	// 调用参数解析函数
	argsMap, err := parseArgs()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// 处理特殊参数
	if toolutil.HasAnyKey(argsMap, "-v") {
		printVersion()
		os.Exit(0)
	}

	if toolutil.HasAnyKey(argsMap, "-h") {
		printHelpInfo()
		os.Exit(0)
	}

	// 运行时初始化系统，指定日志文件名
	loglevel := logutil.WARN
	if val, ok := argsMap["-d"]; ok {
		if level, err := logutil.ParseLogLevel(val); err == nil {
			loglevel = level
		} else {
			fmt.Fprintf(os.Stderr, "警告: %v，使用默认等级 WARN\n", err)
		}
	}

	// 日志文件的名字支持传进来
	logFileName := "com_tsd.log"
	if toolutil.HasAnyKey(argsMap, "-l") {
		logFileName = argsMap["-l"]
	}

	initutil.InitSystem(logFileName, loglevel)
	logutil.Debug("this is %v", argsMap)

	// 获取数据并解析输出 -a 后面的参数名就是解析器名字
	var parserName string
	if toolutil.HasAnyKey(argsMap, "-a") {
		parserName = argsMap["-a"]
	} else {
		printHelpInfo()
		os.Exit(0)
	}

	parser, exists := webbase.GetParser(parserName)
	logutil.Debug(
		"parserName: %v parser: %v exists: %v", parserName, parser, exists)
	if !exists {
		logutil.Error("未找到解析器: %s", parserName)
		os.Exit(1)
	}

	// 初始化解析器
	// :TODO: 错误退出码需要统一处理，不能错误只退出1
	err = parser.InitSelf(argsMap)
	if err != nil {
		logutil.Error("初始化失败: %s", err)
		os.Exit(1)
	}

	// 使用解析器处理逻辑
	err = parser.ProcessXML()
	if err != nil {
		logutil.Error("解析失败: %s", err)
		os.Exit(1)
	}

	// 保存 JSON
	err = parser.SaveJSON(parser)
	if err != nil {
		logutil.Error("保存 JSON 失败:: %s", err)
		os.Exit(1)
	}

	os.Exit(0)
}