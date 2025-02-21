package proxy

import (
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func FuzzGetProxy(f *testing.F) {
	f.Add(uint32(1), uint32(10))
	f.Fuzz(func(t *testing.T, index uint32, urlCounts uint32) {
		r := roundRobinSwitcher{}
		r.index = index
		r.proxyURLs = make([]*url.URL, urlCounts)

		for i := 0; i < int(urlCounts); i++ {
			r.proxyURLs[i] = &url.URL{}
			r.proxyURLs[i].Host = strconv.Itoa(i)
		}
		p, err := r.GetProxy(nil)
		if err != nil && strings.Contains(err.Error(),"proxy URL list empty") {
			t.Skip()
		}
		assert.Nil(t, err)

		e := r.proxyURLs[index%urlCounts]
		if !reflect.DeepEqual(p, e) {
			t.Fatalf("expect %v, got %v", e, p)
		}

	})

}
