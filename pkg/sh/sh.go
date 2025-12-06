package sh

import (
	"fmt"
	"strings"
)

const (
	// 处理波浪号
	// FlagTildeSpecial = 1 // flags & 1
	// 处理空格和TAB
	FlagShellBlank = 2 // flags & 2
)

var DefaultBSTab = buildDefaultBSTab()

// 和Bash源码一些不同的地方
// 1. 47 / 这里转义了，极端情况下防止语义冲突
// 2. 61 = 这里转义了，防止语义冲突
// 3. 126 ~ 这里直接转义了
func buildDefaultBSTab() [256]byte {
	var table [256]byte

	for _, i := range []int{
		9, 10, // tab, newline
		32, 33, 34, 36, 38, 39, 40, 41, 42, 44, 47, // 基本符号
		59, 60, 61, 62, 63, // ; < = > ?
		91, 92, 93, 94, // [ \ ] ^
		96, 123, 124, 125, 126, // ` { | } ~
	} {
		table[i] = 1
	}

	return table
}

// ShBackslashQuote mimics Bash-style quoting via backslash escape
func ShBackslashQuote(inStr string, table *[256]byte, flags int) string {
	if table == nil {
		table = &DefaultBSTab
	}

	var b strings.Builder
	for i, c := range inStr {
		ci := int(c)
		switch {
		case ci < 256 && table[ci] == 1:
			b.WriteString(`\` + string(c))
		case c == '#' && i == 0:
			b.WriteString(`\#`)
		// 原始的Bash代码中是这样判断的，既然表中强制转义了，那么这个判断也不用了
		// case flags&FlagTildeSpecial != 0 && c == '~' && (i == 0 || strings.ContainsRune(":=,", rune(inStr[i-1]))):
		// 	b.WriteString(`\~`)
		case flags&FlagShellBlank != 0 && (c == ' ' || c == '\t'):
			b.WriteString(`\` + string(c))
		default:
			// 编码成UTF-8 后写入
			b.WriteRune(c)
		}
	}

	return b.String()
}

// 下面用来测试
// $'\a\b\t\n\v\f\r\E\\\'\000\001ABC中文'
// BashANSIQuote 将任意字符串转为 $'...' 形式的 ANSI-C 样式安全字符串
func BashANSIQuote(s string) string {
	var b strings.Builder
	b.WriteString("$'")

	for _, r := range s {
		switch r {
		case 27: // Escape (ASCII 27)
			b.WriteString(`\E`)
		case '\a':
			b.WriteString(`\a`)
		case '\b':
			b.WriteString(`\b`)
		case '\t':
			b.WriteString(`\t`)
		case '\n':
			b.WriteString(`\n`)
		case '\v':
			b.WriteString(`\v`)
		case '\f':
			b.WriteString(`\f`)
		case '\r':
			b.WriteString(`\r`)
		case '\\':
			b.WriteString(`\\`)
		case '\'':
			b.WriteString(`\'`)
		default:
			if r < 32 || r == 127 {
				// 对不可打印字符使用 \ooo 八进制转义
				b.WriteString(fmt.Sprintf(`\%03o`, r))
			} else {
				b.WriteRune(r)
			}
		}
	}

	b.WriteString("'")
	return b.String()
}

func BuildCommandLineQuoted(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = BashANSIQuote(arg)
	}
	return strings.Join(quoted, " ")
}
