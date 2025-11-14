package unionfind

// UnionFind 是并查集结构，支持路径压缩和按秩合并
type UnionFind struct {
	parent []int
	rank   []int
	size   []int // 每个集合的大小，可选
}

// NewUnionFind 初始化并查集，元素范围为 [0, n)
func NewUnionFind(n int) *UnionFind {
	parent := make([]int, n)
	rank := make([]int, n)
	size := make([]int, n)
	for i := range parent {
		parent[i] = i
		size[i] = 1
	}
	return &UnionFind{parent: parent, rank: rank, size: size}
}

// Find 查找元素所在集合的根节点（带路径压缩）
func (uf *UnionFind) Find(x int) int {
	if uf.parent[x] != x {
		uf.parent[x] = uf.Find(uf.parent[x])
	}
	return uf.parent[x]
}

// Union 合并两个集合（按秩优化）
func (uf *UnionFind) Union(x, y int) bool {
	rootX := uf.Find(x)
	rootY := uf.Find(y)
	if rootX == rootY {
		return false // 已经在同一个集合
	}

	if uf.rank[rootX] < uf.rank[rootY] {
		uf.parent[rootX] = rootY
		uf.size[rootY] += uf.size[rootX]
	} else if uf.rank[rootX] > uf.rank[rootY] {
		uf.parent[rootY] = rootX
		uf.size[rootX] += uf.size[rootY]
	} else {
		uf.parent[rootY] = rootX
		uf.rank[rootX]++
		uf.size[rootX] += uf.size[rootY]
	}
	return true
}

// Connected 判断两个元素是否在同一个集合
func (uf *UnionFind) Connected(x, y int) bool {
	return uf.Find(x) == uf.Find(y)
}

// Size 返回某个集合的大小
func (uf *UnionFind) Size(x int) int {
	return uf.size[uf.Find(x)]
}
