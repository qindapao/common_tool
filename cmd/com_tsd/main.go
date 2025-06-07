package main

import (
	"fmt"
	"html"
	"net/http"
	"time"

	"github.com/tiaguinho/gosoap"
)

func main() {
	// 创建 HTTP 客户端
	httpClient := &http.Client{
		Timeout: 15 * time.Second,
	}

	// 使用 `SoapClient` 代替 `NewClient`
	client, err := gosoap.SoapClient(
		"http://10.116.145.15/TsdWebService/DesignerService.asmx?wsdl", httpClient)
	if err != nil {
		fmt.Println("创建 SOAP 客户端失败:", err)
		return
	}

	// sn := "033RWK1000000001"
	// dns_name := "develop"
	bomCode := "03033RWK"

	// 工位
	// test_scene := "ST-MP1"
	// test_station := "MP1"
	tsd_doc_ver := "1.004"
	action := "GetSoftInfoExt"
	// sendExtendPara := "null"

	inXml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><GetSoftInfoExt><Import><HardwareBomCode>%s</HardwareBomCode><PlatformType></PlatformType><DocumentVersion>%s</DocumentVersion></Import></GetSoftInfoExt>`, bomCode, tsd_doc_ver)

	fmt.Println(inXml)

	params := gosoap.Params{
		"sTaskType": action,
		"sImport":   inXml,
		"sExport":   "",
	}
	res, err := client.Call("Get_Info_Frmbarcode", params)
	if err != nil {
		fmt.Println("SOAP 请求失败:", err)
		return
	}

	// [q00546874@bin]$ ./com_tsd
	// <?xml version="1.0" encoding="UTF-8"?><GetSoftInfoExt><Import><HardwareBomCode>03033RWK</HardwareBomCode><PlatformType></PlatformType><DocumentVersion>1.004</DocumentVersion></Import></GetSoftInfoExt>
	// SOAP 响应 Body: <Get_Info_FrmbarcodeResponse xmlns="http://Auto.huawei.com.cn/"><sExport><?xml version="1.0" encoding="UTF-8"?><GetSoftInfoExt><Export><ResultData>
	// SoftCode Location RelationDomain ProgrammingStation Ict FT1 FT2 ST PreInstall SparePart TSS RTS Aging
	// 05022XEX NA PXM NA NA NA Load_MP1_Group1 NA NA NA NA NA NA
	// 05023AHX ProductStrategyConfig;BiosSelfTest; Software NA NA NA NA NA NA NA NA NA NA
	// elabel fru=1000(power); NA NA NA NA Load&Verify NA NA NA Verify NA NA
	// </ResultData><Message><ErrorCode>0</ErrorCode><ErrorMsg>success</ErrorMsg></Message></Export></GetSoftInfoExt></sExport></Get_Info_FrmbarcodeResponse>
	// SOAP 响应 Header:
	// SOAP 响应 Payload: <soap:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
	//     <soap:Body>
	//         <Get_Info_Frmbarcode xmlns="http://Auto.huawei.com.cn/">
	//             <sExport></sExport>
	//             <sTaskType>GetSoftInfoExt</sTaskType>
	//             <sImport><?xml version="1.0" encoding="UTF-8"?><GetSoftInfoExt><Import><HardwareBomCode>03033RWK</HardwareBomCode><PlatformType></PlatformType><DocumentVersion>1.004</DocumentVersion></Import></GetSoftInfoExt></sImport>
	//         </Get_Info_Frmbarcode>
	//     </soap:Body>
	// </soap:Envelope>
	// [q00546874@bin]$

	fmt.Println("SOAP 响应 Body:", html.UnescapeString(string(res.Body)))
	fmt.Println("SOAP 响应 Header:", html.UnescapeString(string(res.Header)))
	fmt.Println("SOAP 响应 Payload:", html.UnescapeString(string(res.Payload)))
}
