package heap

import (
	"common_tool/pkg/treeprinter"
	"container/heap"
	"fmt"
	"strings"
)

const (
	nodeVisitFirst        = iota // 0
	nodeVisitReadyToPrint        // 1
	nodeVisitDone                // 2
)

const (
	branchUpper = 1
	branchLower = 0
	branchRoot  = -1
)

type GenericHeap[T any] struct {
	data []T
	less func(a, b T) bool
}

func New[T any](less func(a, b T) bool) *GenericHeap[T] {
	h := &GenericHeap[T]{less: less}
	heap.Init(h)
	return h
}

func (h GenericHeap[T]) Len() int           { return len(h.data) }
func (h GenericHeap[T]) Less(i, j int) bool { return h.less(h.data[i], h.data[j]) }
func (h GenericHeap[T]) Swap(i, j int)      { h.data[i], h.data[j] = h.data[j], h.data[i] }

func (h *GenericHeap[T]) Push(x any) {
	h.data = append(h.data, x.(T))
}

func (h *GenericHeap[T]) Pop() any {
	n := len(h.data)
	x := h.data[n-1]
	h.data = h.data[:n-1]
	return x
}

func (h *GenericHeap[T]) PushItem(x T) {
	heap.Push(h, x)
}

func (h *GenericHeap[T]) PopItem() T {
	return heap.Pop(h).(T)
}

func (h *GenericHeap[T]) Peek() T {
	return h.data[0]
}

func (h *GenericHeap[T]) PrintTree(format func(T) string, style int, direction int) string {
	if h == nil || len(h.data) == 0 {
		return "index_count=0\nheap_array=[]\n"
	}

	treeStr := treeprinter.PrintTreeGeneric(treeprinter.TreePrinter[int]{
		Root: 0,
		GetChild: func(i int, dir string) int {
			if dir == "left" {
				return 2*i + 1
			}
			return 2*i + 2
		},
		GetValue: func(i int) string {
			return fmt.Sprintf("%s(%d)", format(h.data[i]), i)
		},
		IsNil: func(i int) bool {
			return i >= len(h.data)
		},
		Style:     style,
		Direction: direction,
	})

	// 追加定制化信息
	var b strings.Builder
	b.WriteString(treeStr)
	fmt.Fprintf(&b, "index_count=%d\n", len(h.data))
	b.WriteString("heap_array=[")
	for i, v := range h.data {
		if i > 0 {
			b.WriteString(" ")
		}
		fmt.Fprintf(&b, "[%d]=%s", i, format(v))
	}
	b.WriteString("]\n")

	return b.String()
}

