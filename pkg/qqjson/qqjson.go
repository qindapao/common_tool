package qqjson

import (
	"common_tool/pkg/errorutil"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// :TODO: 需要增加输出到一行 JSON 的功能

var (
	GjsonVersion  string
	SjsonVersion  string
	PrettyVersion string
	QqjsonVersion string = "v1.0.0"
)

// 数组和字典指定写入可以不在这个工具中做
// 在bash中自己处理
type CLIOptions struct {
	// 从文件或者标准输入中来
	Kind string
	// 直接从参数中获取数据
	InArg      string
	Format     string
	Path       string
	UseArgPath bool
	ArgPath    []string
	StrInput   string
	JSONInput  string
	FileInput  string
	Input      any
	Mode       string
	JSONFormat JSONFormat
}

type JSONFormat string

const (
	JSONFormatOne   JSONFormat = "one"
	JSONFormatMul   JSONFormat = "mul"
	JSONFormatRaw   JSONFormat = "raw"
	JSONFormatHuman JSONFormat = "human"

	JSONTypeNull    = 1
	JSONTypeTrue    = 2
	JSONTypeFalse   = 3
	JSONTypeNumber  = 4
	JSONTypeString  = 5
	JSONTypeArray   = 6
	JSONTypeObject  = 7
	JSONTypeUnknown = 8
)

// 为了让 VarP 接收自定义类型，实现 flag.Value 接口(String Set Type)即可：
func (f *JSONFormat) String() string { return string(*f) }

func (f *JSONFormat) Set(val string) error {
	switch val {
	case string(JSONFormatMul),
		string(JSONFormatOne),
		string(JSONFormatRaw),
		string(JSONFormatHuman):
		*f = JSONFormat(val)
		return nil
	default:
		return fmt.Errorf("无效的 jsonformat 值: %s", val)
	}
}

func (f *JSONFormat) Type() string {
	return "jsonformat" // 这个字符串用于帮助文档与类型提示
}

// 列出所有的合法值
func (JSONFormat) Values() []string {
	return []string{
		string(JSONFormatMul),
		string(JSONFormatOne),
		string(JSONFormatRaw),
		string(JSONFormatHuman),
	}
}

func qJsonEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, ".", `\.`)
	s = strings.ReplaceAll(s, "[", `\[`)
	s = strings.ReplaceAll(s, "]", `\]`)
	s = strings.ReplaceAll(s, "*", `\*`)
	s = strings.ReplaceAll(s, "?", `\?`)
	s = strings.ReplaceAll(s, "|", `\|`)
	s = strings.ReplaceAll(s, "@", `\@`)
	s = strings.ReplaceAll(s, "!", `\!`)
	s = strings.ReplaceAll(s, "#", `\#`)
	s = strings.ReplaceAll(s, "{", `\{`)
	s = strings.ReplaceAll(s, "}", `\}`)
	return s
}

// 嵌套创建的时候如果遇到数字键要强制设置为字典
// xx.:1.:2用冒号指定
func qJsonEscapeAndJoin(paths []string) string {
	escaped := make([]string, len(paths))
	for i, p := range paths {
		escaped[i] = qJsonEscape(p)
	}
	return strings.Join(escaped, ".")
}

// JSON 子命令封装
func JsonCmd() *cobra.Command {
	opts := &CLIOptions{}

	cmd := &cobra.Command{
		Use:   "json",
		Short: "处理 JSON 的读取、写入、删除、属性读取",
		Long: `处理 JSON 的读取、写入、删除、属性读取
Examples:
1. 属性读取

以下面的 JSON 文件为例进行说明
{
    "key1": {
        "key2": {
            "key3": [
                null,
                null,
                {
                    "other1": "xx",
                    "other2": 2,
					"True": true,
					"False": false,
					"Num": 145.3,
					"str": "xxx",
					"null": null
                }
            ]
        }
    }
}
gobolt json -m r -t type -k file -i demo.json -p key1.key2.1
gobolt json -m r -t type -k str -i "$json_str" -p key1.key2.1

命令的退出码说明:
	JSONTypeNull    = 1
	JSONTypeTrue    = 2
	JSONTypeFalse   = 3
	JSONTypeNumber  = 4
	JSONTypeString  = 5
	JSONTypeArray   = 6
	JSONTypeObject  = 7
	JSONTypeUnknown = 8

gobolt json -m t

这也会打印出工具中的每种类型的JSON对象的枚举值。

2. 读取内容

读取命令除了读取出内容外，命令的返回码也体现出读取到的数据类型，类型码和上面列出
的完全相同。

以下面的 JSON 文件为例进行说明
{
    "key1": {
        "key2": [
            null,
            null,
            null,
            {
                "key4": "value1",
                "specialkey.[]": "value1"
            }
        ]
    }
}
(1). 读取到txt格式
gobolt json -m r -t txt -k file -i demo.json -p key1.key2
效果是直接打印出 key2 数组的 JSON 字符串

(2). 读取到bash数据结构

<a>. 读取到普通的字符串
RESULT=$(gobolt json -m r -t sh -k file -i demo.json -p key1.key2.3.key4) ; RESULT=${RESULT%?}

这里 RESULT=${RESULT%?}的目的是删除字符串最后一个字符，因为$()这种方式默认会去掉字符
串末尾所有的空格，为了保留原始数据，输出的数据中末尾会增加一个非空字符，这不是本来就
该有的，所以我们要手动删除它。

<b>. 读取到数组变量

declare -a myarr=()
eval -- myarr=($(gobolt json -m r -t sh -k file -i demo.json -p key1.key2))

数组的括号中不能用""把$()引用起来，那样所有的输出会被当成一个数组元素。如果数组元素的
某个值本身也是一个JSON对象那么可以获取这个值继续进行解码。

<c>. 读取到关联数组变量

declare -A mymap=()
eval -- mymap=($(gobolt json -m r -t sh -k file -i demo.json -p key1.key2.3))

同样，关联数组中的括号中不能用""把$()引用起来。如果关联数组元素的某个值本身也是
一个JOSN对象，那么可以获取这个值继续进行解码。

(3). 不美化输出的读取(默认以ugly的方式输出,节省空间)

只需要在常规的读取命令里面加上: -F raw 选项即可。其它和上面的读取命令完全相同。

3. 写入

(1). sjson路径写入

gobolt json -m w -k file -i demo.json -p key1.key2.key3 -s "value1"

如果 -p 后面的参数除路径分隔符中包含 . [ ] 三种符号，需要加反斜杠转义
gobolt json -m w -k file -i demo.json -p ke\\.y1.ke\\[y2.k\\]ey3 -s "value1"

(2). 包装后的路径写入

解决上面的 . [ ] 3 种特殊符号的问题
gobolt json -m w -k file -i demo.json -s "value1" -P -- "ke.y1" "ke[y2" "k]ey3" "[]"

(3). 路径默认创建

默认情况下如果 JSON 中没有包含路径，那么会自动创建，创建的自动规则是
	1). 如果是纯数字的键，那么按照数组创建
	2). 如果是非纯数字的键，那么按照字典创建

比如:
gobolt json -m w -k file -i demo.json -p key1.key2.3.key4 -s "value1"
最终创建出来的 demo.json:
{
    "key1": {
        "key2": [
            null,
            null,
            null,
            {
                "key4": "value1"
            }
        ]
    }
}

(4). 路径强制创建

如果想要数字键也用字典来创建，那么需要使用 : 冒号强制指定格式

强烈建议!
	写入 JSON 的时候，如果是字典，那么总是用 : 冒号在前面，这样可以保证就算键里面
	本身就包含前置冒号的情况下也不会被错误处理！

比如:
gobolt json -m w -k file -i demo.json -p key1.key2.:3.key4 -s "value1"
最终创建出来的 demo.json:
{
    "key1": {
        "key2": {
            "3": {
                "key4": "value1"
            }
        }
    }
}

可以看到和上面的差别是所有的层级都是字典。

(5). 除了使用 -s 进行字符串写入外，还可以使用 -j 直接写入一个 JSON 字符串。

1). 写入浮点 空值 bool值
./gobolt json -m w -k file -i demo.json -p key1.key2.3.specialkey\\.\\[\\].0 -j "1.78"
./gobolt json -m w -k file -i demo.json -p key1.key2.3.specialkey\\.\\[\\].1 -j "null"
./gobolt json -m w -k file -i demo.json -p key1.key2.3.specialkey\\.\\[\\].2 -j "false"

{
    "key1": {
        "key2": [
            null,
            null,
            null,
            {
                "key4": "value1",
                "specialkey.[]": [
                    1.78,
                    null,
                    false
                ]
            }
        ]
    }
}

2). 写入数组
gobolt json -m w -k file -i demo.json -p key1.key2.3.specialkey\\.\\[\\].3 -j '["xx1", "yy2"]'

3). 写入字典
gobolt json -m w -k file -i demo.json -p key1.key2.3.specialkey\\.\\[\\].4 -j '{"xx3", "yy4"}'

4). 可以通过文件中的内容，来追加到我们的 JSON 中

由于 OS 的命令行参数是有限制的，所以如果要写入的内容特别大，那么把它们放到一个文件中，
然后通过 -f 参数指定文件名，然后把文件的内容写入，文件中的格式必须是一个合法的 JSON 文件

./gobolt json -m w -k file -i demo.json -p key1.key3 -f deme.json

上面的含义是把 deme.json 中的内容追加到 demo.json 中的 key1.key3 键里面。


4. 删除

删除一个键很简单。

./gobolt json -m d -k file -i demo.json -p key1.key2.3

如果要删除顶级键或者索引

./gobolt json -m d -k file -i demo.json -p key1

5. 普通字符串转换成合法的JSON字符串

./gobolt json -m s -k str -i "$str"

或者

printf "%s" "str" | ./gobolt json -m s -k stdin

转换结果会在标准输出中打印出来。

6. 普通的字符串转换成合法的gjson的路径字符串(escape)

./gobolt json -m e -k str -i "$str"

或者

printf "%s" "$str" | ./gobolt json -m e -k stdin
转换结果会在标准输出中打印出来。
		`,

		// 如果子命令还想嵌套子命令可以下面这么干
		// jsonCmd.AddCommand(ReadCmd())   // gobolt json read
		// jsonCmd.AddCommand(WriteCmd())  // gobolt json write

		RunE: func(cmd *cobra.Command, args []string) error {
			// ./gobolt json -f result.json -j "0" -m w -p Action.xx
			// -j 后面跟 true false 就是布尔值
			//	         null 空值
			//	         1.876 可以直接写入浮点值
			// 最后写入的是数字0而不是字符串0

			// 解析剩下来的参数
			if opts.UseArgPath {
				opts.ArgPath = args
				opts.Path = qJsonEscapeAndJoin(opts.ArgPath)
			}

			if opts.FileInput != "" {
				jsonData, err := os.ReadFile(opts.FileInput)
				if err != nil {
					return fmt.Errorf("无法读取文件 %s: %w", opts.FileInput, err)
				}
				if err := json.Unmarshal(jsonData, &opts.Input); err != nil {
					return fmt.Errorf("无效的 JSON 文件内容: %v", err)
				}
			} else if opts.JSONInput != "" {
				if err := json.Unmarshal([]byte(opts.JSONInput), &opts.Input); err != nil {
					return fmt.Errorf("无效的 JSON 字符串: %v", err)
				}
			} else {
				opts.Input = opts.StrInput
			}

			// 处理Path

			switch opts.Mode {
			case "r":
				return opts.readValueFromJSON()
			case "w":
				return opts.modifyJSON(opts.Input, func(jsonData []byte, path string, val any) ([]byte, error) {
					return sjson.SetBytes(jsonData, path, val)
				})
			case "d":
				return opts.modifyJSON(nil, func(jsonData []byte, path string, _ any) ([]byte, error) {
					return sjson.DeleteBytes(jsonData, path)
				})
			case "s":
				return opts.strToJsonStr()
			case "e":
				return opts.strToEscapeStr()
			case "v":
				return opts.printVer()
			case "t":
				return opts.printTypeCode()
			default:
				return fmt.Errorf("未知模式: %q，请使用 r / w / d", opts.Mode)
			}
		},
	}

	// flag 定义
	// :TODO: 是否需要做参数互斥检查？
	cmd.Flags().StringVarP(&opts.Mode, "mode", "m", "", "r / w / d / s / e / v / t 操作模式")
	cmd.Flags().StringVarP(&opts.Path, "path", "p", "", "gjson / sjson 原始路径，保留原始格式，但是并不建议使用，原因见范例")
	cmd.Flags().BoolVarP(&opts.UseArgPath, "argpath", "P", false, "从命令行中读取路径（需置于最后，空格分隔，强烈建议都用这种格式）")
	cmd.Flags().StringVarP(&opts.Kind, "kind", "k", "", "json来源类别（默认 stdin / file / str）")
	cmd.Flags().StringVarP(&opts.InArg, "inarg", "i", "", "json来源的值")
	cmd.Flags().StringVarP(&opts.Format, "format", "t", "txt", "输出格式：txt/sh/type")
	cmd.Flags().StringVarP(&opts.StrInput, "strinput", "s", "", "写入的字符串值")
	cmd.Flags().StringVarP(&opts.JSONInput, "jsoninput", "j", "", "写入的 JSON 字符串")
	// 如果要写入的内容特别大只能通过文件传递进来
	// 并且文件中只能放JSON格式数据
	cmd.Flags().StringVarP(&opts.FileInput, "fileinput", "f", "", "写入的 JSON 文件")
	// 输出的JSON文件的格式 (一行/多行美化打印)
	opts.JSONFormat = JSONFormatMul
	cmd.Flags().VarP(&opts.JSONFormat, "jsonformat", "F", "输出的 JSON 的格式(mul|one|raw|human)，代表多行/一行/原始格式/人类可读输出")

	cmd.MarkFlagRequired("mode")

	return cmd
}

func (opts *CLIOptions) printTypeCode() error {
	fmt.Println("JSON Type Codes:")
	fmt.Println("JSONTypeNull    =", JSONTypeNull)
	fmt.Println("JSONTypeTrue    =", JSONTypeTrue)
	fmt.Println("JSONTypeFalse   =", JSONTypeFalse)
	fmt.Println("JSONTypeNumber  =", JSONTypeNumber)
	fmt.Println("JSONTypeString  =", JSONTypeString)
	fmt.Println("JSONTypeArray   =", JSONTypeArray)
	fmt.Println("JSONTypeObject  =", JSONTypeObject)
	fmt.Println("JSONTypeUnknown =", JSONTypeUnknown)
	return nil
}

func (opts *CLIOptions) printVer() error {
	fmt.Println("gobolt json version:", QqjsonVersion)
	fmt.Println("gjson version:", GjsonVersion)
	fmt.Println("sjson version:", SjsonVersion)
	fmt.Println("pretty version:", PrettyVersion)
	return nil
}

func (opts *CLIOptions) strToEscapeStr() error {
	raw, err := opts.readInput()
	if err != nil {
		return err
	}

	// 末尾增加一个字符是为了防止$()语法中的末尾换行被bash自动去除
	fmt.Printf("%sX", qJsonEscape(string(raw)))

	return err
}

func (opts *CLIOptions) strToJsonStr() error {
	raw, err := opts.readInput()
	if err != nil {
		return err
	}

	// 使用 encoding/json 来生成合法的 JSON 字符串
	encoded, err := json.Marshal(string(raw))
	if err != nil {
		return err
	}

	fmt.Printf("%s", encoded)
	return nil
}

func (opts *CLIOptions) readInput() ([]byte, error) {
	var reader io.Reader

	switch opts.Kind {
	case "file":
		f, err := os.Open(opts.InArg)
		if err != nil {
			return nil, fmt.Errorf("无法打开文件: %w", err)
		}
		defer f.Close()
		reader = f
	case "str":
		reader = strings.NewReader(opts.InArg)
	default:
		reader = os.Stdin
	}

	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("读取失败: %w", err)
	}
	return raw, nil
}

func (opts *CLIOptions) readValueFromJSON() error {
	formatter, ok := formatters[opts.Format]
	if !ok {
		return fmt.Errorf("不支持的格式: %s", opts.Format)
	}

	var err error

	raw, err := opts.readInput()
	if err != nil {
		return err
	}

	// 校验 JSON 格式
	if !gjson.ValidBytes(raw) {
		err = fmt.Errorf("输入内容不是有效的 JSON")
		return err
	}

	var result gjson.Result
	if strings.TrimSpace(opts.Path) == "" {
		// 解析整个JOSN，作为顶级映射返回
		result = gjson.ParseBytes(raw)
	} else {
		result = gjson.GetBytes(raw, opts.Path)
		if !result.Exists() {
			err = fmt.Errorf("字段 %q 不存在", opts.Path)
			return err
		}
	}

	errFormat := formatter.Format(result, opts.JSONFormat)
	if errFormat.Code == errorutil.CodeSuccess &&
		errFormat.CmdExitCode == errorutil.CodeSuccess {
		return nil
	} else {
		return errFormat
	}
}

// ./gobolt -w a1.Data9.xx\\.yy -d xx -f result2.jso
// 本身带点号的键需要这样传入
func (opts *CLIOptions) modifyJSON(
	value any,
	operation func([]byte, string, any) ([]byte, error)) error {

	var jsonData []byte
	var err error

	switch opts.Kind {
	case "file":
		if _, err := os.Stat(opts.InArg); os.IsNotExist(err) {
			f, err := os.Create(opts.InArg)
			if err != nil {
				return fmt.Errorf("无法创建文件: %w", err)
			}
			defer f.Close()
		}
		jsonData, err = os.ReadFile(opts.InArg)
		if err != nil {
			return err
		}
	case "str":
		jsonData = []byte(opts.InArg)
	// 没有任何参数的情况 或者 stdin 的情况
	default:
		jsonData, err = io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
	}

	// 应用传入的操作函数（设置或删除）
	updatedJSON, err := operation(jsonData, opts.Path, value)
	if err != nil {
		return err
	}

	formatted := formatJSON(updatedJSON, opts.JSONFormat)

	if opts.Kind == "file" {
		return os.WriteFile(opts.InArg, formatted, 0644)
	}
	// 字符串的情况和 str 的情况都输出到标准输出
	_, err = os.Stdout.Write(formatted)
	return err
}
