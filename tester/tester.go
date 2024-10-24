package tester

import (
	"context"
	"fmt"
	"github.com/go-resty/resty/v2"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/0x10240/mihomo-speedtest/config"
	"github.com/0x10240/mihomo-speedtest/result"
	cutils "github.com/metacubex/mihomo/common/utils"
	C "github.com/metacubex/mihomo/constant"
)

func TestProxiesDelay(proxies map[string]config.CProxy, delayTestUrl string, timeout time.Duration) []result.Result {
	results := make([]result.Result, 0, len(proxies))
	mu := sync.Mutex{} // 用于保护 results 切片的并发写操作
	expectedStatus, _ := cutils.NewUnsignedRanges[uint16]("200")

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 16) // 并发限制为16

	for name, proxy := range proxies {
		wg.Add(1)
		// 启动一个 goroutine
		go func(name string, proxy config.CProxy) {
			defer wg.Done()
			semaphore <- struct{}{} // 获取一个令牌，控制并发数

			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			delay, err := proxy.URLTest(ctx, delayTestUrl, expectedStatus)
			if err != nil {
				delay = 9999
			}
			res := result.Result{Name: name, Delay: delay}
			if delay != 9999 {
				setProxyOutboundIP(proxy, &res, timeout)
			}
			// 使用互斥锁保护 results 的写入
			mu.Lock()
			results = append(results, res)
			mu.Unlock()

			<-semaphore // 释放令牌
		}(name, proxy)
	}

	wg.Wait() // 等待所有的 goroutine 完成
	return results
}

func TestProxies(names []string, proxies map[string]config.CProxy, sizeMB int, timeout time.Duration, concurrent int, livenessObject string) []result.Result {
	results := make([]result.Result, 0, len(names))
	fmt.Printf("%-42s\t%-12s\t%-12s\n", "Node", "Bandwidth", "Latency")

	for _, name := range names {
		proxy := proxies[name]
		switch proxy.Type() {
		case C.Shadowsocks, C.ShadowsocksR, C.Snell, C.Socks5, C.Http, C.Vmess, C.Vless, C.Trojan, C.Hysteria, C.Hysteria2, C.WireGuard, C.Tuic:
			downloadSize := sizeMB * 1024 * 1024
			res := testProxyConcurrent(name, proxy, downloadSize, timeout, concurrent, livenessObject)
			res.Print()
			results = append(results, res)
		default:
			continue // Skip unsupported proxy types
		}
	}
	return results
}

func getProxyTransport(proxy C.Proxy) *http.Transport {
	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, portStr, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			port, err := strconv.ParseUint(portStr, 10, 16)
			if err != nil {
				return nil, err
			}
			return proxy.DialContext(ctx, &C.Metadata{
				Host:    host,
				DstPort: uint16(port),
			})
		},
	}
}

func setProxyOutboundIP(proxy C.Proxy, res *result.Result, timeout time.Duration) {
	client := resty.New()
	transport := getProxyTransport(proxy)
	client.SetTimeout(timeout)
	client.SetTransport(transport)
	resp, err := client.R().Get("https://speed.cloudflare.com/__down?bytes=1")
	if err != nil {
		return
	}
	res.OutBoundIp = resp.Header().Get("Cf-Meta-Ip")
	res.Country = resp.Header().Get("Cf-Meta-Country")
	//fmt.Printf("%v outbount ip: %v\n", res.Name, res.OutBoundIp)
}

func testProxyConcurrent(name string, proxy C.Proxy, downloadSize int, timeout time.Duration, concurrentCount int, livenessObject string) result.Result {
	if concurrentCount <= 0 {
		concurrentCount = 1
	}

	chunkSize := downloadSize / concurrentCount
	totalTTFB := int64(0)
	downloaded := int64(0)

	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < concurrentCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, bytes := testProxy(name, proxy, chunkSize, timeout, livenessObject)
			if bytes != 0 {
				atomic.AddInt64(&downloaded, bytes)
				atomic.AddInt64(&totalTTFB, int64(res.TTFB))
			}
		}()
	}
	wg.Wait()

	downloadTime := time.Since(start)
	avgTTFB := time.Duration(totalTTFB / int64(concurrentCount))
	bandwidth := float64(downloaded) / downloadTime.Seconds()

	res := result.Result{
		Name:      name,
		Bandwidth: bandwidth,
		TTFB:      avgTTFB,
	}

	setProxyOutboundIP(proxy, &res, timeout)
	return res
}

func testProxy(name string, proxy C.Proxy, downloadSize int, timeout time.Duration, livenessObject string) (result.Result, int64) {
	client := &http.Client{
		Timeout:   timeout,
		Transport: getProxyTransport(proxy),
	}

	start := time.Now()
	resp, err := client.Get(fmt.Sprintf(livenessObject, downloadSize))
	if err != nil {
		return result.Result{Name: name, Bandwidth: -1, TTFB: -1}, 0
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return result.Result{Name: name, Bandwidth: -1, TTFB: -1}, 0
	}

	ttfb := time.Since(start)
	written, _ := io.Copy(io.Discard, resp.Body)
	if written == 0 {
		return result.Result{Name: name, Bandwidth: -1, TTFB: -1}, 0
	}

	downloadTime := time.Since(start) - ttfb
	bandwidth := float64(written) / downloadTime.Seconds()

	return result.Result{Name: name, Bandwidth: bandwidth, TTFB: ttfb}, written
}
