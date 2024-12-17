package doubangroup

import (
	"fmt"
	"regexp"
	"time"

	"github.com/awaketai/crawler/collect"
)

var DouBanGroupTask = &collect.Task{
	Propety: collect.Propety{
		Name:     "find_douban_sun_room",
		WaitTime: 1 * time.Second,
		MaxDepth: 5,
		Cookie:   cookie,
	},
	Rule: collect.RuleTree{
		Root: func() ([]*collect.Request, error) {
			var roots []*collect.Request
			for i := 0; i < 25; i += 25 {
				str := fmt.Sprintf("https://www.douban.com/group/280198/discussion?start=%d&type=new", i)
				roots = append(roots, &collect.Request{
					Priority: 1,
					Url:      str,
					Method:   "GET",
					RuleName: "解析网站URL",
				})
			}

			return roots, nil
		},
		Trunk: map[string]*collect.Rule{
			"解析网站URL": {ParseFunc: ParseGroupUrl},
			"解析阳台房":   {ParseFunc: GetSunRoom},
		},
	},
}

const cityListRe = `href="(https://www.douban.com/group/topic/[0-9a-zA-Z]+/)"[^>]*>([^<]+)</a>`
const ContentRe = `<div class="topic-content">[\s\S]*?阳台[\s\S]*?<div`

func ParseGroupUrl(ctx *collect.CrawlerContext) (collect.ParseResult, error) {
	re := regexp.MustCompile(cityListRe)
	mathes := re.FindAllSubmatch(ctx.Body, -1)
	result := collect.ParseResult{}
	for _, m := range mathes {
		u := string(m[1])
		result.Requests = append(result.Requests, &collect.Request{
			Method:   "GET",
			Task:     ctx.Req.Task,
			Url:      u,
			Depth:    ctx.Req.Depth + 1,
			RuleName: "解析阳台房",
		})

	}

	return result, nil
}

func GetSunRoom(ctx *collect.CrawlerContext) (collect.ParseResult, error) {
	re := regexp.MustCompile(ContentRe)
	ok := re.Match(ctx.Body)
	if !ok {
		return collect.ParseResult{
			Items: []any{},
		}, nil
	}
	result := collect.ParseResult{
		Items: []any{ctx.Req.Url},
	}

	return result, nil
}

func ParseURL(con []byte, req *collect.Request) collect.ParseResult {
	re := regexp.MustCompile(cityListRe)
	matches := re.FindAllSubmatch(con, -1)
	result := collect.ParseResult{}
	for _, m := range matches {
		u := string(m[1])
		result.Requests = append(result.Requests, &collect.Request{
			Task:  req.Task,
			Url:   u,
			Depth: req.Depth + 1,
			ParseFunc: func(c []byte, req *collect.Request) collect.ParseResult {
				return GetContent(c, u)
			},
		})
	}

	return result
}

func GetContent(con []byte, url string) collect.ParseResult {
	re := regexp.MustCompile(ContentRe)
	ok := re.Match(con)
	if !ok {
		return collect.ParseResult{}
	}

	return collect.ParseResult{
		Items: []any{url},
	}
}
