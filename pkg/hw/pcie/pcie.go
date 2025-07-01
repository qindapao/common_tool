package pcie

import (
	"common_tool/pkg/logutil"
	"common_tool/pkg/toolutil"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"common_tool/pkg/toolutil/hex"
	"common_tool/pkg/toolutil/str"

	"github.com/spf13/cobra"
)

// 定义 各种 Feature 名字
const (
	FeatureNameBridge  = "bridge"
	FeatureNameAER     = "aer"
	FeatureNameHotplug = "hotplug"
	FeatureNameSRIOV   = "sriov"
	FeatureNameVGA     = "vga"
	FeatureNameStorage = "storage"
	// ...
)

// Base Class Codes (bits 23:16 of Class Code)
const (
	PciClassBridge  = 0x06
	PciClassStorage = 0x01
	PciClassDisplay = 0x03
	// ...
)

// Subclass Codes for Bridge (bits 15:8 if base == 0x06)
const (
	PciSubClassPciToPciBridge = 0x04
	PciSubClassCardBusBridge  = 0x07
	// ...
)

// ProgIF Codes for specific subclass (bits 7:0, context-dependent)
const (
	PciProgIFPciToPciStandard = 0x00
)

// Subclass Codes for Bridge (bits 15:8 if base == 0x06)
const (
	PciCfgOffsetPrimaryBus     = 0x18
	PciCfgOffsetSecondaryBus   = 0x19
	PciCfgOffsetSubordinateBus = 0x1A
)

// PCI Header Common
const (
	PciCfgOffsetVendorID     = 0x00
	PciCfgOffsetDeviceID     = 0x02
	PciCfgOffsetHeaderType   = 0x0E
	PciCfgOffsetClassCode    = 0x0B // bits 23:16
	PciCfgOffsetSubClass     = 0x0A
	PciCfgOffsetProgIF       = 0x09
	PciCfgOffsetRevisionID   = 0x08
	PciCfgOffsetInterruptPin = 0x3D
)

var (
	// 把常量改成可变变量，预设默认 sysfs 路径
	sysfsRootDefault = "/sys/bus/pci/devices"
)

type DeviceFeature interface {
	Name() string            // 标识功能类型（用于查找）
	FromConfig([]byte) error // 从配置空间解析
	Describe() string        // 可选：用于人类可读描述、UI 等
}

// ErrorMaps 收集三类 AER 错误计数
type ErrorMaps struct {
	Correctable map[string]int `json:"correctable"`
	NonFatal    map[string]int `json:"non_fatal"`
	Fatal       map[string]int `json:"fatal"`
}

// 实现 DeviceFeature 接口
type PciBridgeInfo struct {
	Primary     byte
	Secondary   byte
	Subordinate byte
}

func (b *PciBridgeInfo) Name() string { return FeatureNameBridge }
func (b *PciBridgeInfo) Describe() string {
	return fmt.Sprintf("Bridge: primary=%02X secondary=%02X subordinate=%02X",
		b.Primary, b.Secondary, b.Subordinate)
}

func (b *PciBridgeInfo) FromConfig(cfg []byte) error {
	if len(cfg) < PciCfgOffsetSubordinateBus+1 {
		return fmt.Errorf("config too short for bridge")
	}
	b.Primary = cfg[PciCfgOffsetPrimaryBus]         // 桥的上游总线号（仅桥有效）
	b.Secondary = cfg[PciCfgOffsetSecondaryBus]     // 桥的下游总线起始号
	b.Subordinate = cfg[PciCfgOffsetSubordinateBus] //桥的下游总线终止号
	return nil
}

// / PCIDevice 描述一个 PCI 设备或桥
// :TODO: 按照当前的架构，其实 Errors (AER) 功能只是一个 Feature
// 不能写死(面向能力编程 一个对象具备哪些能力)
type PCIDevice struct {
	Address  string          // PCI 地址，例如 "0000:00:1f.6"
	Domain   uint16          // PCI 域号
	Bus      uint8           // 总线号
	VendorID string          // 厂商 ID（0x1234）
	DeviceID string          // 设备 ID（0xabcd）
	Class    string          // 类别代码（0x0604）
	Errors   ErrorMaps       // 三类 AER 错误计数
	Parent   string          // 父设备地址
	Children []*PCIDevice    // 子设备列表
	Features []DeviceFeature // PCIE设备特性
}

// 添加功能
func (d *PCIDevice) AddFeature(f DeviceFeature, cfg []byte) error {
	if err := f.FromConfig(cfg); err != nil {
		return err
	}
	d.Features = append(d.Features, f)
	return nil
}

// 查找功能
func (d *PCIDevice) GetFeature(name string) DeviceFeature {
	for _, f := range d.Features {
		if f.Name() == name {
			return f
		}
	}
	return nil
}

// 判断一个设备是否具有桥的能力
func (d *PCIDevice) IsBridge() bool {
	return d.GetFeature(FeatureNameBridge) != nil
}

// 列出某个设备支持的所有能力列表
func (d *PCIDevice) ListFeatureNames() []string {
	var names []string
	for _, f := range d.Features {
		names = append(names, f.Name())
	}
	return names
}

// Node 内部用于构建树形结构
type Node struct {
	D        *PCIDevice
	Parent   *Node
	Children []*Node
}

// PCIECmd 定义根命令 "pcie"
func PCIECmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pcie",
		Short: "PCIe 工具",
	}

	// 注册子命令 error_read 到根命令下
	cmd.AddCommand(PCIEErrorRead())
	return cmd
}

// mockers 注册所有 mock 场景：simple、complex、random、multi-domain
var mockers = map[string]func(root string) error{
	"simple":       MockSimple,
	"complex":      MockComplex,
	"random":       MockRandom, // 如果需要，可以自己实现
	"multi-domain": MockMultiDomain,
}

// MockDev 描述单个 mock 设备属性
// 架构调整了但是这里的Mock结构体不用改，我们只是用它来生成文件而已
type MockDev struct {
	Addr                  string
	IsBridge              bool
	PciBridge             PciBridgeInfo
	Vendor, Device, Class string
	WithAER               bool
}

// MockSimple 生成一个固定的“简单链路”场景
func MockSimple(root string) error {
	// … 把之前 MockFunc 的内容搬过来即可 …
	return mockSetup(root, []MockDev{
		// 4 级桥链
		{"0000:00:00.0", true, PciBridgeInfo{0, 1, 3}, "0x1234", "0xabcd", "0x0604", false},
		{"0000:01:00.0", true, PciBridgeInfo{1, 2, 3}, "0x1234", "0xbcde", "0x0604", true},
		{"0000:02:00.0", true, PciBridgeInfo{2, 3, 3}, "0x1234", "0xcdef", "0x0604", false},
		{"0000:03:00.0", false, PciBridgeInfo{0, 0, 0}, "0x1234", "0xdef0", "0x0300", true},
		// 3 级桥链
		{"0000:10:00.0", true, PciBridgeInfo{0x10, 0x11, 0x12}, "0x1111", "0x2222", "0x0604", true},
		{"0000:11:00.0", true, PciBridgeInfo{0x11, 0x12, 0x12}, "0x1111", "0x3333", "0x0604", false},
		{"0000:12:00.0", false, PciBridgeInfo{0, 0, 0}, "0x1111", "0x4444", "0x0300", true},
		// 2 级桥链
		{"0000:20:00.0", true, PciBridgeInfo{0x20, 0x21, 0x21}, "0x5555", "0x6666", "0x0604", false},
		{"0000:21:00.0", false, PciBridgeInfo{0, 0, 0}, "0x5555", "0x7777", "0x0300", true},
		// 孤立节点
		{"0000:30:00.0", false, PciBridgeInfo{0, 0, 0}, "0x9999", "0xaaaa", "0x0300", false},
	})
}

// MockComplex 生成一个交错跳级的复杂场景
func MockComplex(root string) error {
	return mockSetup(root, []MockDev{
		// 1 级单节点
		{"0000:40:00.0", false, PciBridgeInfo{0, 0, 0},
			"0xfeed", "0x0001", "0x0300", false},
		// 2 级链：50 → 51
		{"0000:50:00.0", true, PciBridgeInfo{0x50, 0x51, 0x51},
			"0xbeef", "0x0101", "0x0604", true},
		{"0000:51:00.0", false, PciBridgeInfo{0, 0, 0},
			"0xbeef", "0x0102", "0x0300", false},
		// 主干 4 级链：70 → 71 → 72 → 73 → 74
		{"0000:70:00.0", true, PciBridgeInfo{0x70, 0x71, 0x7D},
			"0xdead", "0x0301", "0x0604", false},
		{"0000:71:00.0", true, PciBridgeInfo{0x71, 0x72, 0x7D},
			"0xdead", "0x0302", "0x0604", true},
		{"0000:72:00.0", true, PciBridgeInfo{0x72, 0x73, 0x73},
			"0xdead", "0x0303", "0x0604", false},
		{"0000:73:00.0", true, PciBridgeInfo{0x73, 0x74, 0x74},
			"", "", "0x0604", true}, // 空 Vendor/Device，强制 ERR
		{"0000:74:00.0", false, PciBridgeInfo{0, 0, 0},
			"", "", "0x0300", true}, // 纯叶子，ERR
		// 同级直接跳：75 单节点（跳过中间层级）
		{"0000:75:00.0", false, PciBridgeInfo{0, 0, 0},
			"0xdead", "0x0350", "0x0300", false},
		// 另一条子链：77 → 78 → 79
		{"0000:77:00.0", true, PciBridgeInfo{0x77, 0x78, 0x78},
			"0xdead", "0x0303", "0x0604", false},
		{"0000:78:00.0", true, PciBridgeInfo{0x78, 0x79, 0x79},
			"", "", "0x0604", true}, // 空 VID/DID，ERR
		{"0000:79:00.0", false, PciBridgeInfo{0, 0, 0},
			"", "", "0x0300", true},
		// 额外叶子，紧跟在 77 同级
		{"0000:7c:00.0", false, PciBridgeInfo{0, 0, 0},
			"0xdead", "0x0303", "0x0300", false},
		{"0000:7d:00.0", false, PciBridgeInfo{0, 0, 0},
			"0xdead", "0x0303", "0x0300", false},
		// 混合孤立节点
		{"0000:80:00.0", false, PciBridgeInfo{0, 0, 0},
			"0xcafe", "0x0401", "0x0300", true},
	})
}

// MockMultiDomain 构造一个跨多个 domain、跳级嵌套、混合深度的场景
func MockMultiDomain(root string) error {
	return mockSetup(root, []MockDev{
		// === Domain 0001: 4 级链 + 跳 bus 级 ===
		// 根桥：0001:00 → 0001:01–04
		{"0001:00:00.0", true, PciBridgeInfo{0x00, 0x01, 0x04}, "0xaaaa", "0x1111", "0x0604", true},
		// 第二级：0001:01 → 0001:02–04
		{"0001:01:00.0", true, PciBridgeInfo{0x01, 0x02, 0x04}, "0xaaaa", "0x2222", "0x0604", false},
		// 第三级：0001:02 → 0001:03–04
		{"0001:02:00.0", true, PciBridgeInfo{0x02, 0x03, 0x04}, "0xaaaa", "0x3333", "0x0604", true},
		// 叶子：0001:04 设备
		{"0001:04:00.0", false, PciBridgeInfo{0, 0, 0}, "0xaaaa", "0x4444", "0x0300", true},

		// 同域跳级：0001:06 → 0001:10
		{"0001:06:00.0", true, PciBridgeInfo{0x06, 0x07, 0x10}, "0xbbbb", "0x5555", "0x0604", true},
		{"0001:10:00.0", false, PciBridgeInfo{0, 0, 0}, "0xbbbb", "0x6666", "0x0300", false},

		// === Domain 0002: 2 级简单链 ===
		{"0002:20:00.0", true, PciBridgeInfo{0x20, 0x21, 0x21}, "0xcccc", "0x7777", "0x0604", false},
		{"0002:21:00.0", false, PciBridgeInfo{0, 0, 0}, "0xcccc", "0x8888", "0x0300", true},

		// === Domain 0003: 单节点 & 无 AER ===
		{"0003:30:00.0", false, PciBridgeInfo{0, 0, 0}, "0xdddd", "0x9999", "0x0300", false},

		// === Domain 0004: 单节点 & 支持 AER ===
		{"0004:40:00.0", false, PciBridgeInfo{0, 0, 0}, "0xeeee", "0xaaaa", "0x0300", true},
	})
}

// MockRandom 占位：未实现随机场景，返回错误
func MockRandom(root string) error {
	// …TODO…
	return fmt.Errorf("MockRandom 未实现")
}

// mockSetup 通用 mock 实现：
// - 清空 root
// - 重建 root
// - 为每个 MockDev 创建目录 & 必要文件
func mockSetup(root string, devs []MockDev) error {
	// 清理旧目录
	if err := os.RemoveAll(root); err != nil {
		return err
	}

	// 创建根目录
	if err := os.MkdirAll(root, 0755); err != nil {
		return err
	}
	// 遍历 mock 设备列表
	for _, m := range devs {
		// 设备目录 = root + 地址
		d := filepath.Join(root, m.Addr)
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
		// 写 vendor、device、class 文件
		os.WriteFile(filepath.Join(d, "vendor"), []byte(m.Vendor), 0644)
		os.WriteFile(filepath.Join(d, "device"), []byte(m.Device), 0644)
		os.WriteFile(filepath.Join(d, "class"), []byte(m.Class), 0644)

		// 如果是桥：写 config 寄存器字段
		if m.IsBridge {
			buf := make([]byte, 0x1B)
			buf[0x18] = m.PciBridge.Primary
			buf[0x19] = m.PciBridge.Secondary
			buf[0x1A] = m.PciBridge.Subordinate
			os.WriteFile(filepath.Join(d, "config"), buf, 0644)
		}
		// 如果需要 AER：写三类错误文件
		if m.WithAER {
			os.WriteFile(filepath.Join(d, "aer_dev_correctable"),
				[]byte("CE 1\nUE 0\n"), 0644)
			os.WriteFile(filepath.Join(d, "aer_dev_nonfatal"),
				[]byte("NF 2\n"), 0644)
			os.WriteFile(filepath.Join(d, "aer_dev_fatal"),
				[]byte("F 3\n"), 0644)
		}
	}
	return nil
}

// PCIEErrorRead 定义子命令 error_read：读取拓扑 & 错误，支持 JSON/Tree/Table 输出
func PCIEErrorRead() *cobra.Command {
	var jsonFile, view, sysfsRoot string
	var mockScenario string

	cmd := &cobra.Command{
		Use:   "error_read",
		Short: "读取 PCIe 拓扑及 AER 错误",
		RunE: func(cmd *cobra.Command, args []string) error {

			// 0. 如果指定了 mock 场景，就先造数据
			if mockScenario != "" && sysfsRoot != sysfsRootDefault {
				defer func() {
					_ = os.RemoveAll(sysfsRoot)
				}()

				// 找到对应的 mock 函数
				m, ok := mockers[mockScenario]
				if !ok {
					return fmt.Errorf("未知 mock 场景：%s", mockScenario)
				}
				if err := m(sysfsRoot); err != nil {
					return err
				}
			}

			// 1. 扫描所有设备，返回扁平 map[address]*PCIDevice
			flat, err := scanAll(sysfsRoot)
			if err != nil {
				return err
			}

			// 2. 计算每个设备的 summary（OK/ERR）及全局 all_summary
			devSummary := make(map[string]string, len(flat))
			allSummary := "OK"
			for addr, dev := range flat {
				sum := "OK"
				if hasAnyErrors(dev.Errors) {
					sum = "ERR"
					allSummary = "ERR"
				}
				devSummary[addr] = sum
			}

			// 3. 构建树形结构，填充 Parent/Children
			rootsByDomain := buildTree(flat)

			// 4. 如果指定输出 JSON，则写入文件
			if jsonFile != "" {
				f, err := os.Create(jsonFile)
				if err != nil {
					return err
				}
				defer f.Close()

				// 构造输出结构：包含 all_summary 和每个设备的 summary/parent/child/errors
				out := make(map[string]interface{}, len(flat)+1)
				out["all_summary"] = allSummary

				for addr, dev := range flat {
					sum := devSummary[addr]
					var parent any
					if dev.Parent != "" {
						parent = dev.Parent
					} else {
						parent = nil
					}

					// 收集所有子设备地址
					var children []string
					for _, c := range dev.Children {
						children = append(children, c.Address)
					}

					// :TODO: 这里可以打印设备支持的所有 Feature
					out[addr] = map[string]interface{}{
						"summary":  sum,
						"parent":   parent,
						"children": children,
						"errors":   dev.Errors,
					}
				}

				// 使用带缩进的 JSON 编码器
				enc := json.NewEncoder(f)
				enc.SetIndent("", "  ")
				if err := enc.Encode(out); err != nil {
					return err
				}
			}

			// 5. 根据 view 参数打印不同视图
			switch view {
			case "tree":
				printTree(rootsByDomain)
			case "table":
				printTable(flat, devSummary)
			case "both":
				printTree(rootsByDomain)
				printTable(flat, devSummary)
			case "none":
			default:
				return fmt.Errorf("未知视图: %s", view)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&jsonFile, "json-file", "", "保存 JSON 到文件")
	cmd.Flags().StringVar(&view, "view", "none", "视图模式: tree|table|both|none")
	cmd.Flags().StringVar(&sysfsRoot, "sysfs-root", sysfsRootDefault, "PCI 设备根目录（用于 mock 测试）")
	cmd.Flags().StringVar(&mockScenario, "mock-scenario", "",
		"指定 mock 场景(simple, complex, random), 为空则不打桩")
	return cmd
}

// scanAll 遍历 root 目录下每个子目录（PCI 地址），读取设备属性及错误
func scanAll(root string) (map[string]*PCIDevice, error) {
	// 列出 root 下所有条目
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	out := make(map[string]*PCIDevice, len(entries))
	for _, e := range entries {
		addr := e.Name() // "0000:02:00.0"
		parts := strings.Split(addr, ":")
		// 解析域和总线号（16 进制字符串）
		dom, _ := strconv.ParseUint(parts[0], 16, 16)
		bus, _ := strconv.ParseUint(parts[1], 16, 8)

		dev := &PCIDevice{
			Address: addr,
			Domain:  uint16(dom),
			Bus:     uint8(bus),
			VendorID: hex.ReadHexStrFf(
				filepath.Join(root, addr, "vendor")),
			DeviceID: hex.ReadHexStrFf(
				filepath.Join(root, addr, "device")),
			Class: strings.TrimSpace(
				str.ReadStrFf(filepath.Join(root, addr, "class"))),
			Errors: ErrorMaps{
				Correctable: readErrorMap(
					filepath.Join(root, addr, "aer_dev_correctable")),
				NonFatal: readErrorMap(
					filepath.Join(root, addr, "aer_dev_nonfatal")),
				Fatal: readErrorMap(
					filepath.Join(root, addr, "aer_dev_fatal")),
			},
		}

		class, _ := hex.ParseHexToUint32(dev.Class) // dev.Class == "0x060400"
		baseClass := byte(class >> 16)

		// PCIE 桥设备(PCIE配置空间寄存器)
		// PCI-to-PCI Bridge（Class code 0x06/Subclass 0x04）
		// PCIE 配置空间是大端语义，内核已经封装好，不受架构限制
		if baseClass == PciClassBridge {
			// 原封不动地映射了这块设备的 PCI 配置空间（Configuration Space）头部的前 256 字节（Type-1 桥接器头）
			// Primary Bus Number （寄存器 0x18） 桥接器上游所在的总线号，也就是这块桥本身“插在哪条”父总线下面。
			// Secondary Bus Number （寄存器 0x19） 桥接器下游第一个子总线的编号，所有直接连在这个桥背后的设备都在这个总线上。
			// Subordinate Bus Number （寄存器 0x1A） 整棵这块桥管辖的所有
			// 子总线（包括孙桥、曾孙桥……）的最大总线号。 也就是说，这个桥
			// 会“转发”从 Secondary 到 Subordinate 范围内所有的 PCI 事务。
			if cfg, err := os.ReadFile(
				filepath.Join(root, addr, "config")); err == nil &&
				len(cfg) > PciCfgOffsetSubordinateBus {
				_ = dev.AddFeature(&PciBridgeInfo{}, cfg)
			}
		}
		out[addr] = dev
	}
	return out, nil
}

// hasAnyErrors 判断三个错误 map 中是否有任意值大于 0
func hasAnyErrors(em ErrorMaps) bool {
	for _, m := range []map[string]int{em.Correctable, em.NonFatal, em.Fatal} {
		for _, v := range m {
			if v > 0 {
				return true
			}
		}
	}
	return false
}

// readErrorMap 从 AER 文件读取 map[错误类型]计数
func readErrorMap(path string) map[string]int {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	m := make(map[string]int)
	// 按行拆分并解析字段
	// SplitSeq惰性按行拆分，减少对内存的压力(大文件的时候才有意义，当前没多大意义)
	for line := range strings.SplitSeq(strings.TrimSpace(string(b)), "\n") {
		f := strings.Fields(line)
		if len(f) != 2 {
			continue
		}
		key, val := f[0], f[1]
		// 跳过 TOTAL_ERR_* 汇总字段
		if strings.HasPrefix(key, "TOTAL_ERR_") {
			continue
		}
		if v, err := strconv.Atoi(val); err == nil {
			m[key] = v
		}
	}
	return m
}

// buildTree 根据每个设备的桥寄存器，构建 Parent/Children 关系
// 返回 map[Domain][]*Node，用于多域环境下分别打印
func buildTree(flat map[string]*PCIDevice) map[uint16][]*Node {
	// 重置 Parent/Children
	for _, d := range flat {
		d.Parent = ""
		d.Children = nil
	}
	// 创建节点索引
	nodes := make(map[string]*Node, len(flat))
	for addr, d := range flat {
		nodes[addr] = &Node{D: d}
	}
	// 查找每个节点的最佳父桥
	for _, n := range nodes {
		var best *Node
		var bestBridge *PciBridgeInfo
		var rangeBest byte
		for _, cand := range nodes {
			// 只考虑桥且同域且地址不同
			if !cand.D.IsBridge() ||
				cand.D.Domain != n.D.Domain ||
				cand.D.Address == n.D.Address {
				continue
			}

			// 如果 n 的总线号在 cand 桥的 Secondary–Subordinate 范围内
			candBridge := cand.D.GetFeature(FeatureNameBridge).(*PciBridgeInfo)
			rangeCand := candBridge.Subordinate - candBridge.Secondary
			if best != nil {
				bestBridge = best.D.GetFeature(FeatureNameBridge).(*PciBridgeInfo)
				rangeBest = bestBridge.Subordinate - bestBridge.Secondary
			}

			if n.D.Bus >= candBridge.Secondary &&
				n.D.Bus <= candBridge.Subordinate {
				// 选择范围最小的桥，作为最近的父
				// 每个 PCI-to-PCI 桥都有两个寄存器：
				// Secondary：它管辖的下游总线起始号
				// Subordinate：它管辖的下游总线结束号 整个下游范围就是 [Secondary, Subordinate]。
				// 当某个设备 n 的总线号 n.D.Bus 落在桥 cand 的这个范围里时——
				// if n.D.Bus >= cand.D.Secondary && n.D.Bus <= cand.D.Subordinate { … }
				// 那它就有资格成为 n 的父桥。
				// 如果同时有多个桥都满足上述条件，就要挑「距离最近」的那条，也就是管得最近那层。 
				// 我们用 (cand.D.Subordinate - cand.D.Secondary) 来量化「管的范围有多大」：
				// 范围越小，就说明这座桥下面的层级关系越“近”——越可能是直接父。
				// 所以比较两个桥的 (Subordinate - Secondary)，取最小的那个。

				// 如果更窄，更新；若一样窄，就比 Secondary（起始 Bus）更小的
				if best == nil {
					best = cand
					continue
				}

				if rangeCand < rangeBest ||
					(rangeCand == rangeBest && candBridge.Secondary < bestBridge.Secondary) {
					best = cand
				}

				if rangeCand == rangeBest && candBridge.Secondary == bestBridge.Secondary {
					logutil.Error("PCIE 拓扑错误，需要人工检查")
				}
			}
		}

		// 如果找到父桥：建立双向关系
		if best != nil {
			n.Parent = best
			best.Children = append(best.Children, n)
			n.D.Parent = best.D.Address
			best.D.Children = append(best.D.Children, n.D)
		}
	}
	// 收集各域根节点（无父）
	roots := make(map[uint16][]*Node)
	for _, n := range nodes {
		if n.Parent == nil {
			roots[n.D.Domain] = append(roots[n.D.Domain], n)
		}
	}
	// 对每个域的根列表按地址排序，保证输出有序
	for d := range roots {
		sort.Slice(roots[d], func(i, j int) bool {
			return roots[d][i].D.Address < roots[d][j].D.Address
		})
	}
	return roots
}

// printTree 以 ASCII 树形结构打印各域下设备
func printTree(roots map[uint16][]*Node) {
	// 收集并排序域号
	var domains []uint16
	for d := range roots {
		domains = append(domains, d)
	}
	sort.Slice(domains, func(i, j int) bool { return domains[i] < domains[j] })

	// 遍历每个域，打印域标识及其子树
	for di, dom := range domains {
		lastDom := di == len(domains)-1
		conn, prefix := "+-", "│  "
		if lastDom {
			conn, prefix = "\\-", "   "
		}
		// 打印域号（十六进制形式）
		fmt.Printf("%s[%04x]\n", conn, dom)
		// 打印该域下每棵子树
		for i, n := range roots[dom] {
			printNode(n, prefix, i == len(roots[dom])-1)
		}
	}
}

// printNode 递归打印单个节点及其子节点
func printNode(n *Node, prefix string, isLast bool) {
	conn := "+-"
	if isLast {
		conn = "\\-"
	}
	// 只打印地址后半部分（去掉域前缀），显示状态和 ID
	part := n.D.Address[strings.Index(n.D.Address, ":")+1:]
	status := "OK"
	if hasAnyErrors(n.D.Errors) {
		status = "ERR"
	}
	fmt.Printf("%s%s %s [%s] %s/%s\n",
		prefix, conn, part, status, n.D.VendorID, n.D.DeviceID)

	// 为子节点计算新的前缀
	childPref := prefix
	if isLast {
		childPref += "   "
	} else {
		childPref += "│  "
	}
	// 按地址排序子节点，保证输出稳定
	sort.Slice(n.Children, func(i, j int) bool {
		return n.Children[i].D.Address < n.Children[j].D.Address
	})
	// 递归打印所有子节点
	for i, c := range n.Children {
		printNode(c, childPref, i == len(n.Children)-1)
	}
}

// printTable 以表格形式打印所有设备详细信息
// 包括父、子、状态、寄存器字段和所有错误计数列
func printTable(
	devs map[string]*PCIDevice,
	devSummary map[string]string,
) {
	// 1. 收集所有出现过的错误码，分别归类到 CE/NF/F
	corr, nf, fat := make(map[string]struct{}), make(map[string]struct{}), make(map[string]struct{})
	for _, d := range devs {
		for t := range d.Errors.Correctable {
			corr[t] = struct{}{}
		}
		for t := range d.Errors.NonFatal {
			nf[t] = struct{}{}
		}
		for t := range d.Errors.Fatal {
			fat[t] = struct{}{}
		}
	}
	cCols := toolutil.SortedKeys(corr)
	nCols := toolutil.SortedKeys(nf)
	fCols := toolutil.SortedKeys(fat)

	// 2. 准备 tabwriter，按列对齐输出
	w := tabwriter.NewWriter(os.Stdout, 4, 0, 2, ' ', 0)
	// 表头：固定列 + 动态错误列
	headers := []string{
		"Device", "Parent", "firstbornChild", "Summary",
		"Domain", "Bus", "Vendor", "DeviceID", "Class",
	}
	for _, t := range cCols {
		headers = append(headers, "C-"+t)
	}
	for _, t := range nCols {
		headers = append(headers, "N-"+t)
	}
	for _, t := range fCols {
		headers = append(headers, "F-"+t)
	}
	fmt.Fprintln(w, strings.Join(headers, "\t"))

	// 3. 按地址排序所有设备
	var addrs []string
	for a := range devs {
		addrs = append(addrs, a)
	}
	sort.Strings(addrs)

	// 4. 输出每一行的数据
	for _, addr := range addrs {
		d := devs[addr]
		parent := str.DefaultStr(d.Parent, "null")
		child := "null"

		if len(d.Children) > 0 {
			addrs := make([]string, len(d.Children))
			for i, c := range d.Children {
				addrs[i] = c.Address
			}
			sort.Strings(addrs) // 字典序升序
			child = addrs[0]
		}

		sum := devSummary[addr]

		// 基本字段
		row := []string{
			d.Address,
			parent,
			child,
			sum,
			fmt.Sprintf("0x%04x", d.Domain),
			fmt.Sprintf("0x%02x", d.Bus),
			d.VendorID,
			d.DeviceID,
			d.Class,
		}
		// 三类错误计数列
		for _, t := range cCols {
			row = append(row, strconv.Itoa(d.Errors.Correctable[t]))
		}
		for _, t := range nCols {
			row = append(row, strconv.Itoa(d.Errors.NonFatal[t]))
		}
		for _, t := range fCols {
			row = append(row, strconv.Itoa(d.Errors.Fatal[t]))
		}
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	_ = w.Flush()
}
