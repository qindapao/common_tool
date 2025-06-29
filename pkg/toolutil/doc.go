//  1. 基本流构建 & Map
//     s := StreamOf([]int{1, 2, 3, 4})
//     squared := Map(s, func(x int) int { return x * x }).ToSlice()
//     // [1 4 9 16]
//
//  2. 连续 Filter + Map + Reduce
//     sum := Reduce(
//     Map(
//     Filter(StreamOf([]int{1, 2, 3, 4, 5}), func(n int) bool { return n%2 == 1 }),
//     func(x int) int { return x * 10 },
//     ),
//     0,
//     func(acc, x int) int { return acc + x },
//     )
//     // 输出: 90 (10+30+50)
//
//  3. Sorted + Distinct
//     nums := []int{5, 1, 3, 5, 1, 2}
//     unique := Distinct(StreamOf(nums), func(a, b int) bool { return a == b })
//     sorted := Sorted(unique, func(a, b int) bool { return a < b })
//     fmt.Println(sorted.ToSlice()) // [1 2 3 5]
//
//  4. 字符串流 + GroupBy 首字母
//     words := []string{"apple", "banana", "avocado", "blueberry", "cherry"}
//
//     groups := GroupBy(StreamOf(words), func(s string) byte {
//     return s[0]
//     })
//
//     // 输出: map[a:[apple avocado] b:[banana blueberry] c:[cherry]]
//
//  5. StreamBuilder 示例
//     builder := NewStreamBuilder[string]().
//     Filter(func(s string) bool { return strings.HasPrefix(s, "a") }).
//     Sorted(func(a, b string) bool { return a < b }).
//     Map(func(s string) string { return strings.ToUpper(s) }).
//     Take(3).
//     Peek(func(s string) { fmt.Println("中间值:", s) })
//
//     data := []string{"apple", "bob", "alex", "andrew", "bob"}
//
//     result := builder.Build(data).ToSlice()
//     // 中间值: ALEX
//     // 中间值: ALICE
//     // 中间值: ANDREW
//     // 最终: ["ALEX", "ALICE", "ANDREW"]
//
//  6. Reverse 配合使用
//     s := StreamOf([]int{1, 2, 3, 4, 5})
//     reversed := Reverse(s).ToSlice()
//     // [5 4 3 2 1]
package toolutil