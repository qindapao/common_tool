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
```
