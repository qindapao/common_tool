package webinfoget

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"web_tool/pkg/logutil"
	"web_tool/pkg/toolutil"
)

// ParserBase 直接内联进来不要嵌套，保持扁平化
// 直接继承 ParserBase 的通用函数
type GetBomItemDetailParser struct {
	ParserBase
	InputBomCode string
	Data         []string
}

// <ResultData><Item>
// 02314MAX 1 no Function Module,PANGEA,STL07M16G,Memory Module,DDR4 RDIMM,16GB,288pin,0.68ns,2933000KHz,1.2V,ECC,1Rank(2G*4bit),prememtest-crc-load
// 05023YGT 1 no Board Software,0231XXX,0231XXX01,Configuration file of independent memory burn-in,Flash,independent memory burn-in,Load
// 06200468-002 1 no Memory Module,DDR4 RDIMM,16GB,288pin,0.68ns,2933000KHz,1.2V,ECC,1Rank(2G*4bit),B21-AK</Item></ResultData><Message><ErrorCode>0</ErrorCode><ErrorMsg>Success;</ErrorMsg></Message>

type bomDetailResultData struct {
	XMLName xml.Name `xml:"ResultData"`
	Items   string   `xml:"Item"`
}

// 实现 Parser 接口所有函数
func (p *GetBomItemDetailParser) GetName() string {
	return "GetBomItemDetail"
}

func (p *GetBomItemDetailParser) ProcessXML() error {
	inXml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><%s>`+
		`<Import><Barcode>%s</Barcode><GetType>GetCurrentBom`+
		`</GetType><SpecifyTime></SpecifyTime>`+
		`</Import></%s>`, p.Action, p.InputBomCode, p.Action)
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

	var result bomDetailResultData
	if err := xml.Unmarshal(
		[]byte(webInfoGet.ResultData), &result); err != nil {
		return err
	}

	p.Data = strings.Split(strings.ReplaceAll(result.Items, "\r\n", "\n"), "\n")
	// 过滤掉空的元素(空字符串或者全是空白字符)
	p.Data = toolutil.Grep(p.Data, `\S+`, false, true)

	return nil
}

func (p *GetBomItemDetailParser) InitSelf(argsMap map[string]string) error {
	if err := p.ParserBase.InitSelf(argsMap); err != nil {
		return err
	}

	// :TODO: 子类中自己的代码
	p.Action = p.GetName()
	// 子类中只处理自己特殊的参数
	p.Action = p.GetName()
	if toolutil.HasAnyKey(argsMap, "-b") {
		p.InputBomCode = argsMap["-b"]
	} else {
		logutil.Error("-s 参数缺失")
		return fmt.Errorf("-s 参数缺失")
	}

	return nil
}

func (p *GetBomItemDetailParser) SaveJSON(subp interface{}) error {
	// 显式调用 `ParserBase` 的方法
	return p.ParserBase.SaveJSON(p)
}

// 首先注册帮助信息(编译的时候插入的)
func init() {
	// 注册帮助信息
	helpStr := `获取条码 BOM 编码
        ./com_mes -a GetBomItemDetail -b 02314MAX -o result.json`
	RegisterHelp("GetBomItemDetail", helpStr)
	// 注册解析器(传入指针)
	RegisterParser(&GetBomItemDetailParser{})
}
