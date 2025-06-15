package logutil

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
)

// :TODO: 增加日志堆栈功能打印

// 定义日志级别
const (
	DEBUG = iota // 0
	INFO         // 1
	WARN         // 2
	ERROR        // 3
)

// 定义日志级别映射字符串
var LOG_LEVELS = map[string]int{
	"DEBUG": DEBUG,
	"INFO":  INFO,
	"WARN":  WARN,
	"ERROR": ERROR,
}

var (
	logger       *log.Logger
	logFile      *os.File
	once         sync.Once
	currentLevel = INFO // 默认日志级别
)

// InitLogger 初始化日志，允许指定输出目标（stdout 或 文件）
func InitLogger(output string, level int) {
	once.Do(func() {
		var err error
		if output == "stdout" {
			logFile = os.Stdout
		} else {
			logFile, err = os.OpenFile(
				// 以追加模式打开日志文件，不会覆盖已有内容
				output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				log.Fatal("无法创建日志文件:", err)
			}
		}
		logger = log.New(logFile, "", log.LstdFlags)
		currentLevel = level // 设置日志级别
	})
}

// logMessage 记录日志，**仅输出符合当前级别的日志**
func logMessage(level int, msg string, args ...any) {
	if logger == nil {
		InitLogger("stdout", INFO) // 默认输出到控制台
	}
	if level >= currentLevel { // 值越小打印得越多
		_, file, line, _ := runtime.Caller(2) // 获取真正调用的文件+行号
		// :TODO: 这里看下效果，但是不一定好
		relPath, err := filepath.Rel(".", file)
		if err == nil {
			relPath = file
		}

		var formattedArgs []any
		for _, arg := range args {
			// 使用了反射效率低点，但是结构体更美观
			v := reflect.ValueOf(arg)
			// 先检查是否是指针，如果是，则解引用
			// :TODO: 本来是指针打印了值(这样好吗？)
			// :TODO: 如果发生了指针解引用,是否要在日志中说明下增加指针地址打印
			if v.Kind() == reflect.Ptr {
				v = v.Elem()
			}

			if v.Kind() == reflect.Struct {
				formattedArgs = append(
					// 格式化结构体
					formattedArgs, PrintStruct(arg, false))
			} else if v.Kind() == reflect.Slice || v.Kind() == reflect.Map {
				// 如果是集合类型，转换为 JSON
				jsonData, err := json.MarshalIndent(arg, "", "    ")
				if err != nil {
					formattedArgs = append(
						formattedArgs, fmt.Sprintf("无法格式化: %v", err))
				} else {
					formattedArgs = append(formattedArgs, string(jsonData))
				}
			} else {
				formattedArgs = append(formattedArgs, arg) // 直接使用原值
			}
		}

		formattedMsg := fmt.Sprintf(msg, formattedArgs...) // 重新格式化消息
		logger.Printf("[%s:%d] %s", relPath, line, formattedMsg)
	}
}

// 设置日志级别
func SetLogLevel(level int) {
	currentLevel = level
}

// Info 记录 INFO 日志
func Info(msg string, args ...any) {
	logMessage(INFO, "[INFO] "+msg, args...)
}

// Warn 记录 WARN 日志
func Warn(msg string, args ...any) {
	logMessage(WARN, "[WARN] "+msg, args...)
}

// Error 记录 ERROR 日志
// :TODO: 这里打印的是绝对路径，但是相对路径的处理相对复杂，暂时不实现
func Error(msg string, args ...any) {
	// 先打印完整的调用堆栈信息
	size := 1024 // 初始缓冲区大小
	for {
		// 在堆上分配内存
		buf := make([]byte, size)
		n := runtime.Stack(buf, false)

		if n < size { // 如果数据小于缓冲区，则不需要扩展
			// :TODO: 如果string(buf[:n])里面本身格式化字符，比如%s，
			// 那么是否会有问题呢？
			logMessage(
				ERROR, "[ERR] "+msg+"\n调用堆栈:\n"+string(buf[:n]), args...)
			return
		}

		// 扩展缓冲区大小，倍增策略
		size *= 2
	}
}

// Debug 记录 DEBUG 日志
func Debug(msg string, args ...any) {
	// 确保参数被展开在传入进去
	logMessage(DEBUG, "[DBG] "+msg, args...)
}

// 关闭日志文件（如果有的话
// :TODO: 是否需要显式调用?
func CloseLogger() error {
	if logFile != nil && logFile != os.Stdout {
		err := logFile.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

// 递归格式化结构体信息
// :TODO: 如果发生了指针解引用,是否要在日志中说明下增加指针地址打印
func formatStruct(s any, indent string) string {
	v := reflect.ValueOf(s)
	// 先检查是否是指针，如果是，则解引用
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()

	// 处理非结构体类型
	if v.Kind() != reflect.Struct {
		return fmt.Sprintf("%s非结构体类型: %#v\n", indent, v.Kind())
	}

	var builder strings.Builder
	// builder.WriteString(fmt.Sprintf("%s结构体详细信息:\n", indent))

	// 遍历结构体字段
	for i := 0; i < v.NumField(); i++ {
		// go 1.22 加入，可以 range 一个整数，但是效率没有普通for循环高
		// for i := range v.NumField() {
		field := t.Field(i)
		value := v.Field(i)

		// 追加字段信息
		if value.Kind() != reflect.Struct {
			// 如果不是嵌套结构体，就直接打印内容
			builder.WriteString(fmt.Sprintf("%s%s: %#v\n", indent, field.Name, value))
		} else {
			// 如果是嵌套结构体,先打印标头,再递归处理
			builder.WriteString(fmt.Sprintf("%s%s: %s\n", indent, field.Name, "v"))
			builder.WriteString(formatStruct(value.Interface(), indent+"    "))
		}
	}

	return builder.String()
}

// 打印结构体信息（支持控制是否输出到标准输出）
func PrintStruct(s any, printToStdout bool) string {
	result := formatStruct(s, "")

	if printToStdout {
		fmt.Print(result) // 直接打印到标准输出
	}

	return result // 返回格式化字符串
}