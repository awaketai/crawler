package doubangroup

import (
	"regexp"

	"github.com/awaketai/crawler/collect"
)

const cityListRe = `href="(https://www.douban.com/group/topic/[0-9a-zA-Z]+/)"[^>]*>([^<]+)</a>`

func ParseURL(con []byte, req *collect.Request) collect.ParseResult {
	re := regexp.MustCompile(cityListRe)
	matches := re.FindAllSubmatch(con, -1)
	result := collect.ParseResult{}
	for _, m := range matches {
		u := string(m[1])
		result.Requests = append(result.Requests, &collect.Request{
			Task: req.Task,
			Url:    u,
			Depth: req.Depth + 1,
			ParseFunc: func(c []byte, req *collect.Request) collect.ParseResult {
				return GetContent(c, u)
			},
		})
	}

	return result
}

const ContentRe = `<div class="topic-content">[\s\S]*?阳台[\s\S]*?<div`

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
