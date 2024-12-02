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
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/antchfx/htmlquery"
	"github.com/awaketai/crawler/collect"
	log2 "github.com/awaketai/crawler/log"
	"github.com/awaketai/crawler/parse/doubangroup"
	"github.com/awaketai/crawler/proxy"
	"github.com/chromedp/chromedp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// 1.使用正则表达式
// 2.使用xpath
// 3.使用css选择器
var headerRe = regexp.MustCompile(`<div class="small_imgposition__PYVLm"><img.*?alt=([\s\S]*?) src`)

func main() {
	// simulation()
	// logTest()
	// pingPong()
	// search2Test()
	// fallIn()
	// subWorkerTest()
	// course()
	douban()
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

var (
	flag  int64 = 0
	count int64 = 0
	mu    sync.WaitGroup
)

func atomicTest() {

	for {
		if atomic.CompareAndSwapInt64(&flag, 0, 1) {
			count++
			atomic.StoreInt64(&flag, 0)
			return
		}
	}
}

// ping pong模式
func pingPong() {
	var ball int
	table := make(chan int)
	go player(table)
	go player(table)
	table <- ball
	time.Sleep(1 * time.Second)
	a := <-table
	fmt.Println("--a:", a)
}

func player(table chan int) {
	for {
		ball := <-table
		ball++
		time.Sleep(100 * time.Millisecond)
		table <- ball
	}
}

// 查找某个文件夹中是否有特殊的关键字
// 并发方式查询

func search(ch chan string, msg string) {
	var i int
	for {
		ch <- fmt.Sprintf("get %s %d", msg, i)
		i++
		time.Sleep(1000 * time.Millisecond)
	}
}

func searchTest() {
	ch := make(chan string)
	go search(ch, "jonson")
	go search(ch, "ola")
	for i := range ch {
		fmt.Println(i)
	}
}

func search2(msg string) chan string {
	var ch = make(chan string)
	go func() {
		var i int
		for {
			ch <- fmt.Sprintf("get %s %d", msg, i)
			i++
			time.Sleep(100 * time.Millisecond)
		}
	}()
	return ch
}

func search2Test() {
	ch1 := search2("josn")
	ch2 := search2("aa")
	for {
		select {
		case msg := <-ch1:
			fmt.Println("msg1:", msg)
		case msg := <-ch2:
			fmt.Println("msg2:", msg)

		}
	}
}

func worker(ch <-chan int, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		task, ok := <-ch
		if !ok {
			return
		}
		d := time.Duration(task) * time.Millisecond
		time.Sleep(d)
		fmt.Println("processing task", task)

	}
}

func pool(wg *sync.WaitGroup, workers, tasks int) {
	ch := make(chan int)
	for i := 0; i < workers; i++ {
		go worker(ch, wg)
	}
	for i := 0; i < tasks; i++ {
		ch <- i
	}
	close(ch)
}

func fallIn() {
	var wg sync.WaitGroup
	wg.Add(36)
	go pool(&wg, 36, 50)
	wg.Wait()
}

const (
	WORKERS    = 5
	SUBWORKERS = 3
	TASKSK     = 20
	SUBTASKS   = 10
)

func subworker(subtasks chan int) {
	for {
		task, ok := <-subtasks
		if !ok {
			return
		}
		time.Sleep(time.Duration(task) * time.Millisecond)
		fmt.Println("sub worker:", task)
	}
}

func worker2(tasks <-chan int, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		task, ok := <-tasks
		if !ok {
			return
		}
		subtasks := make(chan int)
		for i := 0; i < SUBTASKS; i++ {
			go subworker(subtasks)
		}
		for i := 0; i < SUBTASKS; i++ {
			task1 := task * i
			subtasks <- task1
		}
		close(subtasks)
	}
}

func subWorkerTest() {
	var wg sync.WaitGroup
	wg.Add(WORKERS)
	tasks := make(chan int)
	for i := 0; i < WORKERS; i++ {
		go worker2(tasks, &wg)
	}

	for i := 0; i < TASKSK; i++ {
		tasks <- i
	}
	close(tasks)
	wg.Wait()
}

// 计算机课程和其前序课程的映射关系
var prereqs = map[string][]string{"algorithms": {"data structures"}, "calculus": {"linear algebra"}, "compilers": {"data structures", "formal languages", "computer organization"}, "data structures": {"discrete math"}, "databases": {"data structures"}, "discrete math": {"intro to programming"}, "formal languages": {"discrete math"}, "networks": {"operating systems"}, "operating systems": {"data structures", "computer organization"}, "programming languages": {"data structures", "computer organization"}}

func course() {
	for i, course := range topoSort(prereqs) {
		fmt.Printf("%d:\t%s\n", i+1, course)
	}
}

func topoSort(m map[string][]string) []string {
	var order []string
	seen := make(map[string]bool)
	var visitAll func(items []string)

	visitAll = func(items []string) {
		for _, item := range items {
			if !seen[item] {
				seen[item] = true
				visitAll(m[item])
				order = append(order, item)
			}
		}
	}
	var keys []string
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	visitAll(keys)
	return order
}

func douban() {
	plugin := log2.NewStdoutPlugin(zapcore.InfoLevel)
	logger := log2.NewLogger(plugin)
	logger.Info("log init end")
	cookie := `viewed="27043167_25863515_10746113_2243615_36667173_1007305_1091086"; __utma=30149280.1138703939.1688435343.1733118222.1733122303.10; ll="108288"; bid=p4zwdHrVY7w; __utmz=30149280.1729597487.8.2.utmcsr=ruanyifeng.com|utmccn=(referral)|utmcmd=referral|utmcct=/; _pk_id.100001.8cb4=18c04f5fb62d2e52.1733118221.; __utmc=30149280; dbcl2="285159894:dMkA02qtf50"; ck=tQmt; push_noty_num=0; push_doumail_num=0; __utmv=30149280.28515; __yadk_uid=3D5K4bndWlX7TLf8CjyAjVV5aB26MFa8; loc-last-index-location-id="108288"; _vwo_uuid_v2=DA5C0F35C5141ECEE7520D43DF2106264|8d200da2a9f789409ca0ce01e00d2789; frodotk_db="4a184671f7672f9cde48d355e6358ed4"; _pk_ses.100001.8cb4=1; __utmb=30149280.26.9.1733123639802; __utmt=1`
	var worklist []*collect.Request
	for i := 0; i <= 25; i += 25 {
		str := fmt.Sprintf("https://www.douban.com/group/280198/discussion?start=%d&type=new", i)
		worklist = append(worklist, &collect.Request{
			Url:       str,
			ParseFunc: doubangroup.ParseURL,
			Cookie:    cookie,
		})
	}
	proxyURLs := []string{"http://127.0.0.1:4780"}
	p, err := proxy.RoundRobinProxySwitcher(proxyURLs...)
	if err != nil {
		logger.Error("RoundRobinProxySwitcher err:", zap.Error(err))
	}

	var f collect.Fetcher = &collect.BrowserFetch{
		Timeout: 10 * time.Second,
		Proxy:   p,
	}

	for len(worklist) > 0 {
		items := worklist
		worklist = nil
		for _, item := range items {
			body, err := f.Get(item)
			time.Sleep(1 * time.Second)
			if err != nil {
				logger.Error("read content err", zap.Error(err))
				continue
			}
			res := item.ParseFunc(body, item)
			for _, item := range res.Items {
				logger.Info("result", zap.String("get url:", item.(string)))
			}
			worklist = append(worklist, res.Requests...)
		}
	}
}
