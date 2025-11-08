package heap_test

import (
	"fmt"
	"strings"
	"testing"

	"common_tool/pkg/heap"
)

func TestIntMinHeap(t *testing.T) {
	h := heap.New(func(a, b int) bool { return a < b }) // 小顶堆

	h.PushItem(5)
	h.PushItem(2)
	h.PushItem(8)
	h.PushItem(1)

	if h.Peek() != 1 {
		t.Errorf("Expected top to be 1, got %d", h.Peek())
	}

	val := h.PopItem()
	if val != 1 {
		t.Errorf("Expected pop to be 1, got %d", val)
	}

	val = h.PopItem()
	if val != 2 {
		t.Errorf("Expected pop to be 2, got %d", val)
	}
}

func TestIntMaxHeap(t *testing.T) {
	h := heap.New(func(a, b int) bool { return a > b }) // 大顶堆

	h.PushItem(5)
	h.PushItem(2)
	h.PushItem(8)
	h.PushItem(1)

	if h.Peek() != 8 {
		t.Errorf("Expected top to be 8, got %d", h.Peek())
	}

	val := h.PopItem()
	if val != 8 {
		t.Errorf("Expected pop to be 8, got %d", val)
	}

	val = h.PopItem()
	if val != 5 {
		t.Errorf("Expected pop to be 5, got %d", val)
	}
}

// 自定义结构体作为堆元素
type Node struct {
	Name string
	Val  int
}

// 简单的 format 函数示例：Name:Val
// 在测试中直接把这个字面量函数传入 PrintTree

func TestPrintString_EmptyHeap_WithStruct(t *testing.T) {
	// 使用 New 创建一个空堆（less 函数任意），避免直接使用未初始化的 nil 指针
	h := heap.New(func(a, b Node) bool { return a.Val < b.Val })
	out := h.PrintTree(func(n Node) string { return fmt.Sprintf("%s:%d", n.Name, n.Val) }, 0, 0)

	if !strings.Contains(out, "index_count=0") {
		t.Fatalf("empty struct-heap: expected index_count=0, got:\n%s", out)
	}
	t.Logf("Empty struct-heap output:\n%s", out)
}

func TestPrintString_SingleNode_WithStruct(t *testing.T) {
	h := heap.New(func(a, b Node) bool { return a.Val < b.Val })
	h.PushItem(Node{Name: "root", Val: 100})

	out := h.PrintTree(func(n Node) string { return fmt.Sprintf("%s:%d", n.Name, n.Val) }, 0, 0)

	if !strings.Contains(out, "index_count=1") {
		t.Fatalf("single struct node: expected index_count=1, got:\n%s", out)
	}
	if !strings.Contains(out, "(0)") {
		t.Fatalf("single struct node: expected root index (0) in output, got:\n%s", out)
	}
	t.Logf("Single struct node heap output:\n%s", out)
}

func TestPrintString_Full3Levels_Struct_ASCII(t *testing.T) {
	// 3 层：最多 7 个节点
	h := heap.New(func(a, b Node) bool { return a.Val < b.Val })
	nodes := []Node{
		{"n1", 10},
		{"n2", 20},
		{"n3", 30},
		{"n4", 40},
		{"n5", 50},
		{"n6", 60},
		{"n7", 70},
	}
	for _, n := range nodes {
		h.PushItem(n)
	}

	out := h.PrintTree(func(n Node) string { return fmt.Sprintf("%s:%d", n.Name, n.Val) }, 0, 0)

	if !strings.Contains(out, "index_count=7") {
		t.Fatalf("3-level struct heap: expected index_count=7, got:\n%s", out)
	}
	if !strings.Contains(out, "(0)") {
		t.Fatalf("3-level struct heap: expected root index (0) in output, got:\n%s", out)
	}
	t.Logf("3-level (7 nodes) struct ASCII output:\n%s", out)
}

func TestPrintString_Full4Levels_Struct_Unicode_Forward(t *testing.T) {
	// 4 层：最多 15 个节点
	h := heap.New(func(a, b Node) bool { return a.Val < b.Val })
	for i := 1; i <= 15; i++ {
		h.PushItem(Node{fmt.Sprintf("n%02d", i), i * 5})
	}

	out := h.PrintTree(func(n Node) string { return fmt.Sprintf("%s:%d", n.Name, n.Val) }, 1, 0)

	if !strings.Contains(out, "index_count=15") {
		t.Fatalf("4-level struct heap: expected index_count=15, got:\n%s", out)
	}
	if !strings.Contains(out, "(0)") {
		t.Fatalf("4-level struct heap: expected root index (0) in output, got:\n%s", out)
	}
	if !strings.Contains(out, "┌──>") && !strings.Contains(out, "└──>") && !strings.Contains(out, "│── ") {
		t.Fatalf("4-level struct heap: expected unicode drawing characters in output, got:\n%s", out)
	}
	t.Logf("4-level (15 nodes) struct Unicode forward output:\n%s", out)
}

// helper 格式化
func nodeFormat(n Node) string { return fmt.Sprintf("%s:%d", n.Name, n.Val) }

// 大顶堆比较函数
func maxHeapLess(a, b Node) bool { return a.Val > b.Val }

// 小顶堆比较函数
func minHeapLess(a, b Node) bool { return a.Val < b.Val }

func TestPrintString_Full3Levels_Struct_MaxHeap_Shuffled(t *testing.T) {
	// 3 层：最多 7 个节点
	h := heap.New(maxHeapLess)
	// 插入顺序故意打乱，值也不连续
	nodes := []Node{
		{"a", 42},
		{"b", 7},
		{"c", 85},
		{"d", 13},
		{"e", 29},
		{"f", 60},
		{"g", 2},
	}
	// 插入时顺序可以随机或固定为打乱的顺序，便于复现
	for _, n := range nodes {
		h.PushItem(n)
	}

	out := h.PrintTree(func(n Node) string { return nodeFormat(n) }, 0, 0) // ASCII, reverse-inorder

	if !strings.Contains(out, "index_count=7") {
		t.Fatalf("3-level max-heap: expected index_count=7, got:\n%s", out)
	}
	if !strings.Contains(out, "(0)") {
		t.Fatalf("3-level max-heap: expected root index (0) in output, got:\n%s", out)
	}
	// heap_array 中应包含至少几个我们插入的元素的字符串片段
	if !strings.Contains(out, "a:42") || !strings.Contains(out, "c:85") || !strings.Contains(out, "f:60") {
		t.Fatalf("3-level max-heap: expected heap_array to contain inserted items, got:\n%s", out)
	}
	t.Logf("3-level (7 nodes) max-heap ASCII output:\n%s", out)
}

func TestPrintString_Full4Levels_Struct_MaxHeap_Complex(t *testing.T) {
	// 4 层：最多 15 个节点
	h := heap.New(maxHeapLess)
	// 值与插入顺序更“真实”：非连续、含较大/较小值交错
	nodes := []Node{
		{"n01", 33}, {"n02", 7}, {"n03", 99}, {"n04", 18},
		{"n05", 65}, {"n06", 2}, {"n07", 77}, {"n08", 41},
		{"n09", 56}, {"n10", 13}, {"n11", 88}, {"n12", 5},
		{"n13", 61}, {"n14", 28}, {"n15", 90},
	}
	for _, n := range nodes {
		h.PushItem(n)
	}

	// Unicode 绘图，保留 direction=0（reverse-inorder）或者你想要的方向
	out := h.PrintTree(func(n Node) string { return nodeFormat(n) }, 1, 0)

	if !strings.Contains(out, "index_count=15") {
		t.Fatalf("4-level max-heap: expected index_count=15, got:\n%s", out)
	}
	if !strings.Contains(out, "(0)") {
		t.Fatalf("4-level max-heap: expected root index (0) in output, got:\n%s", out)
	}
	// 检查几项存在以确认 heap_array 打印
	if !strings.Contains(out, "n03:99") || !strings.Contains(out, "n15:90") || !strings.Contains(out, "n11:88") {
		t.Fatalf("4-level max-heap: expected heap_array to contain several inserted items, got:\n%s", out)
	}
	// 检查 Unicode 绘图字符存在
	if !strings.Contains(out, "┌──>") && !strings.Contains(out, "└──>") && !strings.Contains(out, "│── ") {
		t.Fatalf("4-level max-heap: expected unicode drawing characters in output, got:\n%s", out)
	}
	t.Logf("4-level (15 nodes) max-heap Unicode ASCII output:\n%s", out)
}

// 1) 5 个节点：层次为 3（indices 0..4），完全但不是满（满三层需 7 个节点）
func TestPrintString_CompleteNotFull_5Nodes(t *testing.T) {
	h := heap.New(maxHeapLess)
	nodes := []Node{
		{"A", 50},
		{"B", 20},
		{"C", 40},
		{"D", 10},
		{"E", 30},
	}
	for _, n := range nodes {
		h.PushItem(n)
	}

	out := h.PrintTree(nodeFormat, 0, 0) // ASCII, reverse-inorder

	if !strings.Contains(out, "index_count=5") {
		t.Fatalf("5-node heap: expected index_count=5, got:\n%s", out)
	}
	// heap_array 应包含所有插入项
	for _, n := range nodes {
		if !strings.Contains(out, fmt.Sprintf("%s:%d", n.Name, n.Val)) {
			t.Fatalf("5-node heap: expected heap_array to contain %s:%d, got:\n%s", n.Name, n.Val, out)
		}
	}
	t.Logf("5-node complete (not full) heap output:\n%s", out)
}

// 2) 8 个节点：层次为 4（indices 0..7），完全但不是满（满四层需 15 个节点）
func TestPrintString_CompleteNotFull_8Nodes(t *testing.T) {
	h := heap.New(maxHeapLess)
	nodes := []Node{
		{"n1", 75},
		{"n2", 12},
		{"n3", 60},
		{"n4", 5},
		{"n5", 30},
		{"n6", 55},
		{"n7", 20},
		{"n8", 45}, // 第四层只填了最左侧的一个位置，使其成为完全但非满
	}
	for _, n := range nodes {
		h.PushItem(n)
	}

	out := h.PrintTree(nodeFormat, 1, 0) // Unicode, reverse-inorder

	if !strings.Contains(out, "index_count=8") {
		t.Fatalf("8-node heap: expected index_count=8, got:\n%s", out)
	}
	// 检查 heap_array 并确保第四层已有最左节点（索引 7）
	if !strings.Contains(out, "heap_array=[") || !strings.Contains(out, "7:") {
		t.Fatalf("8-node heap: expected heap_array with index 7 present, got:\n%s", out)
	}
	t.Logf("8-node complete (not full) heap output:\n%s", out)
}

// 3) 10 个节点：层次为 4（indices 0..9），完全但非满（第四层部分填充）
func TestPrintString_CompleteNotFull_10Nodes(t *testing.T) {
	h := heap.New(minHeapLess)
	nodes := []Node{
		{"x1", 9},
		{"x2", 22},
		{"x3", 7},
		{"x4", 50},
		{"x5", 33},
		{"x6", 15},
		{"x7", 28},
		{"x8", 3},
		{"x9", 44},
		{"x10", 19}, // 填充到索引 9（包含部分第四层）
	}
	for _, n := range nodes {
		h.PushItem(n)
	}

	out := h.PrintTree(nodeFormat, 0, 1) // ASCII, forward-inorder

	if !strings.Contains(out, "index_count=10") {
		t.Fatalf("10-node heap: expected index_count=10, got:\n%s", out)
	}
	// 验证 heap_array 包含索引 9（最后一个插入位置）
	if !strings.Contains(out, "9:") {
		t.Fatalf("10-node heap: expected heap_array to contain index 9, got:\n%s", out)
	}
	t.Logf("10-node complete (not full) heap output:\n%s", out)
}
