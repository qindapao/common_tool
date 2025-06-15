package com_tsd_test

import (
	"testing"

	"common_tool/internal/testutils"
)

// Go的测试用例必须以 Test 开头
func TestGetSoftInfoExt(t *testing.T) {
	cmd := testutils.Services{
		Name:    "./com_tsd",
		Command: "./com_tsd",
		Args:    []string{"-a", "GetSoftInfoExt", "-t", "1.004", "-b", "03033RWK", "-o", "result.json"},
	}

	// 预期的 JSON 数据
	expected := &testutils.ResultJSON1{
		ResultCommon: testutils.ResultCommon{
			Action:     "GetSoftInfoExt",
			ErrorCode:  "0",
			ErrorMsg:   "success",
			OutputFile: "result.json",
		},
		Data: []string{
			"SoftCode Location RelationDomain ProgrammingStation Ict FT1 FT2 ST PreInstall SparePart TSS RTS Aging",
			"05022XEX NA PXM NA NA NA Load_MP1_Group1 NA NA NA NA NA NA",
			"05023AHX ProductStrategyConfig;BiosSelfTest; Software NA NA NA NA NA NA NA NA NA NA",
			"elabel fru=1000(power); NA NA NA NA Load&Verify NA NA NA Verify NA NA",
		},
		InputBomCode: "03033RWK",
		TsdDocVer:    "1.004",
	}

	testutils.RunCliTest(t, cmd, expected, "result.json")
}