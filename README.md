# common_tool

Command-line public tools provide a unified input/output interface.

## Tool usage instructions

### asciiplay

This is a micro player that can play text frames and image frames，you can in [asciimotion](https://github.com/qindapao/asciimotion) project to see it's effect.

#### Build tools

1. [ImageMagick](https://imagemagick.org/script/download.php#windows)，Then add it to the environment variables before building：

```bash
export PATH=$PATH:/d/go/bin
export PATH=${PATH}:/d/programs/imagemagick
```

2. rsrc


Installation method：`go install github.com/akavel/rsrc@latest`

After the installation is complete, it should be in：`go env GOPATH`in the path queried by the command. Also add environment variables。

```bash
export PATH=$PATH:/c/users/xx/go/bin
```

To sum up, it might be as follows(Depends on your directory structure):

```bash
echo 'export PATH=${PATH}:/d/Go/bin' >> ~/.bashrc
echo 'export PATH=${PATH}:/d/programes/imagemagick' >> ~/.bashrc
echo 'export PATH=${PATH}:/c/Users/pc/go' >> ~/.bashrc
echo 'export PATH=${PATH}:/c/Users/pc/go/bin' >> ~/.bashrc
source ~/.bashrc
```

The `make` and `gcc` tools are required in the pre-compilation environment. Please install the tools first and then compile.

### gobolt

Tool introduction:

```bash
pc@DESKTOP-0MVRMOU UCRT64 ~
$ gobolt
       .-.                       _             _  _
      (o o)         __ _   ___  | |__    ___  | || |_
      | O \        / _` | / _ \ | '_ \  / _ \ | || __|
      \    \      | (_| || (_) || |_) || (_) || || |_
       `~~~'       \__, | \___/ |_.__/  \___/ |_| \__|
                   |___/

Gobolt v1.0.0+20250619 是一个多功能 CLI 工具，支持 json/ipmi/lspci/setpci 等子命令

Usage:
  gobolt [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  json        处理 JSON 的读取、写入、删除、属性读取
  pcie        PCIe 工具
  ssh         远程 SSH 操作

Flags:
  -h, --help                 help for gobolt
  -l, --log-file string      日志文件名(默认gobolt.log，stdout 表示标准输出) (default "gobolt.log")
  -e, --log-level LogLevel   日志等级(DEBUG/INFO/WARN/ERROR) (default WARN)

Use "gobolt [command] --help" for more information about a command.
```


The tool supports multiple subcommands, and you can find detailed descriptions of each subcommand through its help documentation.

```bash
pc@DESKTOP-0MVRMOU UCRT64 ~
$ gobolt ssh -h
通过 SSH 执行命令、发送或获取文件或者目录

Usage:
  gobolt ssh [command]

Available Commands:
  cmd         执行远端命令，命令跟在最后的 -- 后面，支持命令带参数
  scp_get     从远端服务器接收 文件/目录(scp)
  scp_send    发送 文件/目录 到远端服务器(scp)
  tftp_get    从远端服务器接收 文件/目录(tftp)
  tftp_send   发送 文件/目录 到远端服务器(tftp)

Flags:
  -h, --help   help for ssh

Global Flags:
  -l, --log-file string      日志文件名(默认gobolt.log，stdout 表示标准输出) (default "gobolt.log")
  -e, --log-level LogLevel   日志等级(DEBUG/INFO/WARN/ERROR) (default WARN)

Use "gobolt ssh [command] --help" for more information about a command.

pc@DESKTOP-0MVRMOU UCRT64 ~
$
```


The current command documentation is in Chinese. Maybe when I have time, I will translate them into English.

