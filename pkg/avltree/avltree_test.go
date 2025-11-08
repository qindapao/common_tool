package avltree_test

import (
	"fmt"
	"testing"
	"time"

	"common_tool/pkg/avltree" // 替换为你的模块路径
	"common_tool/pkg/diffutil"
)

func intLess(a, b int) bool {
	return a < b
}

func formatInt(v int) string {
	return fmt.Sprintf("%d", v)
}

func printTreeInt(t *avltree.AVLTree[int]) {
	fmt.Println(t.PrintTree(formatInt, 1, 0))
}

func TestInsertBasic(t *testing.T) {
	tree := avltree.NewAVLTree(intLess)
	values := []int{10, 20, 5, 4, 6, 15, 30}
	for _, v := range values {
		tree.Insert(v)
	}
	fmt.Println("* TestInsertBasic:")
	printTreeInt(tree)
}

func TestLeftRotation(t *testing.T) {
	tree := avltree.NewAVLTree(intLess)
	values := []int{10, 20, 30} // 插入顺序会触发左旋
	for _, v := range values {
		tree.Insert(v)
	}
	fmt.Println("* TestLeftRotation:")
	printTreeInt(tree)
}

func TestRightRotation(t *testing.T) {
	tree := avltree.NewAVLTree(intLess)
	values := []int{30, 20, 10} // 插入顺序会触发右旋
	for _, v := range values {
		tree.Insert(v)
	}
	fmt.Println("* TestRightRotation:")
	printTreeInt(tree)
}

func TestLeftRightRotation(t *testing.T) {
	tree := avltree.NewAVLTree(intLess)
	values := []int{30, 10, 20} // 插入顺序会触发左-右旋
	for _, v := range values {
		tree.Insert(v)
	}
	fmt.Println("* TestLeftRightRotation:")
	printTreeInt(tree)
}

func TestRightLeftRotation(t *testing.T) {
	tree := avltree.NewAVLTree(intLess)
	values := []int{10, 30, 20} // 插入顺序会触发右-左旋
	for _, v := range values {
		tree.Insert(v)
	}
	fmt.Println("* TestRightLeftRotation:")
	printTreeInt(tree)
}

func TestDeleteLeaf(t *testing.T) {
	tree := avltree.NewAVLTree(intLess)
	values := []int{20, 10, 30}
	for _, v := range values {
		tree.Insert(v)
	}
	tree.Delete(10) // 删除叶子节点
	fmt.Println("* TestDeleteLeaf:")
	printTreeInt(tree)
}

func TestDeleteNodeWithOneChild(t *testing.T) {
	tree := avltree.NewAVLTree(intLess)
	values := []int{20, 10, 30, 25}
	for _, v := range values {
		tree.Insert(v)
	}
	tree.Delete(30) // 删除有一个左子节点的节点
	fmt.Println("* TestDeleteNodeWithOneChild:")
	printTreeInt(tree)
}

func TestDeleteNodeWithTwoChildren(t *testing.T) {
	tree := avltree.NewAVLTree(intLess)
	values := []int{20, 10, 30, 25, 35}
	for _, v := range values {
		tree.Insert(v)
	}
	tree.Delete(30) // 删除有两个子节点的节点
	fmt.Println("* TestDeleteNodeWithTwoChildren:")
	printTreeInt(tree)
}

func TestSearch(t *testing.T) {
	tree := avltree.NewAVLTree(intLess)
	values := []int{10, 20, 5}
	for _, v := range values {
		tree.Insert(v)
	}
	if !tree.Search(10) {
		t.Error("Expected to find 10")
	}
	if tree.Search(100) {
		t.Error("Did not expect to find 100")
	}
	fmt.Println("* TestSearch:")
	printTreeInt(tree)
}

func TestDuplicateInsert(t *testing.T) {
	tree := avltree.NewAVLTree(intLess)
	tree.Insert(10)
	tree.Insert(10) // 重复插入
	fmt.Println("* TestDuplicateInsert:")
	printTreeInt(tree)
}

func TestComplexTree(t *testing.T) {
	tree := avltree.NewAVLTree(intLess)
	values := []int{
		50, 20, 70, 10, 30, 60, 80,
		5, 15, 25, 35, 55, 65, 75, 85,
		1, 6, 14, 16, 24, 26, 34, 36,
		54, 56, 64, 66, 74, 76, 84, 86,
	}
	for _, v := range values {
		tree.Insert(v)
	}
	fmt.Println("* TestComplexTree:")
	printTreeInt(tree)
}

func TestDeleteStorm(t *testing.T) {
	tree := avltree.NewAVLTree(intLess)
	values := []int{
		50, 20, 70, 10, 30, 60, 80,
		5, 15, 25, 35, 55, 65, 75, 85,
		1, 6, 14, 16, 24, 26, 34, 36,
		54, 56, 64, 66, 74, 76, 84, 86,
	}
	for _, v := range values {
		tree.Insert(v)
	}
	fmt.Println("$ 初始树结构:")
	printTreeInt(tree)

	// 删除风暴：每次删除一个节点并打印
	toDelete := []int{
		1, 6, 14, 16, 24, 26, 34, 36, // 删除叶子节点
		25, 35, 55, 65, // 删除单子树节点
		20, 30, 60, 70, // 删除双子树节点
		50, // 删除根节点
	}

	for i, v := range toDelete {
		tree.Delete(v)
		fmt.Printf("X 删除第 %d 个节点: %d\n", i+1, v)
		printTreeInt(tree)
	}
}

func TestDeleteRootCascade(t *testing.T) {
	tree := avltree.NewAVLTree(intLess)
	values := []int{
		50, 20, 70, 10, 30, 60, 80,
		5, 15, 25, 35, 55, 65, 75, 85,
		1, 6, 14, 16, 24, 26, 34, 36,
		54, 56, 64, 66, 74, 76, 84, 86,
	}
	for _, v := range values {
		tree.Insert(v)
	}
	fmt.Println("$ 初始树结构:")
	printTreeInt(tree)

	// 删除根节点（高度最高）
	fmt.Println("# 删除根节点 50（高度最高）")
	tree.Delete(50)
	printTreeInt(tree)

	// 连续删除几个高节点
	highNodes := []int{70, 30, 60, 20}
	for i, v := range highNodes {
		fmt.Printf("# 删除高节点 %d: %d\n", i+1, v)
		tree.Delete(v)
		printTreeInt(tree)
	}
}

func clearScreen() {
	fmt.Print("\033[H\033[2J") // ANSI 清屏指令
}
func pause() {
	time.Sleep(500 * time.Millisecond) // 每步停顿 0.5 秒
}
func TestAnimatedDeleteStormWithDiff(t *testing.T) {
	tree := avltree.NewAVLTree(intLess)
	values := []int{
		50, 20, 70, 10, 30, 60, 80,
		5, 15, 25, 35, 55, 65, 75, 85,
		1, 6, 14, 16, 24, 26, 34, 36,
		54, 56, 64, 66, 74, 76, 84, 86,
	}
	for _, v := range values {
		tree.Insert(v)
	}
	clearScreen()
	fmt.Println("$ 初始树结构:")
	printTreeInt(tree)
	pause()

	toDelete := []int{50, 70, 30, 60, 20, 25, 55, 65}
	for i, v := range toDelete {
		before := tree.PrintTree(formatInt, 1, 0)
		tree.Delete(v)
		after := tree.PrintTree(formatInt, 1, 0)

		clearScreen()
		fmt.Printf("X 删除第 %d 个节点: %d\n", i+1, v)
		diff := diffutil.CompareMultiline(before, after)
		fmt.Println(diffutil.FormatSideBySide(diff))
		pause()
	}
}
