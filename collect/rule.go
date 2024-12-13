package collect

import (
	"regexp"
	"time"
)

// RuleTree 采集规则树
type RuleTree struct {
	// Root 根节点-执行入口
	Root func() ([]*Request,error)
	// Trunk 规则哈希表
	Trunk map[string]*Rule
}

type Rule struct {
	ParseFunc func(*CrawlerContext) (ParseResult,error)
}

type CrawlerContext struct {
	Body []byte
	Req  *Request
}

type RuleMode struct {
	Name      string `json:"name"`
	ParseFunc string `json:"parse_script"`
}

type OutputData struct {
	Data map[string]any
}

func (c *CrawlerContext) GetRule(ruleName string) *Rule {
	return c.Req.Task.Rule.Trunk[ruleName]
}

func (c *CrawlerContext) Output(data any) *OutputData {
	res := &OutputData{
		Data: map[string]any{},
	}
	res.Data["Rule"] = c.Req.RuleName
	res.Data["Data"] = data
	res.Data["Url"] = c.Req.Url
	res.Data["Time"] = time.Now().Format("2006-01-02 15:04:05")
	return res
}

func (c *CrawlerContext) ParseJSReg(name, reg string) ParseResult {
	re := regexp.MustCompile(reg)
	matches := re.FindAllSubmatch(c.Body, -1)
	result := ParseResult{}
	for _, m := range matches {
		u := string(m[1])
		result.Requests = append(result.Requests, &Request{
			Method:   "GET",
			Task:     c.Req.Task,
			Url:      u,
			Depth:    c.Req.Depth + 1,
			RuleName: name,
		})
	}

	return result
}

func (c *CrawlerContext) OutputJS(reg string) ParseResult {
	re := regexp.MustCompile(reg)
	ok := re.Match(c.Body)
	if !ok {
		return ParseResult{
			Items: []any{},
		}
	}

	return ParseResult{
		Items: []any{c.Req.Url},
	}
}
