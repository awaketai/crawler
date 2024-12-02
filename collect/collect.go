package collect

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/awaketai/crawler/proxy"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

type Fetcher interface {
	Get(*Request) ([]byte, error)
}

type BaseFetch struct {
}

func (BaseFetch) Get(req *Request) ([]byte, error) {
	resp, err := http.Get(req.Url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("error http status:%v %v", resp.StatusCode, resp.Status)
		return nil, fmt.Errorf("error http status:%v", resp.Status)
	}
	bodyReader := bufio.NewReader(resp.Body)
	e := DeterminEncoding(bodyReader)
	utf8Reader := transform.NewReader(bodyReader, e.NewDecoder())

	return io.ReadAll(utf8Reader)
}

type BrowserFetch struct {
	Timeout time.Duration
	Proxy proxy.ProxyFunc
}

func (b BrowserFetch) Get(req *Request) ([]byte, error) {
	client := &http.Client{
		Timeout: b.Timeout,
	}
	// 设置代理服务
	if b.Proxy != nil {
		trasnport := http.DefaultTransport.(*http.Transport)
		trasnport.Proxy = b.Proxy
		client.Transport = trasnport
	}
	
	request, err := http.NewRequest("GET", req.Url, nil)
	if err != nil {
		return nil, err
	}
	if len(req.Cookie) > 0 {
		request.Header.Set("Cookie",req.Cookie)
	}
	request.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:109.0) Gecko/20100101 Firefox/117.0")
	resp, err := client.Do(request)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("error http status:%v %v", resp.StatusCode, resp.Status)
		return nil, fmt.Errorf("error http status:%v", resp.Status)
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
