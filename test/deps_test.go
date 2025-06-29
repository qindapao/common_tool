package deps_test

import (
	"bytes"
	"common_tool/internal/testutils"
	"common_tool/pkg/graph"
	"os"
	"os/exec"
	"testing"

	"github.com/awalterschulze/gographviz"
)

func TestNoImportCycles(t *testing.T) {
	wd, _ := os.Getwd()
	projectRoot, err := testutils.FindGoModRoot(wd)
	if err != nil {
		t.Fatalf("找不到 go.mod 根目录: %v", err)
	}

	t.Logf("projectRoot: %v", projectRoot)

	cmd := exec.Command("goda", "graph", "./...")
	cmd.Dir = projectRoot
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("执行 goda 失败: %v\n输出:\n%s", err, stdout.String())
	}

	graphAst, err := gographviz.Parse(stdout.Bytes())
	if err != nil {
		t.Fatalf("无法解析 DOT 输出: %v", err)
	}

	t.Logf("show graphAst: %s", graphAst.String())

	g := gographviz.NewGraph()
	if err := gographviz.Analyse(graphAst, g); err != nil {
		t.Fatalf("无法分析 DOT 图: %v", err)
	}

	// ➤ 将 gographviz.Graph 转为邻接表
	adj := graph.ToAdjacencyMap(g)

	t.Logf("show adj: %s", adj)

	// ➤ 调用 Johnson 算法查找所有 simple cycles
	rawCycles := graph.FindSimpleCycles(adj)

	// ➤ 去重 + 美化打印
	uniqueCycles := graph.DeduplicateCycles(rawCycles)

	if len(uniqueCycles) > 0 {
		for _, c := range uniqueCycles {
			t.Logf("\U0001f300 循环依赖路径: %s", graph.FormatCycle(c))
		}
		t.Fatalf("共检测到 %d 个循环依赖", len(uniqueCycles))
	}
}