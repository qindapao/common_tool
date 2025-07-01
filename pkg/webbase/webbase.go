package webbase

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"common_tool/pkg/logutil"
	"common_tool/pkg/toolutil"
	"common_tool/pkg/toolutil/structutil"
)

// 维护所有子模块的帮助信息
var HelpRegistry [][]string

// 运行时分配在堆内存中，保存全局指针
// 编译的时候只是保留声明
var HttpClient = &http.Client{Timeout: 120 * time.Second}

// 供包中的子模块注册帮助信息
func RegisterHelp(command, description string) {
	HelpRegistry = append(HelpRegistry, []string{command, description})
}

type InfoGet struct {
	AcTion     string
	ErrorCode  string
	ErrorMsg   string
	ResultData string
}

// 解析器通用结构体
type ParserBase struct {
	Action          string
	OutputFile      string
	SendRequestTime string
	RevRequestTime  string
	ErrorCode       string
	ErrorMsg        string
}

// 让 `ResultData` 既能存储纯文本，也能存储嵌套 XML
type Export struct {
	ResultData string  `xml:",innerxml"`
	Message    Message `xml:"Message"`
}

// Message 定义了 <Message> 节点的结构体
type Message struct {
	ErrorCode string `xml:"ErrorCode"`
	ErrorMsg  string `xml:"ErrorMsg"`
}

// parseExport 封装了解析逻辑，输入 XML 字符串，直接返回 <Export> 节点对应的 Export 结构体
func ParseExport(xmlStr string) (Export, error) {
	decoder := xml.NewDecoder(strings.NewReader(xmlStr))
	var exp Export

	// 遍历 XML 中的所有 token，查找标签名为 "Export" 的开始元素
	for {
		token, err := decoder.Token()
		if err != nil {
			// 如果token遍历结束仍未找到，则返回错误
			break
		}
		switch se := token.(type) {
		case xml.StartElement:
			logutil.Debug("show se.Name.Local: %v", se.Name.Local)
			if se.Name.Local == "Export" {
				// 找到 Export 标签，直接解析该节点下的全部内容
				if err := decoder.DecodeElement(&exp, &se); err != nil {
					return exp, err
				}
				return exp, nil
			}
		}
	}
	return exp, errors.New("未找到 Export 标签")
}

// :TODO: 基于当前的不规范的实现，可以写一个脏的
// 其它字符还原，但是 &amp; 不还原(&)
func DirtyHtmlUnescape(xmlStr string) string {
	// 手动处理 XML 转义字符
	replacements := map[string]string{
		"&lt;":   "<",
		"&gt;":   ">",
		"&quot;": `"`,
		"&apos;": "'",
	}

	for encoded, decoded := range replacements {
		xmlStr = strings.ReplaceAll(xmlStr, encoded, decoded)
	}

	return xmlStr
}

// ResultData 可能是空的，在外部解析比较好
// ResultData: <ResultData></ResultData><Message><ErrorCode>1</ErrorCode><ErrorMsg>Not found any matched information</ErrorMsg></Message>
// Message: v
//     ErrorCode: 1
//     ErrorMsg: Not found any matched information

func EnsureWsdlSuffix(url string) string {
	if !strings.HasSuffix(url, "?wsdl") {
		return url + "?wsdl"
	}
	return url
}

func (w *InfoGet) GetInfo(acTion string, xmlInput string) error {
	return nil
}

// 定义一个接口，用于每个子模块定义解析器
type Parser interface {
	// 初始化
	InitSelf(argsMap map[string]string) error
	// 获取 并解析 XML
	ProcessXML() error
	SaveJSON(subp any) error
	// 解析器名称
	GetName() string
}

// 存储所有已注册的解析器
var parserRegistry = make(map[string]Parser)

// 注册解析器
func RegisterParser(p Parser) {
	parserRegistry[p.GetName()] = p
}

// 获取解析器
func GetParser(name string) (Parser, bool) {
	logutil.Debug("show parserRegistry: %v", parserRegistry)
	parser, exists := parserRegistry[name]
	return parser, exists
}

// 实现 Parser 接口所有函数
func (p *ParserBase) GetName() string {
	return ""
}

func (p *ParserBase) ProcessXML() error {
	return nil
}

func (p *ParserBase) InitSelf(argsMap map[string]string) error {
	if toolutil.HasAnyKey(argsMap, "-o") {
		p.OutputFile = argsMap["-o"]
	} else {
		logutil.Error("-o 参数缺失")
		return fmt.Errorf("-o 参数缺失")
	}

	// 参数检查
	return nil
}

// 这里必须传递子类，不然父类结构体无法获取到子类的特殊字段
// 导致最终生成的JOSN不全
func (p *ParserBase) SaveJSON(subp any) error {

	logutil.Debug("show subp: %v", subp)

	jsonData := structutil.StructToMap(subp)

	// 手动控制写入JSON的情况
	file, err := os.Create(p.OutputFile)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	// 保持良好的缩进
	encoder.SetIndent("", "    ")
	// 禁止自动转义html特殊字符
	encoder.SetEscapeHTML(false)
	// 写入文件
	err = encoder.Encode(jsonData)
	if err != nil {
		logutil.Error("写入文件 %s 失败: %w", p.OutputFile, err)
		return err
	}

	return nil
}