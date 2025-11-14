package unionfind

import "testing"

func TestUnionFind(t *testing.T) {
	uf := NewUnionFind(10)

	// 初始状态：每个元素独立
	if uf.Connected(1, 2) {
		t.Errorf("Expected 1 and 2 not connected")
	}

	// 合并 1 和 2
	uf.Union(1, 2)
	if !uf.Connected(1, 2) {
		t.Errorf("Expected 1 and 2 connected")
	}

	// 合并 2 和 3
	uf.Union(2, 3)
	if !uf.Connected(1, 3) {
		t.Errorf("Expected 1 and 3 connected")
	}

	// 检查集合大小
	if uf.Size(1) != 3 {
		t.Errorf("Expected size of set containing 1 to be 3, got %d", uf.Size(1))
	}

	// 合并不同集合
	uf.Union(4, 5)
	if !uf.Connected(4, 5) {
		t.Errorf("Expected 4 and 5 connected")
	}

	// 检查未合并的元素
	if uf.Connected(1, 4) {
		t.Errorf("Expected 1 and 4 not connected")
	}
}
