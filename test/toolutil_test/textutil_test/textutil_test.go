package textutil_test

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"common_tool/pkg/errorutil"
	"common_tool/pkg/toolutil/textutil"
)

func TestBuilder_SplitSep_Index_First(t *testing.T) {
	raw := "2021-05-20_error-404_log"
	code := textutil.NewBuilder(raw).
		SplitSep("_").
		Index(1). // "error-404"
		SplitSep("-").
		Index(1). // "404"
		First()

	if code != "404" {
		t.Errorf("expected '404', got '%s'", code)
	}
}

func TestBuilder_Filter_Map(t *testing.T) {
	raw := "INFO123, ERR001 , WARNING42 ,ERR500"
	codes := textutil.NewBuilder(raw).
		SplitSep(",").
		Map(strings.TrimSpace).
		Filter(func(s string) bool {
			return strings.HasPrefix(s, "ERR")
		}).
		Result()

	expected := []string{"ERR001", "ERR500"}
	if len(codes) != len(expected) {
		t.Fatalf("expected %d results, got %d", len(expected), len(codes))
	}
	for i := range expected {
		if codes[i] != expected[i] {
			t.Errorf("at index %d: expected '%s', got '%s'", i, expected[i], codes[i])
		}
	}
}
func TestBuilder_AllFunctions(t *testing.T) {
	input := "ERR001, WARN002, INF003,ERR404,    ERR666  , infoX"

	result := textutil.NewBuilder(input).
		SplitSep(",").
		Map(strings.TrimSpace).
		Filter(func(s string) bool {
			return strings.HasPrefix(s, "ERR")
		}).
		Regex(`\d{3}`).
		Substr(1, 2).
		Result()

	expected := []string{"01", "04", "66"}
	if len(result) != len(expected) {
		t.Fatalf("expected %d elements, got %d", len(expected), len(result))
	}
	for i := range expected {
		if result[i] != expected[i] {
			t.Errorf("at index %d: expected '%s', got '%s'", i, expected[i], result[i])
		}
	}
}

func TestBuilder_RegexStartEnd(t *testing.T) {
	input := `user_abc123
user_def456
user_789xyz
begin:match
end:endthing
notEndX`

	tests := []struct {
		pattern  string
		expected []string
	}{
		{`^user_\w+`, []string{"user_abc123", "user_def456", "user_789xyz"}}, // 所有以 user_ 开头的行
		{`^begin:\w+`, []string{"begin:match"}},                              // 精确匹配开头
		{`.+endthing$`, []string{"end:endthing"}},                            // 匹配以 endthing 结尾
		{`^not.*`, []string{"notEndX"}},                                      // 自定义行匹配
	}

	for _, tt := range tests {
		got := textutil.NewBuilderByLines(input).
			Regex(tt.pattern).
			Result()
		if len(got) != len(tt.expected) {
			t.Errorf("pattern '%s': expected %d results, got %d", tt.pattern, len(tt.expected), len(got))
			continue
		}
		for i := range tt.expected {
			if got[i] != tt.expected[i] {
				t.Errorf("pattern '%s' at %d: expected '%s', got '%s'", tt.pattern, i, tt.expected[i], got[i])
			}
		}
	}
}

func TestBuilder_RegexGroup(t *testing.T) {
	input := `ERR404 OK200 WARN301
host[addr=192.168.0.1, port=8080]
user:alice id:42, user:bob id:84`

	tests := []struct {
		name     string
		builder  *textutil.Builder
		expected []string
	}{
		{
			name: "DefaultExtractFirstNonEmptyGroup",
			builder: textutil.NewBuilderByLines(input).
				RegexGroup(`ERR(\d{3})|OK(\d{3})|WARN(\d{3})`),
			expected: []string{"404", "200", "301"},
		},
		{
			name: "ExtractSingleGroup",
			builder: textutil.NewBuilderByLines(input).
				RegexGroup(`addr=(\d+\.\d+\.\d+\.\d+), port=(\d+)`, 2),
			expected: []string{"8080"},
		},
		{
			name: "ExtractMultipleGroups",
			builder: textutil.NewBuilderByLines(input).
				RegexGroup(`addr=(\d+\.\d+\.\d+\.\d+), port=(\d+)`, 1, 2),
			expected: []string{"192.168.0.1", "8080"},
		},
		{
			name: "ExtractNonConsecutiveGroups",
			builder: textutil.NewBuilderByLines(input).
				RegexGroup(`user:(\w+) id:(\d+)`, 1),
			expected: []string{"alice", "bob"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.builder.Result()
			if len(got) != len(tt.expected) {
				t.Fatalf("expected %d results, got %d", len(tt.expected), len(got))
			}
			for i := range tt.expected {
				if got[i] != tt.expected[i] {
					t.Errorf("index %d: expected '%s', got '%s'", i, tt.expected[i], got[i])
				}
			}
		})
	}
}

func TestBuilder_RegexGroupRange(t *testing.T) {
	input := `point(12,34)
box[width=200,height=100]
rgb(255,128,64)
log[ERR001] - caused by [timeout]
coords: x=1 y=2 z=3`

	tests := []struct {
		name     string
		pattern  string
		from     int
		to       int
		expected []string
	}{
		{
			name:     "ExtractTupleGroups",
			pattern:  `point\((\d+),(\d+)\)`,
			from:     1,
			to:       2,
			expected: []string{"12", "34"},
		},
		{
			name:     "ExtractBoxDimensions",
			pattern:  `width=(\d+),height=(\d+)`,
			from:     1,
			to:       2,
			expected: []string{"200", "100"},
		},
		{
			name:     "ExtractRGB",
			pattern:  `rgb\((\d+),(\d+),(\d+)\)`,
			from:     1,
			to:       3,
			expected: []string{"255", "128", "64"},
		},
		{
			name:     "ExtractNestedGroups",
			pattern:  `\[(ERR\d+)\].*\[(\w+)\]`,
			from:     1,
			to:       2,
			expected: []string{"ERR001", "timeout"},
		},
		{
			name:     "ExtractPartialCoords",
			pattern:  `x=(\d+)\s+y=(\d+)\s+z=(\d+)`,
			from:     2,
			to:       3,
			expected: []string{"2", "3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := textutil.NewBuilderByLines(input).
				RegexGroupRange(tt.pattern, tt.from, tt.to).
				Result()

			if len(got) != len(tt.expected) {
				t.Fatalf("expected %d results, got %d", len(tt.expected), len(got))
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("index %d: expected '%s', got '%s'", i, tt.expected[i], got[i])
				}
			}
		})
	}
}

func TestBuilder_FilterAndJoinLines(t *testing.T) {
	input := `INFO Startup complete
ERR001 Disk failure
WARN Network unstable
ERR500 Timeout
INFO All services healthy`

	expected := `ERR001 Disk failure
ERR500 Timeout`

	got := textutil.NewBuilderByLines(input).
		Filter(func(s string) bool {
			return strings.HasPrefix(s, "ERR")
		}).
		JoinLines()

	if got != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, got)
	}
}

func TestBuilder_RegexFilterAndJoinLines(t *testing.T) {
	input := `host:localhost port:8080
host:192.168.1.1 port:443
config:invalid entry
host:10.0.0.1 port:22`

	expected := `host:192.168.1.1 port:443
host:10.0.0.1 port:22`

	got := textutil.NewBuilderByLines(input).
		Regex(`^host:\d+\.\d+\.\d+\.\d+ port:\d+$`).
		JoinLines()

	if got != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, got)
	}
}

func TestBuilder_RegexLinesToConfig(t *testing.T) {
	input := `# CONFIG
key:alpha value:1
key:beta value:2
note:this is a comment
key:gamma value:3`

	expected := `key:alpha value:1
key:beta value:2
key:gamma value:3`

	got := textutil.NewBuilderByLines(input).
		Regex(`^key:\w+ value:\d+$`).
		JoinLines()

	if got != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, got)
	}
}

func TestBuilder_FilterThenJoinLines_CRLF(t *testing.T) {
	input := "ok:123\r\nerr:404\r\nwarn:789\r\nerr:500"

	builder := textutil.NewBuilderByLines(input).
		Filter(func(s string) bool {
			return strings.HasPrefix(s, "err:")
		})

	joined := builder.JoinLines()
	expected := "err:404\r\nerr:500"

	if joined != expected {
		t.Errorf("JoinLines() after Filter with CRLF failed:\nexpected: %q\ngot:      %q", expected, joined)
	}
}

func TestBuilder_JoinLinesFinal(t *testing.T) {
	input := "a:1\r\nb:2\r\nc:3\r\n" // 包含尾部 CRLF
	expected := "b:2\r\nc:3\r\n"

	got := textutil.NewBuilderByLines(input).
		Filter(func(s string) bool {
			return strings.HasPrefix(s, "b:") || strings.HasPrefix(s, "c:")
		}).
		JoinLinesFinal()

	if got != expected {
		t.Errorf("expected with trailing newline:\n%q\ngot:\n%q", expected, got)
	}
}

func TestBuilder_AssertCount_MultipleLevels(t *testing.T) {
	input := `token:abc123
token:def456
user:admin
env:prod`

	builder := textutil.NewBuilderByLines(input).
		Regex(`^token:\w+`). // 匹配 token 行，有2行
		AssertCount("token 行必须有2个", 2).
		Regex(`^user:\w+`). // 匹配 user 行，刚好 1 个
		AssertCount("user 行必须唯一").
		Regex(`^env:\w+`). // 匹配 env 行，刚好 1 个
		AssertCount("env 行必须唯一")

	// 最终统一判断错误
	if err := builder.Error(); err != nil {
		// 在 Go 里，接口变量即使底层是某种类型，也需要显式断言才能访问原始值的字段或方法。
		// Go 是静态类型语言，不允许“推测性访问”接口下的字段，所以：
		// 你声明的是 err error，接口视角；
		// 编译器不能肯定 err 里面一定是你那个 *ExitErrorWithCode；
		// 所以需要类型断言：
		e := err.(*errorutil.ExitErrorWithCode)
		t.Logf("预期断言命中 ✅: Code=%d, Message=%s, Detail=%v", e.Code, e.Message, e.Err)
	} else {
		t.Errorf("预期断言应命中，但没有触发 ❌")
	}
}

func TestBuilder_AssertCount_MultipleLevels_OK(t *testing.T) {
	input := `env:prod token:abc123
user:admin
env:prod
log:started
token:def456
`
	builder := textutil.NewBuilderByLines(input).
		Filter(func(s string) bool {
			return strings.HasPrefix(s, "env:")
		}).
		Map(func(s string) string {
			return strings.ToUpper(s)
		}).                      // 变成 "TOKEN:ABC123" 和 "TOKEN:DEF456"
		Regex(`.*TOKEN:ABC123`). // 只剩一条
		AssertCount("必须有且只有一个 ABC123").
		Regex(`^ENV:.+$`). // 匹配 env 行
		AssertCount("必须存在一个 env 行").
		Regex(`.*123$`). // 匹配 user 行
		AssertCount("必须123结尾")

	if err := builder.Error(); err != nil {
		e := err.(*errorutil.ExitErrorWithCode)
		t.Errorf("不应触发断言错误，但捕获到: Code=%d, Message=%s, Detail=%v", e.Code, e.Message, e.Err)
	} else {
		t.Log("所有断言均满足 ✅")
	}
}

func TestLazyBuilderBasic_Filter_Map(t *testing.T) {
	mockData := `
error: failed to load
info: retrying
error: disk not found
warn: timeout
`
	builder := textutil.NewLazyBuilderFromString(mockData).
		Filter(func(s string) bool {
			return strings.HasPrefix(s, "error")
		}).
		Map(strings.ToUpper)

	got := builder.Collect()
	want := []string{
		"ERROR: FAILED TO LOAD",
		"ERROR: DISK NOT FOUND",
	}

	if len(got) != len(want) {
		t.Fatalf("Expected %d results, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Mismatch at %d: want %q, got %q", i, want[i], got[i])
		}
	}
}

func TestLazyBuilder_Regex(t *testing.T) {
	mockData := `
result: OK
result: FAIL
result: OK
result: ERR123
xx
yy
`
	builder := textutil.NewLazyBuilderFromString(mockData).
		Regex(`OK|FAIL|ERR\d+`)

	got := builder.Collect()
	want := []string{"OK", "FAIL", "OK", "ERR123"}

	if len(got) != len(want) {
		t.Fatalf("Expected %d matches, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Mismatch at %d: want %q, got %q", i, want[i], got[i])
		}
	}
}

func TestLazyBuilder_Each(t *testing.T) {
	mockData := `foo
bar
baz
`
	builder := textutil.NewLazyBuilderFromString(mockData)

	var result []string
	builder.Each(func(s string) {
		result = append(result, s+"-done")
	})

	want := []string{"foo-done", "bar-done", "baz-done"}

	for i := range want {
		if result[i] != want[i] {
			t.Errorf("Mismatch at %d: want %q, got %q", i, want[i], result[i])
		}
	}
}

func TestLazyBuilder_Regex_Each_Report(t *testing.T) {
	mockLog := `
[INFO] service started
[DEBUG] ping ok
[ERROR] ERR1001: database timeout
[ERROR] ERR3021: auth failed
[WARN] something flaky
[ERROR] not formatted
[ERROR] ERR5000: unknown error
`
	var reported []string

	textutil.NewLazyBuilderFromString(mockLog).
		Regex(`ERR\d+`).
		Each(func(code string) {
			reported = append(reported, code)
			// 假设上报错误码：ReportError(code)
			fmt.Printf("Reporting: %s\n", code)
		})

	want := []string{"ERR1001", "ERR3021", "ERR5000"}
	if len(want) != len(reported) {
		t.Fatalf("expected %d reports, got %d", len(want), len(reported))
	}
	for i := range want {
		if reported[i] != want[i] {
			t.Errorf("mismatch at %d: want %s, got %s", i, want[i], reported[i])
		}
	}
}

func TestLazyBuilder_Limit(t *testing.T) {
	mock := `
one
two
three
four
five
`

	builder := textutil.NewLazyBuilderFromString(mock).
		FilterNonEmpty().
		Limit(3)

	got := builder.Collect()
	want := []string{"one", "two", "three"}

	if len(got) != len(want) {
		t.Fatalf("Limit failed: expected %d items, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Mismatch at %d: want %q, got %q", i, want[i], got[i])
		}
	}
}

func TestLazyBuilder_TakeWhile(t *testing.T) {
	mock := `
[INFO] service start
[DEBUG] warmup OK
[DEBUG] connected
[ERROR] boom!
[INFO] post-error still running
`

	got := textutil.NewLazyBuilderFromString(mock).
		FilterNonEmpty().
		TakeWhile(func(s string) bool {
			return strings.HasPrefix(s, "[INFO]") || strings.HasPrefix(s, "[DEBUG]")
		}).
		Collect()

	want := []string{
		"[INFO] service start",
		"[DEBUG] warmup OK",
		"[DEBUG] connected",
	}

	if len(got) != len(want) {
		t.Fatalf("expected %d, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("mismatch at %d: want %q, got %q", i, want[i], got[i])
		}
	}
}

func TestLazyBuilder_SkipWhile(t *testing.T) {
	mock := `
# --- HEADER ---
# some debug msg
# log started at 14:00
ERR1001 database failed
ERR2102 auth error
INFO recovery ok
`

	got := textutil.NewLazyBuilderFromString(mock).
		FilterNonEmpty().
		SkipWhile(func(s string) bool {
			return strings.HasPrefix(s, "#")
		}).
		Collect()

	want := []string{
		"ERR1001 database failed",
		"ERR2102 auth error",
		"INFO recovery ok",
	}

	if len(got) != len(want) {
		t.Fatalf("Expected %d items, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Mismatch at %d: want %q, got %q", i, want[i], got[i])
		}
	}
}

func TestBuilder_RemovePrefix(t *testing.T) {
	got := textutil.NewBuilderByLines(`[ERROR] boom
[INFO] ok
[Error] case mismatch`).
		MapTrim().
		RemovePrefix("[ERROR] ").
		Result()

	want := []string{
		"boom",
		"[INFO] ok",             // 保留
		"[Error] case mismatch", // 保留（大小写不符）
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Mismatch at %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBuilder_RemovePrefixFold(t *testing.T) {
	got := textutil.NewBuilderByLines(`[ERROR] boom
[Error] mismatch
[INFO] running`).
		MapTrim().
		RemovePrefixFold("[error] ").
		Result()

	want := []string{
		"boom",           // ✔️ 移除
		"mismatch",       // ✔️ 移除
		"[INFO] running", // ❌ 保留（不匹配）
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Mismatch at %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLazyBuilder_RemovePrefix(t *testing.T) {
	got := textutil.NewLazyBuilderFromString(`[WARN] too hot
[DEBUG] cooling
[warn] mismatch`).
		MapTrim().
		RemovePrefix("[WARN] ").
		Collect()

	want := []string{
		"too hot",
		"[DEBUG] cooling", // 保留
		"[warn] mismatch", // 保留
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Mismatch at %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLazyBuilder_RemovePrefixFold(t *testing.T) {
	got := textutil.NewLazyBuilderFromString(`[warn] overheating
[WARN] critical
[info] running`).
		MapTrim().
		RemovePrefixFold("[warn] ").
		Collect()

	want := []string{
		"overheating",
		"critical",
		"[info] running", // 保留
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Mismatch at %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBuilder_RemoveSuffix(t *testing.T) {
	got := textutil.NewBuilderByLines(`log.OK
foo.done
bar.DONE
baz.done`).
		MapTrim().
		RemoveSuffix(".done").
		Result()

	want := []string{
		"log.OK",
		"foo",      // ✔️ 移除（完全匹配）
		"bar.DONE", // ❌ 保留（大小写不符）
		"baz",      // ✔️ 移除
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("RemoveSuffix failed at %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBuilder_RemoveSuffixFold(t *testing.T) {
	got := textutil.NewBuilderByLines(`log.OK
foo.done
bar.DONE
baz.DoNe`).
		MapTrim().
		RemoveSuffixFold(".done").
		Result()

	want := []string{
		"log.OK",
		"foo", // ✔️ 移除
		"bar", // ✔️ 移除
		"baz", // ✔️ 移除
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("RemoveSuffixFold failed at %d: got %q, want %q", i, got[i], want[i])
		}
	}
}
func TestLazyBuilder_RemoveSuffix(t *testing.T) {
	got := textutil.NewLazyBuilderFromString(`[INFO] finished.ok
[WARN] closed.DONE
[AUDIT] flushed.done`).
		MapTrim().
		RemoveSuffix(".done").
		Collect()

	want := []string{
		"[INFO] finished.ok",
		"[WARN] closed.DONE", // ❌ 不移除
		"[AUDIT] flushed",    // ✔️ 精准匹配
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Lazy RemoveSuffix mismatch at %d: got %q, want %q", i, got[i], want[i])
		}
	}
}
func TestLazyBuilder_RemoveSuffixFold(t *testing.T) {
	got := textutil.NewLazyBuilderFromString(`INFO: closed.DONE
DEBUG: task.done
WARN: retry.DoNe
trace.ok`).
		MapTrim().
		RemoveSuffixFold(".done").
		Collect()

	want := []string{
		"INFO: closed",
		"DEBUG: task",
		"WARN: retry",
		"trace.ok", // ✔️ 保留
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Lazy RemoveSuffixFold mismatch at %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLazyBuilder_Fork2(t *testing.T) {
	src := textutil.NewLazyBuilderFromString(`[INFO] start
[WARN] cpu high
[ERROR] boom
[WARN] disk slow`).
		MapTrim().
		FilterNonEmpty()

	warn, err := src.Fork2()

	warns := warn.FilterStartsWith("[WARN]").Collect()
	errs := err.FilterStartsWith("[ERROR]").Collect()

	if len(warns) != 2 || len(errs) != 1 {
		t.Fatalf("Expected 2 warns and 1 error, got %d / %d", len(warns), len(errs))
	}
}

func TestLazyBuilder_ForkN(t *testing.T) {
	src := textutil.NewLazyBuilderFromString(`
[INFO] service started
[ERROR] db crash
[WARN] high CPU
[DEBUG] trace ready
[ERROR] out of memory
`).
		MapTrim().
		FilterNonEmpty()

	branches := src.ForkN(3) // 拆成 3 个懒链

	if len(branches) != 3 {
		t.Fatalf("Expected 3 branches, got %d", len(branches))
	}

	// 分别提取各自关注内容
	errors := branches[0].FilterStartsWith("[ERROR]").Collect()
	warns := branches[1].FilterContainsFold("warn").Collect()
	debugs := branches[2].FilterContains("[DEBUG]").Collect()

	// 验证内容正确
	wantErrs := []string{"[ERROR] db crash", "[ERROR] out of memory"}
	wantWarns := []string{"[WARN] high CPU"}
	wantDebugs := []string{"[DEBUG] trace ready"}

	check := func(name string, got, want []string) {
		if len(got) != len(want) {
			t.Errorf("%s: expected %d items, got %d", name, len(want), len(got))
			return
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s mismatch at %d: want %q, got %q", name, i, want[i], got[i])
			}
		}
	}

	check("errors", errors, wantErrs)
	check("warns", warns, wantWarns)
	check("debugs", debugs, wantDebugs)
}

// TestLazyBuilder_ForkMap_LazyVersion 演示懒执行版 ForkMapWait 的使用。
// 先启动分支广播，然后 <-done 等待完成，再安全断言各分支内容。
func TestLazyBuilder_ForkMap_LazyVersion(t *testing.T) {
	raw := `
[INFO] start
[DEBUG] preparing
[ERROR] disk failed
[WARN] cpu high
[ERROR] OOM
[DEBUG] detail
`

	// 1) 构造源 LazyBuilder，并做 MapTrim + FilterNonEmpty 清洗
	src := textutil.NewLazyBuilderFromString(raw).
		MapTrim().
		FilterNonEmpty()

	// 2) 路由函数：对每种日志级别独立 append，不提前 return
	router := func(s string) []string {
		var tags []string
		if strings.Contains(s, "[INFO]") {
			tags = append(tags, "info")
		}
		if strings.Contains(s, "[DEBUG]") {
			tags = append(tags, "debug")
		}
		if strings.Contains(s, "[WARN]") {
			tags = append(tags, "warn")
		}
		if strings.Contains(s, "[ERROR]") {
			// ERROR 同时发给 "error" 和 "alert"
			tags = append(tags, "error", "alert")
		}
		return tags
	}

	// 3) 调用 ForkMapWait，拿到 branches map 和 完成信号 done
	branches, done := src.ForkMapWait(router)

	// 4) 等待广播和分支注册完成
	<-done

	// 5) 确认所有期望分支都已创建
	wantTags := []string{"error", "alert", "info", "debug", "warn"}
	sort.Strings(wantTags)
	var gotTags []string
	for tag := range branches {
		gotTags = append(gotTags, tag)
	}
	sort.Strings(gotTags)
	if len(gotTags) != len(wantTags) {
		t.Fatalf("branches tags mismatch: got %v, want %v", gotTags, wantTags)
	}
	for i, tag := range wantTags {
		if gotTags[i] != tag {
			t.Fatalf("branch tag[%d] = %q, want %q", i, gotTags[i], tag)
		}
	}

	// 6) 对每个分支调用 Collect 并断言数据
	assertEqual(t, "error", branches["error"].Collect(), []string{
		"[ERROR] disk failed",
		"[ERROR] OOM",
	})
	assertEqual(t, "alert", branches["alert"].Collect(), []string{
		"[ERROR] disk failed",
		"[ERROR] OOM",
	})
	assertEqual(t, "info", branches["info"].Collect(), []string{
		"[INFO] start",
	})
	assertEqual(t, "debug", branches["debug"].Collect(), []string{
		"[DEBUG] preparing",
		"[DEBUG] detail",
	})
	assertEqual(t, "warn", branches["warn"].Collect(), []string{
		"[WARN] cpu high",
	})
}

// TestLazyBuilder_ForkMap_Eager 确保 ForkMap 一次性分桶后，
// branches map 里能立刻拿到所有标签，且 Collect 能正确返回数据。
func TestLazyBuilder_ForkMap_Eager(t *testing.T) {
	raw := `
[INFO] start
[DEBUG] preparing
[ERROR] disk failed
[WARN] cpu high
[ERROR] OOM
[DEBUG] detail
`

	// 构造源懒链，做必要的 Trim/Filter
	src := textutil.NewLazyBuilderFromString(raw).
		MapTrim().
		FilterNonEmpty()

	// router 函数，不用 else/return，所有标签独立 append
	router := func(s string) []string {
		var tags []string
		if strings.Contains(s, "[INFO]") {
			tags = append(tags, "info")
		}
		if strings.Contains(s, "[DEBUG]") {
			tags = append(tags, "debug")
		}
		if strings.Contains(s, "[WARN]") {
			tags = append(tags, "warn")
		}
		if strings.Contains(s, "[ERROR]") {
			tags = append(tags, "error", "alert")
		}
		return tags
	}

	// 1. 调用 ForkMap —— 此时就能拿到完整的 branches map
	branches := src.ForkMap(router)

	// 2. 立刻断言 map 中的 key 都存在，无需等待
	wantTags := []string{"error", "alert", "info", "debug", "warn"}
	for _, tag := range wantTags {
		if _, ok := branches[tag]; !ok {
			t.Fatalf("分支 %q 应该存在，但没有在 branches 中找到", tag)
		}
	}

	// 3. Collect 并断言每条流内容
	assertEqual(t, "error", branches["error"].Collect(), []string{"[ERROR] disk failed", "[ERROR] OOM"})
	assertEqual(t, "alert", branches["alert"].Collect(), []string{"[ERROR] disk failed", "[ERROR] OOM"})
	assertEqual(t, "info", branches["info"].Collect(), []string{"[INFO] start"})
	assertEqual(t, "debug", branches["debug"].Collect(), []string{"[DEBUG] preparing", "[DEBUG] detail"})
	assertEqual(t, "warn", branches["warn"].Collect(), []string{"[WARN] cpu high"})
}

// assertEqual 是个小断言，放到测试文件底部
func assertEqual(t *testing.T, name string, got, want []string) {
	if len(got) != len(want) {
		t.Fatalf("[%s] 长度不符：got %d, want %d", name, len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%s] 索引 %d 不符：got %q, want %q", name, i, got[i], want[i])
		}
	}
}

// :TODO:
// 在流中途「插队」做统计、打点（Tap 操作）
// 支持分支超时或落后时自动丢弃（Dead-branch 保护）
// 增加对无限流的 back-pressure 控制
// 可以往 ForkMapWait 里加「缓冲队列溢出保护」
// 「动态 maxBuffer」「限速」等高级特性。或者做个高阶 LazyRouter，
// 支持按标签 count/slice/paginate……
// 总之，下一站是「流式中间件」或「事件总线」级别
