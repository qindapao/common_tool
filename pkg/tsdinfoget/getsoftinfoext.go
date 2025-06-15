package tsdinfoget

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"common_tool/pkg/logutil"
	"common_tool/pkg/toolutil"
	"common_tool/pkg/webbase"
)

// TsdParserBase 直接内联进来不要嵌套，保持扁平化
// 直接继承 TsdParserBase 的通用函数
type GetSoftInfoExtParser struct {
	TsdParserBase
	InputBomCode string
	TsdDocVer    string
	Data         []string
}

// 实现 Parser 接口所有函数
func (p *GetSoftInfoExtParser) GetName() string {
	return "GetSoftInfoExt"
}

// 根据命令行的参数,初始化结构体中的字段
// 使用指针持久化对象
func (p *GetSoftInfoExtParser) InitSelf(argsMap map[string]string) error {
	if err := p.TsdParserBase.InitSelf(argsMap); err != nil {
		return err
	}

	// 子类中只处理自己特殊的参数
	p.Action = p.GetName()
	if toolutil.HasAnyKey(argsMap, "-b") {
		p.InputBomCode = argsMap["-b"]
	} else {
		logutil.Error("-s 参数缺失")
		return fmt.Errorf("-s 参数缺失")
	}

	if toolutil.HasAnyKey(argsMap, "-t") {
		p.TsdDocVer = argsMap["-t"]
	} else {
		logutil.Error("-t 参数缺失")
		return fmt.Errorf("-t 参数缺失")
	}

	// 调试打印结构体的内容
	logutil.Debug("show p %v", p)

	// 参数检查
	return nil
}

// chardata 用于提取存文本
type getSoftInfoExtResultData struct {
	XMLName xml.Name `xml:"ResultData"`
	Content string   `xml:",chardata"`
}

// :TODO: 更细化的错误处理
// 使用指针持久化对象
func (p *GetSoftInfoExtParser) ProcessXML() error {
	// 调试打印结构体的内容
	logutil.Debug("show p %v", p)
	inXml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><GetSoftInfoExt><Import><HardwareBomCode>%s</HardwareBomCode><PlatformType></PlatformType><DocumentVersion>%s</DocumentVersion></Import></GetSoftInfoExt>`, p.InputBomCode, p.TsdDocVer)
	tsdInfoGet := TsdInfoGet{}

	logutil.Debug("show inXml:\n%s p.Action: %s", inXml, p.Action)

	// 记录请求发送时间
	p.SendRequestTime = time.Now().Format("2006-01-02 15:04:05.000")

	err := tsdInfoGet.GetInfo(p.Action, inXml)
	// 记录请求结束时间
	p.RevRequestTime = time.Now().Format("2006-01-02 15:04:05.000")
	if err != nil {
		return err
	}

	// 解析xml, 然后放置到 集合中返回
	// 只有ErrorCode为0的时候才取到了值
	p.ErrorCode = tsdInfoGet.ErrorCode
	p.ErrorMsg = tsdInfoGet.ErrorMsg
	if p.ErrorCode != "0" {
		return nil
	}

	logutil.Debug("show ResultData: %v", tsdInfoGet.ResultData)

	var result getSoftInfoExtResultData
	if err := xml.Unmarshal(
		[]byte(tsdInfoGet.ResultData), &result); err != nil {
		return err
	}

	p.Data = strings.Split(
		strings.ReplaceAll(result.Content, "\r\n", "\n"), "\n")
	// 过滤掉空的元素(空字符串或者全是空白字符)
	p.Data = toolutil.Grep(p.Data, `\S+`, false, true)
	return nil
}

func (p *GetSoftInfoExtParser) SaveJSON(subp any) error {
	// 显式调用 `TsdParserBase` 的方法
	logutil.Debug("show p %v", p)
	return p.TsdParserBase.SaveJSON(p)
}

// 首先注册帮助信息(运行的时候早于main执行,并不是编译期执行)
func init() {
	// 注册帮助信息(这里要详细说明入参和出参的格式)
	helpStr := `获取某个TSD配置编码对应版本的软件编程策略信息
        ./com_tsd -a GetSoftInfoExt -t 1.004 -b 03033RWK -o result.json
		返回的 Data 字段是一个列表，保存了所有的配置行`
	webbase.RegisterHelp("GetSoftInfoExt", helpStr)
	// 注册解析器
	webbase.RegisterParser(&GetSoftInfoExtParser{})
}
