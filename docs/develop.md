# develop

开发说明

## 编译环境初始化

1. 初始化项目

```bash
go mod init
go mod tidy
```

2. 设置共享包的位置

```bash
# windows的环境下
go env -w GOMODCACHE=C:\\Users\\q00208337\\go\\pkg\\mod
go mod tidy
```

3. 获取共享包

1. 查询一个包的所有可用版本：

```bash
go list -m -versions github.com/Enflick/gosoap
```

2. 在 `go.mod` 中写入

```bash
require github.com/Enflick/gosoap v1.0.3
```

3. 其实如果代码中如果直接使用了，即使 `go.mod` 中没有内容，执行 `go mod tidy` 的时候会自动下载。

## 自动化测试

```bash
# 进入 test 目录或者项目根目录执行下面的命令
# 如果不加v那么只会简单显示包级测试简单结果
go test ./... -v
# 可以通过grep过滤检查是否有执行失败
go test ./... -v | grep FAIL
# 在第一个失败的时候就停止
go test ./... -failfast -v
# 基准测试
go test -bench=. -benchmem
# 对上面的基准测试的解释
# go test：运行当前包下的所有测试文件（即 *_test.go 文件）
# -bench=.：匹配所有以 BenchmarkXxx 命名的基准测试函数
# . 是正则中的“匹配任意”，所以表示“跑所有 benchmark”
# -benchmem 告诉 Go 工具“顺便测一下内存分配情况
# 结果的解释:
#     ns/op	每次操作耗时（纳秒）
#     B/op	每次操作分配的内存（字节）
#     allocs/op	每次操作发生多少次内存分配（通常越少越好）
```

### 常用的基准测试指令


```bash
go test -bench=.	                    跑所有基准测试
go test -bench=MyFunc	                只 benchmark BenchmarkMyFunc
go test -bench=My.*	                    跑所有 BenchmarkMy... 开头的测试
go test -bench=. -benchtime=5s	        每个测试持续 5 秒（默认是 1s）
go test -bench=. -count=5	            跑 5 次并取平均值
go test -bench=. -benchmem -run=^$	    跳过所有单元测试（只跑 benchmark）
go test -bench=. -cpuprofile=cpu.out    导出 CPU profile 可视化分析
```

## 循环依赖检查

```bash
# 安装goda命令
go install github.com/loov/goda@latest

# 执行生成依赖关系图
goda graph ./... | dot -Tpng -o deps.png
# 生成 dot 文件
goda graph ./... > deps.dot

```

## 编译

一、纯 Go 代码（CGO_ENABLED=0）

打开 cmd 或 PowerShell，切到项目目录；

设置环境变量，然后直接 go build：

Windows CMD 下：

bat
set CGO_ENABLED=0
set GOOS=linux
set GOARCH=arm
set GOARM=7       ← 如果要编译给 ARMv7（32 位）跑的。  
go build -o myapp-linux-arm   ./cmd/yourapp
PowerShell 下：

powershell
$env:CGO_ENABLED = "0"
$env:GOOS        = "linux"
$env:GOARCH      = "arm"
$env:GOARM       = "7"
go build -o myapp-linux-arm ./cmd/yourapp
运行完就能在 Windows 下得到可在 Linux/ARMv7 上跑的 myapp-linux-arm 二进制。

如果要编 aarch64（ARM64），把 GOARCH=arm64，去掉 GOARM 即可。

二、依赖 CGO（调用 C 库） Go 自带的交叉编译不带交叉 C 编译器，这时得：

安装一个 Linux-ARM 的交叉工具链（比如在 MSYS2 或从你信任的源拿到 arm-linux-gnueabi-gcc、aarch64-linux-gnu-gcc 等）；

设置额外环境：

powershell
$env:CGO_ENABLED = "1"
$env:GOOS        = "linux"
$env:GOARCH      = "arm"
$env:GOARM       = "7"
$env:CC_for_target = "arm-linux-gnueabi-gcc"    # 或者 aarch64-linux-gnu-gcc
go build -o myapp-linux-arm ./cmd/yourapp
这样 Go 在编译时就会调用你指定的交叉 C 编译器，把 C 部分也一起编进最终 ELF。

总结

纯 Go 最简单：CGO_ENABLED=0 GOOS=linux GOARCH=arm[ GOARM=...] go build；

需要 cgo 时，多装个交叉 gcc，并把 CC_for_target（或 CC) 指向它。

照着上面配，就能在 Windows 一键生成 Linux ARM 的 Go 可执行。
