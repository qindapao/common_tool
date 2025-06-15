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

3. 文件的编译初始化函数 `init()` 中注册子解析器。
