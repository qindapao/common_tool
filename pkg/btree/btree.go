package btree

import (
	"sort"

	"common_tool/pkg/treeprinter"

	"github.com/google/btree"
)

// 泛型 PrintNode
type PrintNode[T any] struct {
	items    []T
	children []*PrintNode[T]
}

func (n *PrintNode[T]) isLeaf() bool { return len(n.children) == 0 }

// 泛型 PrinterBTree
type PrinterBTree[T any] struct {
	order int
	root  *PrintNode[T]
	toKey func(T) int // 提取 key 的函数
}

func NewPrinterBTree[T any](order int, toKey func(T) int) *PrinterBTree[T] {
	return &PrinterBTree[T]{order: order, toKey: toKey}
}

func (bt *PrinterBTree[T]) insertAt(node *PrintNode[T], item T) *PrintNode[T] {
	key := bt.toKey(item)

	if node.isLeaf() {
		// 插入到叶子
		i := sort.Search(len(node.items), func(i int) bool {
			return bt.toKey(node.items[i]) >= key
		})
		if i < len(node.items) && bt.toKey(node.items[i]) == key {
			return node // 去重
		}
		node.items = append(node.items, item)
		copy(node.items[i+1:], node.items[i:])
		node.items[i] = item

		// 分裂
		if len(node.items) > bt.order-1 {
			mid := len(node.items) / 2
			leftItems := make([]T, mid)
			copy(leftItems, node.items[:mid])
			rightItems := make([]T, len(node.items[mid+1:]))
			copy(rightItems, node.items[mid+1:])
			left := &PrintNode[T]{items: leftItems}
			right := &PrintNode[T]{items: rightItems}
			return &PrintNode[T]{items: []T{node.items[mid]}, children: []*PrintNode[T]{left, right}}
		}
		return node
	}

	// 内部节点
	i := sort.Search(len(node.items), func(i int) bool {
		return bt.toKey(node.items[i]) >= key
	})
	child := node.children[i]
	newChild := bt.insertAt(child, item)
	if newChild != child {
		// 子节点分裂，上推
		mid := newChild.items[0]
		node.items = append(node.items, mid)
		copy(node.items[i+1:], node.items[i:])
		node.items[i] = mid

		left := newChild.children[0]
		right := newChild.children[1]
		node.children = append(node.children, nil)
		copy(node.children[i+2:], node.children[i+1:])
		node.children[i] = left
		node.children[i+1] = right
	}

	// 分裂当前节点
	if len(node.items) > bt.order-1 {
		mid := len(node.items) / 2
		// 拷贝 items（左右两边）
		leftItems := make([]T, mid)
		copy(leftItems, node.items[:mid])

		rightItems := make([]T, len(node.items[mid+1:]))
		copy(rightItems, node.items[mid+1:])

		// 拷贝 children（仅内部节点，长度分别是 mid+1 和余下部分）
		leftChildren := make([]*PrintNode[T], mid+1)
		copy(leftChildren, node.children[:mid+1])

		rightChildren := make([]*PrintNode[T], len(node.children[mid+1:]))
		copy(rightChildren, node.children[mid+1:])

		left := &PrintNode[T]{items: leftItems, children: leftChildren}
		right := &PrintNode[T]{items: rightItems, children: rightChildren}

		// 上推中间键（注意：这是 B-Tree 分裂返回的新“临时根”）
		return &PrintNode[T]{
			items:    []T{node.items[mid]},
			children: []*PrintNode[T]{left, right},
		}
	}
	return node
}

func (bt *PrinterBTree[T]) Insert(item T) {
	if bt.root == nil {
		bt.root = &PrintNode[T]{items: []T{item}}
		return
	}
	newRoot := bt.insertAt(bt.root, item)
	if newRoot != bt.root {
		bt.root = newRoot
	}
}

// 泛型 BuildBTreeAsMultiTree
func BuildBTreeAsMultiTree[T btree.Item](tr *btree.BTree, order int, toKey func(T) int) *treeprinter.MultiNode {
	bt := NewPrinterBTree(order, toKey)
	tr.Ascend(func(i btree.Item) bool {
		bt.Insert(i.(T))
		return true
	})
	return convert(bt.root)
}

func convert[T any](n *PrintNode[T]) *treeprinter.MultiNode {
	if n == nil {
		return nil
	}
	children := []*treeprinter.MultiNode{}
	for _, c := range n.children {
		children = append(children, convert(c))
	}
	return &treeprinter.MultiNode{Data: n.items, Children: children}
}

// 泛型 PrettyPrintBTree
func PrettyPrintBTree[T btree.Item](tr *btree.BTree, order int, toKey func(T) int, formatFn func(*treeprinter.MultiNode) string) string {
	root := BuildBTreeAsMultiTree(tr, order, toKey)
	pr := treeprinter.MultiTreePrinter{
		Root:     root,
		Style:    1,
		FormatFn: formatFn,
	}
	return treeprinter.PrintMultiTree(pr)
}
