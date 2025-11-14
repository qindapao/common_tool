package rbtree

import (
	"fmt"
	"strings"
	"testing"

	rbt "github.com/emirpasic/gods/trees/redblacktree"
)

func PrintRBTreeValue(n *rbt.Node) string {
	return fmt.Sprintf("%v(%v)", n.Key, n.Value)
}

func TestPrintRBTree(t *testing.T) {
	// 创建红黑树
	tree := rbt.NewWithIntComparator()

	// 插入一些元素
	tree.Put(10, "中文")
	tree.Put(20, "b")
	tree.Put(5, "c")
	tree.Put(15, "d")
	tree.Put(25, "e")

	// 打印树结构（库自带的打印）
	output := tree.String()
	fmt.Println("=== tree.String() ===")
	fmt.Println(output)

	// 打印树结构（自定义打印，默认只显示 Key）
	fmt.Println("=== PrintRBTree 默认 ===")
	fmt.Println(PrintRBTree(tree))

	// 打印树结构（自定义打印，显示 Key:Value）
	fmt.Println("=== PrintRBTree Key:Value ===")
	fmt.Println(PrintRBTree(tree, func(n *rbt.Node) string {
		return fmt.Sprintf("%v:%v", n.Key, n.Value)
	}))

	fmt.Println("=== PrintRBTree Key(Value) ===")
	checkCustomStr := PrintRBTree(tree, PrintRBTreeValue)
	fmt.Println(checkCustomStr)

	// 简单断言：输出里应该包含某些关键节点
	if !strings.Contains(output, "10") ||
		!strings.Contains(output, "20") ||
		!strings.Contains(output, "5") {
		t.Errorf("打印结果缺少关键节点: \n%s", output)
	}

	if !strings.Contains(checkCustomStr, "10(中文)") ||
		!strings.Contains(checkCustomStr, "20") ||
		!strings.Contains(checkCustomStr, "5") {
		t.Errorf("打印结果缺少关键节点: \n%s", checkCustomStr)
	}
}
