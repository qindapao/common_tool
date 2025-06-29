// Package textutil 提供一个链式文本处理构建器 Builder，
// 支持文本切分、正则提取、过滤映射、字段定位、结构断言和链路调试追踪。
//
// \U0001f517 Builder：处理文本，如同操作列表（Segment-Oriented）
//
// —— 想象一下你不是在处理 string，而是在操作字符串“片段切片”
// —— 每次调用都是一段数据流的重构、过滤与压缩
//
// ✅ 核心理念：
//   • 将文本看作“片段列表”：[]string，不再手动处理索引、Split、Len
//   • 支持链式调用：如 Split → Regex → Filter → Index → Join
//   • 保留每一步中间状态：便于调试、断言和测试
//   • 断言失败统一挂载：b.Error()，支持错误聚合与信息丰富化
//
// ✅ 典型应用场景：
//   • 日志提取
//   • 配置片段分析
//   • 脚本参数校验
//   • 行级内容清洗
//
// 示例用法：
//   raw := "2021-05-20_error-404_log"
//   code, err := textutil.NewBuilder(raw).
//     SplitSep("_").       // ["2021-05-20", "error-404", "log"]
//     Index(1).            // ["error-404"]
//     SplitSep("-").       // ["error", "404"]
//     Index(1).            // ["404"]
//     MustFirst("必须提取出一个错误码")
//
//   if err != nil {
//     panic(err)
//   }
//   fmt.Println(code) // 输出 "404"
//
// 示例2：提取所有以 ERR 开头的字段
//   raw := " OK1, ERR_404, OK2, ERR_503 "
//   errCodes := textutil.NewBuilder(raw).
//     SplitSep(",").
//     Map(strings.TrimSpace).
//     Filter(func(s string) bool {
//       return strings.HasPrefix(s, "ERR")
//     }).
//     Result() // []string{"ERR_404", "ERR_503"}
//
// \U0001f4cc 默认开启 trace 追踪机制，可通过 textutil.DisableGlobalTrace() 全局关闭追踪。
// \U0001f50d 模块内部结构导览（供维护者使用）：
//
//  • 操作链式 API：SplitSep, SplitLen, Regex, RegexGroup, Index, Map, Filter...
//  • 断言 API：AssertCount, MustFirst
//  • 出口 API：Result, First, Last, Join, JoinLines...
//  • 错误追踪：traceStep, DumpTrace, traceEnabled
//  • 初始化：NewBuilder, NewBuilderByLines
//  • 全局行为控制：EnableGlobalTrace, DisableGlobalTrace

package textutil

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"common_tool/pkg/errorutil"
)

var globalTraceEnabled = true

func EnableGlobalTrace() {
	globalTraceEnabled = true
}

// 关闭错误追踪，默认打开
func DisableGlobalTrace() {
	globalTraceEnabled = false
}

// SplitLines 兼容多平台换行符，统一按 \n 拆分
func SplitLines(s string) []string {
	normalized := strings.ReplaceAll(s, "\r\n", "\n")       // Windows
	normalized = strings.ReplaceAll(normalized, "\r", "\n") // Old macOS
	return strings.Split(normalized, "\n")
}

// Builder 把字符串当作一系列片段来操作
type Builder struct {
	segs        []string
	lineSepHint string // 记录换行符："\n"、"\r\n"、"\r"，默认为"\n"
	// 实现了 error 接口, err 不需要导出了
	err errorutil.ExitErrorWithCode
	// trace 每一步操作前的片段记录
	trace        []string
	traceEnabled bool
}

func (b *Builder) traceStep(label string) {
	if b.traceEnabled {
		b.trace = append(b.trace, fmt.Sprintf("\n[%s] segments: %#v\n", label, b.segs))
	}
}

// Error 实现 error 接口，用于链式处理后统一返回断言错误（若存在）
func (b *Builder) Error() error {
	if b.err.Code == 0 {
		return nil
	}
	return &b.err
}

// NewBuilder 用原始字符串初始化一个 Builder
func NewBuilder(s string) *Builder {
	return &Builder{
		segs:         []string{s},
		lineSepHint:  "\n",
		traceEnabled: globalTraceEnabled,
	}
}

func NewBuilderByLines(s string) *Builder {
	var sep string
	switch {
	case strings.Contains(s, "\r\n"):
		sep = "\r\n"
	case strings.Contains(s, "\r"):
		sep = "\r"
	default:
		sep = "\n"
	}
	return &Builder{
		segs:         SplitLines(s),
		lineSepHint:  sep,
		traceEnabled: globalTraceEnabled,
	}
}

// SplitLen 按固定长度切分（最后一段可能不足）
func (b *Builder) SplitLen(n int) *Builder {
	b.traceStep("SplitLen")
	if n <= 0 {
		b.segs = nil
		return b
	}
	var out []string
	for _, seg := range b.segs {
		for i := 0; i < len(seg); i += n {
			end := i + n
			if end > len(seg) {
				end = len(seg)
			}
			out = append(out, seg[i:end])
		}
	}
	b.segs = out
	return b
}

// SplitSep 按指定分隔符拆分
func (b *Builder) SplitSep(sep string) *Builder {
	b.traceStep("SplitSep")
	var out []string
	for _, seg := range b.segs {
		parts := strings.Split(seg, sep)
		out = append(out, parts...)
	}
	b.segs = out
	return b
}

// Regex 用正则在每个片段里抽取所有匹配
func (b *Builder) Regex(pattern string) *Builder {
	b.traceStep("Regex")
	re := regexp.MustCompile(pattern)
	var out []string
	for _, seg := range b.segs {
		matches := re.FindAllString(seg, -1)
		out = append(out, matches...)
	}
	b.segs = out
	return b
}

// RegexGroup 默认或指定 groupIndex（单个或多个）
func (b *Builder) RegexGroup(pattern string, groupIndexes ...int) *Builder {
	b.traceStep("RegexGroup")
	re := regexp.MustCompile(pattern)
	var out []string

	for _, seg := range b.segs {
		matches := re.FindAllStringSubmatch(seg, -1)
		for _, m := range matches {
			// case 1: 无参数 ⇒ 默认取第一个非空 group
			if len(groupIndexes) == 0 {
				for i := 1; i < len(m); i++ {
					if m[i] != "" {
						out = append(out, m[i])
						break
					}
				}
				continue
			}

			// case 2/4: 指定 groupIndex（可多个）
			for _, idx := range groupIndexes {
				if idx >= 1 && idx < len(m) {
					out = append(out, m[idx])
				}
			}
		}
	}
	b.segs = out
	return b
}

// RegexGroupRange 提取闭区间内的所有 group，例如 (1~3)
func (b *Builder) RegexGroupRange(pattern string, from, to int) *Builder {
	b.traceStep("RegexGroupRange")
	re := regexp.MustCompile(pattern)
	var out []string

	for _, seg := range b.segs {
		matches := re.FindAllStringSubmatch(seg, -1)
		for _, m := range matches {
			for i := from; i <= to && i < len(m); i++ {
				if i >= 1 {
					out = append(out, m[i])
				}
			}
		}
	}
	b.segs = out
	return b
}

// Index 只保留第 i 个片段，越界则结果为空
func (b *Builder) Index(i int) *Builder {
	b.traceStep("Index")
	if i < 0 || i >= len(b.segs) {
		b.segs = []string{}
	} else {
		b.segs = []string{b.segs[i]}
	}
	return b
}

// Substr 对每个片段再做一次子串截取
// start<0 或 >=len ⇒ 该段变空，length<=0 ⇒ 取剩余
func (b *Builder) Substr(start, length int) *Builder {
	b.traceStep("Substr")
	var out []string
	for _, seg := range b.segs {
		if start < 0 || start >= len(seg) {
			out = append(out, "")
			continue
		}
		end := start + length
		if length <= 0 || end > len(seg) {
			end = len(seg)
		}
		out = append(out, seg[start:end])
	}
	b.segs = out
	return b
}

// Result 返回所有片段
func (b *Builder) Result() []string {
	b.traceStep("Result")
	if len(b.segs) == 0 {
		// 直接返回一个长度为 0、但非 nil 的空切片
		return []string{}
	}
	return b.segs
}

// Result 返回所有片段
func (b *Builder) ResultSafe() ([]string, error) {
	if err := b.Error(); err != nil {
		return []string{}, err
	}
	return b.Result(), nil
}

// First 快速取第一个片段（常用出口）
func (b *Builder) First() string {
	if len(b.segs) == 0 {
		return ""
	}
	return b.segs[0]
}

// First 判断断言的版本
func (b *Builder) FirstSafe() (string, error) {
	if err := b.Error(); err != nil {
		return "", err
	}
	return b.First(), nil
}

// 出口函数，必须只有一个片段
func (b *Builder) MustFirst(msg string) (string, error) {
	b.AssertCount(msg)
	if err := b.Error(); err != nil {
		return "", err
	}
	return b.First(), nil
}

// Last 快速取最后一个片段
func (b *Builder) Last() string {
	b.traceStep("Last")
	if len(b.segs) == 0 {
		return ""
	}
	return b.segs[len(b.segs)-1]
}

func (b *Builder) LastSafe() (string, error) {
	if err := b.Error(); err != nil {
		return "", err
	}
	return b.Last(), nil
}

// 在 Builder 里放两行，就能链式用
func (b *Builder) Map(f func(string) string) *Builder {
	b.traceStep("Map")
	for i, seg := range b.segs {
		b.segs[i] = f(seg)
	}
	return b
}

func (b *Builder) Filter(predicate func(string) bool) *Builder {
	b.traceStep("Filter")
	var out []string
	for _, seg := range b.segs {
		if predicate(seg) {
			out = append(out, seg)
		}
	}
	b.segs = out
	return b
}

// FilterNonEmpty 过滤掉空行或全空白的文本片段
func (b *Builder) FilterNonEmpty() *Builder {
	return b.Filter(func(s string) bool {
		return strings.TrimSpace(s) != ""
	})
}

// Join 用指定分隔符拼接所有片段
func (b *Builder) Join(sep string) string {
	b.traceStep("Join")
	return strings.Join(b.segs, sep)
}

func (b *Builder) JoinSafe(sep string) (string, error) {
	if err := b.Error(); err != nil {
		return "", err
	}
	return strings.Join(b.segs, sep), nil
}

// 用空字符链接
func (b *Builder) String() string {
	b.traceStep("String")
	return strings.Join(b.segs, "")
}

func (b *Builder) StringSafe() (string, error) {
	if err := b.Error(); err != nil {
		return "", err
	}
	return strings.Join(b.segs, ""), nil
}

func (b *Builder) JoinLines() string {
	b.traceStep("JoinLines")
	return strings.Join(b.segs, b.lineSepHint)
}

func (b *Builder) JoinLinesSafe() (string, error) {
	if err := b.Error(); err != nil {
		return "", err
	}
	return strings.Join(b.segs, b.lineSepHint), nil
}

func (b *Builder) JoinLinesFinal() string {
	b.traceStep("JoinLinesFinal")
	return strings.Join(b.segs, b.lineSepHint) + b.lineSepHint
}

func (b *Builder) JoinLinesFinalSafe() (string, error) {
	if err := b.Error(); err != nil {
		return "", err
	}
	return strings.Join(b.segs, b.lineSepHint) + b.lineSepHint, nil
}

// 断言某一步是否符合预期啊
func (b *Builder) AssertCount(msg string, expected ...int) *Builder {
	// 错误默认0值
	if b.err.Code != 0 {
		// 已有错误则不再继续断言
		// 保证断言的准确性和第一次发生的地方
		return b
	}

	want := 1
	if len(expected) > 0 {
		want = expected[0]
	}

	if len(b.segs) != want {
		b.err = errorutil.ExitErrorWithCode{
			Code:    errorutil.CodeAssertionFailed,
			Message: fmt.Sprintf("msg: %s, segs: %#v, trace: %v", msg, b.segs, b.trace),
			Err:     fmt.Errorf("assertion failed: expected %d item(s), got %d", want, len(b.segs)),
		}
	}
	return b
}

// 超大文本处理器

// LazyBuilder 支持行级懒执行文本处理
type LazyBuilder struct {
	next func() (string, bool) // 获取下一个文本段
}

// NewLazyBuilderFromFile 构造一个按行扫描的懒处理器
func NewLazyBuilderFromFile(filePath string) (*LazyBuilder, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	scanner := bufio.NewScanner(f)

	return &LazyBuilder{
		next: func() (string, bool) {
			if scanner.Scan() {
				return scanner.Text(), true
			}
			return "", false
		},
	}, nil
}

// NewLazyBuilderFromString 用于测试，模拟从多行字符串构造 LazyBuilder
func NewLazyBuilderFromString(content string) *LazyBuilder {
	lines := strings.Split(content, "\n")
	i := 0
	return &LazyBuilder{
		next: func() (string, bool) {
			if i >= len(lines) {
				return "", false
			}
			s := lines[i]
			i++
			return s, true
		},
	}
}

// Map 将每行做变换
func (lb *LazyBuilder) Map(fn func(string) string) *LazyBuilder {
	prev := lb.next
	return &LazyBuilder{
		next: func() (string, bool) {
			s, ok := prev()
			if !ok {
				return "", false
			}
			return fn(s), true
		},
	}
}

// MapTrim 对每个片段应用 strings.TrimSpace，用于清洗首尾空白。
//
// 常用于：
// • 文本规范化（如去除 YAML、JSON、日志片段中的空格噪声）
// • 与 FilterNonEmpty 组合做标准化清洗
func (b *Builder) MapTrim() *Builder {
	return b.Map(strings.TrimSpace)
}

// Filter 过滤行
func (lb *LazyBuilder) Filter(fn func(string) bool) *LazyBuilder {
	prev := lb.next
	return &LazyBuilder{
		next: func() (string, bool) {
			for {
				s, ok := prev()
				if !ok {
					return "", false
				}
				if fn(s) {
					return s, true
				}
			}
		},
	}
}

// FilterStartsWith 过滤出所有以指定前缀开头的片段（区分大小写）
//
// 示例用法：FilterStartsWith("ERR") 只保留前缀为 "ERR" 的内容
func (b *Builder) FilterStartsWith(prefix string) *Builder {
	return b.Filter(func(s string) bool {
		return strings.HasPrefix(s, prefix)
	})
}

// FilterStartsWithFold 过滤出所有以指定前缀开头的片段（忽略大小写）
// 示例：FilterStartsWithFold("err") 会匹配 "ERR404", "err100", "ErrXYZ"
func (b *Builder) FilterStartsWithFold(prefix string) *Builder {
	prefixLower := strings.ToLower(prefix)
	return b.Filter(func(s string) bool {
		return strings.HasPrefix(strings.ToLower(s), prefixLower)
	})
}

func (b *Builder) FilterEndsWith(suffix string) *Builder {
	return b.Filter(func(s string) bool {
		return strings.HasSuffix(s, suffix)
	})
}

func (b *Builder) FilterEndsWithFold(suffix string) *Builder {
	sfx := strings.ToLower(suffix)
	return b.Filter(func(s string) bool {
		return strings.HasSuffix(strings.ToLower(s), sfx)
	})
}

// MapToUpper 将每个片段转换为大写字母（如 "abc" → "ABC"）
func (b *Builder) MapToUpper() *Builder {
	return b.Map(strings.ToUpper)
}

// MapToLower 将每个片段转换为小写字母（如 "ABC" → "abc"）
func (b *Builder) MapToLower() *Builder {
	return b.Map(strings.ToLower)
}

// FilterContains 保留包含指定子串的片段（区分大小写）
func (b *Builder) FilterContains(substr string) *Builder {
	return b.Filter(func(s string) bool {
		return strings.Contains(s, substr)
	})
}

// FilterContainsFold 保留包含指定子串的片段（忽略大小写）
func (b *Builder) FilterContainsFold(substr string) *Builder {
	sub := strings.ToLower(substr)
	return b.Filter(func(s string) bool {
		return strings.Contains(strings.ToLower(s), sub)
	})
}

// RemovePrefix 移除所有片段中以 prefix 开头的部分（区分大小写）
func (b *Builder) RemovePrefix(prefix string) *Builder {
	return b.Map(func(s string) string {
		if strings.HasPrefix(s, prefix) {
			return s[len(prefix):]
		}
		return s
	})
}

// RemovePrefixFold 移除所有片段中以 prefix 开头的部分（忽略大小写）
// 会判断实际是否前缀等于 prefix（不区分大小写），但保留原始大小写内容。
func (b *Builder) RemovePrefixFold(prefix string) *Builder {
	plen := len(prefix)
	plow := strings.ToLower(prefix)
	return b.Map(func(s string) string {
		if len(s) < plen {
			return s
		}
		if strings.ToLower(s[:plen]) == plow {
			return s[plen:]
		}
		return s
	})
}

func (b *Builder) RemoveSuffix(suffix string) *Builder {
	return b.Map(func(s string) string {
		if strings.HasSuffix(s, suffix) {
			return s[:len(s)-len(suffix)]
		}
		return s
	})
}

func (b *Builder) RemoveSuffixFold(suffix string) *Builder {
	sfx := strings.ToLower(suffix)
	slen := len(suffix)
	return b.Map(func(s string) string {
		if len(s) < slen {
			return s
		}
		if strings.ToLower(s[len(s)-slen:]) == sfx {
			return s[:len(s)-slen]
		}
		return s
	})
}

// Regex 匹配每行中的所有匹配项
func (lb *LazyBuilder) Regex(pattern string) *LazyBuilder {
	re := regexp.MustCompile(pattern)
	prev := lb.next
	var matches []string
	var i int

	return &LazyBuilder{
		next: func() (string, bool) {
			for i >= len(matches) {
				s, ok := prev()
				if !ok {
					return "", false
				}
				matches = re.FindAllString(s, -1)
				i = 0
			}
			res := matches[i]
			i++
			return res, true
		},
	}
}

// Each 消费流中每个元素，逐条执行给定函数，常用于流式打印或边处理边写入。
// 只产生副作用，不改变原始数据
// \U0001f4cc 注意：Each 是终端操作，会驱动整个链条执行。
// 与 Collect 不同，它不保留任何中间结果，而是边拿边处理，适合处理大规模数据流。
func (lb *LazyBuilder) Each(fn func(string)) {
	for {
		s, ok := lb.next()
		if !ok {
			break
		}
		fn(s)
	}
}

// Collect 收集所有结果
func (lb *LazyBuilder) Collect() []string {
	var out []string
	for {
		s, ok := lb.next()
		if !ok {
			break
		}
		out = append(out, s)
	}
	return out
}

// Limit 限制最多返回 n 项，后续流会被自动截断
func (lb *LazyBuilder) Limit(n int) *LazyBuilder {
	if n <= 0 {
		// 直接返回空流
		return &LazyBuilder{
			next: func() (string, bool) {
				return "", false
			},
		}
	}
	prev := lb.next
	count := 0
	return &LazyBuilder{
		next: func() (string, bool) {
			if count >= n {
				return "", false
			}
			s, ok := prev()
			if !ok {
				return "", false
			}
			count++
			return s, true
		},
	}
}

// FilterNonEmpty 过滤空行或全是空白字符的内容
func (lb *LazyBuilder) FilterNonEmpty() *LazyBuilder {
	return lb.Filter(func(s string) bool {
		return strings.TrimSpace(s) != ""
	})
}

// MapTrim 对每个字符串执行 strings.TrimSpace，去除首尾空白
//
// 常用于清洗日志字段、去除空格噪声；等价于 .Map(strings.TrimSpace)
func (lb *LazyBuilder) MapTrim() *LazyBuilder {
	return lb.Map(strings.TrimSpace)
}

func (lb *LazyBuilder) FilterStartsWith(prefix string) *LazyBuilder {
	return lb.Filter(func(s string) bool {
		return strings.HasPrefix(s, prefix)
	})
}

func (lb *LazyBuilder) FilterStartsWithFold(prefix string) *LazyBuilder {
	prefixLower := strings.ToLower(prefix)
	return lb.Filter(func(s string) bool {
		return strings.HasPrefix(strings.ToLower(s), prefixLower)
	})
}

func (lb *LazyBuilder) FilterEndsWith(suffix string) *LazyBuilder {
	return lb.Filter(func(s string) bool {
		return strings.HasSuffix(s, suffix)
	})
}

func (lb *LazyBuilder) FilterEndsWithFold(suffix string) *LazyBuilder {
	sfx := strings.ToLower(suffix)
	return lb.Filter(func(s string) bool {
		return strings.HasSuffix(strings.ToLower(s), sfx)
	})
}

// MapToUpper 将流中每个字符串转换为大写（适合日志归一化）
func (lb *LazyBuilder) MapToUpper() *LazyBuilder {
	return lb.Map(strings.ToUpper)
}

// MapToLower 将流中每个字符串转换为小写（常用于大小写不敏感清洗）
func (lb *LazyBuilder) MapToLower() *LazyBuilder {
	return lb.Map(strings.ToLower)
}

// FilterContains 保留流中包含指定子串的行（区分大小写）
func (lb *LazyBuilder) FilterContains(substr string) *LazyBuilder {
	return lb.Filter(func(s string) bool {
		return strings.Contains(s, substr)
	})
}

// FilterContainsFold 保留流中包含指定子串的行（忽略大小写）
func (lb *LazyBuilder) FilterContainsFold(substr string) *LazyBuilder {
	sub := strings.ToLower(substr)
	return lb.Filter(func(s string) bool {
		return strings.Contains(strings.ToLower(s), sub)
	})
}

func (lb *LazyBuilder) RemovePrefix(prefix string) *LazyBuilder {
	return lb.Map(func(s string) string {
		if strings.HasPrefix(s, prefix) {
			return s[len(prefix):]
		}
		return s
	})
}

func (lb *LazyBuilder) RemovePrefixFold(prefix string) *LazyBuilder {
	plen := len(prefix)
	plow := strings.ToLower(prefix)
	return lb.Map(func(s string) string {
		if len(s) < plen {
			return s
		}
		if strings.ToLower(s[:plen]) == plow {
			return s[plen:]
		}
		return s
	})
}

func (lb *LazyBuilder) RemoveSuffix(suffix string) *LazyBuilder {
	return lb.Map(func(s string) string {
		if strings.HasSuffix(s, suffix) {
			return s[:len(s)-len(suffix)]
		}
		return s
	})
}

func (lb *LazyBuilder) RemoveSuffixFold(suffix string) *LazyBuilder {
	sfx := strings.ToLower(suffix)
	slen := len(suffix)
	return lb.Map(func(s string) string {
		if len(s) < slen {
			return s
		}
		if strings.ToLower(s[len(s)-slen:]) == sfx {
			return s[:len(s)-slen]
		}
		return s
	})
}

// TakeWhile 保留流中从头开始连续满足 predicate 条件的元素，
// 一旦遇到第一条不满足的项，即终止整个流后续处理。
// TakeWhile() 是流式处理里的“条件式闸门”，它的意思是：
// “只要满足条件，就继续处理；一旦遇到第一条不满足的，就立刻停止整个链”。
// 它适合提取前缀片段、按条件中止处理、模拟“直到出现错误为止”的场景
func (lb *LazyBuilder) TakeWhile(predicate func(string) bool) *LazyBuilder {
	prev := lb.next
	stopped := false
	return &LazyBuilder{
		next: func() (string, bool) {
			if stopped {
				return "", false
			}
			s, ok := prev()
			if !ok {
				return "", false
			}
			if !predicate(s) {
				stopped = true
				return "", false
			}
			return s, true
		},
	}
}

// SkipWhile 跳过前面所有满足 predicate 条件的元素，
// 一旦遇到第一条“不满足”的，就从该元素开始正常返回后续数据。
// ✅ 使用场景：
// - 跳过无关前缀（如日志头、Banner）
// - 略过结构化之前的数据段
// - 搭配 Filter + Limit 做高性能预处理
func (lb *LazyBuilder) SkipWhile(predicate func(string) bool) *LazyBuilder {
	prev := lb.next
	skipped := false
	return &LazyBuilder{
		next: func() (string, bool) {
			for {
				s, ok := prev()
				if !ok {
					return "", false
				}
				if skipped {
					return s, true
				}
				if !predicate(s) {
					skipped = true
					return s, true
				}
				// 当前满足 predicate，继续跳
			}
		},
	}
}

// Fork2 将当前 LazyBuilder 拆成两个“独立懒链分支”，让它们能同时消费同一份原始流。
// 用于：一次读取，一份数据，被两种不同方式独立处理。
func (lb *LazyBuilder) Fork2() (*LazyBuilder, *LazyBuilder) {
	// 每个分支都要有自己的“独立水管”，不能共用。
	// 64 是缓冲容量，防止读写阻塞（你可以调整）。
	ch1 := make(chan string, 64) // 第一个分支的数据通道
	ch2 := make(chan string, 64) // 第二个分支的数据通道

	// 启动一个 goroutine（后台数据搬运工），负责从原始流里一
	// 条条读取，然后分别发送给两个分支
	go func() {
		defer close(ch1) // 整个读取完毕后，关闭两个通道，告诉消费者没得读了
		defer close(ch2)
		for {
			s, ok := lb.next() // 从源 LazyBuilder 读取下一条
			if !ok {
				return // 没有更多内容了，退出 goroutine
			}
			// 把这一行内容同时发送给两个通道，作为副本
			ch1 <- s
			ch2 <- s
		}
	}()

	// 对上面代码的解释
	// 每读取一条数据，就「广播」给两个通道；
	// 注意：不是复制 channel，而是复制数据行本身；
	// goroutine 只跑一次，并自动结束；
	// defer close(...) 是为了通知消费者“没了”。

	// 构造两个 LazyBuilder 分支，每个都从各自的通道里读取数据
	makeBranch := func(ch <-chan string) *LazyBuilder {
		return &LazyBuilder{
			next: func() (string, bool) {
				s, ok := <-ch // 每次调用 next() 就从该通道读出一条
				return s, ok
			},
		}
	}

	// 对上面代码的解释
	// 这个函数就是分支的构造器；
	// 每个 LazyBuilder 拿到一个独立的 next() 实现逻辑；
	// 只要通道没关、数据没读完，它就会一直给出下一条内容。

	// 最后返回两个独立的 LazyBuilder 实例
	return makeBranch(ch1), makeBranch(ch2)

	// 使用者拿到这两个实例后，就可以像平时一样 .Filter() .Map() .Collect() 分别处理了；
	// 两个分支逻辑上共享源头，但消费上是独立的；
	// 数据只从源读取一次，避免重复 IO 开销。
	//
	// 函数总结:
	// 一个原始 LazyBuilder；
	// .Fork2() 把它拆出 两路独立流；
	// 源流只读一次，副本由 goroutine 广播发出；
	// 每个分支有自己的 next()，只看自己水管里的内容。
	//   源 LazyBuilder
	//        ↓
	//   +----------+
	//   | goroutine|
	//   +----------+
	//    ↓       ↓
	// [ch1]    [ch2]
	//    ↓       ↓
	//分支1     分支2
	// goroutine 是广播者，channel 是专属水管，LazyBuilder 是水龙头本身
}

// ForkN 将当前 LazyBuilder 拆分成 n 个“独立懒链分支”，它们可以并行消费相同的源流内容。
// 注意：每个分支都只能消费一次，不可重复 Collect；且数据流只从原始 LazyBuilder 中读取一次。
func (lb *LazyBuilder) ForkN(n int) []*LazyBuilder {
	if n <= 0 {
		return nil // 如果 n 不合法，直接返回空
	}

	// 创建 n 个缓冲通道，每个对应一个分支
	chs := make([]chan string, n)
	for i := 0; i < n; i++ {
		chs[i] = make(chan string, 64) // 可根据应用调整缓冲大小
	}

	// 启动一个 goroutine 用于从原始 LazyBuilder 中读取流，并广播给每个分支通道
	go func() {
		defer func() {
			// 在读取完毕后，关闭所有通道，通知分支结束
			for _, ch := range chs {
				close(ch)
			}
		}()

		for {
			s, ok := lb.next() // 从源流中获取一行数据
			if !ok {
				return // 读完了，退出
			}
			// 广播模式：将该数据行分别写入所有通道
			for _, ch := range chs {
				ch <- s // 阻塞写入，缓冲区满会等待消费
			}
		}
	}()

	// 构造对应数量的 LazyBuilder 分支，每个绑定自己独立的通道
	branches := make([]*LazyBuilder, n)
	for i := 0; i < n; i++ {
		ch := chs[i] // 当前分支的通道
		branches[i] = &LazyBuilder{
			next: func() (string, bool) {
				s, ok := <-ch // 从通道中读取数据
				return s, ok
			},
		}
	}

	return branches
}

// 上面两个Fork函数
// 功能点	说明
// \U0001f4e6 流广播	所有分支都能完整接收到数据
// \U0001f9ca 支持懒消费	每个分支可以独立处理，延迟执行
// \U0001f6ab 一次性消费	每个分支 .Collect() 后不可再用
// ⚖️ 内存控制	缓冲区大小需根据并发消费情况调整
// ⚠️ 慎用于死链	如果有分支不消费，将导致阻塞或 goroutine 卡死（可加超时或保护机制）
//
//
//

// ForkMapWait 是懒执行版的多标签分发：
//   - router(s) 返回这个行要发送到哪些标签。
//   - branches map 会在后台广播过程中动态创建分支。
//   - done chan 在所有源行广播完成、所有分支通道关闭后才被 close。
//   - 用户在 <-done 后，能安全地访问 branches map 中所有分支。
func (lb *LazyBuilder) ForkMapWait(
	router func(string) []string,
) (
	branches map[string]*LazyBuilder,
	done <-chan struct{},
) {
	// 1. 通道表：标签名 -> 对应 channel
	chMap := make(map[string]chan string)
	// 2. branches 返回表：标签名 -> *LazyBuilder
	branches = make(map[string]*LazyBuilder)
	// 3. 完成信号
	doneCh := make(chan struct{})

	// getOrCreate 确保每个 tag 只创建一次通道和 LazyBuilder
	getOrCreate := func(tag string) chan string {
		if ch, ok := chMap[tag]; ok {
			return ch
		}
		// 新建一个缓冲通道，容量可调
		ch := make(chan string, 64)
		chMap[tag] = ch
		// 封装成 LazyBuilder 分支，next() 从 ch 读取
		branches[tag] = &LazyBuilder{
			next: func() (string, bool) {
				s, ok := <-ch
				return s, ok
			},
		}
		return ch
	}

	// 后台广播 goroutine
	go func() {
		defer func() {
			// 关闭所有分支通道，通知分支流结束
			for _, ch := range chMap {
				close(ch)
			}
			// 发出完成信号
			close(doneCh)
		}()
		// 从源流中依次拉取、路由、广播
		for {
			s, ok := lb.next()
			if !ok {
				return // 源流拉完，退出
			}
			tags := router(s)
			for _, tag := range tags {
				ch := getOrCreate(tag)
				ch <- s
			}
		}
	}()

	return branches, doneCh
}

// ForkMap 会一次性从源流把所有数据拉取完毕，并按 router 的规则分桶，
// 最终返回一个 map[string]*LazyBuilder，保证每个 key 在返回时就已存在。
// 这样，你不再需要在测试里做“异步等待”或担心 nil 分支的问题。
func (lb *LazyBuilder) ForkMap(router func(string) []string) map[string]*LazyBuilder {
	// 1. 读取所有条目，按标签聚集到内存桶 buckets
	//    key: 标签名    value: 收到该标签的所有行
	buckets := make(map[string][]string)

	for {
		s, ok := lb.next() // 从源流拉取一行
		if !ok {
			break // 全部拉完，退出循环
		}
		// 根据 router 决定要把 s 放到哪些标签桶里
		tags := router(s)
		for _, tag := range tags {
			buckets[tag] = append(buckets[tag], s)
		}
	}

	// 2. 将每个桶“打包”成一个 LazyBuilder 分支，写入 result
	result := make(map[string]*LazyBuilder, len(buckets))
	for tag, arr := range buckets {
		// 为该标签创建一个容量刚好的 channel
		ch := make(chan string, len(arr))
		// 先把所有 arr 中的内容推入通道
		for _, line := range arr {
			ch <- line
		}
		close(ch) // 推完就关，消费者会读到 ok==false

		// 用这个通道封装一个 LazyBuilder 分支
		// 由于 ch 在本次循环中是新变量，闭包内用的就是正确的通道
		result[tag] = &LazyBuilder{
			next: func() (string, bool) {
				s, ok := <-ch // 从通道里拿条数据
				return s, ok
			},
		}
	}

	// 3. 返回完整的分支映射，此时所有 key 都已创建完毕！
	return result
}