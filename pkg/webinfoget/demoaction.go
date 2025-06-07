package webinfoget

// ParserBase 直接内联进来不要嵌套，保持扁平化
// 直接继承 ParserBase 的通用函数
type DemoActionParser struct {
	ParserBase
	InputXX string
	Data []string
}

// 实现 Parser 接口所有函数
func (p *DemoActionParser) GetName() string {
	return "DemoAction"
}

func (p *DemoActionParser) ProcessXML() error {
	return nil
}

func (p *DemoActionParser) InitSelf(argsMap map[string]string) error {
	if err := p.ParserBase.InitSelf(argsMap); err != nil {
        return err
    }

	// :TODO: 子类中自己的代码
	p.Action = p.GetName()
    // ... ...

	return nil
}

func (p *DemoActionParser) SaveJSON(subp interface{}) error {
	// 显式调用 `ParserBase` 的方法
    return p.ParserBase.SaveJSON(p)
}

// 首先注册帮助信息(编译的时候插入的)
func init() {
    // 注册帮助信息
	helpStr := `动作范例
        ./com_mes -a DemoAction -s 2102314MAX250100002E -o result.json`
	RegisterHelp("DemoAction",  helpStr)
	// 注册解析器(传入指针)
	RegisterParser(&DemoActionParser{})
}