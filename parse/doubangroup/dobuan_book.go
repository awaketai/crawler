package doubangroup

import (
	"regexp"
	"strconv"
	"time"

	"github.com/awaketai/crawler/collect"
)

var DoubanBookTask = &collect.Task{
	Propety: collect.Propety{
		Name:     "douban_book_list",
		WaitTime: 1 * time.Second,
		MaxDepth: 5,
		Cookie:   cookie,
	},
	Rule: collect.RuleTree{
		Root: func() ([]*collect.Request, error) {
			roots := []*collect.Request{
				{
					Priority: 1,
					Url:      "https://book.douban.com",
					Method:   "GET",
					RuleName: "数据tag",
				},
			}
			return roots, nil
		},
		Trunk: map[string]*collect.Rule{
			"数据tag": {
				ParseFunc: parseTag,
			},
			"书籍列表": &collect.Rule{
				ParseFunc: parseBookList,
			},
			"书籍简介": {
				ParseFunc: parseBookDetail,
				ItemFields: []string{
					"书名",
					"作者",
					"页数",
					"出版社",
					"得分",
					"价格",
					"简介",
				},
			},
		},
	},
}

const regexpStr = `<a href="([^"]+)" class="tag">([^<]+)</a>`

func parseTag(ctx *collect.CrawlerContext) (collect.ParseResult, error) {
	re := regexp.MustCompile(regexpStr)
	matches := re.FindAllSubmatch(ctx.Body, -1)
	result := collect.ParseResult{}
	for _, m := range matches {
		result.Requests = append(result.Requests, &collect.Request{
			Url:      "https://book.douban.com" + string(m[1]),
			Depth:    ctx.Req.Depth + 1,
			Method:   "GET",
			RuleName: "书籍列表",
			Task:     ctx.Req.Task,
		})
	}
	// 减少抓取数量，防止被封
	if len(result.Requests) > 0 {
		result.Requests = result.Requests[:1]
	}
	return result, nil
}

const BooklistRe = `<a.*?href="([^"]+)" title="([^"]+)"`

func parseBookList(ctx *collect.CrawlerContext) (collect.ParseResult, error) {
	re := regexp.MustCompile(BooklistRe)
	matches := re.FindAllSubmatch(ctx.Body, -1)
	result := collect.ParseResult{}
	for _, m := range matches {
		req := &collect.Request{
			Method:   "GET",
			Task:     ctx.Req.Task,
			Url:      string(m[1]),
			Depth:    ctx.Req.Depth + 1,
			RuleName: "书籍简介",
		}
		req.TmpData = &collect.Tmp{}
		req.TmpData.Set("book_name", string(m[2]))
		result.Requests = append(result.Requests, req)
	}
	if len(result.Requests) > 0 {
		result.Requests = result.Requests[:1]
	}

	return result, nil
}

var autoRe = regexp.MustCompile(`<span class="pl"> 作者</span>:[\d\D]*?<a.*?>([^<]+)</a>`)
var public = regexp.MustCompile(`<span class="pl">出版社:</span>([^<]+)<br/>`)
var pageRe = regexp.MustCompile(`<span class="pl">页数:</span> ([^<]+)<br/>`)
var priceRe = regexp.MustCompile(`<span class="pl">定价:</span>([^<]+)<br/>`)
var scoreRe = regexp.MustCompile(`<strong class="ll rating_num " property="v:average">([^<]+)</strong>`)
var intoRe = regexp.MustCompile(`<div class="intro">[\d\D]*?<p>([^<]+)</p></div>`)

func parseBookDetail(ctx *collect.CrawlerContext) (collect.ParseResult, error) {
	bookName := ctx.Req.TmpData.Get("book_name")
	page, _ := strconv.Atoi(ExtraString(ctx.Body, pageRe))
	book := map[string]any{
		"书名":  bookName,
		"作者":  ExtraString(ctx.Body, autoRe),
		"页数":  page,
		"出版社": ExtraString(ctx.Body, public),
		"得分":  ExtraString(ctx.Body, scoreRe),
		"价格":  ExtraString(ctx.Body, intoRe),
	}
	data := ctx.Output(book)
	result := collect.ParseResult{
		Items: []any{data},
	}

	return result, nil
}

func ExtraString(con []byte, re *regexp.Regexp) string {
	match := re.FindSubmatch(con)
	if len(match) > 2 {
		return string(match[1])
	}

	return ""
}
