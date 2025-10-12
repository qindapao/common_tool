# common_tool

命令行公共工具，提供统一的输入输出入口

## 工具使用说明

命令行执行`./com_mes -h`会打印工具的帮助信息。

## 工具扩展

### com_mes

1. 注册帮助信息

参考 `webinfoget` 包中的 `init()` 函数的内容。

2. 实现`Parser`接口的所有函数

范例见 `demoaction.go`，以及已经实现了的子解析器。

- `SaveJSON` 方法使用默认即可。
- `ProcessXML` 方法需要单独实现，需要定义一个内部使用的独立结构体处理 `ResultData`信息。
- `InitSelf` 方法需要单独实现，可以借用内嵌结构体中的对应方法。
- `GetName` 方法需要单独实现。

3. 文件的运行初始化函数 `init()` 中注册子解析器。

### asciiplay

#### 构建工具

1. [ImageMagick](https://imagemagick.org/script/download.php#windows)，然后构建前要添加到环境变量中：

```bash
export PATH=$PATH:/d/go/bin
export PATH=${PATH}:/d/programs/imagemagick
```

2. rsrc


安装方法：`go install github.com/akavel/rsrc@latest`

按照完成后，应该在：`go env GOPATH`命令查询到的路径中。同样加入环境变量。

```bash
export PATH=$PATH:/c/users/xx/go/bin
```

总结起来可能是下面这样(依你的目录结构决定):

```bash
echo 'export PATH=${PATH}:/d/Go/bin' >> ~/.bashrc
echo 'export PATH=${PATH}:/d/programes/imagemagick' >> ~/.bashrc
echo 'export PATH=${PATH}:/c/Users/pc/go' >> ~/.bashrc
echo 'export PATH=${PATH}:/c/Users/pc/go/bin' >> ~/.bashrc
source ~/.bashrc
```

编译前环境中需要有`make`和`gcc`工具，请先安装工具后再编译。

