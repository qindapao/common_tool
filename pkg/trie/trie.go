package trie

import (
	"fmt"
	"slices"
	"strings"

	"github.com/armon/go-radix"
)

// Node 用来构建临时的层级树
type Node struct {
	children map[rune]*Node
	val      any
}

// 构建一个临时 Trie
func buildTrie(r *radix.Tree) *Node {
	root := &Node{children: make(map[rune]*Node)}
	r.Walk(func(key string, value any) bool {
		cur := root
		for _, ch := range key {
			if cur.children[ch] == nil {
				cur.children[ch] = &Node{children: make(map[rune]*Node)}
			}
			cur = cur.children[ch]
		}
		cur.val = value
		return false
	})
	return root
}

// 递归生成字符串（逐字符分层）
func buildString(n *Node, prefix string, depth int, sb *strings.Builder) {
	indent := strings.Repeat("  ", depth)

	// 打印当前节点（只显示最后一个字符）
	if prefix != "" {
		ch := string(prefix[len(prefix)-1])
		if n.val != nil {
			sb.WriteString(fmt.Sprintf("%s%s : %v\n", indent, ch, n.val))
		} else {
			sb.WriteString(fmt.Sprintf("%s%s\n", indent, ch))
		}
	}

	// 子节点排序后打印
	keys := make([]rune, 0, len(n.children))
	for k := range n.children {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	for _, ch := range keys {
		buildString(n.children[ch], prefix+string(ch), depth+1, sb)
	}
}

// PrettyPrintRadix 返回 Radix 树的层级结构字符串
func PrettyPrintRadix(r *radix.Tree) string {
	root := buildTrie(r)
	var sb strings.Builder
	buildString(root, "", 0, &sb)
	return sb.String()
}

// PrettyPrintRadixCompressed 打印 radix 的压缩前缀结构：
// - 节点集合：分支点前缀（相邻键的最长公共前缀） ∪ 完整键
// - 父节点：在节点集合中，所有真前缀里选“最长”的作为父（键也可作为父）
// - 打印规则：分支点无值则只打印前缀；有值或完整键打印 "key : value"
func PrettyPrintRadixCompressed(r *radix.Tree) string {
	var sb strings.Builder

	// 收集并排序所有键和值
	var keys []string
	values := make(map[string]any)
	r.Walk(func(key string, value any) bool {
		keys = append(keys, key)
		values[key] = value
		return false
	})
	if len(keys) == 0 {
		return ""
	}
	slices.Sort(keys)

	// 计算分支点：相邻键的最长公共前缀
	branchSet := make(map[string]struct{})
	for i := 1; i < len(keys); i++ {
		lcp := longestCommonPrefix(keys[i-1], keys[i])
		if lcp != "" {
			branchSet[lcp] = struct{}{}
		}
	}

	// 节点集合（去重）
	nodeSet := make(map[string]struct{})
	for p := range branchSet {
		nodeSet[p] = struct{}{}
	}
	for _, k := range keys {
		nodeSet[k] = struct{}{}
	}

	// 列表并排序
	nodes := make([]string, 0, len(nodeSet))
	for n := range nodeSet {
		nodes = append(nodes, n)
	}
	slices.Sort(nodes)

	// 建立父子关系：父为“节点集合中、是该节点真前缀的最长候选”
	children := make(map[string][]string)
	hasParent := make(map[string]bool)

	isPrefix := func(p, s string) bool {
		rp, rs := []rune(p), []rune(s)
		if len(rp) >= len(rs) {
			return false
		}
		for i := range rp {
			if rp[i] != rs[i] {
				return false
			}
		}
		return true
	}

	for _, node := range nodes {
		var parent string
		// 选最长的真前缀作为父
		for _, cand := range nodes {
			if cand == node {
				continue
			}
			if isPrefix(cand, node) {
				if len([]rune(cand)) > len([]rune(parent)) {
					parent = cand
				}
			}
		}
		if parent != "" {
			children[parent] = append(children[parent], node)
			hasParent[node] = true
		}
	}

	// 顶层：没有父的节点
	roots := make([]string, 0)
	for _, n := range nodes {
		if !hasParent[n] {
			roots = append(roots, n)
		}
	}
	slices.Sort(roots)

	// 打印（分支点若有值，打印 "前缀 : 值"；否则仅前缀）
	var printNode func(node string, depth int)
	printNode = func(node string, depth int) {
		indent := strings.Repeat("  ", depth)
		if val, ok := values[node]; ok && val != nil {
			sb.WriteString(fmt.Sprintf("%s%s : %v\n", indent, node, val))
		} else {
			sb.WriteString(fmt.Sprintf("%s%s\n", indent, node))
		}
		if ch, ok := children[node]; ok {
			slices.Sort(ch)
			for _, c := range ch {
				printNode(c, depth+1)
			}
		}
	}

	for _, root := range roots {
		printNode(root, 0)
	}

	return sb.String()
}

// 按 rune 的最长公共前缀
func longestCommonPrefix(a, b string) string {
	ra, rb := []rune(a), []rune(b)
	n := 0
	for n < len(ra) && n < len(rb) && ra[n] == rb[n] {
		n++
	}
	return string(ra[:n])
}
