package toolutil

import (
	"sort"

	"golang.org/x/exp/constraints"

	"github.com/mohae/deepcopy"
)

// Stream 是一个数据流容器，支持链式数据处理
type Stream[T any] struct {
	data []T
}

// StreamOf 将切片包装为 Stream 对象
func StreamOf[T any](data []T) Stream[T] {
	return Stream[T]{data}
}

func (s Stream[T]) ToSlice() []T {
	return s.data
}

func (s Stream[T]) Count() int {
	return len(s.data)
}

func (s Stream[T]) First() (T, bool) {
	if len(s.data) == 0 {
		var zero T
		return zero, false
	}
	return s.data[0], true
}

func (s Stream[T]) Last() (T, bool) {
	if len(s.data) == 0 {
		var zero T
		return zero, false
	}
	return s.data[len(s.data)-1], true
}

// 干活但是不回报，副作用函数，不能改变原始数据
func (s Stream[T]) ForEach(f func(T)) {
	for _, v := range s.data {
		f(v)
	}
}

// Filter 过滤切片中满足条件的元素
func Filter[T any](s Stream[T], pred func(T) bool) Stream[T] {
	var out []T
	for _, v := range s.data {
		if pred(v) {
			out = append(out, v)
		}
	}
	return Stream[T]{out}
}

func FilterSafe[T any](s Stream[T], pred func(T) bool) Stream[T] {
	var out []T
	for _, v := range s.data {
		if pred(v) {
			// 深拷贝 v，再追加
			m := deepcopy.Copy(v).(T)
			out = append(out, m)
		}
	}
	return Stream[T]{out}
}

// Map 映射元素为另一个类型
func Map[T any, R any](s Stream[T], f func(T) R) Stream[R] {
	out := make([]R, len(s.data))
	for i, v := range s.data {
		out[i] = f(v)
	}
	return Stream[R]{out}
}

// MapSafe 使用 deepcopy 保护每个返回值，防止引用泄漏
func MapSafe[T any, R any](s Stream[T], f func(T) R) Stream[R] {
	out := make([]R, len(s.data))
	for i, v := range s.data {
		mapped := f(v)
		out[i] = deepcopy.Copy(mapped).(R) // 强转回目标类型
	}
	return Stream[R]{out}
}

// Reduce 将流中的元素归约为一个值
func Reduce[T any, R any](s Stream[T], init R, comb func(R, T) R) R {
	acc := init
	for _, v := range s.data {
		acc = comb(acc, v)
	}
	return acc
}

// Distinct 去重元素，eq 用于判断是否相等
func Distinct[T any](s Stream[T], eq func(T, T) bool) Stream[T] {
	var result []T
	for _, v := range s.data {
		found := false
		for _, r := range result {
			if eq(v, r) {
				found = true
				break
			}
		}
		if !found {
			result = append(result, v)
		}
	}
	return Stream[T]{result}
}

// DistinctSafe 深拷贝去重元素，避免引用穿透
func DistinctSafe[T any](s Stream[T], eq func(T, T) bool) Stream[T] {
	var result []T
	for _, v := range s.data {
		found := false
		for _, r := range result {
			if eq(v, r) {
				found = true
				break
			}
		}
		if !found {
			cloned := deepcopy.Copy(v).(T)
			result = append(result, cloned)
		}
	}
	return Stream[T]{result}
}

// GroupBy 按某个 key 对元素进行分组
func GroupBy[T any, K comparable](s Stream[T], keyFunc func(T) K) map[K][]T {
	result := make(map[K][]T)
	for _, v := range s.data {
		k := keyFunc(v)
		result[k] = append(result[k], v)
	}
	return result
}

// GroupBySafe 将元素按 keyFunc 分组，并对每个元素执行深拷贝
func GroupBySafe[T any, K comparable](s Stream[T], keyFunc func(T) K) map[K][]T {
	result := make(map[K][]T)
	for _, v := range s.data {
		k := keyFunc(v)
		clone := deepcopy.Copy(v).(T)
		result[k] = append(result[k], clone)
	}
	return result
}

// Peek 对每个元素执行副作用操作（如打印），返回原流
func Peek[T any](s Stream[T], f func(T)) Stream[T] {
	for _, v := range s.data {
		f(v)
	}
	return s
}

func sortStream[T any](data []T, less func(T, T) bool) Stream[T] {
	sort.Slice(data, func(i, j int) bool {
		return less(data[i], data[j])
	})
	return Stream[T]{data}
}

// Sorted 按指定比较函数排序
func Sorted[T any](s Stream[T], less func(T, T) bool) Stream[T] {
	cloned := make([]T, len(s.data))
	copy(cloned, s.data)
	return sortStream(cloned, less)
}

// Sorted 的深度拷贝版本，性能比较低但是安全，防止引用类型
func SortedSafe[T any](s Stream[T], less func(T, T) bool) Stream[T] {
	cloned := deepcopy.Copy(s.data).([]T)
	return sortStream(cloned, less)
}

// Take 获取前 n 项
func Take[T any](s Stream[T], n int) Stream[T] {
	if n >= len(s.data) {
		return s
	}
	return Stream[T]{s.data[:n]}
}

// TakeSafe 返回最多前 n 个元素的深拷贝，防止底层引用共享
func TakeSafe[T any](s Stream[T], n int) Stream[T] {
	if n >= len(s.data) {
		// 还是要 deepcopy 整个 data，避免 s.data 被引用出去改动
		cloned := deepcopy.Copy(s.data).([]T)
		return Stream[T]{cloned}
	}

	// 截断 + 拷贝截断后的元素
	cloned := deepcopy.Copy(s.data[:n]).([]T)
	return Stream[T]{cloned}
}

// Skip 跳过前 n 项
func Skip[T any](s Stream[T], n int) Stream[T] {
	if n >= len(s.data) {
		return Stream[T]{}
	}
	return Stream[T]{s.data[n:]}
}

// SkipSafe 深拷贝跳过后的元素，避免引用类型修改影响原始数据
func SkipSafe[T any](s Stream[T], n int) Stream[T] {
	if n >= len(s.data) {
		return Stream[T]{}
	}
	cloned := deepcopy.Copy(s.data[n:]).([]T)
	return Stream[T]{cloned}
}

// Reverse 反转流中元素顺序
func Reverse[T any](s Stream[T]) Stream[T] {
	cloned := make([]T, len(s.data))
	copy(cloned, s.data)

	for i, j := 0, len(cloned)-1; i < j; i, j = i+1, j-1 {
		cloned[i], cloned[j] = cloned[j], cloned[i]
	}
	return Stream[T]{cloned}
}

// ReverseSafe 深拷贝反转后的元素，防止引用字段联动污染
func ReverseSafe[T any](s Stream[T]) Stream[T] {
	cloned := deepcopy.Copy(s.data).([]T)

	for i, j := 0, len(cloned)-1; i < j; i, j = i+1, j-1 {
		cloned[i], cloned[j] = cloned[j], cloned[i]
	}
	return Stream[T]{cloned}
}

// Any 只要有一个元素满足条件就是真
func Any[T any](s Stream[T], pred func(T) bool) bool {
	for _, v := range s.data {
		if pred(v) {
			return true
		}
	}
	return false
}

func All[T any](s Stream[T], pred func(T) bool) bool {
	for _, v := range s.data {
		if !pred(v) {
			return false
		}
	}
	return true
}

func None[T any](s Stream[T], pred func(T) bool) bool {
	for _, v := range s.data {
		if pred(v) {
			return false
		}
	}
	return true
}

// Find 返回满足条件的第一个元素
func Find[T any](s Stream[T], pred func(T) bool) (T, bool) {
	for _, v := range s.data {
		if pred(v) {
			return v, true
		}
	}
	var zero T
	return zero, false
}

// 找不到索引返回 -1
func IndexOf[T any](s Stream[T], pred func(T) bool) int {
	for i, v := range s.data {
		if pred(v) {
			return i
		}
	}
	return -1
}

func LastIndexOf[T any](s Stream[T], pred func(T) bool) int {
	for i := len(s.data) - 1; i >= 0; i-- {
		if pred(s.data[i]) {
			return i
		}
	}
	return -1
}

func Max[T constraints.Ordered](s Stream[T]) (T, bool) {
	if len(s.data) == 0 {
		var zero T
		return zero, false
	}
	max := s.data[0]
	for _, v := range s.data[1:] {
		if v > max {
			max = v
		}
	}
	return max, true
}

func Min[T constraints.Ordered](s Stream[T]) (T, bool) {
	if len(s.data) == 0 {
		var zero T
		return zero, false
	}
	min := s.data[0]
	for _, v := range s.data[1:] {
		if v < min {
			min = v
		}
	}
	return min, true
}

// 泛型约束要求支持 + 号运算
func Sum[T constraints.Ordered](s Stream[T]) (T, bool) {
	if len(s.data) == 0 {
		var zero T
		return zero, false
	}
	var sum T
	for _, v := range s.data {
		sum += v
	}
	return sum, true
}

// 平均值只对浮点有意义
type Float interface {
	~float32 | ~float64
}

func Average[T Float](s Stream[T]) (T, bool) {
	if len(s.data) == 0 {
		var zero T
		return zero, false
	}
	var sum T
	for _, v := range s.data {
		sum += v
	}
	return sum / T(len(s.data)), true
}

// 把流按照大小切片成小数组
func Chunk[T any](s Stream[T], size int) [][]T {
	if size <= 0 {
		return nil
	}
	var chunks [][]T
	data := s.data
	for len(data) > 0 {
		end := size
		if len(data) < size {
			end = len(data)
		}
		chunks = append(chunks, data[:end])
		data = data[end:]
	}
	return chunks
}

func ChunkSafe[T any](s Stream[T], size int) [][]T {
	if size <= 0 {
		return nil
	}
	var chunks [][]T
	data := s.data
	for len(data) > 0 {
		end := size
		if len(data) < size {
			end = len(data)
		}
		// 深度拷贝这段切片
		block := deepcopy.Copy(data[:end]).([]T)
		chunks = append(chunks, block)
		data = data[end:]
	}
	return chunks
}

func Partition[T any](s Stream[T], pred func(T) bool) ([]T, []T) {
	matched := make([]T, 0)
	unmatched := make([]T, 0)

	for _, v := range s.data {
		if pred(v) {
			matched = append(matched, v)
		} else {
			unmatched = append(unmatched, v)
		}
	}
	return matched, unmatched
}

func PartitionSafe[T any](s Stream[T], pred func(T) bool) ([]T, []T) {
	matched := make([]T, 0)
	unmatched := make([]T, 0)

	for _, v := range s.data {
		if pred(v) {
			m := deepcopy.Copy(v).(T)
			matched = append(matched, m)
		} else {
			u := deepcopy.Copy(v).(T)
			unmatched = append(unmatched, u)
		}
	}
	return matched, unmatched
}

// Window 操作其实就是把一条流“按窗口”切片，常见有两种玩法：
// Tumbling Window（跃动窗口） 不重叠、按固定大小切分： 例如 data=[1,2,3,4,5,6], size=2 → [[1,2],[3,4],[5,6]]
// Sliding Window（滑动窗口） 可以重叠、指定步长： size=3, step=1 → [[1,2,3],[2,3,4],[3,4,5],[4,5,6]] size=3, step=2 → [[1,2,3],[3,4,5]]
// Window: size 为窗口大小，step 为滑动步长（step<=0 时等同于 size）
func Window[T any](s Stream[T], size, step int) [][]T {
	if size <= 0 {
		return nil
	}
	if step <= 0 {
		step = size
	}
	var out [][]T
	data := s.data
	for start := 0; start+size <= len(data); start += step {
		// 这里简单地返回底层切片，可酌情用 make+copy 做深拷贝
		out = append(out, data[start:start+size])
	}
	return out
}

// WindowSafe: size 为窗口大小，step 为滑动步长（step<=0 时等同于 size）
// 只产生完整窗口，不包含不足 size 的尾部。
func WindowSafe[T any](s Stream[T], size, step int) [][]T {
	if size <= 0 {
		return nil
	}
	if step <= 0 {
		step = size
	}

	var out [][]T
	n := len(s.data)
	for start := 0; start+size <= n; start += step {
		// 深拷贝每个元素，构造一个独立的 window 切片
		win := make([]T, size)
		for i := 0; i < size; i++ {
			win[i] = deepcopy.Copy(s.data[start+i]).(T)
		}
		out = append(out, win)
	}
	return out
}

type StreamBuilder[T any] struct {
	steps []func(Stream[T]) Stream[T]
}

// 新建构建器
func NewStreamBuilder[T any]() *StreamBuilder[T] {
	return &StreamBuilder[T]{}
}

// Filter
func (b *StreamBuilder[T]) Filter(f func(T) bool) *StreamBuilder[T] {
	b.steps = append(b.steps, func(s Stream[T]) Stream[T] {
		return Filter(s, f)
	})
	return b
}

func (b *StreamBuilder[T]) FilterSafe(pred func(T) bool) *StreamBuilder[T] {
	b.steps = append(b.steps, func(s Stream[T]) Stream[T] {
		return FilterSafe(s, pred)
	})
	return b
}

// Map(就地修改并且return自己，保持链条不断)
func (b *StreamBuilder[T]) Map(f func(T) T) *StreamBuilder[T] {
	b.steps = append(b.steps, func(s Stream[T]) Stream[T] {
		return Map(s, f)
	})
	return b
}

// MapSafe
func (b *StreamBuilder[T]) MapSafe(f func(T) T) *StreamBuilder[T] {
	b.steps = append(b.steps, func(s Stream[T]) Stream[T] {
		return MapSafe(s, f)
	})
	return b
}

// Sorted
func (b *StreamBuilder[T]) Sorted(less func(T, T) bool) *StreamBuilder[T] {
	b.steps = append(b.steps, func(s Stream[T]) Stream[T] {
		return Sorted(s, less)
	})
	return b
}

// SortedSafe
func (b *StreamBuilder[T]) SortedSafe(less func(T, T) bool) *StreamBuilder[T] {
	b.steps = append(b.steps, func(s Stream[T]) Stream[T] {
		return SortedSafe(s, less)
	})
	return b
}

// Distinct
func (b *StreamBuilder[T]) Distinct(eq func(T, T) bool) *StreamBuilder[T] {
	b.steps = append(b.steps, func(s Stream[T]) Stream[T] {
		return Distinct(s, eq)
	})
	return b
}

// DistinctSafe
func (b *StreamBuilder[T]) DistinctSafe(eq func(T, T) bool) *StreamBuilder[T] {
	b.steps = append(b.steps, func(s Stream[T]) Stream[T] {
		return DistinctSafe(s, eq)
	})
	return b
}

// Take
func (b *StreamBuilder[T]) Take(n int) *StreamBuilder[T] {
	b.steps = append(b.steps, func(s Stream[T]) Stream[T] {
		return Take(s, n)
	})
	return b
}

// TakeSafe
func (b *StreamBuilder[T]) TakeSafe(n int) *StreamBuilder[T] {
	b.steps = append(b.steps, func(s Stream[T]) Stream[T] {
		return TakeSafe(s, n)
	})
	return b
}

// Skip
func (b *StreamBuilder[T]) Skip(n int) *StreamBuilder[T] {
	b.steps = append(b.steps, func(s Stream[T]) Stream[T] {
		return Skip(s, n)
	})
	return b
}

// SkipSafe
func (b *StreamBuilder[T]) SkipSafe(n int) *StreamBuilder[T] {
	b.steps = append(b.steps, func(s Stream[T]) Stream[T] {
		return SkipSafe(s, n)
	})
	return b
}

// Reverse
func (b *StreamBuilder[T]) Reverse() *StreamBuilder[T] {
	b.steps = append(b.steps, func(s Stream[T]) Stream[T] {
		return Reverse(s)
	})
	return b
}

// ReverseSafe
func (b *StreamBuilder[T]) ReverseSafe() *StreamBuilder[T] {
	b.steps = append(b.steps, func(s Stream[T]) Stream[T] {
		return ReverseSafe(s)
	})
	return b
}

// Peek（仅副作用）
func (b *StreamBuilder[T]) Peek(f func(T)) *StreamBuilder[T] {
	b.steps = append(b.steps, func(s Stream[T]) Stream[T] {
		return Peek(s, f)
	})
	return b
}

// Any和Build一样是终结函数
func (b *StreamBuilder[T]) Any(data []T, pred func(T) bool) bool {
	return Any(b.Build(data), pred)
}

// All和Build一样是终结函数
func (b *StreamBuilder[T]) All(data []T, pred func(T) bool) bool {
	return All(b.Build(data), pred)
}

// None 终结函数
func (b *StreamBuilder[T]) None(data []T, pred func(T) bool) bool {
	return None(b.Build(data), pred)
}

// Find 满足条件的第一个值，终结函数
func (b *StreamBuilder[T]) Find(data []T, pred func(T) bool) (T, bool) {
	return Find(b.Build(data), pred)
}

func (b *StreamBuilder[T]) IndexOf(data []T, pred func(T) bool) int {
	return IndexOf(b.Build(data), pred)
}

func (b *StreamBuilder[T]) LastIndexOf(data []T, pred func(T) bool) int {
	return LastIndexOf(b.Build(data), pred)
}

func (b *StreamBuilder[T]) Chunk(data []T, size int) [][]T {
	return Chunk(b.Build(data), size)
}

func (b *StreamBuilder[T]) Partition(data []T, pred func(T) bool) ([]T, []T) {
	return Partition(b.Build(data), pred)
}

func (b *StreamBuilder[T]) PartitionSafe(data []T, pred func(T) bool) ([]T, []T) {
	return PartitionSafe(b.Build(data), pred)
}

func (b *StreamBuilder[T]) ChunkSafe(data []T, size int) [][]T {
	return ChunkSafe(b.Build(data), size)
}

func (b *StreamBuilder[T]) Window(data []T, size, step int) [][]T {
	return Window(b.Build(data), size, step)
}

func (b *StreamBuilder[T]) WindowSafe(data []T, size, step int) [][]T {
	return WindowSafe(b.Build(data), size, step)
}

// 执行构建并返回最终 Stream
func (b *StreamBuilder[T]) Build(data []T) Stream[T] {
	stream := StreamOf(data)
	for _, step := range b.steps {
		stream = step(stream)
	}
	return stream
}

// 保留链式调用，同时在类型系统里告诉编译器：放心，流里的元素是能比较大小的
type OrderedStreamBuilder[T constraints.Ordered] struct {
	StreamBuilder[T]
}

func NewOrderedStreamBuilder[T constraints.Ordered]() *OrderedStreamBuilder[T] {
	return &OrderedStreamBuilder[T]{StreamBuilder: *NewStreamBuilder[T]()}
}

func (b *OrderedStreamBuilder[T]) Max(data []T) (T, bool) {
	return Max(b.Build(data))
}

// Filter
func (b *OrderedStreamBuilder[T]) Filter(f func(T) bool) *OrderedStreamBuilder[T] {
	b.StreamBuilder.Filter(f)
	return b
}

func (b *OrderedStreamBuilder[T]) FilterSafe(f func(T) bool) *OrderedStreamBuilder[T] {
	b.StreamBuilder.FilterSafe(f)
	return b
}

func (b *OrderedStreamBuilder[T]) Min(data []T) (T, bool) {
	return Min(b.Build(data))
}

func (b *OrderedStreamBuilder[T]) Map(f func(T) T) *OrderedStreamBuilder[T] {
	b.StreamBuilder.Map(f)
	return b
}

// 在 OrderedStreamBuilder 上加一个 MapSafe
func (b *OrderedStreamBuilder[T]) MapSafe(f func(T) T) *OrderedStreamBuilder[T] {
	// 就地在 b.StreamBuilder.steps 里 append 一个安全映射 step
	b.StreamBuilder.MapSafe(f)
	// 返回的依旧是同一个 OrderedStreamBuilder，继续链式调用
	return b
}

func (b *OrderedStreamBuilder[T]) Sum(data []T) (T, bool) {
	return Sum(b.Build(data))
}

type FloatStreamBuilder[T Float] struct {
	StreamBuilder[T]
}

func (b *FloatStreamBuilder[T]) Average(data []T) (T, bool) {
	return Average(b.Build(data))
}

// 这些转发函数不是多余的，保证返回的类型正确，不至于变成StreamBuilder
// 终结函数就不用重写了
func (b *FloatStreamBuilder[T]) Map(f func(T) T) *FloatStreamBuilder[T] {
	b.StreamBuilder.Map(f)
	return b
}

// 在 FloatStreamBuilder 上添加 MapSafe
func (b *FloatStreamBuilder[T]) MapSafe(f func(T) T) *FloatStreamBuilder[T] {
	b.StreamBuilder.MapSafe(f)
	return b
}

func (b *FloatStreamBuilder[T]) Filter(f func(T) bool) *FloatStreamBuilder[T] {
	b.StreamBuilder.Filter(f)
	return b
}

func (b *FloatStreamBuilder[T]) FilterSafe(f func(T) bool) *FloatStreamBuilder[T] {
	b.StreamBuilder.FilterSafe(f)
	return b
}

func NewFloatStreamBuilder[T Float]() *FloatStreamBuilder[T] {
	return &FloatStreamBuilder[T]{StreamBuilder: *NewStreamBuilder[T]()}
}

// zip 函数是原子函数
// zip 操作多个流对象，和当前流构建器语义不一样，所以单独成立为函数
func Zip[A any, B any, R any](a []A, b []B, f func(A, B) R) []R {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	result := make([]R, 0, n)
	for i := 0; i < n; i++ {
		result = append(result, f(a[i], b[i]))
	}
	return result
}

func ZipSafe[A any, B any, R any](a []A, b []B, f func(A, B) R) []R {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	result := make([]R, 0, n)
	for i := 0; i < n; i++ {
		ra := deepcopy.Copy(a[i]).(A)
		rb := deepcopy.Copy(b[i]).(B)
		r := f(ra, rb)
		result = append(result, deepcopy.Copy(r).(R))
	}
	return result
}