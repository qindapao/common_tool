package sshclient

import (
	"bufio"
	"bytes"
	"common_tool/pkg/logutil"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"common_tool/pkg/errorutil"
	"common_tool/pkg/sh"

	"github.com/dustin/go-humanize"
	"github.com/pkg/sftp"
	"github.com/povsister/scp"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

// :TODO: 当前使用 TFTP 获取和发送文件不会保留元数据
// client.Stat(path)：获取远程文件的元信息（权限、时间戳等）
// os.Chtimes(localPath, atime, mtime)：设置本地文件的访问时间和修改时间
// os.Chmod(localPath, mode)：设置本地文件的权限
const (
	SCP_RECEIVE_FLAG  = "scp_get"
	SCP_SEND_FLAG     = "scp_send"
	TFTP_RECEIVE_FLAG = "tftp_get"
	TFTP_SEND_FLAG    = "tftp_send"
)

// 缓冲区大小为 64MB
const bufferSize = 64 * 1024 * 1024

type CLIOptionsBase struct {
	Host     string
	Port     string
	Timeout  time.Duration
	User     string
	Password string
}

type CLIOptionsCmd struct {
	CLIOptionsBase
	Cmd string
}

type CLIOptionsTransfer struct {
	CLIOptionsBase
	LocalPath  string
	RemotePath string
	Direction  string
}

// 新增通用连接函数
func createSSHClient(base CLIOptionsBase) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User:            base.User,
		Auth:            []ssh.AuthMethod{ssh.Password(base.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         base.Timeout,
	}
	return ssh.Dial("tcp", base.Host+":"+base.Port, config)
}

type SSHSFTPClient struct {
	SSH  *ssh.Client
	SFTP *sftp.Client
}

// 重构SFTP初始化函数（供发送/接收共用）
func createSSHAndSFTP(base CLIOptionsBase) (*SSHSFTPClient, error) {
	sshClient, err := createSSHClient(base)
	if err != nil {
		return nil, err
	}

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		sshClient.Close()
		return nil, err
	}

	return &SSHSFTPClient{SSH: sshClient, SFTP: sftpClient}, nil
}

func (c *SSHSFTPClient) Close() {
	c.SFTP.Close()
	c.SSH.Close()
}

// :TODO: 目录默认远端是LINUX服务器，并且用的bash/zsh
// 其它情况不行
func (opts *CLIOptionsCmd) RunRemoteCommand() error {
	conn, err := createSSHClient(opts.CLIOptionsBase)
	if err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("创建 session 失败: %w", err)
	}
	defer session.Close()

	// 执行命令并检查状态
	// 这里是否绑定到 os.Stdout os.Stderr 更好？
	var outputBuf bytes.Buffer
	var errorBuf bytes.Buffer
	session.Stdout = &outputBuf
	session.Stderr = &errorBuf

	err = session.Run(opts.Cmd)
	output := outputBuf.String() + errorBuf.String()
	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			// 将原始输出打印出来（按流分发）
			os.Stdout.Write(outputBuf.Bytes())
			os.Stderr.Write(errorBuf.Bytes())

			return errorutil.NewCmdFailure(
				exitErr.ExitStatus(),
				fmt.Sprintf("远程命令失败（ExitCode=%d）：%s", exitErr.ExitStatus(), output),
				err,
			)
		}

		// 非命令失败类型错误，依然打印原始输出
		os.Stdout.Write(outputBuf.Bytes())
		os.Stderr.Write(errorBuf.Bytes())

		return errorutil.NewExitErrorWithMessage(
			errorutil.CodeSSHError,
			fmt.Sprintf("SSH 执行失败: %s", output),
			err,
		)
	}

	// 成功路径：只打印 stdout
	os.Stdout.Write(outputBuf.Bytes())
	return nil
}

// ./gobolt ssh scp_get -H 10.43.111.20 -U root -P xx -p 50956 -L ./ -R '//home/xx/xx.txt'
// gitbash 传路径要用 // 不然会被自动转换
func (opts *CLIOptionsTransfer) SendDirOrFileToRemote() error {
	switch opts.Direction {
	case SCP_SEND_FLAG:
		return opts.sendViaSCP()
	case TFTP_SEND_FLAG:
		return opts.sendViaSFTP()
	default:
		return fmt.Errorf("暂不支持的下载方式: %s", opts.Direction)
	}
}

func (opts *CLIOptionsTransfer) ReceiveDirOrFileFromRemote() error {
	switch opts.Direction {
	case SCP_RECEIVE_FLAG:
		return opts.receiveViaSCP()
	case TFTP_RECEIVE_FLAG:
		return opts.receiveViaSFTP()
	default:
		return fmt.Errorf("暂不支持的下载方式: %s", opts.Direction)
	}
}

func (opts *CLIOptionsTransfer) sendViaSFTP() error {
	client, err := createSSHAndSFTP(opts.CLIOptionsBase)
	if err != nil {
		return fmt.Errorf("SFTP 初始化失败: %w", err)
	}
	defer client.Close()

	// 判断 src 是目录还是文件
	info, err := os.Stat(opts.LocalPath)
	if err != nil {
		return fmt.Errorf("源路径无效: %w", err)
	}

	if info.IsDir() {
		logutil.Debug("is dir, show LocalPath: %v RemotePath: %v", opts.LocalPath, opts.RemotePath)
		return uploadDirectory(client.SFTP, opts.LocalPath, opts.RemotePath)
	} else {
		logutil.Debug("is file, show LocalPath: %v RemotePath: %v", opts.LocalPath, opts.RemotePath)
		// 转换为绝对路径
		absSrc, _ := filepath.Abs(opts.LocalPath)
		absDir := filepath.Dir(absSrc)
		fmt.Println()
		fmt.Println("Uploading from:", absDir)
		fmt.Println("          to:  ", path.Dir(opts.RemotePath))
		fmt.Println()
		return uploadFile(client.SFTP, opts.LocalPath, opts.RemotePath, absDir)
	}
}

// 进度监控结构体
type ProgressReader struct {
	Filename          string
	Reader            io.Reader
	bytesRead         int64
	total             int64
	startTime         time.Time
	lastReport        time.Time
	lastBytesReported int64
}

func (r *ProgressReader) Read(p []byte) (n int, err error) {
	// 初始化 startTime（只设置一次）
	if r.startTime.IsZero() {
		now := time.Now()
		r.startTime = now
		r.lastReport = now
	}

	n, err = r.Reader.Read(p)
	r.bytesRead += int64(n)

	now := time.Now()
	elapsed := now.Sub(r.lastReport).Seconds()

	if elapsed > 1.0 {
		r.printProgress(elapsed)
	}

	if err == io.EOF {
		r.printProgress(elapsed)
		fmt.Fprintln(os.Stdout)
	}
	return
}

func (r *ProgressReader) printProgress(elapsedSec float64) {
	var percent float64
	if r.total == 0 {
		percent = 100.0
	} else {
		percent = float64(r.bytesRead) / float64(r.total) * 100
	}

	// 瞬时速度
	var speed float64
	delta := r.bytesRead - r.lastBytesReported
	if elapsedSec > 0 {
		speed = float64(delta) / elapsedSec / (1024 * 1024)
	}

	// 平均速度
	totalElapsed := time.Since(r.startTime).Seconds()
	var avgSpeed float64
	if totalElapsed > 0 {
		avgSpeed = float64(r.bytesRead) / totalElapsed / (1024 * 1024)
	}

	fmt.Fprintf(os.Stdout,
		"\r%-40s %-18s %-18s %-18s %-18s",
		"["+r.Filename+"]",
		fmt.Sprintf("Progress: %.2f%%", percent),
		fmt.Sprintf("(%s/%s)",
			humanize.Bytes(uint64(r.bytesRead)),
			humanize.Bytes(uint64(r.total))),
		fmt.Sprintf("avg:(%.2f MB/s)", avgSpeed),
		fmt.Sprintf("cur:(%.2f MB/s)", speed),
	)

	r.lastReport = time.Now()
	r.lastBytesReported = r.bytesRead
}

type FlushWriter struct {
	*bufio.Writer
}

// 牺牲性能，但是节省内存
func (fw *FlushWriter) Write(p []byte) (n int, err error) {
	n, err = fw.Writer.Write(p)
	if err == nil && n > 0 {
		fw.Writer.Flush()
	}
	return
}

// 上传单个文件
// :TODO: 文件的路径太长了，是否可以提取公共前缀？
// :TODO: 当前TFTP的上传很慢(只有600K)，原因未知
func uploadFile(
	client *sftp.Client, localPath, remotePath, srcDir string) error {
	// 打开本地文件
	srcFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open local file failed: %w", err)
	}
	defer srcFile.Close()

	// 创建带有进度监控的Reader
	localRel, _ := filepath.Abs(localPath)
	relPath, err := filepath.Rel(srcDir, localRel)
	if err != nil {
		return fmt.Errorf(
			"can not get real path from srcDir: %v, localPath: %v", srcDir, localPath)
	}
	fileInfo, _ := srcFile.Stat()
	progressReader := &ProgressReader{
		Reader:     bufio.NewReaderSize(srcFile, bufferSize),
		total:      fileInfo.Size(),
		lastReport: time.Now(),
		Filename:   relPath,
	}

	// 创建远程文件
	dstFile, err := client.Create(remotePath)
	if err != nil {
		return fmt.Errorf("create remote file failed: %w", err)
	}
	defer dstFile.Close()

	// 使用带缓冲的写入
	bufWriter := bufio.NewWriterSize(dstFile, bufferSize)
	if _, err := io.Copy(&FlushWriter{Writer: bufWriter}, progressReader); err != nil {
		return fmt.Errorf("file transfer failed: %w", err)
	}

	// :TODO: 后续可以优化 goroutine 每秒刷新一次，最后再刷新一次收尾
	// 我们当前已经做了实时写，可靠性更好一些，暂时不用做并发
	// 不然文件太大一次性写入可能不保险，内存可能也不够
	// go func() {
	// 	ticker := time.NewTicker(time.Second)
	// 	defer ticker.Stop()
	// 	for range ticker.C {
	// 		bufWriter.Flush()
	// 	}
	// }()

	// 最后刷新一次保证数据完全写入
	if err := bufWriter.Flush(); err != nil {
		return fmt.Errorf("file Flush failed: %w", err)
	}

	if stat, err := os.Stat(localPath); err == nil {
		if err := client.Chmod(remotePath, stat.Mode()); err != nil {
			logutil.Warn("设置文件权限失败: %v", err)
		}
	}

	// 保留修改时间
	if stat, err := os.Stat(localPath); err == nil {
		mtime := stat.ModTime()
		atime := time.Now() // 访问时间通常设为当前时间
		if err := client.Chtimes(remotePath, atime, mtime); err != nil {
			logutil.Warn("设置文件时间失败: %v", err)
		}
	}

	// 文件属性修改失败不报错
	return nil
}

// 递归上传目录，忽略符号链接
func uploadDirectory(client *sftp.Client, srcDir, destDir string) error {
	absDir, _ := filepath.Abs(srcDir)
	fmt.Println()
	fmt.Println("Uploading from:", absDir)
	fmt.Println("          to:  ", destDir)
	fmt.Println()

	return filepath.Walk(srcDir, func(localPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 忽略符号链接
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, localPath)
		if err != nil {
			return fmt.Errorf("计算相对路径失败: %w", err)
		}

		// 统一为 Linux 风格路径
		relPath = filepath.ToSlash(relPath)

		remotePath := path.Join(destDir, relPath)
		logutil.Debug(
			"destDir: %v srcDir: %v localpath: %v relPath: %v remotePath: %v",
			destDir, srcDir, localPath, relPath, remotePath)

		if info.IsDir() {
			logutil.Debug("remotePath: %v, is dir.", remotePath)
			return client.MkdirAll(remotePath)
		}

		return uploadFile(client, localPath, remotePath, absDir)
	})
}

func tryDownloadAsDirectoryViaScp(
	client *scp.Client, remotePath, localPath string, opt *scp.DirTransferOption) error {
	// 检查本地路径是否已存在
	_, statErr := os.Stat(localPath)
	createdLocalDir := false

	if os.IsNotExist(statErr) {
		if err := os.MkdirAll(localPath, 0755); err != nil {
			return fmt.Errorf("创建本地目录失败: %w", err)
		}
		createdLocalDir = true
	}

	// 拉取目录
	dirErr := client.CopyDirFromRemote(remotePath, localPath, opt)
	if dirErr == nil {
		return nil
	}

	// 如果失败，并且我们创建了本地目录，则删除它
	if createdLocalDir {
		if rmErr := os.RemoveAll(localPath); rmErr != nil {
			logutil.Warn("拉取失败，尝试删除创建的本地目录失败: %v", rmErr)
		} else {
			logutil.Debug("拉取失败，删除了新创建的本地目录: %v", localPath)
		}
	}

	return fmt.Errorf("目录拉取失败: %w", dirErr)
}

func (opts *CLIOptionsTransfer) receiveViaSCP() error {
	sshConf := scp.NewSSHConfigFromPassword(opts.User, opts.Password)
	sshConf.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	sshConf.Timeout = opts.Timeout

	client, err := scp.NewClient(opts.Host+":"+opts.Port, sshConf, &scp.ClientOption{})
	if err != nil {
		return fmt.Errorf("创建 SCP 客户端失败: %w", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	// logutil.Error("show RemotePath: %v", opts.RemotePath)

	// 尝试作为目录拉取
	dirErr := tryDownloadAsDirectoryViaScp(client, opts.RemotePath, opts.LocalPath, &scp.DirTransferOption{
		Context:      ctx,
		PreserveProp: true,
	})
	if dirErr == nil {
		return nil
	}

	// 尝试作为文件拉取
	fileErr := client.CopyFileFromRemote(opts.RemotePath, opts.LocalPath, &scp.FileTransferOption{
		Context:      ctx,
		PreserveProp: true,
	})
	if fileErr == nil {
		return nil
	}

	return fmt.Errorf("拉取失败:\n作为目录错误: %v\n作为文件错误: %v RemotePath: %v", dirErr, fileErr, opts.RemotePath)
}

func (opts *CLIOptionsTransfer) sendViaSCP() error {
	sshConf := scp.NewSSHConfigFromPassword(opts.User, opts.Password)
	sshConf.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	sshConf.Timeout = opts.Timeout

	client, err := scp.NewClient(opts.Host+":"+opts.Port, sshConf, &scp.ClientOption{})
	if err != nil {
		return fmt.Errorf("创建 SCP 客户端失败: %w", err)
	}
	defer client.Close()

	fi, err := os.Stat(opts.LocalPath)
	if err != nil {
		return fmt.Errorf("本地路径无效: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	if fi.IsDir() {
		// 保持和 TFTP 的传参方式一致,所以下面的代码就不要了
		// 自动保留目录名，例如 /data/conf/ -> /tmp/conf/
		// baseName := filepath.Base(opts.LocalPath)
		// // 默认远端是linux格式的
		// remoteTarget := path.Join(opts.RemotePath, baseName)
		// logutil.Debug("is dir")
		// logutil.Debug("baseName: %v, LocalPath: %v, remoteTarget: %v", baseName, opts.LocalPath, opts.RemotePath)
		// logutil.Debug("remoteTarget: %v", remoteTarget)

		return client.CopyDirToRemote(opts.LocalPath, opts.RemotePath, &scp.DirTransferOption{
			Context: ctx,
			// 保留原始文件的属性，比如：
			// 文件权限
			// 修改时间
			// 访问时间
			PreserveProp: true,
		})
	}

	logutil.Debug("is file")
	logutil.Debug("LocalPath: %v RemotePath: %v", opts.LocalPath, opts.RemotePath)
	return client.CopyFileToRemote(opts.LocalPath, opts.RemotePath, &scp.FileTransferOption{
		Context:      ctx,
		PreserveProp: true,
	})
}

func (opts *CLIOptionsTransfer) receiveViaSFTP() error {
	client, err := createSSHAndSFTP(opts.CLIOptionsBase)
	if err != nil {
		return fmt.Errorf("SFTP 初始化失败: %w", err)
	}
	defer client.Close()

	info, err := client.SFTP.Stat(opts.RemotePath)
	if err != nil {
		return fmt.Errorf("远端路径无效: %w", err)
	}

	if info.IsDir() {
		return downloadDirectory(client.SFTP, opts.RemotePath, opts.LocalPath)
	}

	localAbsPath, _ := filepath.Abs(opts.LocalPath)
	localAbsDir := filepath.Dir(localAbsPath)
	fmt.Println()
	fmt.Println("Downloading from:", path.Dir(opts.RemotePath))
	fmt.Println("            to:  ", localAbsDir)
	fmt.Println()

	return downloadFile(client.SFTP, opts.RemotePath, opts.LocalPath, path.Dir(opts.RemotePath))
}

// RelativeRemotePath 计算 remoteFile 相对于 remoteRootDir 的路径
// remoteRootDir 和 remoteFile 都应该是类 Unix 风格路径（用 path 包而非 filepath）
func RelativeRemotePath(remoteRootDir, remoteFile string) (string, error) {
	remoteRootDir = path.Clean(remoteRootDir)
	remoteFile = path.Clean(remoteFile)

	if !strings.HasPrefix(remoteFile, remoteRootDir) {
		return "", fmt.Errorf("远程文件 [%s] 不在远程根目录 [%s] 下", remoteFile, remoteRootDir)
	}

	relPath := strings.TrimPrefix(remoteFile, remoteRootDir)
	relPath = strings.TrimPrefix(relPath, "/") // 移除多余的斜杠
	return relPath, nil
}

func downloadFile(
	client *sftp.Client, remoteFile, localPath, remoteRootDir string) error {
	srcFile, err := client.Open(remoteFile)
	if err != nil {
		return fmt.Errorf("打开远端文件失败: %w", err)
	}
	defer srcFile.Close()

	relRemotePath, _ := RelativeRemotePath(remoteRootDir, remoteFile)

	fileInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("获取远端文件信息失败: %w", err)
	}

	// 进度监控 + 带缓冲的读取器
	progressReader := &ProgressReader{
		Reader:     bufio.NewReaderSize(srcFile, bufferSize),
		total:      fileInfo.Size(),
		lastReport: time.Now(),
		Filename:   relRemotePath,
	}

	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("创建本地目录失败: %w", err)
	}

	dstFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("创建本地文件失败: %w", err)
	}
	defer dstFile.Close()

	// 加上缓冲写入（更高效）
	bufWriter := bufio.NewWriterSize(dstFile, bufferSize)
	if _, err := io.Copy(&FlushWriter{Writer: bufWriter}, progressReader); err != nil {
		return fmt.Errorf("复制文件失败: %w", err)
	}
	// 写入完数据后刷新缓冲
	if err := bufWriter.Flush(); err != nil {
		return fmt.Errorf("写入缓冲区失败: %w", err)
	}

	// 还原权限
	if err := os.Chmod(localPath, fileInfo.Mode()); err != nil {
		logutil.Warn("设置文件权限失败: %v", err)
	}

	// 还原时间
	mtime := fileInfo.ModTime()
	atime := time.Now()
	if err := os.Chtimes(localPath, atime, mtime); err != nil {
		logutil.Warn("设置文件时间失败: %v", err)
	}

	// 文件属性修改失败不报错
	return nil
}

func downloadDirectory(client *sftp.Client, remoteDir, localRoot string) error {
	localAbsDir, _ := filepath.Abs(localRoot)
	fmt.Println()
	fmt.Println("Downloading from:", remoteDir)
	fmt.Println("            to  :", localAbsDir)
	fmt.Println()

	// 校验远端路径合法性
	info, err := client.Stat(remoteDir)
	if err != nil {
		return fmt.Errorf("远端目录无法访问: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("远端路径不是目录: %s", remoteDir)
	}

	// 确保本地根路径存在
	if err := os.MkdirAll(localRoot, 0755); err != nil {
		return fmt.Errorf("创建本地根目录失败: %w", err)
	}

	// 遍历远端目录结构
	logutil.Debug("show localRoot: %v", localRoot)
	walker := client.Walk(remoteDir)
	for walker.Step() {
		if err := walker.Err(); err != nil {
			return err
		}

		remotePath := walker.Path()
		logutil.Debug("show remoteDir: %v remotePath: %v", remoteDir, remotePath)
		relPath, _ := filepath.Rel(remoteDir, remotePath) // 计算相对路径
		localPath := filepath.Join(localRoot, relPath)    // 拼接目标路径

		logutil.Debug("show relPath: %v localPath: %v", relPath, localPath)

		info := walker.Stat()
		if info == nil {
			continue
		}

		if info.Mode()&os.ModeSymlink != 0 {
			continue // 跳过符号链接
		}

		if info.IsDir() {
			logutil.Debug("is dir, localPath: %v", localPath)
			if err := os.MkdirAll(localPath, 0755); err != nil {
				return fmt.Errorf("创建本地目录失败: %w", err)
			}
			continue
		}

		if err := downloadFile(client, remotePath, localPath, remoteDir); err != nil {
			return fmt.Errorf("下载文件失败 [%s]: %w", remotePath, err)
		}
	}
	return nil
}

func SSHCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssh",
		Short: "远程 SSH 操作",
		Long:  "通过 SSH 执行命令、发送或获取文件或者目录",
	}

	cmd.AddCommand(sshCmdCmd())
	cmd.AddCommand(sshTftpSendCmd())
	cmd.AddCommand(sshTftpReceiveCmd())
	cmd.AddCommand(sshScpSendCmd())
	cmd.AddCommand(sshScpReceiveCmd())

	return cmd
}

// ssh 子命令下面的子命令
// 所有的root的全局设置都是默认继承的
func sshCmdCmd() *cobra.Command {
	opts := &CLIOptionsCmd{}

	cmd := &cobra.Command{
		Use:   "cmd",
		Short: "执行远端命令，命令跟在最后的 -- 后面，支持命令带参数",
		Long: `执行远端命令，命令跟在最后的 -- 后面，支持命令带参数
举例:
gobolt ssh cmd -H 10.43.111.20 -U xx -P xx -p 50956 -- ls -l "/home/"
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 获取要执行的命令
			opts.Cmd = sh.BuildCommandLineQuoted(args)
			// 连接 SSH，执行命令
			return opts.RunRemoteCommand()
		},
	}

	bindCommonSSHFlags(cmd, &opts.CLIOptionsBase)

	// :TODO: 哪些参数必须带需要检查
	// 注意检查的时候只需要检查长选项，短选项只是语法糖的作用，长选项和短选项
	// 只要有一个存在就可以了
	// cmd.MarkFlagRequired("host")

	return cmd
}

func sshTftpSendCmd() *cobra.Command {
	return newSSHTransferCommand(TFTP_SEND_FLAG, "发送 文件/目录 到远端服务器(tftp)", func(opts *CLIOptionsTransfer) error {
		opts.Direction = TFTP_SEND_FLAG
		return opts.SendDirOrFileToRemote()
	})
}

func sshTftpReceiveCmd() *cobra.Command {
	return newSSHTransferCommand(TFTP_RECEIVE_FLAG, "从远端服务器接收 文件/目录(tftp)", func(opts *CLIOptionsTransfer) error {
		opts.Direction = TFTP_RECEIVE_FLAG
		return opts.ReceiveDirOrFileFromRemote()
	})
}

func sshScpSendCmd() *cobra.Command {
	return newSSHTransferCommand(SCP_SEND_FLAG, "发送 文件/目录 到远端服务器(scp)", func(opts *CLIOptionsTransfer) error {
		opts.Direction = SCP_SEND_FLAG
		// 这里也可以加区分逻辑，比如后续区分协议方式
		return opts.SendDirOrFileToRemote()
	})
}

func sshScpReceiveCmd() *cobra.Command {
	return newSSHTransferCommand(SCP_RECEIVE_FLAG, "从远端服务器接收 文件/目录(scp)", func(opts *CLIOptionsTransfer) error {
		opts.Direction = SCP_RECEIVE_FLAG
		return opts.ReceiveDirOrFileFromRemote()
	})
}

func newSSHTransferCommand(name, short string, action func(*CLIOptionsTransfer) error) *cobra.Command {
	opts := &CLIOptionsTransfer{}

	cmd := &cobra.Command{
		Use:   name,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			return action(opts)
		},
	}

	bindCommonSSHFlags(cmd, &opts.CLIOptionsBase)
	cmd.Flags().StringVarP(&opts.LocalPath, "local", "L", "", "本地 文件/目录 路径")
	cmd.Flags().StringVarP(&opts.RemotePath, "remote", "R", "", "远端 文件/目录 路径")

	return cmd
}

// 通用参数绑定
func bindCommonSSHFlags(cmd *cobra.Command, base *CLIOptionsBase) {
	cmd.Flags().StringVarP(&base.Host, "host", "H", "", "目标主机")
	cmd.Flags().StringVarP(&base.Port, "port", "p", "22", "SSH 链接端口，默认 22")
	cmd.Flags().DurationVarP(&base.Timeout, "timeout", "t", 20*time.Second, "连接超时，默认20秒(1s 2m2s 1h32m12s 20ms 这样的格式)")
	cmd.Flags().StringVarP(&base.User, "user", "U", "", "用户名，默认为空")
	cmd.Flags().StringVarP(&base.Password, "password", "P", "", "密码，默认为空")
}
