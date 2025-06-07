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
go env -w GOMODCACHE=C:\\Users\\pc\\go\\pkg\\mod
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

4. 如果开发环境是windows，可能要注意下项目代码的路径和go的安装路径最好是在一个盘符下，防止某些工具无法正确找到go的编译二进制文件。


