package btree

import (
	"common_tool/pkg/treeprinter"
	"fmt"
	"strings"
	"testing"

	"github.com/google/btree"
)

// 定义一个简单的整型包装
type IntWrapper struct {
	Key int
}

func (a IntWrapper) Less(b btree.Item) bool {
	return a.Key < b.(IntWrapper).Key
}

// 定义一个键值对包装
type KVWrapper struct {
	Key   int
	Value string
}

func (a KVWrapper) Less(b btree.Item) bool {
	return a.Key < b.(KVWrapper).Key
}
func TestPrettyPrintBTree_FourLevels_IntWrapper(t *testing.T) {
	tr := btree.New(3)
	for _, v := range []int{5, 10, 15, 20, 25, 30, 35, 40,
		45, 50, 55, 60, 65, 70, 75, 80,
		85, 90, 95, 100} {
		tr.ReplaceOrInsert(IntWrapper{Key: v})
	}

	output := PrettyPrintBTree(tr, 3,
		func(x IntWrapper) int { return x.Key },
		func(n *treeprinter.MultiNode) string {
			if items, ok := n.Data.([]IntWrapper); ok {
				parts := []string{}
				for _, it := range items {
					parts = append(parts, fmt.Sprintf("%d", it.Key))
				}
				return "[" + strings.Join(parts, ", ") + "]"
			}
			return fmt.Sprintf("%v", n.Data)
		},
	)
	t.Log("\n" + output)
}

func TestPrettyPrintBTree_FiveLevels_IntWrapper(t *testing.T) {
	tr := btree.New(3)
	for v := 1; v <= 50; v++ {
		tr.ReplaceOrInsert(IntWrapper{Key: v})
	}

	output := PrettyPrintBTree(tr, 3,
		func(x IntWrapper) int { return x.Key },
		func(n *treeprinter.MultiNode) string {
			if items, ok := n.Data.([]IntWrapper); ok {
				parts := []string{}
				for _, it := range items {
					parts = append(parts, fmt.Sprintf("%d", it.Key))
				}
				return "[" + strings.Join(parts, ", ") + "]"
			}
			return fmt.Sprintf("%v", n.Data)
		},
	)
	t.Log("\n" + output)
}

func TestPrettyPrintBTree_KVWrapper(t *testing.T) {
	tr := btree.New(3)
	for _, kv := range []KVWrapper{
		{Key: 10, Value: "A"},
		{Key: 20, Value: "B"},
		{Key: 30, Value: "C"},
		{Key: 40, Value: "D"},
		{Key: 50, Value: "E"},
		{Key: 60, Value: "F"},
		{Key: 70, Value: "G"},
		{Key: 80, Value: "H"},
	} {
		tr.ReplaceOrInsert(kv)
	}

	output := PrettyPrintBTree(tr, 3,
		func(x KVWrapper) int { return x.Key },
		func(n *treeprinter.MultiNode) string {
			if items, ok := n.Data.([]KVWrapper); ok {
				parts := []string{}
				for _, kv := range items {
					parts = append(parts, fmt.Sprintf("%d:%s", kv.Key, kv.Value))
				}
				return "[" + strings.Join(parts, ", ") + "]"
			}
			return fmt.Sprintf("%v", n.Data)
		},
	)
	t.Log("\n" + output)
}
