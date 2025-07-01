package hex

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// 从文件读取十六进制字符串，补全前缀 "0x"
func ReadHexStrFf(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return "0x0000"
	}
	s := strings.TrimSpace(string(b))
	if !strings.HasPrefix(s, "0x") {
		s = "0x" + s
	}
	return s
}

func parseHex(s string, bits int) (uint64, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "\uFEFF") // 去除 BOM
	s = strings.TrimPrefix(strings.ToLower(s), "0x")
	return strconv.ParseUint(s, 16, bits)
}

// 文件中16进制字符串读取成正整数
func ReadHexToUint64Ff(path string) (uint64, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read file failed: %w", err)
	}

	return ParseHexToUint64(string(b))
}

func ReadHexToUint16Ff(path string) (uint16, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read file failed: %w", err)
	}

	return ParseHexToUint16(string(b))
}

func ReadHexToUint32Ff(path string) (uint32, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read file failed: %w", err)
	}

	return ParseHexToUint32(string(b))
}

func ParseHexToUint16(s string) (uint16, error) {
	v, err := parseHex(s, 16)
	return uint16(v), err
}

func ParseHexToUint32(s string) (uint32, error) {
	v, err := parseHex(s, 32)
	return uint32(v), err
}

func ParseHexToUint64(s string) (uint64, error) {
	return parseHex(s, 64)
}