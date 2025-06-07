package webinfoget

import (
	"encoding/xml"
	"fmt"
	"time"

	"web_tool/pkg/logutil"
	"web_tool/pkg/toolutil"
)

// ParserBase 直接内联进来不要嵌套，保持扁平化
// 直接继承 ParserBase 的通用函数
type GetBarcodeBOMParser struct {
	ParserBase
	InputSn string
	Data    string
}

// 实现 Parser 接口所有函数
func (p *GetBarcodeBOMParser) GetName() string {
	return "GetBarcodeBOM"
}

// 根据命令行的参数,初始化结构体中的字段
// 使用指针持久化对象
func (p *GetBarcodeBOMParser) InitSelf(argsMap map[string]string) error {
	if err := p.ParserBase.InitSelf(argsMap); err != nil {
		return err
	}

	// 子类中只处理自己特殊的参数
	p.Action = p.GetName()
	if toolutil.HasAnyKey(argsMap, "-s") {
		p.InputSn = argsMap["-s"]
	} else {
		logutil.Error("-s 参数缺失")
		return fmt.Errorf("-s 参数缺失")
	}

	// 调试打印结构体的内容
	logutil.Debug("show p %v", p)

	// 参数检查
	return nil
}

// <ResultData><Bom>02314MAX</Bom><Rev></Rev><Sn>2102314MAX250100002E</Sn>
// </ResultData>
// <Message><ErrorCode>0</ErrorCode><ErrorMsg>Success</ErrorMsg>
// </Message>
// 定义结构体，映射 XML 结构
type barcodeBomResultData struct {
	XMLName xml.Name `xml:"ResultData"`
	Bom     string   `xml:"Bom"`
	Rev     string   `xml:"Rev"`
	Sn      string   `xml:"Sn"`
}

// :TODO: 更细化的错误处理
// 使用指针持久化对象
func (p *GetBarcodeBOMParser) ProcessXML() error {
	// 调试打印结构体的内容
	logutil.Debug("show p %v", p)

	inXml := fmt.Sprintf(
		`<?xml version="1.0" encoding="UTF-8"?>`+
			`<%s>`+
			`<Import>`+
			`<Barcode>%s</Barcode>`+
			`</Import>`+
			`</%s>`, p.Action, p.InputSn, p.Action)
	webInfoGet := WebInfoGet{}

	logutil.Debug("show inXml:\n%s p.Action: %s", inXml, p.Action)

	// 记录请求发送时间
	p.SendRequestTime = time.Now().Format("2006-01-02 15:04:05.000")

	err := webInfoGet.GetInfo(p.Action, inXml)
	// 记录请求结束时间
	p.RevRequestTime = time.Now().Format("2006-01-02 15:04:05.000")
	if err != nil {
		return err
	}

	// 解析xml, 然后放置到 集合中返回
	// 只有ErrorCode为0的时候才取到了值
	p.ErrorCode = webInfoGet.ErrorCode
	p.ErrorMsg = webInfoGet.ErrorMsg
	if p.ErrorCode != "0" {
		return nil
	}

	var result barcodeBomResultData
	if err := xml.Unmarshal(
		[]byte(webInfoGet.ResultData), &result); err != nil {
		return err
	}

	p.Data = result.Bom
	return nil
}

func (p *GetBarcodeBOMParser) SaveJSON(subp interface{}) error {
	// 显式调用 `ParserBase` 的方法
	logutil.Debug("show p %v", p)
	return p.ParserBase.SaveJSON(p)
}

// 首先注册帮助信息(编译的时候插入的)
func init() {
	// 注册帮助信息(这里要详细说明入参和出参的格式)
	helpStr := `获取条码 BOM 编码
        ./com_mes -a GetBarcodeBOM -s 2102314MAX250100002E -o result.json`
	RegisterHelp("GetBarcodeBOM", helpStr)
	// 注册解析器
	RegisterParser(&GetBarcodeBOMParser{})
}
