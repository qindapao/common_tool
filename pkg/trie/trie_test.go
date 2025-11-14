package trie

import (
	"strings"
	"testing"

	"github.com/armon/go-radix"
)

func TestPrettyPrintRadix(t *testing.T) {
	r := radix.New()
	r.Insert("cat", "动物")
	r.Insert("car", "交通工具")
	r.Insert("dog", "动物")
	r.Insert("cart", "购物车")

	output := PrettyPrintRadix(r)

	// 打印出来方便肉眼查看
	t.Log("\n" + output)

	// 预期包含的关键行
	expectedLines := []string{
		"c",
		"  a",
		"    r : 交通工具",
		"      t : 购物车",
		"    t : 动物",
		"d",
		"  o",
		"    g : 动物",
	}

	for _, line := range expectedLines {
		if !strings.Contains(output, line) {
			t.Errorf("输出缺少预期行: %q\n实际输出:\n%s", line, output)
		}
	}
}

func TestPrettyPrintRadixCompressed_Simple(t *testing.T) {
	r := radix.New()
	r.Insert("car", "交通工具")
	r.Insert("cart", "购物车")
	r.Insert("cat", "动物")
	r.Insert("dog", "动物")

	output := PrettyPrintRadixCompressed(r)
	t.Log("\n" + output)

	expectedLines := []string{
		"car : 交通工具",
		"  cart : 购物车",
		"cat : 动物",
		"dog : 动物",
	}

	for _, line := range expectedLines {
		if !strings.Contains(output, line) {
			t.Errorf("缺少预期行: %q\n实际输出:\n%s", line, output)
		}
	}
}
func TestPrettyPrintRadixCompressed_Complex(t *testing.T) {
	r := radix.New()
	r.Insert("app", "应用")
	r.Insert("apple", "水果")
	r.Insert("application", "应用程序")
	r.Insert("apply", "动词：应用")
	r.Insert("banana", "水果")
	r.Insert("band", "乐队")
	r.Insert("bandwidth", "带宽")
	r.Insert("bank", "银行")

	output := PrettyPrintRadixCompressed(r)
	t.Log("\n" + output)

	expectedLines := []string{
		"app : 应用",
		"  apple : 水果",
		"    application : 应用程序",
		"    apply : 动词：应用",
		"banana : 水果",
		"  band : 乐队",
		"    bandwidth : 带宽",
		"  bank : 银行",
	}

	for _, line := range expectedLines {
		if !strings.Contains(output, line) {
			t.Errorf("缺少预期行: %q\n实际输出:\n%s", line, output)
		}
	}
}
func TestPrettyPrintRadixCompressed_Complex_Example1(t *testing.T) {
	r := radix.New()
	// app 族群
	r.Insert("app", "应用")
	r.Insert("apple", "水果")
	r.Insert("application", "应用程序")
	r.Insert("apply", "动词：应用")
	r.Insert("apt", "恰当的")
	r.Insert("apron", "围裙")
	// aq 族群
	r.Insert("aqua", "水")
	r.Insert("aquaculture", "水产养殖")
	r.Insert("aquarium", "水族馆")
	// ban 族群
	r.Insert("banana", "水果")
	r.Insert("band", "乐队")
	r.Insert("bandage", "绷带")
	r.Insert("bandwidth", "带宽")
	r.Insert("bank", "银行")
	r.Insert("banker", "银行家")
	r.Insert("banking", "银行业务")
	r.Insert("banner", "横幅")

	output := PrettyPrintRadixCompressed(r)
	t.Log("\n" + output)

	expectedLines := []string{
		"app : 应用",
		"  apple : 水果",
		"    application : 应用程序",
		"    apply : 动词：应用",
		"apt : 恰当的",
		"apron : 围裙",
		"aqua : 水",
		"  aquaculture : 水产养殖",
		"  aquarium : 水族馆",
		"banana : 水果",
		"  band : 乐队",
		"    bandage : 绷带",
		"    bandwidth : 带宽",
		"  bank : 银行",
		"    banker : 银行家",
		"    banking : 银行业务",
		"  banner : 横幅",
	}

	for _, line := range expectedLines {
		if !strings.Contains(output, line) {
			t.Errorf("缺少预期行: %q\n实际输出:\n%s", line, output)
		}
	}
}
func TestPrettyPrintRadixCompressed_Complex_Example2(t *testing.T) {
	r := radix.New()
	// do 族群
	r.Insert("do", "助动词")
	r.Insert("dog", "动物")
	r.Insert("door", "门")
	r.Insert("doom", "厄运")
	r.Insert("dove", "鸽子")
	r.Insert("dot", "点")
	// ca 族群
	r.Insert("car", "交通工具")
	r.Insert("cart", "购物车")
	r.Insert("cat", "动物")
	// emoji 族群
	r.Insert("book", "书")
	r.Insert("bookmark", "书签")
	r.Insert("bookshelf", "书架")
	// 中文
	r.Insert("书", "book")
	r.Insert("书本", "books")
	r.Insert("书签", "bookmark")

	output := PrettyPrintRadixCompressed(r)
	t.Log("\n" + output)

	expectedLines := []string{
		"do : 助动词",
		"  dog : 动物",
		"  door : 门",
		"  doom : 厄运",
		"  dove : 鸽子",
		"  dot : 点",
		"car : 交通工具",
		"  cart : 购物车",
		"cat : 动物",
		"book : 书",
		"  bookmark : 书签",
		"  bookshelf : 书架",
		"书 : book",
		"  书本 : books",
		"  书签 : bookmark",
	}

	for _, line := range expectedLines {
		if !strings.Contains(output, line) {
			t.Errorf("缺少预期行: %q\n实际输出:\n%s", line, output)
		}
	}
}
