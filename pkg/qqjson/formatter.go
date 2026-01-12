package qqjson

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode/utf8"

	"common_tool/pkg/errorutil"
	"common_tool/pkg/sh"

	"github.com/tidwall/gjson"
	"github.com/tidwall/pretty"
)

const maxCharsPerLine = 80

type OutputFormatter interface {
	Format(res gjson.Result, jsonFormat JSONFormat, trieSep string) *errorutil.ExitErrorWithCode
}

type BashFormatter struct{}

func (f BashFormatter) Format(res gjson.Result, _ JSONFormat, _ string) *errorutil.ExitErrorWithCode {
	return outputBash(res)
}

type TextFormatter struct{}

func (f TextFormatter) Format(res gjson.Result, jsonFormat JSONFormat, trieSep string) *errorutil.ExitErrorWithCode {
	return outputText(res, jsonFormat, trieSep)
}

type TypeFormatter struct{}

func (f TypeFormatter) Format(res gjson.Result, jsonFormat JSONFormat, _ string) *errorutil.ExitErrorWithCode {
	return outputType(res)
}

var formatters = map[string]OutputFormatter{
	"sh":   BashFormatter{},
	"txt":  TextFormatter{},
	"type": TypeFormatter{},
}

func writeCustomJSON(buf *bytes.Buffer, v any, indent int) {
	indentStr := strings.Repeat(" ", indent)
	switch val := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			v2 := val[k]
			keyLines := strings.Split(k, "\n")
			for i, kl := range keyLines {
				if i < len(keyLines)-1 {
					fmt.Fprintf(buf, "%s%s\n", indentStr, kl)
				} else {
					if isLeaf(v2) {
						fmt.Fprintf(buf, "%s%s => ", indentStr, kl)
						writeLeaf(buf, v2, indent+4)
					} else {
						fmt.Fprintf(buf, "%s%s => \n", indentStr, kl)
						writeCustomJSON(buf, v2, indent+4)
					}
				}
			}
		}
	case []any:
		for i, item := range val {
			if isLeaf(item) {
				fmt.Fprintf(buf, "%s[%d] = ", indentStr, i)
				writeLeaf(buf, item, indent+4)
			} else {
				fmt.Fprintf(buf, "%s[%d] = \n", indentStr, i)
				writeCustomJSON(buf, item, indent+4)
			}
		}
	default:
		writeLeaf(buf, val, indent)
	}
}

func writeTrieJSON(buf *bytes.Buffer, v any, path string, trieSep string) {
	switch val := v.(type) {

	case map[string]any:
		if len(val) == 0 {
			writeTrieLeaf(buf, val, path, trieSep)
			return
		}
		for k, v2 := range val {
			writeTrieJSON(buf, v2, path+"{"+k+"}"+trieSep, trieSep)
		}
	case []any:
		if len(val) == 0 {
			writeTrieLeaf(buf, val, path, trieSep)
			return
		}

		for i, item := range val {
			writeTrieJSON(buf, item, fmt.Sprintf("%s[%v]%s", path, i, trieSep), trieSep)
		}
	default:
		writeTrieLeaf(buf, val, path, trieSep)
	}
}

func isLeaf(v any) bool {
	switch val := v.(type) {
	case map[string]any:
		return len(val) == 0
	case []any:
		return len(val) == 0
	default:
		return true
	}
}

func writeLeaf(buf *bytes.Buffer, v any, indent int) {
	indentStr := strings.Repeat(" ", indent)
	switch val := v.(type) {
	case string:
		lines := strings.Split(val, "\n")
		if len(lines) == 1 {
			fmt.Fprintf(buf, "s:%s\n", val)
		} else {
			fmt.Fprintf(buf, "s:%s\n", lines[0])
			for _, line := range lines[1:] {
				fmt.Fprintf(buf, "%s%s\n", indentStr, line)
			}
		}
	case float64:
		fmt.Fprintf(buf, "i:%v\n", val)
	case bool:
		if val {
			fmt.Fprintf(buf, "t:true\n")
		} else {
			fmt.Fprintf(buf, "f:false\n")
		}
	case nil:
		fmt.Fprintf(buf, "n:null\n")
	case []any:
		fmt.Fprintf(buf, "a:[]\n")
	case map[string]any:
		fmt.Fprintf(buf, "o:{}\n")
	default:
		fmt.Fprintf(buf, "unknown type\n")
	}
}

func writeTrieLeaf(buf *bytes.Buffer, v any, path string, trieSep string) {
	switch val := v.(type) {
	case string:
		fmt.Fprintf(buf, " [%s]=%s", sh.BashANSIQuote(path), sh.BashANSIQuote(val))
	case float64:
		fmt.Fprintf(buf, " [%s]=%s",
			sh.BashANSIQuote(path),
			sh.BashANSIQuote(fmt.Sprintf("%v", val)+trieSep),
		)
	case bool:
		if val {
			fmt.Fprintf(buf, " [%s]=%s", sh.BashANSIQuote(path), sh.BashANSIQuote("true"+trieSep))
		} else {
			fmt.Fprintf(buf, " [%s]=%s", sh.BashANSIQuote(path), sh.BashANSIQuote("false"+trieSep))
		}
	case nil:
		fmt.Fprintf(buf, " [%s]=%s", sh.BashANSIQuote(path), sh.BashANSIQuote("null"+trieSep))
	case []any:
		fmt.Fprintf(buf, " [%s]=%s", sh.BashANSIQuote(path), sh.BashANSIQuote("[]"+trieSep))
	case map[string]any:
		fmt.Fprintf(buf, " [%s]=%s", sh.BashANSIQuote(path), sh.BashANSIQuote("{}"+trieSep))
	}
}

// 命令的退出码输出对象类型
func outputType(res gjson.Result) *errorutil.ExitErrorWithCode {
	typeErr := &errorutil.ExitErrorWithCode{Code: errorutil.CodeSuccess, Message: "", Err: nil, CmdExitCode: errorutil.CodeSuccess}

	var typeMap = map[gjson.Type]struct {
		Code    int
		Message string
	}{
		gjson.String: {JSONTypeString, "string"},
		gjson.Number: {JSONTypeNumber, "number"},
		gjson.Null:   {JSONTypeNull, "null"},
		gjson.True:   {JSONTypeTrue, "true"},
		gjson.False:  {JSONTypeFalse, "false"},
	}

	if res.IsObject() {
		typeErr.CmdExitCode = JSONTypeObject
		typeErr.Message = "object"
	} else if res.IsArray() {
		typeErr.CmdExitCode = JSONTypeArray
		typeErr.Message = "array"
	} else if info, ok := typeMap[res.Type]; ok {
		typeErr.CmdExitCode = info.Code
		typeErr.Message = info.Message
	} else {
		typeErr.CmdExitCode = JSONTypeUnknown
		typeErr.Message = "unknown"
	}

	return typeErr
}

func prefixValue(v gjson.Result) string {
	var prefix string
	suffix := v.String()
	switch {
	case v.IsObject():
		prefix = "o:"
	case v.IsArray():
		prefix = "a:"
	default:
		switch v.Type {
		case gjson.String:
			prefix = "s:"
		case gjson.Number:
			prefix = "i:"
		case gjson.True:
			prefix = "t:"
		case gjson.False:
			prefix = "f:"
		case gjson.Null:
			prefix = "n:"
			suffix = v.Raw
		default:
			prefix = "u:"
		}
	}
	return prefix + suffix
}

// 使用 declare 确保是局部变量
// s: 字符串
// i: 数字
// f: bool false
// t: bool true
// o: 对象
// a: 数组
// n: null
// declare -A mymap=(
//     ["name"]="s:Alice"
//     ["age"]="i:30"
//     ["active"]="t:true"
//     ["active2"]="f:false"
//     ["profile"]="o:{\"email\":\"alice@example.com\"}"
//     ["tags"]="a:[\"dev\",\"ops\"]"
//     ["deleted"]="n:null"
// )

func outputBash(res gjson.Result) *errorutil.ExitErrorWithCode {

	// 创建一个和数组/对象大小一摸一样的切片避免扩容提升性能
	// 其它对象返回0,就是一个空切片
	parts := make([]string, 0, res.Get("#").Int())

	if res.IsArray() {
		res.ForEach(func(_, v gjson.Result) bool {
			parts = append(parts, sh.BashANSIQuote(prefixValue(v)))
			return true
		})
		fmt.Printf("%s", strings.Join(parts, " "))
	} else if res.IsObject() {
		res.ForEach(func(k, v gjson.Result) bool {
			parts = append(parts, fmt.Sprintf("[%s]=%s",
				sh.BashANSIQuote(k.String()),
				sh.BashANSIQuote(prefixValue(v)),
			))
			return true
		})
		fmt.Printf("%s", strings.Join(parts, " "))
	} else {
		// 这里不处理null,因为会和字符串的null冲突
		// 结尾增加一个补充字符是为了防止Bash中$()自动去掉结尾的换行符行为
		fmt.Printf("%sX", res.String())
	}

	return outputType(res)
}

func formatJSON(data []byte, format JSONFormat, trieSep string) []byte {
	switch format {
	case JSONFormatMul:
		return pretty.PrettyOptions(data, &pretty.Options{
			Indent:   "    ",
			SortKeys: true,
			Width:    0,
			Prefix:   "",
		})
	case JSONFormatOne:
		return pretty.Ugly(data)
	case JSONFormatRaw:
		return data
	case JSONFormatHuman:
		var raw any
		if err := json.Unmarshal(data, &raw); err != nil {
			return fmt.Appendf(nil, "error: %v", err)
		}
		var buf bytes.Buffer
		buf.WriteString("\n--- Human JSON ---\n")
		writeCustomJSON(&buf, raw, 4)

		rawJSON := pretty.Ugly(data)
		writeWrappedRawJSON(&buf, rawJSON)

		return buf.Bytes()
	case JSONFormatTrie:
		var raw any
		if err := json.Unmarshal(data, &raw); err != nil {
			return fmt.Appendf(nil, "error: %v", err)
		}
		var buf bytes.Buffer
		writeTrieJSON(&buf, raw, "", trieSep)

		return buf.Bytes()
	default:
		return data
	}
}

func writeWrappedRawJSON(buf *bytes.Buffer, raw []byte) {
	buf.WriteString("--- RAW JSON(convert multiple lines into a single line before parsing.) ---\n")

	line := bytes.Buffer{}
	charCount := 0
	for len(raw) > 0 {
		r, size := utf8.DecodeRune(raw)
		if r == utf8.RuneError && size == 1 {
			// Invalid UTF-8 byte, write as-is
			line.WriteByte(raw[0])
			raw = raw[1:]
		} else {
			line.Write(raw[:size])
			raw = raw[size:]
		}
		charCount++

		if charCount >= maxCharsPerLine {
			line.WriteString("\n")
			buf.Write(line.Bytes())
			line.Reset()
			charCount = 0
		}
	}
	if line.Len() > 0 {
		line.WriteString("\n")
		buf.Write(line.Bytes())
	}
}

func outputText(res gjson.Result, jsonFormat JSONFormat, trieSep string) *errorutil.ExitErrorWithCode {
	raw := []byte(res.Raw)
	errInit := &errorutil.ExitErrorWithCode{
		Code:        errorutil.CodeSuccess,
		Message:     "",
		Err:         nil,
		CmdExitCode: errorutil.CodeSuccess}

	_, err := os.Stdout.Write(formatJSON(raw, jsonFormat, trieSep))
	errInit.Err = err

	if err != nil {
		errInit.Code = errorutil.CodeIOError
		return errInit
	}
	return outputType(res)
	// }
}
