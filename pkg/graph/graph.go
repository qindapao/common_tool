package graph

import (
	"sort"
	"strings"

	"github.com/awalterschulze/gographviz"
)

// HasCycleDFS 遍历有向图并判断是否存在至少一个环路（即图中存在某个节点能沿边最终回到自己）
// 它使用深度优先搜索（DFS）结合递归栈，检测是否存在回到当前递归路径中的“祖先节点”
// 若发现环，将返回 true，并返回首次形成闭环的起点节点名称
func HasCycleDFS(graph *gographviz.Graph) (bool, string) {
	visited := make(map[string]bool)  // 标记是否访问过节点，避免重复访问
	recStack := make(map[string]bool) // 当前 DFS 路径中的节点（递归栈），用于判断回到祖先

	var dfs func(string) bool
	dfs = func(node string) bool {
		// 当前节点已在栈中，说明出现了回到祖先的“环”
		if recStack[node] {
			return true
		}
		// 已经访问过但不在栈中，说明这条路径处理过了，没有环
		if visited[node] {
			return false
		}
		visited[node] = true  // 标记当前节点已访问
		recStack[node] = true // 加入递归路径栈

		// 遍历当前节点出边
		for _, dst := range graph.Edges.SrcToDsts[node] {
			for _, edge := range dst {
				// 如果子节点递归中发现环，直接返回 true
				if dfs(edge.Dst) {
					return true
				}
			}
		}
		// 当前节点 DFS 完成，移出栈（不在当前路径）
		recStack[node] = false
		return false
	}

	// 遍历所有节点作为起点（图不一定是连通图）
	for _, node := range graph.Nodes.Nodes {
		if dfs(node.Name) {
			return true, node.Name // 找到环，返回结果和起点
		}
	}
	return false, "" // 无环
}

// FindAllCycles 遍历有向图中的所有节点，寻找从任意节点出发、沿边最终回到自身形成的所有闭环路径
// 它采用递归 DFS 算法并维护一个 path 栈记录当前路径、一个 onStack 状态标记递归路径上的节点
// 一旦在递归路径中再次访问某节点，说明存在环，立即回溯构建完整的路径片段形成一个环
// 所有满足条件的环将以 string 切片形式返回，其中每个环都以“路径形式”存储
func FindAllCycles(g *gographviz.Graph) [][]string {
	var (
		allCycles [][]string // 所有找到的环
		path      []string   // 当前 DFS 路径
		visited   = make(map[string]bool)
		onStack   = make(map[string]bool)
	)

	var dfs func(string)
	dfs = func(node string) {
		visited[node] = true      // 标记已访问
		onStack[node] = true      // 当前节点加入栈
		path = append(path, node) // 记录路径

		for _, dstGroup := range g.Edges.SrcToDsts[node] {
			for _, edge := range dstGroup {
				target := edge.Dst
				if !visited[target] {
					dfs(target)
				} else if onStack[target] {
					// 出现环（路径中再次访问到 onStack 节点）
					cycle := append([]string{}, target)
					// 从 path 中回溯拼出完整路径
					for i := len(path) - 1; i >= 0; i-- {
						cycle = append([]string{path[i]}, cycle...)
						if path[i] == target && i != len(path)-1 {
							break
						}
					}
					allCycles = append(allCycles, cycle)
				}
			}
		}
		onStack[node] = false     // 离开栈
		path = path[:len(path)-1] // 路径弹出
	}

	for _, node := range g.Nodes.Nodes {
		name := node.Name
		if !visited[name] {
			dfs(name)
		}
	}
	return allCycles
}

// FormatCycle 将一段环路径（如 [A, B, C, A]）格式化为可读字符串
// 输出样式如 "A → B → C → A"，方便在日志或测试输出中查看结构
func FormatCycle(path []string) string {
	return strings.Join(path, " → ")
}

// FindSimpleCycles 使用 Johnson 算法在有向图中枚举所有“简单环”
// 所谓简单环是指：路径首尾相同，且中间节点不重复（无交叉重复）
// 它通过对图中所有节点依次作为起点进行搜索，每次只处理从该点开始的子图
// 使用 blocked/unblock 机制避免重复路径搜索，以及 stack 记录当前路径轨迹
// 所有找到的简单环路径将被以 [][]string 形式返回
// TODO(johnson-cycle): 当前使用递归 + blocked 回溯策略实现 Johnson 算法
// 后续可考虑改写为显式栈或迭代版本，降低栈溢出风险 & 可调试性更强
// 参考：Tarjan + Johnson 非递归变种实现
func FindSimpleCycles(graph map[string][]string) [][]string {
	var (
		blocked  = map[string]bool{}            // 节点是否被当前搜索路径“屏蔽”掉（暂时不可重复访问）
		blockMap = map[string]map[string]bool{} // 每个节点对应的“屏蔽者”集合（哪些点导致它被 block）
		stack    []string                       // 当前正在访问的路径栈
		cycles   [][]string                     // 存储所有检测到的环
	)

	// unblock 操作：取消该点的屏蔽状态，同时解除所有依赖它的节点的屏蔽
	var unblock func(string)
	unblock = func(u string) {
		blocked[u] = false
		for w := range blockMap[u] {
			if blocked[w] {
				unblock(w)
			}
		}
		delete(blockMap, u)
	}

	// 核心 DFS 递归逻辑
	var circuit func(string, string) bool
	circuit = func(v, s string) bool {
		foundCycle := false
		stack = append(stack, v)
		blocked[v] = true

		for _, w := range graph[v] {
			if w == s {
				// 从起点走一圈回来了，构造环
				cycle := append([]string{}, stack...)
				cycles = append(cycles, append(cycle, s))
				foundCycle = true
			} else if !blocked[w] {
				if circuit(w, s) {
					foundCycle = true
				}
			}
		}

		if foundCycle {
			unblock(v) // 递归路径中有成功找到环，解除 block
		} else {
			// 没找到路径，则记录：我试图访问了哪些后继节点失败了
			for _, w := range graph[v] {
				if blockMap[w] == nil {
					blockMap[w] = make(map[string]bool)
				}
				blockMap[w][v] = true
			}
		}

		stack = stack[:len(stack)-1] // 当前点出栈
		return foundCycle
	}

	// 所有节点按字典序逐个尝试作为起点 s
	nodes := []string{}
	for v := range graph {
		nodes = append(nodes, v)
	}
	sort.Strings(nodes)

	for i := 0; i < len(nodes); i++ {
		s := nodes[i]

		// 构建从 s 开始的子图
		subgraph := subGraphFrom(graph, s)
		if len(subgraph) == 0 {
			continue
		}

		// 重置状态
		for k := range subgraph {
			blocked[k] = false
			blockMap[k] = nil
		}
		stack = nil

		_ = circuit(s, s)
	}

	return cycles
}

// subGraphFrom 提取从给定起始节点 start（按字典序）开始的子图（用于 Johnson 算法）
// 从原图 g 中只保留起点和终点都大于等于 start 的边
// 这样做的目的是：分阶段查环，避免重复枚举同一个环在不同起点下的排列
func subGraphFrom(g map[string][]string, start string) map[string][]string {
	sub := map[string][]string{}
	for from, tos := range g {
		// 只考虑 from 节点不小于 start 的部分	
		if from >= start {
			for _, to := range tos {
				// 只添加终点也不小于 start 的边
				if to >= start {
					sub[from] = append(sub[from], to)
				}
			}
		}
	}
	return sub
}

// NormalizeCycle 对一个环路径进行归一化处理
// 目标是将环的多种等价表示（起点不同但结构相同）统一为相同的标识字符串
// 例如 [B C A B]、[C A B C] 都归一为 "A→B→C"
// 步骤：
//  1. 如果路径以闭环形式存在（首尾相同），去掉最后一个重复节点
//  2. 通过旋转所有可能起点，找出字典序最小的那一个作为标准排列
//  3. 使用 → 连接作为唯一 key（便于去重）
func NormalizeCycle(cycle []string) string {
	n := len(cycle)
	if n == 0 {
		return ""
	}

	// 去掉尾部自闭合起点（避免 A→B→C→A→ 再次封闭）
	if cycle[0] == cycle[len(cycle)-1] {
		cycle = cycle[:len(cycle)-1]
	}

	// 初始化最小排列为原始顺序
	min := make([]string, len(cycle))
	copy(min, cycle)

	// 尝试所有可能的旋转排列，找出字典序最小的那个
	for i := 1; i < len(cycle); i++ {
		rotated := append(cycle[i:], cycle[:i]...)
		if lessThan(rotated, min) {
			copy(min, rotated)
		}
	}

	return strings.Join(min, "→")
}

// lessThan 用于比较两个字符串切片的字典序
// 返回 true 表示 a 在字典序上比 b 更小
// 用于 NormalizeCycle 中寻找最小旋转版本
func lessThan(a, b []string) bool {
	for i := 0; i < len(a); i++ {
		if a[i] < b[i] {
			return true
		} else if a[i] > b[i] {
			return false
		}
	}
	// 所有字符都相等
	return false
}

// ToAdjacencyMap 将 gographviz.Graph 图结构转换为邻接表形式
// 输入图 g 通常来自 goda 生成的 DOT 格式图
// 转换后得到 map[节点]出边列表，可直接用于算法分析（如 Johnson 算法）
func ToAdjacencyMap(g *gographviz.Graph) map[string][]string {
	adj := make(map[string][]string)
	for src, dstGroup := range g.Edges.SrcToDsts {
		for _, edges := range dstGroup {
			for _, edge := range edges {
				adj[src] = append(adj[src], edge.Dst)
			}
		}
	}
	return adj
}

// DeduplicateCycles 去除语义上重复的环路径，只保留结构唯一的那些
// 同一环（如 A→B→C→A）可能以不同起点方式出现（B→C→A→B），本质相同
// 使用 NormalizeCycle 将路径归一化为标准 key，再借助 map 去重
// 最终返回的结果是环路径去重后的子集
func DeduplicateCycles(cycles [][]string) [][]string {
	if len(cycles) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	var result [][]string

	for _, cycle := range cycles {
		if len(cycle) == 0 {
			continue // 忽略空环
		}
		key := NormalizeCycle(cycle)
		if !seen[key] {
			seen[key] = true
			result = append(result, cycle)
		}
	}
	return result
}