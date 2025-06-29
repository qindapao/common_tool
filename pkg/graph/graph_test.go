package graph_test

import (
	"common_tool/pkg/graph"
	"testing"

	"github.com/awalterschulze/gographviz"
)

func newTestGraph() *gographviz.Graph {
	g := gographviz.NewGraph()
	g.SetName("G")
	g.SetDir(true)
	return g
}

func TestHasCycleDFS(t *testing.T) {
	tests := []struct {
		name      string
		edges     [][2]string
		wantCycle bool
		wantStart string
	}{
		{
			name: "no cycle",
			edges: [][2]string{
				{"A", "B"},
				{"B", "C"},
			},
			wantCycle: false,
		},
		{
			name: "simple cycle A→B→C→A",
			edges: [][2]string{
				{"A", "B"},
				{"B", "C"},
				{"C", "A"},
			},
			wantCycle: true,
			wantStart: "A",
		},
		{
			name: "self loop",
			edges: [][2]string{
				{"X", "X"},
			},
			wantCycle: true,
			wantStart: "X",
		},
		{
			name:      "empty graph",
			edges:     nil,
			wantCycle: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newTestGraph()
			nodeSet := make(map[string]bool)
			for _, e := range tt.edges {
				if !nodeSet[e[0]] {
					g.AddNode("G", e[0], nil)
					nodeSet[e[0]] = true
				}
				if !nodeSet[e[1]] {
					g.AddNode("G", e[1], nil)
					nodeSet[e[1]] = true
				}
				g.AddEdge(e[0], e[1], true, nil)
			}

			has, entry := graph.HasCycleDFS(g)
			if has != tt.wantCycle {
				t.Errorf("HasCycleDFS = %v, want %v", has, tt.wantCycle)
			}
			if tt.wantCycle && entry != tt.wantStart {
				t.Errorf("Cycle entry = %v, want %v", entry, tt.wantStart)
			}
		})
	}
}

func TestFindAllCycles(t *testing.T) {
	g := newTestGraph()
	// 添加 A→B→C→A 和 D→E→D 两个环
	for _, n := range []string{"A", "B", "C", "D", "E"} {
		g.AddNode("G", n, nil)
	}
	g.AddEdge("A", "B", true, nil)
	g.AddEdge("B", "C", true, nil)
	g.AddEdge("C", "A", true, nil)

	g.AddEdge("D", "E", true, nil)
	g.AddEdge("E", "D", true, nil)

	cycles := graph.FindAllCycles(g)

	if len(cycles) != 2 {
		t.Fatalf("应找到 2 个环，实际找到 %d 个", len(cycles))
	}
	for _, c := range cycles {
		t.Logf("检测到环: %s", graph.FormatCycle(c))
	}
}

func TestFindSimpleCycles_Complex(t *testing.T) {
	// 构造简单邻接表表示图结构（适配 FindSimpleCycles 函数）
	g := map[string][]string{
		// 环 A→B→C→A
		"A": {"B"},
		"B": {"C"},
		"C": {"A"},

		// 环 X→Y→Z→X
		"X": {"Y"},
		"Y": {"Z"},
		"Z": {"X"},

		// 重叠环 M→N→O→M 和 M→P→Q→N→M
		"M": {"N", "P"},
		"N": {"O"},
		"O": {"M"},
		"P": {"Q"},
		"Q": {"N"},

		// 噪声路径（不成环）
		"U": {"V"},

		// 自环
		"S": {"S"},
		"T": {"T"},
	}

	cycles := graph.FindSimpleCycles(g)
	t.Logf("共检测到 %d 个环:", len(cycles))
	for _, c := range cycles {
		t.Logf("\U0001f300 %s", graph.FormatCycle(c))
	}

	deduped := graph.DeduplicateCycles(cycles)
	t.Logf("\U0001f3af 去重后仅保留 %d 个语义唯一环:", len(deduped))
	for _, c := range deduped {
		t.Logf("\U0001f300 %s", graph.FormatCycle(c))
	}

	// 期望环数：6（A-B-C-A, X-Y-Z-X, M-N-O-M, M-P-Q-N-M, S→S, T→T）
	expectedMinCycles := 6
	if len(cycles) < expectedMinCycles {
		t.Errorf("应至少检测到 %d 个环，实际为 %d", expectedMinCycles, len(cycles))
	}
}