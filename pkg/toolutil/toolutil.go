package toolutil

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"reflect"
)

// Grep 函数：从文本中筛选符合条件的行
func Grep(
	lines []string,
	pattern string,
	caseInsensitive bool,
	useRegex bool) []string {
	var result []string
	var matcher func(string) bool

	if useRegex {
		// 正则匹配
		if caseInsensitive {
			pattern = "(?i)" + pattern // (?i) 让正则忽略大小写
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			fmt.Println("正则表达式错误:", err)
			return nil
		}
		matcher = re.MatchString
	} else {
		// 关键字匹配（开头匹配）
		if caseInsensitive {
			pattern = strings.ToLower(pattern)
			matcher = func(s string) bool {
				return strings.HasPrefix(strings.ToLower(s), pattern)
			}
		} else {
			matcher = func(s string) bool {
				return strings.HasPrefix(s, pattern)
			}
		}
	}

	// 遍历文本列表进行匹配
	for _, line := range lines {
		if matcher(line) {
			result = append(result, line)
		}
	}

	return result
}

// map 函数：应用某个转换逻辑
func MapStrings(lines []string, transform func(string) string) []string {
	result := make([]string, len(lines))
	for i, line := range lines {
		result[i] = transform(line)
	}
	return result
}

// Go 1.8 才支持，先屏蔽掉
// // 泛型 Map 函数：对切片中的每个元素应用转换函数
// // :TODO: vscode语法检查这里报错，不知道原因
// func Map[T any, R any](slice []T, transform func(T) R) []R {
//     result := make([]R, len(slice))
//     for i, v := range slice {
//         result[i] = transform(v)
//     }
//     return result
// }

// 读取文件并返回按行拆分的字符串列表，适用于所有操作系统
func ReadFileToLines(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("无法打开文件 %s: %w", filePath, err)
	}

	// 使用 defer + 匿名函数捕获 file.Close() 错误
	var closeErr error
	defer func() {
		if cerr := file.Close(); cerr != nil {
			closeErr = fmt.Errorf("关闭文件 %s 失败: %w\n", filePath, cerr)
		}
	}()

	var lines []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		// 自动处理不同操作系统的换行符
		lines = append(lines, scanner.Text())
	}

	readErr := scanner.Err()
	if readErr != nil {
		readErr = fmt.Errorf("读取文件 %s 出错: %w", filePath, readErr)
	}

	if readErr != nil || closeErr != nil {
		return lines, errors.Join(readErr, closeErr)
	}

	return lines, nil
}

// :TODO: 模拟awk的字符串截取，但是不完全相同(无分隔符打印第一个字段的情况)
type StringProcessor struct {
	str string
}

// 创建新的 StringProcessor 实例
func NewStringProcessor(s string) *StringProcessor {
	return &StringProcessor{str: s}
}

// 以指定分隔符拆分字符串，并更新为指定索引的字段
func (p *StringProcessor) Split(delim string, index int) *StringProcessor {
	fields := strings.Split(p.str, delim)
	if index >= 0 && index < len(fields) {
		p.str = fields[index]
	} else {
		p.str = "" // 索引超出范围时返回空串
	}
	return p
}

// 获取最终结果
func (p *StringProcessor) Res() string {
	return p.str
}

// Int64 Min 函数
// :TODO: 如果Go 1.18支持泛型可以用泛型实现通用的Min函数
// 泛型中的比较也比较麻烦
func MinInt64(values ...int64) int64 {
	if len(values) == 0 {
		panic("MinInt64 requires at least one argument")
	}
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func MaxInt64(values ...int64) int64 {
	if len(values) == 0 {
		panic("MaxInt64 requires at least one argument")
	}
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

// 判断map中是否至少有一个字符串键
func HasAnyKey(m map[string]string, keys ...string) bool {
	for _, key := range keys {
		if _, exists := m[key]; exists {
			return true
		}
	}
	return false
}

// StructToMap 递归转换结构体到 map
func StructToMap(obj any) map[string]any {
	result := make(map[string]any)
	val := reflect.ValueOf(obj)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		// 处理嵌套结构体
		if fieldValue.Kind() == reflect.Struct {
			if field.Anonymous {
				// **扁平化匿名嵌套结构体（即 Go 结构体继承，如 ParserBase）**
				subMap := StructToMap(fieldValue.Interface())
				for k, v := range subMap {
					result[k] = v // 合并，而不是嵌套
				}
			} else {
				// **真正的嵌套结构体，保持层级**
				result[field.Name] = StructToMap(fieldValue.Interface())
			}
		} else {
			result[field.Name] = fieldValue.Interface()
		}
	}

	return result
}