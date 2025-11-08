package diffutil

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

type TextInfo struct {
	CharsCount   int
	DisplayWidth int
}

// len(s): 字节宽度
// utf8.RuneCountInString(s): 字符数
// runewidth.StringWidth(s): 显示宽度
// fmt.Sprintf 是按照字符数打印
// 总字符数 = 字符数 + (最大显示宽度 - 当前行显示宽度)
func FormatSideBySide(diff []DiffLine) string {
	type TextInfo struct {
		CharsCount   int
		DisplayWidth int
	}

	// 修正宽度判断(模糊字符按照宽度1计算)
	runewidth.DefaultCondition.EastAsianWidth = false

	maxDisPlayWidth := 0
	leftCharsNum := make([]TextInfo, 0, len(diff))
	for _, d := range diff {
		disPlayLen := runewidth.StringWidth(d.Left)
		if disPlayLen > maxDisPlayWidth {
			maxDisPlayWidth = disPlayLen
		}
		leftCharsNum = append(leftCharsNum, TextInfo{
			CharsCount:   utf8.RuneCountInString(d.Left),
			DisplayWidth: runewidth.StringWidth(d.Left),
		})
	}

	var out []string
	header := fmt.Sprintf("%-*s  %s  %s", maxDisPlayWidth, "* Before", " ", "* After")
	out = append(out, header)
	out = append(out, strings.Repeat("-", len(header)))

	for idx, d := range diff {
		// 原始的字符数+补充的空格数
		out = append(out, fmt.Sprintf(
			"%-*s  %s  %s",
			leftCharsNum[idx].CharsCount+maxDisPlayWidth-leftCharsNum[idx].DisplayWidth,
			d.Left, d.Mark, d.Right))
	}

	return strings.Join(out, "\n")
}
