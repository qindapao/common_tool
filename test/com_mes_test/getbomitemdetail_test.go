package com_mes_test

import (
	"testing"

	"common_tool/internal/testutils"
)

// Go的测试用例必须以 Test 开头
func TestGetBomItemDetail(t *testing.T) {
	cmd := testutils.Services{
		Name:    "GetBomItemDetail",
		Command: "./com_mes",
		Args:    []string{"-a", "GetBomItemDetail", "-b", "02314MAX", "-o", "result.json"},
	}

	// 预期的 JSON 数据
	// :TODO: 为什么获取BOM明细的Success中还有分号?
	expected := &testutils.ResultJSON3{
		ResultCommon: testutils.ResultCommon{
			Action:     "GetBomItemDetail",
			ErrorCode:  "0",
			ErrorMsg:   "Success;",
			OutputFile: "result.json",
		},
		Data: []string{
			"02314MAX 1 no Function Module,PANGEA,STL07M16G,Memory Module,DDR4 RDIMM,16GB,288pin,0.68ns,2933000KHz,1.2V,ECC,1Rank(2G*4bit),prememtest-crc-load",
			"05023YGT 1 no Board Software,0231XXX,0231XXX01,Configuration file of independent memory burn-in,Flash,independent memory burn-in,Load",
			"06200468-002 1 no Memory Module,DDR4 RDIMM,16GB,288pin,0.68ns,2933000KHz,1.2V,ECC,1Rank(2G*4bit),B21-AK",
		},
		InputBomCode: "02314MAX",
	}

	testutils.RunCliTest(t, cmd, expected, "result.json")
}