package webinfoget

import (
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"time"

	"common_tool/pkg/initutil"
	"common_tool/pkg/logutil"
	"common_tool/pkg/toolutil"
	"common_tool/pkg/webbase"
)

type WebInfoGet struct {
	webbase.InfoGet
}

// 解析器通用结构体
type MesParserBase struct {
	webbase.ParserBase
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

// 这个函数没有作用了，有更防呆的解析方法
// // **提取 `<Export>` 部分的内容**
// func extractExportContent(sExportXML string) string {
// 	// 使用正则匹配 `<Export>...</Export>` 部分
// 	// 可能有多行 (?s) 让 . 能匹配多行
// 	re := regexp.MustCompile(`(?s)<Export>(.*?)</Export>`)
// 	match := re.FindString(sExportXML)

// 	return match // 直接返回匹配到的 `<Export>` 部分
// }

func (w *WebInfoGet) GetInfo(acTion string, xmlInput string) error {
	globalConfig := initutil.GetConfig()
	w.AcTion = acTion

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
			// convertsExportXML := webbase.DirtyHtmlUnescape(sExportXML)

			logutil.Debug("show convertsExportXML: %v", sExportXML)

			export, err := webbase.ParseExport(sExportXML)
			if err != nil {
				logutil.Error("SOAP 解析错误: %w", err)
				return fmt.Errorf("SOAP 解析错误: %w", err)
			}

			logutil.Debug("export: %v", export)

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

func sendPostRequest(
	url, payload string, headers map[string]string) (string, error) {
	req, err := http.NewRequest("POST", url, strings.NewReader(payload))
	if err != nil {
		// %w 是 fmt.Errorf() 的格式化标志，包装原始错误，用于外部
		// errors.Unwrap() 或者 errors.Is() 进行错误处理
		return "", fmt.Errorf("请求创建失败: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := webbase.HttpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP 请求发送失败: %w", err)
	}

	var closeErr error
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			closeErr = fmt.Errorf("关闭 Body 体失败: %w", cerr)
		}
	}()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应数据失败: %w", err)
	}

	return string(bodyBytes), closeErr
}

// 实现 Parser 接口所有函数
func (p *MesParserBase) GetName() string {
	return ""
}

func (p *MesParserBase) ProcessXML() error {
	return nil
}

// InitSelf 直接继承不需要处理

// :TODO: 更细化的错误处理
// 这里必须传递子类，不然父类结构体无法获取到子类的特殊字段
// 导致最终生成的JOSN不全
func (p *MesParserBase) SaveJSON(subp any) error {
	return p.ParserBase.SaveJSON(subp)
}