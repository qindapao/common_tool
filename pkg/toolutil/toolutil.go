package toolutil

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const ProjectPrefix = "common_tool/"

func TrimToProjectPath(file string) string {
	// 统一分隔符，确保在不同操作系统下都表现一致
	path := filepath.ToSlash(file)

	// 查找前缀位置
	if idx := strings.Index(path, ProjectPrefix); idx >= 0 {
		return path[idx+len(ProjectPrefix):]
	}
	return path
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
			closeErr = fmt.Errorf("关闭文件 %s 失败: %w", filePath, cerr)
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

// 把任意对象转换成JSON格式
func ToJSONIndent(obj any) string {
	data, err := json.MarshalIndent(obj, "", "    ")
	if err != nil {
		return fmt.Sprintf(`{"error":"failed to marshalindent object: %s"}`, err)
	}
	return string(data)
}

// 把任意对象转换成JSON格式(紧凑)
func ToJSON(obj any) string {
	data, err := json.Marshal(obj)
	if err != nil {
		return fmt.Sprintf(`{"error":"failed to marshal object: %s"}`, err)
	}
	return string(data)
}

// sortedKeys 将 map[string]struct{} 的 key 排序后返回
func SortedKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}