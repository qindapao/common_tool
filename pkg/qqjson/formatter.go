package qqjson

import (
	"encoding/json"
	"fmt"
	"strings"

	"common_tool/pkg/sh"

	"github.com/tidwall/gjson"
)

type OutputFormatter interface {
	Format(res gjson.Result, varName string)
	// 可选: 用于错误退出前执行的清理
	Cleanup(varName string)
}

type BashFormatter struct{}

func (f BashFormatter) Format(res gjson.Result, varName string) {
	outputBash(varName, res)
}

func (f BashFormatter) Cleanup(varName string) {
	if varName == "" {
		varName = "RESULT"
	}
	// 再次 unset 只是示意，具体逻辑你可根据需要调整
	fmt.Printf("unset -v %s\n", varName)
}

type TextFormatter struct{}

func (f TextFormatter) Format(res gjson.Result, _ string) {
	outputText(res)
}

func (f TextFormatter) Cleanup(varName string) {
	// 空实现，不做任何事
}

var formatters = map[string]OutputFormatter{
	"sh":  BashFormatter{},
	"txt": TextFormatter{},
}

// 使用 declare 确保是局部变量
func outputBash(name string, res gjson.Result) {
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
		return
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
		return
	}

	// 原始值(确保是局部变量)
	fmt.Printf("unset -v %s ; declare %s=%s\n", name, name, sh.BashANSIQuote(res.String()))
}

func outputText(res gjson.Result) {
	// if res.IsArray() {
	// 	res.ForEach(func(_, v gjson.Result) bool {
	// 		fmt.Println(v.String())
	// 		return true
	// 	})
	// } else {
	str := res.Raw // 更安全的方式获取原始 JSON 字符串

	var pretty any
	if err := json.Unmarshal([]byte(str), &pretty); err != nil {
		// 如果不是有效 JSON，就直接打印原始字符串
		fmt.Println(res.String())
		return
	}

	formatted, err := json.MarshalIndent(pretty, "", "    ")
	if err != nil {
		// 出错时也回退打印原始值
		fmt.Println(res.String())
		return
	}

	fmt.Println(string(formatted))
	// }
}