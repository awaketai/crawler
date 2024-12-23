package proxy

import (
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"
)

type ProxyFunc func(*http.Request) (*url.URL, error)

func RoundRobinProxySwitcher(proxyURLs ...string) (ProxyFunc, error) {
	if len(proxyURLs) < 1 {
		return nil, fmt.Errorf("proxy URL list empty")
	}
	urls := make([]*url.URL, len(proxyURLs))

	for i, u := range proxyURLs {
		parsedU, err := url.Parse(u)
		if err != nil {
			return nil, err
		}
		urls[i] = parsedU
	}

	return (&roundRobinSwitcher{urls, 0}).GetPorxy, nil
}

type roundRobinSwitcher struct {
	proxyURLs []*url.URL
	index     uint32
}

func (r *roundRobinSwitcher) GetPorxy(_ *http.Request) (*url.URL, error) {
	index := atomic.AddUint32(&r.index, 1) - 1
	u := r.proxyURLs[index%uint32(len(r.proxyURLs))]

	return u, nil
}
