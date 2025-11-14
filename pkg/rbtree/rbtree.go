package rbtree

import (
	"common_tool/pkg/treeprinter"
	"fmt"

	rbt "github.com/emirpasic/gods/trees/redblacktree"
	"github.com/emirpasic/gods/utils"
)

// 打印红黑树，第二个参数可选
func PrintRBTree(tree *rbt.Tree, formats ...func(*rbt.Node) string) string {
	// 默认格式：只打印 Key
	defaultFormat := func(n *rbt.Node) string {
		return fmt.Sprintf("%v", n.Key)
	}

	// 如果用户传了函数，就用第一个；否则用默认的
	var getValue func(*rbt.Node) string
	if len(formats) > 0 && formats[0] != nil {
		getValue = formats[0]
	} else {
		getValue = defaultFormat
	}

	return treeprinter.PrintTreeGeneric(treeprinter.TreePrinter[*rbt.Node]{
		Root: tree.Root,
		GetChild: func(n *rbt.Node, dir string) *rbt.Node {
			if dir == "left" {
				return n.Left
			}
			return n.Right
		},
		GetValue: getValue,
		IsNil: func(n *rbt.Node) bool {
			return n == nil
		},
		Style:     1,
		Direction: 0,
	})
}

// 打桩函数：先简单用一下库里的 API
func StubUsage() {
	// 创建一个红黑树，使用 IntComparator
	tree := rbt.NewWith(utils.IntComparator)

	// 插入一些元素
	tree.Put(1, "a")
	tree.Put(2, "b")

	// 简单查找
	if value, found := tree.Get(1); found {
		fmt.Println("Stub 查找到 key=1, value=", value)
	}

	// 删除一个元素
	tree.Remove(2)

	// 打印剩余的键值
	fmt.Println("=== PrintRBTree Key:Value ===")
	fmt.Println(PrintRBTree(tree, func(n *rbt.Node) string {
		return fmt.Sprintf("%v:%v", n.Key, n.Value)
	}))
}
