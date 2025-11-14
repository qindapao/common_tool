package treeprinter

import (
	"fmt"
	"strings"
)

const (
	nodeVisitFirst        = 0
	nodeVisitReadyToPrint = 1
	nodeVisitDone         = 2

	BranchUpper = 1
	BranchLower = 0
	BranchRoot  = -1
)

// TreeNode 是任意类型的节点标识符（可以是索引、指针等）
type TreeNode any

// TreePrinter 是通用打印器的配置结构
type TreePrinter[T TreeNode] struct {
	Root      T
	GetChild  func(T, string) T // 获取左右子节点
	GetValue  func(T) string    // 获取节点值的字符串表示
	IsNil     func(T) bool      // 判断节点是否为空
	Style     int               // 0 = ascii, 1 = unicode
	Direction int               // 0 = right-root-left, 1 = left-root-right
}

// PrintTreeGeneric 是通用的树结构打印函数
func PrintTreeGeneric[T TreeNode](printer TreePrinter[T]) string {
	var (
		vert     string
		uRArrow  string
		dRArrow  string
		rootSign string
	)

	if printer.Style == 1 {
		vert = "│"
		uRArrow = "┌──>"
		dRArrow = "└──>"
		rootSign = "│── "
	} else {
		vert = "|"
		uRArrow = ".-->"
		dRArrow = "'-->"
		rootSign = "|-- "
	}

	printDir := "right"
	printOpoDir := "left"
	if printer.Direction == 1 {
		printDir = "left"
		printOpoDir = "right"
	}

	type stackEntry struct {
		node          T
		branchPos     int
		pre           string
		hasUpperChild int
		hasLowerChild int
		nodeSts       int
	}

	if printer.IsNil(printer.Root) {
		return "tree is empty\n"
	}

	stack := []stackEntry{
		{node: printer.Root, branchPos: BranchRoot, pre: "", hasUpperChild: 1, hasLowerChild: 1, nodeSts: nodeVisitFirst},
	}

	var b strings.Builder

	for len(stack) > 0 {
		idx := len(stack) - 1
		top := stack[idx]

		switch top.nodeSts {
		case nodeVisitFirst:
			top.nodeSts = nodeVisitReadyToPrint
			stack[idx] = top

			child := printer.GetChild(top.node, printDir)
			if !printer.IsNil(child) {
				newPre := top.pre
				if top.hasUpperChild == 1 {
					newPre += "    "
				} else {
					newPre += vert + "   "
				}
				top.hasUpperChild = 1
				stack[idx] = top

				stack = append(stack, stackEntry{
					node:          child,
					branchPos:     BranchUpper,
					pre:           newPre,
					hasUpperChild: 1,
					hasLowerChild: 0,
					nodeSts:       nodeVisitFirst,
				})
			}
		case nodeVisitReadyToPrint:
			stack[idx].nodeSts = nodeVisitDone
			valStr := printer.GetValue(top.node)
			switch top.branchPos {
			case BranchUpper:
				fmt.Fprintf(&b, "%s%s%s\n", top.pre, uRArrow, valStr)
			case BranchLower:
				fmt.Fprintf(&b, "%s%s%s\n", top.pre, dRArrow, valStr)
			default:
				fmt.Fprintf(&b, "%s%s\n", rootSign, valStr)
			}
		case nodeVisitDone:
			savedNode := top.node
			savedPre := top.pre
			savedHasL := top.hasLowerChild

			stack = stack[:idx]

			child := printer.GetChild(savedNode, printOpoDir)
			if !printer.IsNil(child) {
				newPre := savedPre
				if savedHasL == 1 {
					newPre += "    "
				} else {
					newPre += vert + "   "
				}
				stack = append(stack, stackEntry{
					node:          child,
					branchPos:     BranchLower,
					pre:           newPre,
					hasUpperChild: 0,
					hasLowerChild: 1,
					nodeSts:       nodeVisitFirst,
				})
			}
		}
	}

	return b.String()
}

type MultiNode struct {
	Data     any // 节点数据，可以是任意类型
	Children []*MultiNode
}

type MultiTreePrinter struct {
	Root     *MultiNode
	Style    int                     // 0 = ascii, 1 = unicode
	FormatFn func(*MultiNode) string // 可选的自定义格式化函数
}

func PrintMultiTree(printer MultiTreePrinter) string {
	if printer.Root == nil {
		return "tree is empty\n"
	}

	var b strings.Builder

	var dfs func(node *MultiNode, prefix string, isLast bool)
	dfs = func(node *MultiNode, prefix string, isLast bool) {
		if node == nil {
			return
		}

		connector := ""
		branch := ""
		space := ""
		if printer.Style == 1 {
			connector = "└── "
			branch = "├── "
			space = "│   "
		} else {
			connector = "'-- "
			branch = ".-- "
			space = "|   "
		}

		// 使用 FormatFn，如果没有就用默认 Data 的字符串
		label := fmt.Sprintf("%v", node.Data)
		if printer.FormatFn != nil {
			label = printer.FormatFn(node)
		}

		if isLast {
			b.WriteString(fmt.Sprintf("%s%s%s\n", prefix, connector, label))
		} else {
			b.WriteString(fmt.Sprintf("%s%s%s\n", prefix, branch, label))
		}

		for i, child := range node.Children {
			newPrefix := prefix
			if isLast {
				newPrefix += "    "
			} else {
				newPrefix += space
			}
			dfs(child, newPrefix, i == len(node.Children)-1)
		}
	}

	dfs(printer.Root, "", true)
	return b.String()
}
