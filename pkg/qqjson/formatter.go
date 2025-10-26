package qqjson

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"common_tool/pkg/errorutil"
	"common_tool/pkg/sh"

	"github.com/tidwall/gjson"
)

type OutputFormatter interface {
	Format(res gjson.Result, varName string, jsonFormat JSONFormat) *errorutil.ExitErrorWithCode
	// 可选: 用于错误退出前执行的清理
	Cleanup(varName string)
}

type BashFormatter struct{}

func (f BashFormatter) Format(res gjson.Result, varName string, _ JSONFormat) *errorutil.ExitErrorWithCode {
	return outputBash(varName, res)
}

func (f BashFormatter) Cleanup(varName string) {
	if varName == "" {
		varName = "RESULT"
	}
	// 再次 unset 只是示意，具体逻辑你可根据需要调整
	fmt.Printf("unset -v %s\n", varName)
}

type TextFormatter struct{}

func (f TextFormatter) Format(res gjson.Result, _ string, jsonFormat JSONFormat) *errorutil.ExitErrorWithCode {
	return outputText(res, jsonFormat)
}

func (f TextFormatter) Cleanup(varName string) {
	// 空实现，不做任何事
}

type TypeFormatter struct{}

func (f TypeFormatter) Format(res gjson.Result, _ string, jsonFormat JSONFormat) *errorutil.ExitErrorWithCode {
	return outputType(res)
}

func (f TypeFormatter) Cleanup(varName string) {
	// 空实现，不做任何事
}

var formatters = map[string]OutputFormatter{
	"sh":   BashFormatter{},
	"txt":  TextFormatter{},
	"type": TypeFormatter{},
}

type JSONFormatter func(any) ([]byte, error)

var JsonFormatters = map[JSONFormat]JSONFormatter{
	JSONFormatMul: func(v any) ([]byte, error) { return json.MarshalIndent(v, "", "    ") },
	JSONFormatOne: func(v any) ([]byte, error) { return json.Marshal(v) },
}

// 命令的退出码输出对象类型
func outputType(res gjson.Result) *errorutil.ExitErrorWithCode {
	typeErr := &errorutil.ExitErrorWithCode{Code: errorutil.CodeSuccess, Message: "", Err: nil, CmdExitCode: errorutil.CodeSuccess}

	if res.IsObject() {
		typeErr.CmdExitCode = JSONTypeObject
		typeErr.Message = "object"
	} else if res.IsArray() {
		typeErr.CmdExitCode = JSONTypeArray
		typeErr.Message = "array"
	} else if res.Type == gjson.String {
		typeErr.CmdExitCode = JSONTypeString
		typeErr.Message = "string"
	} else if res.Type == gjson.Number {
		typeErr.CmdExitCode = JSONTypeNumber
		typeErr.Message = "number"
	} else if res.Type == gjson.Null {
		typeErr.CmdExitCode = JSONTypeNull
		typeErr.Message = "null"
	} else if res.Type == gjson.True {
		typeErr.CmdExitCode = JSONTypeTrue
		typeErr.Message = "true"
	} else if res.Type == gjson.False {
		typeErr.CmdExitCode = JSONTypeFalse
		typeErr.Message = "false"
	} else {
		typeErr.CmdExitCode = JSONTypeUnknown
		typeErr.Message = "unknown"
	}

	return typeErr
}

// 使用 declare 确保是局部变量
func outputBash(name string, res gjson.Result) *errorutil.ExitErrorWithCode {

	err := &errorutil.ExitErrorWithCode{Code: errorutil.CodeSuccess, Message: "", Err: nil, CmdExitCode: errorutil.CodeSuccess}

	if name == "" {
		name = "RESULT"
	}

	if res.IsArray() {
		var parts []string
		res.ForEach(func(_, v gjson.Result) bool {
			parts = append(parts, sh.BashANSIQuote(v.String()))
			return true
		})
		fmt.Printf("unset -v %s ; declare -a %s=(%s)\n", name, name, strings.Join(parts, " "))
		return err
	}

	if res.IsObject() {
		fmt.Printf("unset -v %s ; declare -A %s=(\n", name, name)
		res.ForEach(func(k, v gjson.Result) bool {
			key := sh.BashANSIQuote(k.String())
			val := sh.BashANSIQuote(v.String())
			fmt.Printf("    [%s]=%s\n", key, val)
			return true
		})
		fmt.Println(")")
		return err
	}

	// 原始值(确保是局部变量)
	fmt.Printf("unset -v %s ; declare %s=%s\n", name, name, sh.BashANSIQuote(res.String()))
	return err
}

func outputText(res gjson.Result, jsonFormat JSONFormat) *errorutil.ExitErrorWithCode {
	str := res.Raw // 更安全的方式获取原始 JSON 字符串

	errInit := &errorutil.ExitErrorWithCode{Code: errorutil.CodeSuccess, Message: "", Err: nil, CmdExitCode: errorutil.CodeSuccess}

	var pretty any
	if err := json.Unmarshal([]byte(str), &pretty); err != nil {
		// 如果不是有效 JSON，就直接打印原始字符串
		fmt.Println(res.String())
		errInit.Message = "Not a valid JSON string"
		errInit.Err = err
		errInit.CmdExitCode = JSONErrNotValidJsonStr
		return errInit
	}

	f, ok := JsonFormatters[jsonFormat]
	if !ok {
		errInit.Message = "format fail"
		errInit.CmdExitCode = JSONErrFormatFail
		return errInit
	}
	formatted, err := f(pretty)
	if err != nil {
		// 出错时也回退打印原始值
		fmt.Println(res.String())
		errInit.Message = "json pretty fail"
		errInit.CmdExitCode = JSONPrettyFail
		return errInit
	}

	os.Stdout.Write(formatted)
	return errInit
	// }
}
