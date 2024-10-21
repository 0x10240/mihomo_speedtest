package tester

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/0x10240/mihomo-speedtest/config"
	"github.com/0x10240/mihomo-speedtest/result"
	C "github.com/metacubex/mihomo/constant"
)

func TestProxies(names []string, proxies map[string]config.CProxy, sizeMB int, timeout time.Duration, concurrent int, livenessObject string) []result.Result {
	results := make([]result.Result, 0, len(names))
	fmt.Printf("%-42s\t%-12s\t%-12s\n", "Node", "Bandwidth", "Latency")

	for _, name := range names {
		proxy := proxies[name]
		switch proxy.Type() {
		case C.Shadowsocks, C.ShadowsocksR, C.Snell, C.Socks5, C.Http, C.Vmess, C.Vless, C.Trojan, C.Hysteria, C.Hysteria2, C.WireGuard, C.Tuic:
			downloadSize := sizeMB * 1024 * 1024
			res := testProxyConcurrent(name, proxy.Proxy, downloadSize, timeout, concurrent, livenessObject)
			res.Print()
			results = append(results, res)
		default:
			continue // Skip unsupported proxy types
		}
	}
	return results
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

	return result.Result{
		Name:      name,
		Bandwidth: bandwidth,
		TTFB:      avgTTFB,
	}
}

func testProxy(name string, proxy C.Proxy, downloadSize int, timeout time.Duration, livenessObject string) (result.Result, int64) {
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
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
		},
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
