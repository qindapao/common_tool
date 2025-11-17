package stream_test

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"common_tool/pkg/toolutil"
)

type Person struct {
	Name string
	Age  int
}

// 测试StreamOf构造器是否如期包装原始切片
func TestStreamOf(t *testing.T) {
	input := []int{1, 2, 3}
	stream := toolutil.StreamOf(input)

	// 测试 ToSlice 输出是否等于原始切片
	got := stream.ToSlice()
	if len(got) != len(input) {
		t.Errorf("Stream.ToSlice length mismatch: got %d, want %d", len(got), len(input))
	}
	for i, v := range input {
		if got[i] != v {
			t.Errorf("Stream.ToSlice mismatch at index %d: got %v, want %v", i, got[i], v)
		}
	}
}

// 测试StreamOf构造器包含空切片的情况
func TestEmptyStream(t *testing.T) {
	var empty []string
	stream := toolutil.StreamOf(empty)

	if stream.Count() != 0 {
		t.Errorf("Empty stream count should be 0, got %d", stream.Count())
	}

	if first, ok := stream.First(); ok {
		t.Errorf("Expected no first element, got %v", first)
	}

	if last, ok := stream.Last(); ok {
		t.Errorf("Expected no last element, got %v", last)
	}
}

// 测试Count
func TestStreamCount(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		expected int
	}{
		{"EmptyStream", []int{}, 0},
		{"OneElement", []int{42}, 1},
		{"MultipleElements", []int{1, 2, 3, 4, 5}, 5},
	}

	for _, tt := range tests {
		// 构建子用例
		t.Run(tt.name, func(t *testing.T) {
			stream := toolutil.StreamOf(tt.input)
			count := stream.Count()
			if count != tt.expected {
				t.Errorf("Count() = %d, want %d", count, tt.expected)
			}
		})
	}
}

// 测试 First
func TestStreamFirst(t *testing.T) {
	tests := []struct {
		name         string
		input        []int
		wantValue    int
		wantHasValue bool
	}{
		{"EmptyStream", []int{}, 0, false},
		{"OneElement", []int{99}, 99, true},
		{"MultipleElements", []int{5, 6, 7}, 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream := toolutil.StreamOf(tt.input)
			val, ok := stream.First()

			if ok != tt.wantHasValue {
				t.Errorf("First() hasValue = %v, want %v", ok, tt.wantHasValue)
			}
			if ok && val != tt.wantValue {
				t.Errorf("First() = %v, want %v", val, tt.wantValue)
			}
		})
	}
}

// 测试Last
func TestStreamLast(t *testing.T) {
	type custom struct {
		Name string
		Age  int
	}

	t.Run("Float64Slice", func(t *testing.T) {
		data := []float64{3.14, 2.71, 1.41}
		stream := toolutil.StreamOf(data)
		v, ok := stream.Last()
		if !ok || v != 1.41 {
			t.Errorf("Last() = %v, want %v", v, 1.41)
		}
	})

	t.Run("StringSlice", func(t *testing.T) {
		data := []string{"go", "perl", "bash"}
		stream := toolutil.StreamOf(data)
		v, ok := stream.Last()
		if !ok || v != "bash" {
			t.Errorf("Last() = %v, want %v", v, "bash")
		}
	})

	t.Run("CustomStructSlice", func(t *testing.T) {
		data := []custom{
			{Name: "Alice", Age: 20},
			{Name: "Bob", Age: 30},
		}
		stream := toolutil.StreamOf(data)
		v, ok := stream.Last()

		expected := custom{Name: "Bob", Age: 30}
		if !ok || !reflect.DeepEqual(v, expected) {
			t.Errorf("Last() = %+v, want %+v", v, expected)
		}
	})

	t.Run("EmptyStream", func(t *testing.T) {
		var data []string
		stream := toolutil.StreamOf(data)
		v, ok := stream.Last()
		if ok {
			t.Errorf("Expected no value, got %v", v)
		}
	})
}

// 测试 ForEach
func TestStreamForEach(t *testing.T) {
	t.Run("CollectElements", func(t *testing.T) {
		data := []string{"go", "perl", "bash"}
		var collected []string

		stream := toolutil.StreamOf(data)
		stream.ForEach(func(s string) {
			collected = append(collected, s)
		})

		if !reflect.DeepEqual(collected, data) {
			t.Errorf("ForEach collected = %v, want %v", collected, data)
		}
	})

	t.Run("CountInvocations", func(t *testing.T) {
		data := []int{1, 2, 3, 4}
		invoked := 0

		stream := toolutil.StreamOf(data)
		// 这里传值但是不使用，所以用 _
		stream.ForEach(func(_ int) {
			invoked++
		})

		if invoked != len(data) {
			t.Errorf("ForEach invoked %d times, want %d", invoked, len(data))
		}
	})

	t.Run("EmptyStream", func(t *testing.T) {
		var data []float64
		called := false

		stream := toolutil.StreamOf(data)
		stream.ForEach(func(_ float64) {
			called = true
		})

		if called {
			t.Errorf("ForEach should not be called on empty stream")
		}
	})
}

// 测试 Filter
func TestFilter(t *testing.T) {
	t.Run("FilterEvenNumbers", func(t *testing.T) {
		data := []int{1, 2, 3, 4, 5, 6}
		stream := toolutil.StreamOf(data)
		result := toolutil.Filter(stream, func(n int) bool {
			return n%2 == 0 // 筛偶数
		}).ToSlice()

		expected := []int{2, 4, 6}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Filter(even) = %v, want %v", result, expected)
		}
	})

	t.Run("FilterShortStrings", func(t *testing.T) {
		data := []string{"go", "perl", "python", "c"}
		stream := toolutil.StreamOf(data)
		result := toolutil.Filter(stream, func(s string) bool {
			return len(s) <= 2 // 字符串长度不超过 2 的
		}).ToSlice()

		expected := []string{"go", "c"}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Filter(short strings) = %v, want %v", result, expected)
		}
	})

	t.Run("FilterFloatsGreaterThanPi", func(t *testing.T) {
		data := []float64{2.7, 3.14, 1.41, 3.5}
		stream := toolutil.StreamOf(data)
		result := toolutil.Filter(stream, func(f float64) bool {
			return f > 3.14
		}).ToSlice()

		expected := []float64{3.5}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Filter(> π) = %v, want %v", result, expected)
		}
	})

	t.Run("FilterStructByField", func(t *testing.T) {
		type person struct {
			Name string
			Age  int
		}
		data := []person{
			{"Alice", 18},
			{"Bob", 25},
			{"Eve", 15},
		}
		stream := toolutil.StreamOf(data)
		result := toolutil.Filter(stream, func(p person) bool {
			return p.Age >= 18
		}).ToSlice()

		expected := []person{
			{"Alice", 18},
			{"Bob", 25},
		}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Filter(adults) = %v, want %v", result, expected)
		}
	})
}

// 测试 Map
func TestMap(t *testing.T) {
	t.Run("IntToString", func(t *testing.T) {
		data := []int{1, 2, 3}
		stream := toolutil.StreamOf(data)
		result := toolutil.Map(stream, func(n int) string {
			return fmt.Sprintf("no.%d", n)
		}).ToSlice()

		expected := []string{"no.1", "no.2", "no.3"}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Map(int→string) = %v, want %v", result, expected)
		}
	})

	t.Run("StringToLength", func(t *testing.T) {
		data := []string{"go", "perl", "python"}
		stream := toolutil.StreamOf(data)
		result := toolutil.Map(stream, func(s string) int {
			return len(s)
		}).ToSlice()

		expected := []int{2, 4, 6}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Map(string→int) = %v, want %v", result, expected)
		}
	})

	t.Run("StructToField", func(t *testing.T) {
		type person struct {
			Name string
			Age  int
		}
		data := []person{
			{"Alice", 18},
			{"Bob", 25},
		}
		stream := toolutil.StreamOf(data)
		result := toolutil.Map(stream, func(p person) string {
			return p.Name
		}).ToSlice()

		expected := []string{"Alice", "Bob"}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Map(person→Name) = %v, want %v", result, expected)
		}
	})
}

// 测试 MapSafe
func TestMapSafeWithDeepCopy(t *testing.T) {
	t.Run("StructCloneWithTags", func(t *testing.T) {
		type user struct {
			Name string
			// 这里是切片，是一个引用类型
			Tags []string
		}
		data := []user{
			{"Alice", []string{"x"}},
			{"Bob", []string{"y"}},
		}
		stream := toolutil.StreamOf(data)

		result := toolutil.MapSafe(stream, func(u user) user {
			return u // 原样返回，由 MapSafe 来 deepcopy
		}).ToSlice()

		result[0].Tags[0] = "MODIFIED"

		if data[0].Tags[0] == "MODIFIED" {
			t.Errorf("MapSafe(deepcopy) failed: original was mutated")
		}
	})
}

// 测试 Reduce
func TestReduce(t *testing.T) {
	t.Run("SumOfInts", func(t *testing.T) {
		data := []int{1, 2, 3, 4, 5}
		stream := toolutil.StreamOf(data)
		result := toolutil.Reduce(stream, 0, func(acc, v int) int {
			return acc + v
		})
		if result != 15 {
			t.Errorf("Reduce(sum) = %v, want 15", result)
		}
	})

	t.Run("ConcatStrings", func(t *testing.T) {
		data := []string{"go", "lang", "rocks"}
		stream := toolutil.StreamOf(data)
		result := toolutil.Reduce(stream, "", func(acc, v string) string {
			return acc + v + "-"
		})
		expected := "go-lang-rocks-"
		if result != expected {
			t.Errorf("Reduce(concat) = %v, want %v", result, expected)
		}
	})

	t.Run("MaxFloat", func(t *testing.T) {
		data := []float64{1.1, 3.5, 2.8, 9.0, 4.2}
		stream := toolutil.StreamOf(data)
		result := toolutil.Reduce(stream, -math.MaxFloat64, func(acc, v float64) float64 {
			if v > acc {
				return v
			}
			return acc
		})
		if result != 9.0 {
			t.Errorf("Reduce(max) = %v, want 9.0", result)
		}
	})

	t.Run("SumStructFields", func(t *testing.T) {
		type item struct {
			Name  string
			Count int
		}
		data := []item{
			{"apple", 3},
			{"banana", 4},
			{"cherry", 2},
		}
		stream := toolutil.StreamOf(data)
		total := toolutil.Reduce(stream, 0, func(acc int, i item) int {
			return acc + i.Count
		})
		if total != 9 {
			t.Errorf("Reduce(item.Count sum) = %v, want 9", total)
		}
	})

	t.Run("ReduceOnEmptyStream", func(t *testing.T) {
		var data []int
		stream := toolutil.StreamOf(data)
		result := toolutil.Reduce(stream, 100, func(acc, v int) int {
			return acc + v
		})
		if result != 100 {
			t.Errorf("Reduce(empty stream) = %v, want 100", result)
		}
	})
}

// 测试 Distinct
func TestDistinct(t *testing.T) {
	t.Run("IntsWithExactMatch", func(t *testing.T) {
		data := []int{1, 2, 2, 3, 3, 3, 4}
		stream := toolutil.StreamOf(data)
		result := toolutil.Distinct(stream, func(a, b int) bool { return a == b }).ToSlice()

		expected := []int{1, 2, 3, 4}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Distinct(ints) = %v, want %v", result, expected)
		}
	})

	t.Run("StringsIgnoreCase", func(t *testing.T) {
		data := []string{"Go", "go", "Golang", "GOLANG", "Perl"}
		stream := toolutil.StreamOf(data)
		result := toolutil.Distinct(stream, func(a, b string) bool {
			return strings.EqualFold(a, b) // 忽略大小写
		}).ToSlice()

		expected := []string{"Go", "Golang", "Perl"}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Distinct(strings, ignore case) = %v, want %v", result, expected)
		}
	})

	t.Run("StructByField", func(t *testing.T) {
		type lang struct {
			Name string
			Tag  string
		}
		data := []lang{
			{"Golang", "go"},
			{"Go", "go"},
			{"Perl", "perl"},
			{"Perl5", "perl"},
			{"Python", "py"},
		}
		stream := toolutil.StreamOf(data)
		result := toolutil.Distinct(stream, func(a, b lang) bool {
			return a.Tag == b.Tag // 按 Tag 字段判断重复
		}).ToSlice()

		expected := []lang{
			{"Golang", "go"},
			{"Perl", "perl"},
			{"Python", "py"},
		}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Distinct(struct by field) = %v, want %v", result, expected)
		}
	})

	t.Run("EmptyStream", func(t *testing.T) {
		var data []int
		stream := toolutil.StreamOf(data)
		result := toolutil.Distinct(stream, func(a, b int) bool { return a == b }).ToSlice()

		if len(result) != 0 {
			t.Errorf("Distinct(empty) = %v, want empty slice", result)
		}
	})
}

// 测试 DistinctSafe
func TestDistinctSafe(t *testing.T) {
	t.Run("StructWithSliceField", func(t *testing.T) {
		type user struct {
			Name string
			Tags []string
		}

		data := []user{
			{"Alice", []string{"x"}},
			{"Bob", []string{"y"}},
			{"ALICE", []string{"x"}},
		}

		stream := toolutil.StreamOf(data)
		distinct := toolutil.DistinctSafe(stream, func(a, b user) bool {
			return strings.EqualFold(a.Name, b.Name) // 忽略大小写
		}).ToSlice()

		// 验证去重结果
		if len(distinct) != 2 {
			t.Errorf("DistinctSafe result length = %d, want 2", len(distinct))
		}

		// 修改结果数据，确保不会影响原始数据
		distinct[0].Tags[0] = "MODIFIED"

		if data[0].Tags[0] == "MODIFIED" || data[2].Tags[0] == "MODIFIED" {
			t.Errorf("DistinctSafe failed: original data was mutated")
		}
	})
}

// 测试 GroupBy
func TestGroupBy(t *testing.T) {
	t.Run("GroupIntsByRemainder", func(t *testing.T) {
		data := []int{1, 2, 3, 4, 5, 6, 7}
		stream := toolutil.StreamOf(data)
		grouped := toolutil.GroupBy(stream, func(n int) int {
			return n % 3 // 取余数作为分组 key
		})

		if len(grouped) != 3 {
			t.Errorf("Expected 3 groups, got %d", len(grouped))
		}
		if !reflect.DeepEqual(grouped[0], []int{3, 6}) {
			t.Errorf("Group[0] = %v, want [3 6]", grouped[0])
		}
		if !reflect.DeepEqual(grouped[1], []int{1, 4, 7}) {
			t.Errorf("Group[1] = %v, want [1 4 7]", grouped[1])
		}
		if !reflect.DeepEqual(grouped[2], []int{2, 5}) {
			t.Errorf("Group[2] = %v, want [2 5]", grouped[2])
		}
	})

	t.Run("GroupStringsByLength", func(t *testing.T) {
		data := []string{"go", "perl", "c", "bash", "node", "r"}
		stream := toolutil.StreamOf(data)
		grouped := toolutil.GroupBy(stream, func(s string) int {
			return len(s)
		})

		expected := map[int][]string{
			1: {"c", "r"},
			2: {"go"},
			4: {"perl", "bash", "node"},
		}

		if !reflect.DeepEqual(grouped, expected) {
			t.Errorf("GroupBy = %v, want %v", grouped, expected)
		}
	})

	t.Run("GroupStructByField", func(t *testing.T) {
		type logLine struct {
			Level string
			Msg   string
		}
		data := []logLine{
			{"INFO", "start"},
			{"ERROR", "fail A"},
			{"INFO", "running"},
			{"ERROR", "fail B"},
			{"DEBUG", "trace"},
		}

		expected := map[string][]logLine{
			"INFO": {
				{"INFO", "start"},
				{"INFO", "running"},
			},
			"ERROR": {
				{"ERROR", "fail A"},
				{"ERROR", "fail B"},
			},
			"DEBUG": {
				{"DEBUG", "trace"},
			},
		}

		stream := toolutil.StreamOf(data)
		grouped := toolutil.GroupBy(stream, func(l logLine) string {
			return l.Level
		})

		if !reflect.DeepEqual(grouped, expected) {
			t.Errorf("GroupBy = %v, want %v", grouped, expected)
		}
	})

	t.Run("EmptyStream", func(t *testing.T) {
		var data []string
		grouped := toolutil.GroupBy(toolutil.StreamOf(data), func(s string) string {
			return s
		})
		if len(grouped) != 0 {
			t.Errorf("Expected empty group map, got %v", grouped)
		}
	})
}

func TestGroupBySafe(t *testing.T) {
	t.Run("GroupByFieldWithDeepCopy", func(t *testing.T) {
		type user struct {
			Name string
			Tags []string
		}
		data := []user{
			{"Alice", []string{"x"}},
			{"Bob", []string{"x"}},
			{"Carol", []string{"y"}},
		}

		stream := toolutil.StreamOf(data)
		grouped := toolutil.GroupBySafe(stream, func(u user) string {
			return u.Tags[0]
		})

		if len(grouped["x"]) != 2 || len(grouped["y"]) != 1 {
			t.Errorf("GroupBySafe group counts incorrect: %v", grouped)
		}

		// 改变 grouped 中的元素
		grouped["x"][0].Tags[0] = "MODIFIED"

		// 验证原始数据未被污染
		if data[0].Tags[0] == "MODIFIED" || data[1].Tags[0] == "MODIFIED" {
			t.Errorf("GroupBySafe failed: original data was mutated")
		}
	})
}

// 测试Peek
func TestPeek(t *testing.T) {
	t.Run("PeekShouldInvokeSideEffectAndPreserveStream", func(t *testing.T) {
		data := []int{1, 2, 3}
		var seen []int

		stream := toolutil.StreamOf(data)
		stream = toolutil.Peek(stream, func(n int) {
			seen = append(seen, n)
		})

		result := stream.ToSlice()

		if !reflect.DeepEqual(seen, data) {
			t.Errorf("Peek side effect failed: got %v, want %v", seen, data)
		}
		if !reflect.DeepEqual(result, data) {
			t.Errorf("Peek mutated stream: got %v, want %v", result, data)
		}
	})

	t.Run("PeekShouldLogStructElements", func(t *testing.T) {
		type user struct {
			Name string
		}
		data := []user{
			{"Alice"},
			{"Bob"},
		}
		var log []string

		stream := toolutil.StreamOf(data)
		stream = toolutil.Peek(stream, func(u user) {
			log = append(log, fmt.Sprintf("%s", u.Name))
		})

		result := stream.ToSlice()

		wantLog := []string{"Alice", "Bob"}
		if !reflect.DeepEqual(log, wantLog) {
			t.Errorf("Peek logging incorrect: got %v, want %v", log, wantLog)
		}
		if !reflect.DeepEqual(result, data) {
			t.Errorf("Peek mutated stream: got %v, want %v", result, data)
		}
	})
}

// 测试 Sorted
func TestSorted(t *testing.T) {
	t.Run("IntsAsc", func(t *testing.T) {
		data := []int{5, 2, 4, 3, 1}
		sorted := toolutil.Sorted(toolutil.StreamOf(data), func(a, b int) bool {
			return a < b
		}).ToSlice()

		expected := []int{1, 2, 3, 4, 5}
		if !reflect.DeepEqual(sorted, expected) {
			t.Errorf("Sorted(int asc) = %v, want %v", sorted, expected)
		}
	})

	t.Run("StringsDesc", func(t *testing.T) {
		data := []string{"go", "perl", "bash"}
		sorted := toolutil.Sorted(toolutil.StreamOf(data), func(a, b string) bool {
			return a > b // 降序
		}).ToSlice()

		expected := []string{"perl", "go", "bash"}
		if !reflect.DeepEqual(sorted, expected) {
			t.Errorf("Sorted(strings desc) = %v, want %v", sorted, expected)
		}
	})

	t.Run("StructByField", func(t *testing.T) {
		type file struct {
			Name string
			Size int
		}
		data := []file{
			{"a.txt", 300},
			{"b.txt", 100},
			{"c.txt", 200},
		}
		sorted := toolutil.Sorted(toolutil.StreamOf(data), func(a, b file) bool {
			return a.Size < b.Size
		}).ToSlice()

		expected := []file{
			{"b.txt", 100},
			{"c.txt", 200},
			{"a.txt", 300},
		}
		if !reflect.DeepEqual(sorted, expected) {
			t.Errorf("Sorted(files by size) = %v, want %v", sorted, expected)
		}
	})

	t.Run("EmptyStream", func(t *testing.T) {
		var data []float64
		sorted := toolutil.Sorted(toolutil.StreamOf(data), func(a, b float64) bool {
			return a < b
		}).ToSlice()

		if len(sorted) != 0 {
			t.Errorf("Sorted(empty) = %v, want empty slice", sorted)
		}
	})
}

// 测试 SortedSafe
func TestSortedSafe(t *testing.T) {
	t.Run("SortStructByFieldWithDeepCopy", func(t *testing.T) {
		type user struct {
			Name string
			Tags []string
		}
		data := []user{
			{"Charlie", []string{"z"}},
			{"Alice", []string{"x"}},
			{"Bob", []string{"y"}},
		}
		stream := toolutil.StreamOf(data)

		sorted := toolutil.SortedSafe(stream, func(a, b user) bool {
			return a.Name < b.Name
		}).ToSlice()

		expected := []user{
			{"Alice", []string{"x"}},
			{"Bob", []string{"y"}},
			{"Charlie", []string{"z"}},
		}
		if !reflect.DeepEqual(sorted, expected) {
			t.Errorf("SortedSafe result = %v, want %v", sorted, expected)
		}

		// 验证深拷贝是否生效
		sorted[0].Tags[0] = "MODIFIED"
		if data[1].Tags[0] == "MODIFIED" {
			t.Errorf("SortedSafe did not deep copy elements properly")
		}
	})

	t.Run("EmptyStream", func(t *testing.T) {
		var data []int
		stream := toolutil.StreamOf(data)
		sorted := toolutil.SortedSafe(stream, func(a, b int) bool {
			return a < b
		}).ToSlice()

		if len(sorted) != 0 {
			t.Errorf("SortedSafe(empty) = %v, want empty slice", sorted)
		}
	})
}

func TestTake(t *testing.T) {
	t.Run("TakeLessThanLength", func(t *testing.T) {
		data := []int{1, 2, 3, 4, 5}
		stream := toolutil.StreamOf(data)
		result := toolutil.Take(stream, 3).ToSlice()

		want := []int{1, 2, 3}
		if !reflect.DeepEqual(result, want) {
			t.Errorf("Take(3) = %v, want %v", result, want)
		}
	})

	t.Run("TakeEqualToLength", func(t *testing.T) {
		data := []string{"a", "b", "c"}
		stream := toolutil.StreamOf(data)
		result := toolutil.Take(stream, 3).ToSlice()

		want := []string{"a", "b", "c"}
		if !reflect.DeepEqual(result, want) {
			t.Errorf("Take(3) = %v, want %v", result, want)
		}
	})

	t.Run("TakeMoreThanLength", func(t *testing.T) {
		data := []float64{1.1, 2.2}
		stream := toolutil.StreamOf(data)
		result := toolutil.Take(stream, 10).ToSlice()

		want := []float64{1.1, 2.2}
		if !reflect.DeepEqual(result, want) {
			t.Errorf("Take(10) = %v, want %v", result, want)
		}
	})

	t.Run("TakeZero", func(t *testing.T) {
		data := []int{1, 2, 3}
		stream := toolutil.StreamOf(data)
		result := toolutil.Take(stream, 0).ToSlice()

		if len(result) != 0 {
			t.Errorf("Take(0) = %v, want empty", result)
		}
	})

	t.Run("TakeOnEmptyStream", func(t *testing.T) {
		var data []int
		stream := toolutil.StreamOf(data)
		result := toolutil.Take(stream, 3).ToSlice()

		if len(result) != 0 {
			t.Errorf("Take(empty, 3) = %v, want empty", result)
		}
	})
}

func TestTakeSafe(t *testing.T) {
	t.Run("TakeSafeDeepCopyEffect", func(t *testing.T) {
		type user struct {
			Name string
			Tags []string
		}
		data := []user{
			{"Alice", []string{"x"}},
			{"Bob", []string{"y"}},
			{"Carol", []string{"z"}},
		}
		stream := toolutil.StreamOf(data)
		result := toolutil.TakeSafe(stream, 2).ToSlice()

		// 修改 result 的深层字段
		result[0].Tags[0] = "MODIFIED"

		// 原始数据应不受影响
		if data[0].Tags[0] == "MODIFIED" {
			t.Errorf("TakeSafe failed: original data was mutated")
		}

		// 验证截取数量
		if len(result) != 2 {
			t.Errorf("TakeSafe returned %d elements, want 2", len(result))
		}
	})

	t.Run("TakeSafeOnEmptyStream", func(t *testing.T) {
		var data []int
		stream := toolutil.StreamOf(data)
		result := toolutil.TakeSafe(stream, 3).ToSlice()
		if len(result) != 0 {
			t.Errorf("TakeSafe on empty = %v, want empty slice", result)
		}
	})
}

func TestSkip(t *testing.T) {
	t.Run("SkipLessThanLength", func(t *testing.T) {
		data := []int{10, 20, 30, 40}
		stream := toolutil.StreamOf(data)
		result := toolutil.Skip(stream, 2).ToSlice()

		want := []int{30, 40}
		if !reflect.DeepEqual(result, want) {
			t.Errorf("Skip(2) = %v, want %v", result, want)
		}
	})

	t.Run("SkipEqualToLength", func(t *testing.T) {
		data := []string{"a", "b", "c"}
		stream := toolutil.StreamOf(data)
		result := toolutil.Skip(stream, 3).ToSlice()

		if len(result) != 0 {
			t.Errorf("Skip(3) = %v, want empty slice", result)
		}
	})

	t.Run("SkipMoreThanLength", func(t *testing.T) {
		data := []float64{1.1, 2.2}
		stream := toolutil.StreamOf(data)
		result := toolutil.Skip(stream, 5).ToSlice()

		if len(result) != 0 {
			t.Errorf("Skip(5) = %v, want empty slice", result)
		}
	})

	t.Run("SkipZero", func(t *testing.T) {
		data := []int{1, 2, 3}
		stream := toolutil.StreamOf(data)
		result := toolutil.Skip(stream, 0).ToSlice()

		if !reflect.DeepEqual(result, data) {
			t.Errorf("Skip(0) = %v, want %v", result, data)
		}
	})

	t.Run("SkipOnEmptyStream", func(t *testing.T) {
		var data []int
		stream := toolutil.StreamOf(data)
		result := toolutil.Skip(stream, 1).ToSlice()

		if len(result) != 0 {
			t.Errorf("Skip on empty = %v, want empty slice", result)
		}
	})
}

func TestSkipSafe(t *testing.T) {
	t.Run("SkipSafeShouldReturnClonedElements", func(t *testing.T) {
		type user struct {
			Name string
			Tags []string
		}
		data := []user{
			{"Alice", []string{"x"}},
			{"Bob", []string{"y"}},
			{"Carol", []string{"z"}},
		}

		stream := toolutil.StreamOf(data)
		result := toolutil.SkipSafe(stream, 1).ToSlice()

		// 修改跳过后的结果，确保不影响原始数据
		result[0].Tags[0] = "MODIFIED"
		if data[1].Tags[0] == "MODIFIED" {
			t.Errorf("SkipSafe failed: original data was mutated")
		}

		// 验证返回值长度正确
		if len(result) != 2 {
			t.Errorf("SkipSafe returned %d elements, want 2", len(result))
		}

		// 验证内容一致
		if result[1].Name != "Carol" || result[1].Tags[0] != "z" {
			t.Errorf("SkipSafe result content mismatch: got %+v", result[1])
		}
	})

	t.Run("SkipSafeAllOrMoreThanLengthShouldReturnEmpty", func(t *testing.T) {
		data := []int{1, 2, 3}
		stream := toolutil.StreamOf(data)

		got := toolutil.SkipSafe(stream, 3).ToSlice()
		if len(got) != 0 {
			t.Errorf("SkipSafe(3) = %v, want empty slice", got)
		}

		gotOverflow := toolutil.SkipSafe(stream, 10).ToSlice()
		if len(gotOverflow) != 0 {
			t.Errorf("SkipSafe(10) = %v, want empty slice", gotOverflow)
		}
	})

	t.Run("SkipSafeOnEmptyStreamShouldReturnEmpty", func(t *testing.T) {
		var data []string
		stream := toolutil.StreamOf(data)

		result := toolutil.SkipSafe(stream, 1).ToSlice()
		if len(result) != 0 {
			t.Errorf("SkipSafe(empty) = %v, want empty", result)
		}
	})

	t.Run("SkipSafeZeroShouldReturnDeepClonedAll", func(t *testing.T) {
		type person struct {
			ID    int
			Notes []string
		}
		data := []person{
			{1, []string{"note1"}},
			{2, []string{"note2"}},
		}
		stream := toolutil.StreamOf(data)
		result := toolutil.SkipSafe(stream, 0).ToSlice()

		// 修改返回值，验证深拷贝有效
		result[1].Notes[0] = "CHANGED"
		if data[1].Notes[0] == "CHANGED" {
			t.Errorf("SkipSafe(0) failed to deep copy entire stream")
		}
	})
}

func TestReverse(t *testing.T) {
	t.Run("ReverseShouldReverseOrder", func(t *testing.T) {
		data := []int{1, 2, 3, 4}
		result := toolutil.Reverse(toolutil.StreamOf(data)).ToSlice()

		want := []int{4, 3, 2, 1}
		if !reflect.DeepEqual(result, want) {
			t.Errorf("Reverse = %v, want %v", result, want)
		}
	})

	t.Run("ReversePreservesOriginalWhenValueType", func(t *testing.T) {
		data := []string{"a", "b", "c"}
		_ = toolutil.Reverse(toolutil.StreamOf(data)).ToSlice()

		if !reflect.DeepEqual(data, []string{"a", "b", "c"}) {
			t.Errorf("Reverse modified original: %v", data)
		}
	})
}

func TestReverseSafe(t *testing.T) {
	t.Run("ReverseSafeShouldReverseAndDeepCopy", func(t *testing.T) {
		type user struct {
			Name string
			Tags []string
		}
		data := []user{
			{"Alice", []string{"x"}},
			{"Bob", []string{"y"}},
			{"Carol", []string{"z"}},
		}
		result := toolutil.ReverseSafe(toolutil.StreamOf(data)).ToSlice()

		// 顺序应反转
		if result[0].Name != "Carol" || result[2].Name != "Alice" {
			t.Errorf("ReverseSafe order incorrect: got %+v", result)
		}

		// 修改 result，原始数据应不变
		result[1].Tags[0] = "MODIFIED"
		if data[1].Tags[0] == "MODIFIED" {
			t.Errorf("ReverseSafe failed: original data was mutated")
		}
	})

	t.Run("ReverseSafeEmptyStream", func(t *testing.T) {
		var data []int
		result := toolutil.ReverseSafe(toolutil.StreamOf(data)).ToSlice()
		if len(result) != 0 {
			t.Errorf("ReverseSafe(empty) = %v, want empty", result)
		}
	})
}

func TestStreamBuilder_FilterMapSorted(t *testing.T) {
	data := []int{7, 2, 5, 2, 9, 4, 1}

	result := toolutil.NewStreamBuilder[int]().
		Filter(func(v int) bool { return v > 3 }).
		Map(func(v int) int { return v * 2 }).
		Sorted(func(a, b int) bool { return a < b }).
		Build(data).
		ToSlice()

	assert.Equal(t, []int{8, 10, 14, 18}, result)
}

func TestStreamBuilder_TakeDistinctReverse(t *testing.T) {
	data := []string{"a", "b", "a", "c", "b", "d"}

	result := toolutil.NewStreamBuilder[string]().
		Distinct(func(a, b string) bool { return a == b }).
		Take(3).
		Reverse().
		Build(data).
		ToSlice()

	assert.Equal(t, []string{"c", "b", "a"}, result)
}

func TestStreamBuilder_SkipMapSafePeek(t *testing.T) {
	data := []int{1, 2, 3, 4, 5}

	var peeked []int
	result := toolutil.NewStreamBuilder[int]().
		SkipSafe(1).
		MapSafe(func(v int) int { return v + 10 }).
		Peek(func(v int) { peeked = append(peeked, v) }).
		Build(data).
		ToSlice()

	assert.Equal(t, []int{12, 13, 14, 15}, result)
	assert.Equal(t, result, peeked)
}

func TestStreamBuilder_EmptyFlow(t *testing.T) {
	data := []int{}

	result := toolutil.NewStreamBuilder[int]().
		Filter(func(v int) bool { return v > 0 }).
		Map(func(v int) int { return v * 2 }).
		Build(data).
		ToSlice()

	assert.Empty(t, result)
}

func TestStreamBuilder_DeepCopyIntegrity(t *testing.T) {
	data := []*Person{
		{Name: "Alice", Age: 30},
		{Name: "Bob", Age: 25},
		{Name: "Alice", Age: 30},
		{Name: "Charlie", Age: 40},
	}

	result := toolutil.NewStreamBuilder[*Person]().
		MapSafe(func(p *Person) *Person {
			// 不修改原始指针，而是创建全新结构体
			return &Person{
				Name: p.Name,
				Age:  p.Age + 1,
			}
		}).
		DistinctSafe(func(a, b *Person) bool {
			return a.Name == b.Name && a.Age == b.Age
		}).
		SortedSafe(func(a, b *Person) bool {
			return a.Age < b.Age
		}).
		TakeSafe(2).
		ReverseSafe().
		Build(data).
		ToSlice()

	// 验证原始数据未被污染
	assert.Equal(t, 30, data[0].Age)
	assert.Equal(t, 25, data[1].Age)
	assert.Equal(t, 30, data[2].Age)
	assert.Equal(t, 40, data[3].Age)

	// 验证输出顺序与值是否正确
	assert.Len(t, result, 2)
	assert.Equal(t, "Alice", result[0].Name)
	assert.Equal(t, 31, result[0].Age)
	assert.Equal(t, "Bob", result[1].Name)
	assert.Equal(t, 26, result[1].Age)
}

func TestStreamBuilder_Any(t *testing.T) {
	data := []string{"hello", "world", "golang", "awesome"}

	containsG := toolutil.NewStreamBuilder[string]().
		Filter(func(s string) bool {
			return len(s) > 3
		}).
		Map(func(s string) string {
			return strings.ToLower(s)
		}).
		Any(data, func(s string) bool {
			return strings.HasPrefix(s, "g")
		})

	assert.True(t, containsG)
}

func TestStreamBuilder_All_AllMatch(t *testing.T) {
	data := []int{6, 8, 10, 12}

	allEven := toolutil.NewStreamBuilder[int]().
		Filter(func(v int) bool { return v > 5 }).
		All(data, func(v int) bool { return v%2 == 0 })

	assert.True(t, allEven)
}

func TestStreamBuilder_All_SomeMismatch(t *testing.T) {
	data := []int{5, 7, 8, 9}

	allGT6 := toolutil.NewStreamBuilder[int]().
		Map(func(v int) int { return v + 1 }).
		All(data, func(v int) bool { return v > 6 })

	assert.False(t, allGT6)
}

func TestStreamBuilder_None_AllMiss(t *testing.T) {
	data := []string{"alpha", "beta", "gamma"}

	hasDigit := toolutil.NewStreamBuilder[string]().
		Filter(func(s string) bool { return len(s) < 10 }).
		None(data, func(s string) bool {
			return strings.ContainsAny(s, "0123456789")
		})

	assert.True(t, hasDigit)
}

func TestStreamBuilder_None_MatchExists(t *testing.T) {
	data := []int{2, 4, 6, 7}

	noneOdd := toolutil.NewStreamBuilder[int]().
		Filter(func(v int) bool { return v > 0 }).
		None(data, func(v int) bool { return v%2 != 0 })

	assert.False(t, noneOdd)
}

func TestStreamBuilder_Find_Found(t *testing.T) {
	data := []int{3, 6, 9, 12}

	val, ok := toolutil.NewStreamBuilder[int]().
		Map(func(v int) int { return v + 1 }).
		Find(data, func(v int) bool { return v%5 == 0 })

	assert.True(t, ok)
	assert.Equal(t, 10, val) // 9+1=10，第一个被 5 整除的数
}

func TestStreamBuilder_Find_NotFound(t *testing.T) {
	data := []string{"cat", "dog", "bird"}

	val, ok := toolutil.NewStreamBuilder[string]().
		Filter(func(s string) bool { return len(s) > 2 }).
		Find(data, func(s string) bool { return strings.Contains(s, "z") })

	assert.False(t, ok)
	assert.Equal(t, "", val)
}

func TestStreamBuilder_IndexOf_Found(t *testing.T) {
	data := []string{"zero", "one", "two", "three", "four"}

	index := toolutil.NewStreamBuilder[string]().
		Map(func(s string) string {
			return strings.ToUpper(s)
		}).
		IndexOf(data, func(s string) bool {
			return strings.HasPrefix(s, "T")
		})

	assert.Equal(t, 2, index) // "TWO" 在第 2 个位置
}

func TestStreamBuilder_IndexOf_NotFound(t *testing.T) {
	data := []int{1, 3, 5, 7}

	index := toolutil.NewStreamBuilder[int]().
		Filter(func(v int) bool {
			return v > 0
		}).
		IndexOf(data, func(v int) bool {
			return v%2 == 0
		})

	assert.Equal(t, -1, index)
}

func TestStreamBuilder_LastIndexOf_Found(t *testing.T) {
	data := []int{3, 5, 7, 8, 5, 9}

	idx := toolutil.NewStreamBuilder[int]().
		Filter(func(v int) bool { return v > 2 }).
		LastIndexOf(data, func(v int) bool { return v == 5 })

	assert.Equal(t, 4, idx) // 最后一个 5 的位置
}

func TestStreamBuilder_LastIndexOf_NotFound(t *testing.T) {
	data := []string{"dog", "cat", "fox"}

	idx := toolutil.NewStreamBuilder[string]().
		Map(func(s string) string { return strings.ToUpper(s) }).
		LastIndexOf(data, func(s string) bool { return strings.Contains(s, "z") })

	assert.Equal(t, -1, idx)
}

// Max很特殊，必须是可以比较的
func TestOrderedStreamBuilder_Max_Int(t *testing.T) {
	data := []int{7, 2, 9, 5}

	max, ok := toolutil.NewOrderedStreamBuilder[int]().
		Filter(func(v int) bool { return v > 3 }).
		Max(data)

	assert.True(t, ok)
	assert.Equal(t, 9, max)
}

func TestOrderedStreamBuilder_Min_Int(t *testing.T) {
	data := []int{8, 3, 10, 6}

	min, ok := toolutil.NewOrderedStreamBuilder[int]().
		Filter(func(v int) bool { return v%2 == 0 }).
		Min(data)

	assert.True(t, ok)
	assert.Equal(t, 6, min) // 被 Filter 剩下的是 [8,10,6] → min 是 6
}

func TestOrderedStreamBuilder_Min_Empty(t *testing.T) {
	var data []string

	min, ok := toolutil.NewOrderedStreamBuilder[string]().
		Map(func(s string) string { return strings.ToLower(s) }).
		Min(data)

	assert.False(t, ok)
	assert.Equal(t, "", min)
}

func TestOrderedStreamBuilder_Sum_Int(t *testing.T) {
	data := []int{1, 2, 3, 4, 5}

	sum, ok := toolutil.NewOrderedStreamBuilder[int]().
		Filter(func(v int) bool { return v%2 != 0 }).
		Sum(data)

	assert.True(t, ok)
	assert.Equal(t, 9, sum) // 1 + 3 + 5
}

func TestOrderedStreamBuilder_Sum_Empty(t *testing.T) {
	var data []float64

	sum, ok := toolutil.NewOrderedStreamBuilder[float64]().
		Map(func(v float64) float64 { return v * 1.5 }).
		Sum(data)

	assert.False(t, ok)
	assert.Equal(t, 0.0, sum)
}

func TestFloatStreamBuilder_Average_MapFilter(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5, 6}

	avg, ok := toolutil.NewFloatStreamBuilder[float64]().
		Map(func(v float64) float64 { return v * 2 }). // [2, 4, 6, 8, 10, 12]
		Filter(func(v float64) bool { return v > 5 }). // [6, 8, 10, 12]
		Average(data)

	assert.True(t, ok)
	assert.InDelta(t, 9.0, avg, 0.0001) // (6+8+10+12)/4 = 9.0
}

func TestStreamBuilder_Chunk_Even(t *testing.T) {
	data := []int{1, 2, 3, 4, 5, 6}

	chunks := toolutil.NewStreamBuilder[int]().
		Map(func(v int) int { return v * 10 }).
		Chunk(data, 2)

	assert.Equal(t, [][]int{{10, 20}, {30, 40}, {50, 60}}, chunks)
}

func TestStreamBuilder_Chunk_Uneven(t *testing.T) {
	data := []string{"a", "b", "c", "d", "e"}

	chunks := toolutil.NewStreamBuilder[string]().
		Filter(func(s string) bool { return s != "c" }).
		Chunk(data, 2)

	assert.Equal(t, [][]string{{"a", "b"}, {"d", "e"}}, chunks)
}

func TestStreamBuilder_Partition_EvenOdd(t *testing.T) {
	data := []int{1, 2, 3, 4, 5, 6}

	evens, odds := toolutil.NewStreamBuilder[int]().
		Map(func(v int) int { return v*2 - 1 }).
		Partition(data, func(v int) bool { return v%2 == 0 })

	assert.Equal(t, []int{}, evens)
	assert.Equal(t, []int{1, 3, 5, 7, 9, 11}, odds)
}

func TestStreamBuilder_Partition_AlphaGroup(t *testing.T) {
	data := []string{"apple", "banana", "apricot", "grape"}

	aWords, others := toolutil.NewStreamBuilder[string]().
		Filter(func(s string) bool { return len(s) > 0 }).
		Partition(data, func(s string) bool { return strings.HasPrefix(s, "a") })

	assert.Equal(t, []string{"apple", "apricot"}, aWords)
	assert.Equal(t, []string{"banana", "grape"}, others)
}

func TestZip_IntString(t *testing.T) {
	ints := []int{1, 2, 3}
	strs := []string{"a", "b", "c"}

	result := toolutil.Zip(ints, strs, func(i int, s string) string {
		return fmt.Sprintf("%d-%s", i, s)
	})

	assert.Equal(t, []string{"1-a", "2-b", "3-c"}, result)
}

func TestZip_LengthMismatch(t *testing.T) {
	a := []int{1, 2}
	b := []string{"x", "y", "z"} // 多余项不会被使用

	result := toolutil.Zip(a, b, func(i int, s string) string {
		return fmt.Sprintf("%d%s", i, s)
	})

	assert.Equal(t, []string{"1x", "2y"}, result)
}

type Pair struct {
	Key   string
	Value int
}

func TestZip_ToStruct(t *testing.T) {
	keys := []string{"apple", "banana"}
	values := []int{10, 20}

	pairs := toolutil.Zip(keys, values, func(k string, v int) Pair {
		return Pair{Key: k, Value: v}
	})

	expected := []Pair{{"apple", 10}, {"banana", 20}}
	assert.Equal(t, expected, pairs)
}

func TestStreamBuilder_ChunkSafe(t *testing.T) {
	data := []int{1, 2, 3, 4, 5}

	// 构造流 → 映射加倍 → 安全切片
	chunks := toolutil.NewStreamBuilder[int]().
		Map(func(v int) int { return v * 10 }).
		ChunkSafe(data, 2)

	expected := [][]int{
		{10, 20},
		{30, 40},
		{50},
	}
	assert.Equal(t, expected, chunks)

	// 修改外部切片，确保不会影响已返回数据
	chunks[0][0] = 999
	assert.Equal(t, 10, data[0]*10)         // 原始数据未变
	assert.NotEqual(t, 999, expected[0][0]) // 深拷贝有效
}

func TestChunkSafe_Direct(t *testing.T) {
	data := []int{7, 8, 9, 10}

	// 先构建流，再调用安全分片
	stream := toolutil.NewStreamBuilder[int]().Build(data)

	chunks := toolutil.ChunkSafe(stream, 3)

	expected := [][]int{
		{7, 8, 9},
		{10},
	}
	assert.Equal(t, expected, chunks)

	// 修改返回数据，验证不会影响源
	chunks[0][0] = 0
	assert.Equal(t, 7, data[0]) // 确保原始数据未被污染
}

func TestStreamBuilder_PartitionSafe(t *testing.T) {
	type User struct {
		Name string
		Age  int
	}
	users := []User{
		{"Alice", 25},
		{"Bob", 30},
		{"Carol", 18},
	}

	adults, minors := toolutil.NewStreamBuilder[User]().
		PartitionSafe(users, func(u User) bool { return u.Age >= 20 })

	assert.Equal(t, []User{{"Alice", 25}, {"Bob", 30}}, adults)
	assert.Equal(t, []User{{"Carol", 18}}, minors)

	// 修改结果，验证不会影响原始数据
	adults[0].Age = 99
	assert.Equal(t, 25, users[0].Age) // 原始数据未被污染
}

func TestZipSafe(t *testing.T) {
	keys := []string{"apple", "banana"}
	values := []int{10, 20}

	result := toolutil.ZipSafe(keys, values, func(k string, v int) Pair {
		return Pair{Key: k, Value: v}
	})

	// 修改输出，确保不影响原始输入
	result[0].Key = "changed"
	assert.Equal(t, "apple", keys[0])
	assert.Equal(t, 10, values[0])
}

func TestStreamBuilder_Window_Tumbling(t *testing.T) {
	data := []int{1, 2, 3, 4, 5, 6}
	win := toolutil.NewStreamBuilder[int]().
		Window(data, 2, 0) // step=0 时等同于 tumbling
	assert.Equal(t, [][]int{{1, 2}, {3, 4}, {5, 6}}, win)
}

func TestStreamBuilder_Window_Sliding(t *testing.T) {
	data := []int{1, 2, 3, 4, 5, 6}
	win := toolutil.NewStreamBuilder[int]().
		Window(data, 3, 1)
	assert.Equal(t, [][]int{{1, 2, 3}, {2, 3, 4}, {3, 4, 5}, {4, 5, 6}}, win)
}
func TestWindowSafe_Tumbling(t *testing.T) {
	data := []int{1, 2, 3, 4, 5, 6}
	// tumbling 窗口：size=2, step=0
	wins := toolutil.NewStreamBuilder[int]().
		WindowSafe(data, 2, 0)

	expected := [][]int{{1, 2}, {3, 4}, {5, 6}}
	assert.Equal(t, expected, wins)

	// 修改返回窗口，确保原 data 不变
	wins[0][0] = 999
	assert.Equal(t, 1, data[0])
}

func TestWindowSafe_Sliding(t *testing.T) {
	data := []int{10, 20, 30, 40, 50}

	// 1）先 Build 出一个 Stream[int]
	stream := toolutil.NewStreamBuilder[int]().Build(data)

	// 2）再调用 WindowSafe
	wins := toolutil.WindowSafe(stream, 3, 1)

	expected := [][]int{
		{10, 20, 30},
		{20, 30, 40},
		{30, 40, 50},
	}
	assert.Equal(t, expected, wins)

	// 修改返回，确保原 data 不变
	wins[1][1] = 0
	assert.Equal(t, 30, data[2])
}

func TestFloatStreamBuilder_MapSafe_Primitive(t *testing.T) {
	data := []float64{1.1, 2.2, 3.3}

	// 安全映射：先深拷贝输入，再应用 f，再深拷贝输出
	builder := toolutil.NewFloatStreamBuilder[float64]().
		MapSafe(func(v float64) float64 {
			return v * 10
		})

	// 执行构建，ToSlice 转换回 切片
	out := builder.Build(data).ToSlice()

	// 结果正确
	assert.Equal(t, []float64{11, 22, 33}, out)

	// 修改输出不影响原始切片
	out[1] = 999
	assert.Equal(t, float64(2.2), data[1])
}

func TestOrderedStreamBuilder_MapSafe_Primitive(t *testing.T) {
	data := []int{1, 2, 3}

	// 安全映射：输入／输出都做深拷贝
	builder := toolutil.NewOrderedStreamBuilder[int]().
		MapSafe(func(v int) int { return v * 3 })

	out := builder.Build(data).ToSlice()
	// 结果正确
	assert.Equal(t, []int{3, 6, 9}, out)

	// 修改输出不影响原切片
	out[1] = -99
	assert.Equal(t, 2, data[1])
}

func TestFilterSafe_Primitive(t *testing.T) {
	data := []int{1, 2, 3, 4, 5}

	out := toolutil.NewStreamBuilder[int]().
		FilterSafe(func(v int) bool { return v%2 == 1 }).
		Build(data).ToSlice()

	assert.Equal(t, []int{1, 3, 5}, out)

	// 修改输出不影响原切片
	out[0] = 999
	assert.Equal(t, 1, data[0])
}

func TestFilterSafe_Struct(t *testing.T) {
	type Pair struct {
		A int
		B []string
	}
	data := []Pair{
		{A: 1, B: []string{"x"}},
		{A: 2, B: []string{"y"}},
	}

	out := toolutil.NewStreamBuilder[Pair]().
		FilterSafe(func(p Pair) bool { return p.A%2 == 0 }).
		Build(data).ToSlice()

	// 只保留 A==2 的项
	assert.Len(t, out, 1)
	// 输出中的 B slice 是深拷贝
	out[0].B[0] = "z"
	// 原始 data[1].B 不变
	assert.Equal(t, "y", data[1].B[0])
}

func TestOrderedStreamBuilder_FilterSafe(t *testing.T) {
	data := []string{"a", "bb", "ccc"}

	out := toolutil.NewOrderedStreamBuilder[string]().
		FilterSafe(func(s string) bool { return len(s) > 1 }).
		Build(data).ToSlice()

	assert.Equal(t, []string{"bb", "ccc"}, out)
}

func TestFloatStreamBuilder_FilterSafe(t *testing.T) {
	data := []float64{1.1, 2.2, 3.3}

	out := toolutil.NewFloatStreamBuilder[float64]().
		FilterSafe(func(f float64) bool { return f > 2 }).Build(data).
		ToSlice()

	assert.Equal(t, []float64{2.2, 3.3}, out)
}
