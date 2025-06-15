package testutils

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// 定义不同的可执行文件路径
// :TODO: windows系统下直接执行有点问题，不知道原因，
// 不过在gitbash下面使用bash解释器是OK的

type Services struct {
	Name    string
	Command string
	Args    []string
}

// 返回的JSON格式1
// RevRequestTime
// SendRequestTime
// 上面两个键忽略

type ResultCommon struct {
	Action     string `json:"Action"`
	ErrorCode  string `json:"ErrorCode"`
	ErrorMsg   string `json:"ErrorMsg"`
	OutputFile string `json:"OutputFile"`
}

type ResultJSON1 struct {
	ResultCommon
	Data         []string `json:"Data"`
	InputBomCode string   `json:"InputBomCode"`
	TsdDocVer    string   `json:"TsdDocVer"`
}

type ResultJSON2 struct {
	ResultCommon
	Data    string `json:"Data"`
	InputSn string `json:"InputSn"`
}

type ResultJSON3 struct {
	ResultCommon
	Data         []string `json:"Data"`
	InputBomCode string   `json:"InputBomCode"`
}

// 读取 JSON 文件的泛型函数
func readJSONFile[T any](filePath string) (*T, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var result T
	err = json.Unmarshal(file, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// 比较 JSON 数据的泛型函数
func compareJSON[T any](actual, expected *T) bool {
	return reflect.DeepEqual(actual, expected)
}

// Go语言的变成规范要求error一般放右边
// 用例初始化
func testCaseInit() (string, error) {
	// 设置初始工作目录
	curPath, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("获取当前工作目录失败: %v", err)
	}

	// 统一转换路径格式（适用于 Windows 和 Unix）
	curPath = filepath.FromSlash(curPath)
	curDir := filepath.Dir(curPath)
	curDir = filepath.Clean(curDir)

	fmt.Printf("show curDir: %v\n", curDir)

	parts := strings.Split(curDir, string(filepath.Separator))
	partsLen := len(parts)
	if partsLen > 0 && parts[partsLen-1] == "common_tool" {
		return curDir, nil
	}

	if partsLen > 1 && parts[partsLen-1] == "test" && parts[partsLen-2] == "common_tool" {
		return curDir, nil
	}

	return curDir, fmt.Errorf("当前的工作目录 不是 项目根目录 也不是 test 目录")
}

// 进入命令行工具对应目录，并且给可执行文件附权限
func enterCliToolDir(curDir string, execToolPath string) error {
	// 进入命令行工具的目录
	// 根据当前的目录进行切换
	// 这里的目录切换非常奇怪是因为测试库会自动控制进入某些目录
	var swithDir string
	if strings.HasSuffix(curDir, "common_tool") {
		swithDir = "./TestPlat/bin"
	} else if strings.HasSuffix(curDir, "test") {
		swithDir = "../TestPlat/bin"
	} else {
		return fmt.Errorf("当前目录: %s 错误", curDir)
	}
	if err := os.Chdir(swithDir); err != nil {
		return fmt.Errorf("切换工作目录失败: %w", err)
	}

	if err := os.Chmod(execToolPath, 0755); err != nil {
		return fmt.Errorf("给可执行文件 %s 赋 0755 权限失败: %w", execToolPath, err)
	}

	// 删除 *.json就够了，因为*.log的文件可能被主进程占用
	// :TODO: *.json 有时候也会报被占用，无权限删除
	cmd := exec.Command("bash", "-c", "rm -f ./*.json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("删除 临时文件 失败: %w, 命令输出: %s", err, output)
	}

	return nil
}

// 命令行测试通用的泛型函数
func RunCliTest[T any](t *testing.T, cmd Services, expected *T, jsonFile string) {
	originalDir, err := testCaseInit()
	if err != nil {
		t.Fatalf("当前工作目录 %s 不对: %v", originalDir, err)
	}

	// t.Logf("show originalDir: %s", originalDir)

	// 函数结束前恢复工作目录
	defer func() {
		fmt.Println("originalDir: ", originalDir)
		err := os.Chdir(originalDir)
		if err != nil {
			fmt.Println("恢复工作目录失败: %w", err)
		}
	}()
	// 进入命令行工具的目录
	if err := enterCliToolDir(originalDir, cmd.Command); err != nil {
		t.Fatalf("切换工作目录失败: %v", err)
	}

	t.Run(cmd.Name, func(t *testing.T) {
		argStr := strings.Join(cmd.Args, " ")
		bashCmd := fmt.Sprintf(`%s %s`, cmd.Command, argStr)
		execCmd := exec.Command("bash", "-c", bashCmd)
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
		// 如果是用 bash 来执行这里不需要指定路径了
		// execCmd.Dir = filepath.Dir(exePath)
		err := execCmd.Run()
		if err != nil {
			t.Logf("cmd: %s 执行失败: %v", cmd.Name, err)
			t.Fatalf("执行失败: %v", err)
		}

		// 判断返回的JSON文件内容是否符合预期
		actual, err := readJSONFile[T]("result.json")
		if err != nil {
			t.Fatalf("读取 result.json 失败: %v", err)
		}

		// t.Logf("show   actual: %v", actual)
		// t.Logf("show expected: %v", expected)
		// 这个时候编译器已经知道了两个数据的类型，不需要显示指定了
		if compareJSON(actual, expected) {
			t.Logf("JSON 数据匹配")
		} else {
			t.Fatalf("JSON 数据不匹配")
		}
	})
}