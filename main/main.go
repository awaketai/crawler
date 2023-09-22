package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"

	"github.com/PuerkitoBio/goquery"
	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// 1.使用正则表达式
// 2.使用xpath
// 3.使用css选择器
var headerRe = regexp.MustCompile(`<div class="small_imgposition__PYVLm"><img.*?alt=([\s\S]*?) src`)

func main() {
	dumpLogOrderId()
	url := "https://www.thepaper.cn/"
	res, err := Fetch(url)
	if err != nil {
		fmt.Println("fetch err:", err)
		return
	}
	// 1.使用正则表达式
	matches := headerRe.FindAllSubmatch(res, -1)
	for _, m := range matches {
		fmt.Println("fetch card news:", string(m[1]))
	}
	// 2.使用xpath
	doc, err := htmlquery.Parse(bytes.NewReader(res))
	if err != nil {
		fmt.Println("htmlquery.Parse err:", err)
		return
	}
	nodes := htmlquery.Find(doc, `//div[@class="small_imgposition__PYVLm"]/img/@alt`)
	for _, node := range nodes {
		fmt.Println("fetch card:", node.FirstChild.Data)
	}
	// 3.使用css选择器
	cssDoc, err := goquery.NewDocumentFromReader(bytes.NewReader(res))
	if err != nil {
		fmt.Println("goquery err:", err)
		return
	}
	cssDoc.Find("div.small_imgposition__PYVLm img").Each(func(i int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		fmt.Println("src:", src, "exists:", exists)
		alt, exists := s.Attr("alt")
		fmt.Println("alt:", alt, "exists:", exists)
	})
}

func Fetch(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("http.Get err:", err)
		return []byte{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("error http status:%v %v", resp.StatusCode, resp.Status)
	}
	bodyReader := bufio.NewReader(resp.Body)
	e := DeterminEncoding(bodyReader)
	utf8Reader := transform.NewReader(bodyReader, e.NewDecoder())

	return io.ReadAll(utf8Reader)

}

func DeterminEncoding(r *bufio.Reader) encoding.Encoding {
	bytes, err := r.Peek(1024)
	if err != nil {
		fmt.Println("determin encoding err:", err)
		return unicode.UTF8
	}
	e, _, _ := charset.DetermineEncoding(bytes, "")
	return e
}

var logReg = regexp.MustCompile(`order_id=(\d+)`)

func dumpLogOrderId() {
	var log = `
	2022-11-22  name=versionReport||order_id=3732978217343693673||trace_id=XXX
	2022-11-22  name=versionReport||order_id=3732978217343693674||trace_id=XXX
	`
	matches := logReg.FindAllStringSubmatch(log, -1)
	for _, match := range matches {
		fmt.Println("match:", match[1])
	}
	// 2.exec command
	cmdName := `grep -oE 'order_id=\d+' | grep -oE '\d+'`
	cmd := exec.Command(cmdName)
	out, err := cmd.CombinedOutput()
	fmt.Println("cmd res:", string(out), "err:", err)

}
