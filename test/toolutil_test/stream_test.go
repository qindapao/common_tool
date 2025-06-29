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