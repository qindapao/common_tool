package diffutil

import (
	"testing"
)

func TestCompareMultiline_BasicStructure(t *testing.T) {
	before := `
┌──>A(h=1)
│   └──>B(h=2)
└──>C(h=3)`
	after := `
┌──>A(h=1)
│   └──>B(h=2)
└──>D(h=3)`

	diff := CompareMultiline(before, after)
	output := FormatSideBySide(diff)
	t.Log("\nBasic Structure Diff:\n" + output)
}

func TestCompareMultiline_ChineseCharacters(t *testing.T) {
	before := `
┌──>你好(h=1)
│   └──>世界(h=2)
└──>测试(h=3)`
	after := `
┌──>你好(h=1)
│   └──>地球(h=2)
└──>测试(h=3)`

	diff := CompareMultiline(before, after)
	output := FormatSideBySide(diff)
	t.Log("\nChinese Character Diff:\n" + output)
}

func TestCompareMultiline_MixedLanguage(t *testing.T) {
	before := `
┌──>Hello(h=1)
│   └──>世界(h=2)
└──>Test(h=3)`
	after := `
┌──>Hello(h=1)
│   └──>地球(h=2)
└──>Test(h=3)`

	diff := CompareMultiline(before, after)
	output := FormatSideBySide(diff)
	t.Log("\nMixed Language Diff:\n" + output)
}

func TestCompareMultiline_EmptyLines(t *testing.T) {
	before := `
┌──>A(h=1)

└──>B(h=2)`
	after := `
┌──>A(h=1)

└──>C(h=2)`

	diff := CompareMultiline(before, after)
	output := FormatSideBySide(diff)
	t.Log("\nEmpty Line Diff:\n" + output)
}

func TestCompareMultiline_MultipleModifications(t *testing.T) {
	before := `
┌──>86(h=1)
│   └──>84(h=1)
└──>80(h=3)`
	after := `
┌──>86(h=1)
│   └──>85(h=2)
└──>80(h=3)`

	diff := CompareMultiline(before, after)
	output := FormatSideBySide(diff)
	t.Log("\nMultiple Modifications Diff:\n" + output)
}
