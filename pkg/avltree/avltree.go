package avltree

import (
	"common_tool/pkg/treeprinter"
	"fmt"
)

type AVLTree[T any] struct {
	root *node[T]
	less func(a, b T) bool
}

type node[T any] struct {
	value  T
	height int
	left   *node[T]
	right  *node[T]
}

func NewAVLTree[T any](less func(a, b T) bool) *AVLTree[T] {
	return &AVLTree[T]{less: less}
}

func (t *AVLTree[T]) Insert(value T) {
	t.root = insert(t.root, value, t.less)
}

func insert[T any](n *node[T], value T, less func(a, b T) bool) *node[T] {
	if n == nil {
		return &node[T]{value: value, height: 1}
	}

	if less(value, n.value) {
		n.left = insert(n.left, value, less)
	} else if less(n.value, value) {
		n.right = insert(n.right, value, less)
	} else {
		// Duplicate value, do nothing
		return n
	}

	updateHeight(n)
	return balance(n)
}

func height[T any](n *node[T]) int {
	if n == nil {
		return 0
	}
	return n.height
}

func updateHeight[T any](n *node[T]) {
	n.height = max(height(n.left), height(n.right)) + 1
}

func balanceFactor[T any](n *node[T]) int {
	return height(n.left) - height(n.right)
}

func balance[T any](n *node[T]) *node[T] {
	bf := balanceFactor(n)

	if bf > 1 {
		if balanceFactor(n.left) < 0 {
			n.left = rotateLeft(n.left)
		}
		return rotateRight(n)
	}
	if bf < -1 {
		if balanceFactor(n.right) > 0 {
			n.right = rotateRight(n.right)
		}
		return rotateLeft(n)
	}
	return n
}

func rotateLeft[T any](z *node[T]) *node[T] {
	if z == nil || z.right == nil {
		return z
	}
	y := z.right
	z.right = y.left
	y.left = z
	updateHeight(z)
	updateHeight(y)
	return y
}

func rotateRight[T any](z *node[T]) *node[T] {
	if z == nil || z.left == nil {
		return z
	}
	y := z.left
	z.left = y.right
	y.right = z
	updateHeight(z)
	updateHeight(y)
	return y
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (t *AVLTree[T]) PrintTree(format func(T) string, style int, direction int) string {
	return treeprinter.PrintTreeGeneric(treeprinter.TreePrinter[*node[T]]{
		Root: t.root,
		GetChild: func(n *node[T], dir string) *node[T] {
			if dir == "left" {
				return n.left
			}
			return n.right
		},
		GetValue: func(n *node[T]) string {
			// return fmt.Sprintf("%v(h=%d,bf=%d)", n.value, n.height, height(n.left)-height(n.right))
			return fmt.Sprintf("%v(h=%d)", n.value, n.height)
		},
		IsNil: func(n *node[T]) bool {
			return n == nil
		},
		Style:     style,
		Direction: direction,
	})
}

func (t *AVLTree[T]) Search(value T) bool {
	cur := t.root
	for cur != nil {
		if t.less(value, cur.value) {
			cur = cur.left
		} else if t.less(cur.value, value) {
			cur = cur.right
		} else {
			return true
		}
	}
	return false
}

func (t *AVLTree[T]) Delete(value T) {
	t.root = deleteNode(t.root, value, t.less)
}

func deleteNode[T any](n *node[T], value T, less func(a, b T) bool) *node[T] {
	if n == nil {
		return nil
	}

	if less(value, n.value) {
		n.left = deleteNode(n.left, value, less)
	} else if less(n.value, value) {
		n.right = deleteNode(n.right, value, less)
	} else {
		// Found the node to delete
		if n.left == nil && n.right == nil {
			return nil
		} else if n.left == nil {
			return n.right
		} else if n.right == nil {
			return n.left
		} else {
			// Two children: find in-order successor (min in right subtree)
			successor := findMin(n.right)
			n.value = successor.value
			n.right = deleteNode(n.right, successor.value, less)
		}
	}

	updateHeight(n)
	return balance(n)
}

func findMin[T any](n *node[T]) *node[T] {
	for n.left != nil {
		n = n.left
	}
	return n
}
