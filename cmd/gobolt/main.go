package main

import (
	"fmt"
	"os"

	"common_tool/pkg/errorutil"
	"common_tool/pkg/hw/pcie"
	"common_tool/pkg/logutil"
	"common_tool/pkg/qqjson"
	"common_tool/pkg/sshclient"

	"github.com/spf13/cobra"
)

const TOOL_VERSION = "1.0.2+20260112"

func main() {
	var rootCmd = &cobra.Command{
		Use:   "gobolt",
		Short: fmt.Sprintf("Gobolt v%s 是一个多功能 CLI 工具，支持 json/ssh/pcie 等子命令", TOOL_VERSION),
		Long: "       .-.                       _             _  _   \n" +
			"      (o o)         __ _   ___  | |__    ___  | || |_ \n" +
			"      | O \\        / _` | / _ \\ | '_ \\  / _ \\ | || __|\n" +
			"      \\    \\      | (_| || (_) || |_) || (_) || || |_ \n" +
			"       `~~~'       \\__, | \\___/ |_.__/  \\___/ |_| \\__|\n" +
			"                   |___/                             \n" +
			fmt.Sprintf("\nGobolt v%s 是一个多功能 CLI 工具，支持 json/ipmi/lspci/setpci 等子命令\n", TOOL_VERSION),
	}

	rootCmd.AddCommand(qqjson.JsonCmd())
	rootCmd.AddCommand(sshclient.SSHCmd())
	rootCmd.AddCommand(pcie.PCIECmd())

	var logFile string
	logLevel := logutil.WARN

	// 定义全局flag(屁股后面带P的函数才支持短选项)
	rootCmd.PersistentFlags().VarP(&logLevel, "log-level", "e", "日志等级(DEBUG/INFO/WARN/ERROR)")
	rootCmd.PersistentFlags().StringVarP(&logFile, "log-file", "l", "gobolt.log", "日志文件名(默认gobolt.log，stdout 表示标准输出)")
	// 阻止 Cobra 在命令参数错误时输出帮助
	rootCmd.SilenceUsage = true
	// 阻止Cobra自动打印RunEs返回的错误内容
	rootCmd.SilenceErrors = true

	// 等待Cobra的flag解析完成后
	// PersistentPreRunE 回调，这个钩子会在用户的命令解析完成、flag 值填充后执行
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		logutil.InitLogger(logFile, logLevel)
		return nil
	}

	exitCode := errorutil.CodeSuccess
	if err := rootCmd.Execute(); err != nil {
		errJsonStr, code, oriCode := errorutil.FormatErrorAndCode(err)
		exitCode = code
		if oriCode != errorutil.CodeSuccess {
			logutil.Error(errJsonStr)
		}
	}
	logutil.CloseLogger()
	os.Exit(exitCode)
}
