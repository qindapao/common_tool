package diffutil

import (
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

type DiffLine struct {
	Left  string
	Right string
	Mark  string // "|", "+", "-", "~"
}

func CompareMultiline(before, after string) []DiffLine {
	dmp := diffmatchpatch.New()
	text1, text2, lineArray := dmp.DiffLinesToChars(before, after)
	diffs := dmp.DiffMain(text1, text2, false)
	dmp.DiffCleanupSemantic(diffs)
	diffs = dmp.DiffCharsToLines(diffs, lineArray)

	var result []DiffLine
	i := 0
	for i < len(diffs) {
		d := diffs[i]
		if d.Type == diffmatchpatch.DiffDelete &&
			i+1 < len(diffs) &&
			diffs[i+1].Type == diffmatchpatch.DiffInsert {

			delLines := strings.Split(d.Text, "\n")
			insLines := strings.Split(diffs[i+1].Text, "\n")

			maxLen := max(len(delLines), len(insLines))
			for i := range make([]struct{}, maxLen) {
				l, r := "", ""
				if i < len(delLines) {
					l = delLines[i]
				}
				if i < len(insLines) {
					r = insLines[i]
				}
				if l == "" && r == "" {
					continue
				}
				result = append(result, DiffLine{Left: l, Right: r, Mark: "~"})
			}
			i += 2
			continue
		}

		lines := strings.Split(d.Text, "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			switch d.Type {
			case diffmatchpatch.DiffEqual:
				result = append(result, DiffLine{Left: line, Right: line, Mark: "|"})
			case diffmatchpatch.DiffDelete:
				result = append(result, DiffLine{Left: line, Right: "", Mark: "-"})
			case diffmatchpatch.DiffInsert:
				result = append(result, DiffLine{Left: "", Right: line, Mark: "+"})
			}
		}
		i++
	}
	return result
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

