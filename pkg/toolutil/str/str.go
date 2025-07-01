// 简单的字符串操作库
package str

import (
	"os"
)

// 从文件读取原始字符串
func ReadStrFf(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

// 字符串为空的时候设置字符串的默认值
func DefaultStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}