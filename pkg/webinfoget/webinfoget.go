package webinfoget

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"web_tool/pkg/initutil"
	"web_tool/pkg/logutil"
	"web_tool/pkg/toolutil"
)

// 维护所有子模块的帮助信息
var HelpRegistry [][]string
var httpClient = &http.Client{Timeout: 120 * time.Second}

// 供包中的子模块注册帮助信息
func RegisterHelp(command, description string) {
	HelpRegistry = append(HelpRegistry, []string{command, description})
}

type WebInfoGet struct {
	acTion     string
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

// Body: v
//     Response: v
//         XMLName: v
//             Space: http://Auto.huawei.com.cn/
//             Local: Get_Info_FrmbarcodeResponse
//         SExport: <sExport><?xml version="1.0" encoding="utf-8"?><GetBarcodeBOM><Export><ResultData></ResultData><Message><ErrorCode>1</ErrorCode><ErrorMsg>Not found any matched information</ErrorMsg></Message></Export></GetBarcodeBOM></sExport>

// 定义 SOAP 响应结构
type GetInfoFrmBarcodeResponse struct {
	XMLName xml.Name `xml:"Get_Info_FrmbarcodeResponse"`
	SExport string   `xml:",innerxml"` // 这里用 `innerxml` 获取完整的嵌套 XML
}

type SoapBody struct {
	Response GetInfoFrmBarcodeResponse `xml:"Get_Info_FrmbarcodeResponse"`
}

type SoapEnvelope struct {
	Body SoapBody `xml:"Body"`
}

// ResultData 可能是空的，在外部解析比较好
// ResultData: <ResultData></ResultData><Message><ErrorCode>1</ErrorCode><ErrorMsg>Not found any matched information</ErrorMsg></Message>
// Message: v
//     ErrorCode: 1
//     ErrorMsg: Not found any matched information

// 处理 Export 内的有效数据
type Export struct {
	// 这里用 `innerxml` 获取完整的嵌套 XML
	// 如果是空的会保存完整的 XML
	ResultData string  `xml:",innerxml"`
	Message    Message `xml:"Message"`
}

// 解析 `Message` 结构
type Message struct {
	ErrorCode string `xml:"ErrorCode"`
	ErrorMsg  string `xml:"ErrorMsg"`
}

// **提取 `<Export>` 部分的内容**
func extractExportContent(sExportXML string) string {
	// 使用正则匹配 `<Export>...</Export>` 部分
	// 可能有多行 (?s) 让 . 能匹配多行
	re := regexp.MustCompile(`(?s)<Export>(.*?)</Export>`)
	match := re.FindString(sExportXML)

	return match // 直接返回匹配到的 `<Export>` 部分
}

func (w *WebInfoGet) GetInfo(acTion string, xmlInput string) error {
	globalConfig := initutil.GetConfig()
	w.acTion = acTion

	// 构建 SOAP 请求
	payload := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
            <soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/">
                <soapenv:Body>
                    <Get_Info_Frmbarcode xmlns="http://Auto.huawei.com.cn/">
                        <sTaskType>%s</sTaskType>
                        <sImport><![CDATA[%s]]></sImport>
                    </Get_Info_Frmbarcode>
                </soapenv:Body>
            </soapenv:Envelope>`, acTion, xmlInput)

	headers := map[string]string{
		"Content-Type":  "text/xml",
		"Cache-Control": "no-cache",
	}

	nowTime := time.Now().Unix()
	allDelayTime := globalConfig.WebConfig.AllDelayTime
	endTime := nowTime + allDelayTime

	originalDelay := globalConfig.WebConfig.EachDelayTime
	eachDelay := originalDelay

	for {
		var errorCode string

		for _, url := range globalConfig.WebConfig.MesWeb {
			result, err := sendPostRequest(url, payload, headers)
			if err != nil {
				logutil.Error("请求失败: %s, 错误: %s", url, err)
				continue
			}

			if result == "" {
				logutil.Error("所有 WebService 请求失败")
				return fmt.Errorf("所有 WebService 请求失败")
			}

			// 获取原始xml
			result = html.UnescapeString(result)
			logutil.Debug("show result: \n%s", result)

			// 解析SOAP响应
			var envelope SoapEnvelope
			err = xml.Unmarshal([]byte(result), &envelope)
			if err != nil {
				logutil.Error("SOAP 解析错误: %s", err.Error())
				return fmt.Errorf("SOAP 解析错误: %s", err.Error())
			}

			logutil.Debug("show envelope: \n%v", envelope)

			// 获取 sExport 内嵌 XML
			sExportXML := envelope.Body.Response.SExport
			sExportXML = strings.TrimSpace(sExportXML) // 去除可能的空白
			exportXML := extractExportContent(sExportXML)

			logutil.Debug("exportXML: %s", exportXML)

			var export Export
			err = xml.Unmarshal([]byte(exportXML), &export)
			if err != nil {
				logutil.Error("SOAP 解析错误: %s", err.Error())
				return fmt.Errorf("SOAP 解析错误: %s", err.Error())
			}

			// 解析返回 XML
			errorCode = export.Message.ErrorCode
			errorMsg := export.Message.ErrorMsg

			logutil.Debug("show export: \n%s", export)

			// 如果错误码符合重试条件，则继续循环迭代
			if errorCode == "5" || errorCode == "6" || errorCode == "7" {
				logutil.Warn(
					"错误码 %s, 错误信息 %s ,进行重试", errorCode, errorMsg)
				continue
			} else {
				// 不是 5 6 7 的情况下返回
				w.ErrorCode = errorCode
				w.ErrorMsg = errorMsg
				w.ResultData = export.ResultData
				return nil
			}
		}
		nowTime = time.Now().Unix()
		// 执行请求
		if nowTime > endTime {
			return fmt.Errorf("重试时间超过上限 (%d 秒)", allDelayTime)
		}

		if errorCode == "5" || errorCode == "6" || errorCode == "7" {
			eachDelay = toolutil.MinInt64(eachDelay*2, 600)
			time.Sleep(time.Duration(eachDelay) * time.Second)
		} else {
			time.Sleep(time.Duration(originalDelay) * time.Second)
		}
	}
}

func sendPostRequest(url, payload string, headers map[string]string) (string, error) {
	req, err := http.NewRequest("POST", url, strings.NewReader(payload))
	if err != nil {
		// %w 是 fmt.Errorf() 的格式化标志，包装原始错误，用于外部
		// errors.Unwrap() 或者 errors.Is() 进行错误处理
		return "", fmt.Errorf("请求创建失败: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP 请求发送失败: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应数据失败: %w", err)
	}

	return string(bodyBytes), nil
}

// 定义一个接口，用于每个子模块定义解析器
type Parser interface {
	// 初始化
	InitSelf(argsMap map[string]string) error
	// 获取 并解析 XML
	ProcessXML() error
	SaveJSON(subp interface{}) error
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

// :TODO: 更细化的错误处理
// 这里必须传递子类，不然父类结构体无法获取到子类的特殊字段
// 导致最终生成的JOSN不全
func (p *ParserBase) SaveJSON(subp interface{}) error {
	jsonData := toolutil.StructToMap(subp)
	fileData, err := json.MarshalIndent(jsonData, "", "    ")
	if err != nil {
		logutil.Error("json 序列化失败, %s", err)
		return err
	}

	logutil.Debug("show p %v", p)

	err = os.WriteFile(p.OutputFile, fileData, 0644)
	if err != nil {
		logutil.Error("写入文件 %s 失败: %s", p.OutputFile, err)
		return err
	}

	return nil
}
