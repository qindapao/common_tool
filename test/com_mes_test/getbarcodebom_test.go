package com_mes_test

import (
	"testing"

	"common_tool/internal/testutils"
)

// Go的测试用例必须以 Test 开头
func TestGetBarcodeBOM(t *testing.T) {
	cmd := testutils.Services{
		Name:    "GetBarcodeBOM",
		Command: "./com_mes",
		Args:    []string{"-a", "GetBarcodeBOM", "-s", "2102314MAX250100002E", "-o", "result.json"},
	}

	// 预期的 JSON 数据
	expected := &testutils.ResultJSON2{
		ResultCommon: testutils.ResultCommon{
			Action:     "GetBarcodeBOM",
			ErrorCode:  "0",
			ErrorMsg:   "Success",
			OutputFile: "result.json",
		},
		Data:    "02314MAX",
		InputSn: "2102314MAX250100002E",
	}

	testutils.RunCliTest(t, cmd, expected, "result.json")
}