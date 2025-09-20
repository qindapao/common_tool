package tsdinfoget

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/tiaguinho/gosoap"

	"common_tool/pkg/initutil"
	"common_tool/pkg/logutil"
	"common_tool/pkg/toolutil"
	"common_tool/pkg/webbase"
)

type TsdInfoGet struct {
	webbase.InfoGet
}

// 解析器通用结构体
type TsdParserBase struct {
	webbase.ParserBase
}

// Body: v
//     Response: v
//         XMLName: v
//             Space: http://Auto.xx.com.cn/
//             Local: Get_Info_FrmbarcodeResponse
//         SExport: <sExport><?xml version="1.0" encoding="utf-8"?><GetBarcodeBOM><Export><ResultData></ResultData><Message><ErrorCode>1</ErrorCode><ErrorMsg>Not found any matched information</ErrorMsg></Message></Export></GetBarcodeBOM></sExport>

// 定义 SOAP 响应结构
type GetInfoFrmBarcodeResponse struct {
	XMLName xml.Name `xml:"Get_Info_FrmbarcodeResponse"`
	SExport SExport  `xml:"sExport"`
}

type SExport struct {
	InnerXML string `xml:",innerxml"`
}

// ResultData 可能是空的，在外部解析比较好
// ResultData: <ResultData></ResultData><Message><ErrorCode>1</ErrorCode><ErrorMsg>Not found any matched information</ErrorMsg></Message>
// Message: v
//     ErrorCode: 1
//     ErrorMsg: Not found any matched information

// :TODO: 函数需要处理两种情况，有些接口是可能支持容灾的但是有些接口不支持
// 后面统一整理后再处理
func (w *TsdInfoGet) GetInfo(acTion string, xmlInput string) error {
	globalConfig := initutil.GetConfig()
	w.AcTion = acTion

	// 构建 SOAP 请求 参数
	params := gosoap.Params{
		"sTaskType": acTion,
		"sImport":   xmlInput,
		"sExport":   "",
	}

	nowTime := time.Now().Unix()
	allDelayTime := globalConfig.WebConfig.AllDelayTime
	endTime := nowTime + allDelayTime

	originalDelay := globalConfig.WebConfig.EachDelayTime
	eachDelay := originalDelay

	for {
		var errorCode string

		for _, url := range globalConfig.WebConfig.TsdWeb {
			// url这里需要处理,如果后缀不是?dswl，那么需要加上
			// http://10.116.145.15/TsdWebService/DesignerService.asmx?wsdl
			url = webbase.EnsureWsdlSuffix(url)
			result, err := sendSoapRequest(url, "Get_Info_Frmbarcode", params)
			if err != nil {
				logutil.Error("请求失败: %s, 错误: %s", url, err)
				continue
			}

			if result == "" {
				logutil.Error("所有 WebService 请求失败")
				return fmt.Errorf("所有 WebService 请求失败")
			}

			// 获取原始xml
			// rawResult := html.UnescapeString(result)
			logutil.Debug("show rawResult: \n%s", result)

			// 直接获取Export标签下的内容

			// 解析SOAP响应(这里使用原始的不要用UnescapeString后的)
			var envelope GetInfoFrmBarcodeResponse
			err = xml.Unmarshal([]byte(result), &envelope)
			if err != nil {
				logutil.Error("SOAP 解析错误: %w", err)
				return fmt.Errorf("SOAP 解析错误: %w", err)
			}

			logutil.Debug(
				"show SExport: \n%v", envelope.SExport.InnerXML)

			// 获取 sExport 内嵌 XML
			sExportXML := envelope.SExport.InnerXML
			// 去除两端可能的留白
			sExportXML = strings.TrimSpace(sExportXML)

			logutil.Debug("show sExportXML: %s", sExportXML)

			convertsExportXML := webbase.DirtyHtmlUnescape(sExportXML)

			logutil.Debug("show convertsExportXML: %s", convertsExportXML)

			export, err := webbase.ParseExport(convertsExportXML)
			if err != nil {
				logutil.Error("SOAP 解析错误: %w", err)
				return fmt.Errorf("SOAP 解析错误: %w", err)
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

// 转义层级不够
// <Get_Info_FrmbarcodeResponse xmlns="http://Auto.xx.com.cn/"><sExport>&lt;?xml version="1.0" encoding="UTF-8"?&gt;&lt;GetSoftInfoExt&gt;&lt;Export&gt;&lt;ResultData&gt;
// SoftCode Location RelationDomain ProgrammingStation Ict FT1 FT2 ST PreInstall SparePart TSS RTS Aging
// 05022XEX NA PXM NA NA NA Load_MP1_Group1 NA NA NA NA NA NA
// 05023AHX ProductStrategyConfig;BiosSelfTest; Software NA NA NA NA NA NA NA NA NA NA
// elabel fru=1000(power); NA NA NA NA Load&amp;Verify NA NA NA Verify NA NA
// &lt;/ResultData&gt;&lt;Message&gt;&lt;ErrorCode&gt;0&lt;/ErrorCode&gt;&lt;ErrorMsg&gt;success&lt;/ErrorMsg&gt;&lt;/Message&gt;&lt;/Export&gt;&lt;/GetSoftInfoExt&gt;</sExport></Get_Info_FrmbarcodeResponse>
// 按照XML的层级，&amp; 应该写成 &amp;amp;

// [xx@bin]$ ./com_tsd // <?xml version="1.0" encoding="UTF-8"?><GetSoftInfoExt><Import><HardwareBomCode>03033RWK</HardwareBomCode><PlatformType></PlatformType><DocumentVersion>1.004</DocumentVersion></Import></GetSoftInfoExt>
// SOAP 响应 Body: <Get_Info_FrmbarcodeResponse xmlns="http://Auto.xx.com.cn/"><sExport><?xml version="1.0" encoding="UTF-8"?><GetSoftInfoExt><Export><ResultData>
// SoftCode Location RelationDomain ProgrammingStation Ict FT1 FT2 ST PreInstall SparePart TSS RTS Aging
// 05022XEX NA PXM NA NA NA Load_MP1_Group1 NA NA NA NA NA NA
// 05023AHX ProductStrategyConfig;BiosSelfTest; Software NA NA NA NA NA NA NA NA NA NA
// elabel fru=1000(power); NA NA NA NA Load&Verify NA NA NA Verify NA NA
// </ResultData><Message><ErrorCode>0</ErrorCode><ErrorMsg>success</ErrorMsg></Message></Export></GetSoftInfoExt></sExport></Get_Info_FrmbarcodeResponse>
// SOAP 响应 Header:
// SOAP 响应 Payload: <soap:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
//
//	<soap:Body>
//	    <Get_Info_Frmbarcode xmlns="http://Auto.xx.com.cn/">
//	        <sExport></sExport>
//	        <sTaskType>GetSoftInfoExt</sTaskType>
//	        <sImport><?xml version="1.0" encoding="UTF-8"?><GetSoftInfoExt><Import><HardwareBomCode>03033RWK</HardwareBomCode><PlatformType></PlatformType><DocumentVersion>1.004</DocumentVersion></Import></GetSoftInfoExt></sImport>
//	    </Get_Info_Frmbarcode>
//	</soap:Body>
//
// </soap:Envelope>
// [xx@bin]$

// bomCode := "03033RWK"

// // 工位
// // test_scene := "ST-MP1"
// // test_station := "MP1"
// tsd_doc_ver := "1.004"
// action := "GetSoftInfoExt"
// // sendExtendPara := "null"

// inXml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><GetSoftInfoExt><Import><HardwareBomCode>%s</HardwareBomCode><PlatformType></PlatformType><DocumentVersion>%s</DocumentVersion></Import></GetSoftInfoExt>`, bomCode, tsd_doc_ver)

func sendSoapRequest(
	url, subFunc string, params gosoap.Params) (string, error) {

	client, err := gosoap.SoapClient(url, webbase.HttpClient)
	if err != nil {
		return "", fmt.Errorf("创建 SOAP 客户端失败: %w", err)
	}

	resp, err := client.Call(subFunc, params)
	if err != nil {
		return "", fmt.Errorf("SOAP 请求失败: %w" , err)
	}

	// :TODO: 不确定这里是否需要做资源清理
	// 但是 SoapClient 本身没有找到 Close 方法
	// Body Header Payload
	// Unmarshal 取原始数据
	logutil.Debug("show resp: %v", resp)
	return string(resp.Body), nil
}

// 实现 Parser 接口所有函数
func (p *TsdParserBase) GetName() string {
	return ""
}

func (p *TsdParserBase) ProcessXML() error {
	return nil
}

// InitSelf 直接继承不需要处理

// 这里必须传递子类，不然父类结构体无法获取到子类的特殊字段
// 导致最终生成的JOSN不全
func (p *TsdParserBase) SaveJSON(subp any) error {
	return p.ParserBase.SaveJSON(subp)
}
