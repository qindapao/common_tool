package errorutil

import (
	"encoding/json"
	"errors"
	"fmt"
)

const (
	CodeSuccess = 0 // 成功执行

	// 60–69: 用户输入或调用错误
	CodeInvalidUsage = 64 // 命令行用法错误（参数不合法等）
	CodeMissingInput = 65 // 缺失必须输入（如文件、路径等）
	CodeInvalidData  = 66 // 用户输入格式错误（数据非法）
	CodePermission   = 67 // 权限不足或操作被拒绝

	CodeAssertionFailed = 68 // 断言失败（数量不符、语义不符等）

	// 70–79: 程序自身或依赖错误
	CodeCmdFailed   = 70 // 命令执行失败（catch-all）
	CodeSSHError    = 71 // SSH 层错误（连接失败、channel 拒绝等）
	CodeIOError     = 72 // 文件或设备读写失败
	CodeInternalErr = 74 // 内部 bug、panic、未捕捉异常

	// 80–89: 外部服务或系统相关错误（可扩展）
	CodeConfigError = 80 // 配置文件有误或缺失
	CodeTempFail    = 81 // 临时失败（可重试的错误）
)

// omitempty 的作用是空字段不出现
type ExitErrorWithCode struct {
	Code        int    `json:"code"`                    // 框架/业务层级错误码
	Message     string `json:"message,omitempty"`       // 可读消息
	CmdExitCode int    `json:"cmd_exit_code,omitempty"` // 原始命令的退出码（仅在执行命令时填充）
	Err         error  `json:"-"`
}

func (e *ExitErrorWithCode) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("Exit with code: %d", e.Code)
}

func (e *ExitErrorWithCode) Unwrap() error {
	return e.Err
}

func NewExitError(code int, err error) error {
	return &ExitErrorWithCode{Code: code, Err: err}
}

// os.Exit(errorutil.ExitCodeFromError(err))
func ExitCodeFromError(err error) int {
	if err == nil {
		return CodeSuccess
	}
	var exitErr *ExitErrorWithCode
	if errors.As(err, &exitErr) {
		return exitErr.Code
	}
	return CodeInternalErr
}

// msg := errorutil.UserMessage(err)
func UserMessage(err error) string {
	if e, ok := err.(*ExitErrorWithCode); ok && e.Message != "" {
		return e.Message
	}
	return ""
}

// 判断当前的错误是否是带退出码的错误
func HasExitCode(err error) bool {
	var exitErr *ExitErrorWithCode
	return errors.As(err, &exitErr)
}

// 提取原始错误
func RootError(err error) error {
	for {
		unwrapped := errors.Unwrap(err)
		if unwrapped == nil {
			return err
		}
		err = unwrapped
	}
}

// 带错误消息的错误，如果有命令退出码需要注入
func NewExitErrorWithMessage(code int, message string, err error, cmdExitCode ...int) error {
	e := &ExitErrorWithCode{Code: code, Message: message, Err: err}
	if len(cmdExitCode) > 0 {
		e.CmdExitCode = cmdExitCode[0]
	}
	return e
}

func (e *ExitErrorWithCode) JSON() string {
	type jsonErr struct {
		Code        int    `json:"code"`
		Message     string `json:"message,omitempty"`
		Err         string `json:"error,omitempty"`
		CmdExitCode int    `json:"cmd_exit_code,omitempty"`
	}

	data := jsonErr{
		Code:        e.Code,
		Message:     e.Message,
		CmdExitCode: e.CmdExitCode,
	}
	if e.Err != nil {
		data.Err = e.Err.Error()
	}
	jsonBytes, _ := json.Marshal(data)
	return string(jsonBytes)
}

// NewCmdFailure 用于构造命令执行失败的结构化错误
func NewCmdFailure(cmdExitCode int, message string, err error) error {
	return &ExitErrorWithCode{
		Code:        CodeCmdFailed, // 框架定义的“命令失败”状态
		CmdExitCode: cmdExitCode,   // 实际命令返回码（如 exit 42）
		Message:     message,       // 可读信息
		Err:         err,           // 错误链原始错误
	}
}

func FormatErrorAndCode(err error) (string, int) {
	// 默认退出码是内部错误
	code := CodeInternalErr
	switch e := err.(type) {
	case *ExitErrorWithCode:
		// 优先使用命令原始退出码（如果设置），否则用结构化错误码
		if e.CmdExitCode != 0 {
			code = e.CmdExitCode
		} else {
			code = e.Code
		}
		return e.JSON(), code
	default:
		// 构建一个临时 ExitErrorWithCode 对象，并直接调用其 JSON() 方法
		return (&ExitErrorWithCode{
			Code:    code,
			Message: "未知错误",
			Err:     err,
		}).JSON(), code
	}
}