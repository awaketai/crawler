package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os/exec"
	"reflect"
	"regexp"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/antchfx/htmlquery"
	log2 "github.com/awaketai/crawler/log"
	"github.com/chromedp/chromedp"
	"go.uber.org/zap/zapcore"
)

// 1.使用正则表达式
// 2.使用xpath
// 3.使用css选择器
var headerRe = regexp.MustCompile(`<div class="small_imgposition__PYVLm"><img.*?alt=([\s\S]*?) src`)

func main() {
	// simulation()
	logTest()
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

func getTitle(res []byte) {
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

func simulation() {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	// 爬取页面，等待一个元素出现，接着模拟鼠标点击，最后获取数据
	var example string
	err := chromedp.Run(ctx,
		chromedp.Navigate(`https://pkg.go.dev/time`),
		chromedp.WaitVisible(`body > footer`),
		chromedp.Click(`#example-After`, chromedp.NodeVisible),
		chromedp.Value(`#example-After textarea`, &example),
	)
	if err != nil {
		log.Fatal("err:", err)
	}
	fmt.Println()
	log.Printf("Go's time.After exampel:\\n%s", example)
}

type Trade struct {
	tradeId int
	Price   int
}

type Student struct {
	Name string
	Age  int
}

func createQUery(q any) string {
	if reflect.ValueOf(q).Kind() == reflect.Struct {
		// 获取结构体名字
		t := reflect.TypeOf(q).Name()
		// 查询语句
		query := fmt.Sprintf("insert into %s values(", t)
		v := reflect.ValueOf(q)
		// 遍历结构体字段
		for i := 0; i < v.NumField(); i++ {
			switch v.Field(i).Kind() {
			case reflect.Int:
				if i == 0 {
					query = fmt.Sprintf("%s%d", query, v.Field(i).Int())
				} else {
					query = fmt.Sprintf("%s,%d", query, v.Field(i).Int())
				}
			case reflect.String:
				if i == 0 {
					query = fmt.Sprintf(`%s"%s"`, query, v.Field(i).String())
				} else {
					query = fmt.Sprintf(`%s ,"%s"`, query, v.Field(i).String())
				}
			}

		}
		query = fmt.Sprintf("%s)", query)
		fmt.Println("query:", query)
		return query

	}
	return ""
}

func handleHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("method:", r.Method)
	resp, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func copyHeader(dst, src http.Header) {
	for k, v := range src {
		for _, vo := range v {
			dst.Add(k, vo)
		}
	}
}

func main2() {
	// 1.direct proxy
	// server := &http.Server{
	// 	Addr: ":8083",
	// 	Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 		handleHTTP(w,r)
	// 	}),
	// }
	// log.Fatal(server.ListenAndServe())
	// 2.tunnel proxy

	// server := &http.Server{

	// 	Addr: ":8084",
	// 	Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 		if r.Method == http.MethodConnect {
	// 			handleTunneling(w,r)
	// 		}else{
	// 			handleHTTP(w,r)
	// 		}
	// 	}),
	// }
	// log.Fatal(server.ListenAndServe())

	// 2.reverse proxy
	proxy, err := NewProxy()
	if err != nil {
		panic(err)
	}
	http.HandleFunc("/", ProxyRequestHandler(proxy))
	log.Fatal(http.ListenAndServe(":8085", nil))
}

func handleTunneling(w http.ResponseWriter, r *http.Request) {
	fmt.Println("method:", r.Method)
	dstConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	// 接管http 请求
	hijaker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	cliConn, _, err := hijaker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	go transfer(dstConn, cliConn)
	go transfer(cliConn, dstConn)
}

func transfer(dest io.WriteCloser, src io.ReadCloser) {
	defer dest.Close()
	defer src.Close()
	io.Copy(dest, src)
}

// 反向代理
func ProxyRequestHandler(proxy *httputil.ReverseProxy) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
		w.Write([]byte("111"))
	}
}

func NewProxy() (*httputil.ReverseProxy, error) {
	// 要代理到的地址
	host := "http://www.baidu.com"
	url, err := url.Parse(host)
	if err != nil {
		return nil, err
	}
	proxy := httputil.NewSingleHostReverseProxy(url)
	return proxy, nil
}

func logTest() {
	plugin, c := log2.NewFilePlugin("./log.log", zapcore.InfoLevel)
	defer c.Close()
	logger := log2.NewLogger(plugin)
	logger.Info("log init end")

}
