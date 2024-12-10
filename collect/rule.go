package collect

// RuleTree 采集规则树
type RuleTree struct{
	// Root 根节点-执行入口
	Root func() []*Request
	// Trunk 规则哈希表
	Trunk map[string]*Rule
}

type Rule struct{
	ParseFunc func(*CrawlerContext) ParseResult
}

type CrawlerContext struct{
	Body []byte
	Req *Request
}