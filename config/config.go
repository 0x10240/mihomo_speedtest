package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/metacubex/mihomo/adapter"
	"github.com/metacubex/mihomo/adapter/provider"
	"github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/tunnel"
	"gopkg.in/yaml.v3"
)

type CProxy = constant.Proxy

type RawConfig struct {
	Providers map[string]map[string]any `yaml:"proxy-providers"`
	Proxies   []map[string]any          `yaml:"proxies"`
}

func LoadAllProxies(configPaths string, proxy string, forwardProxy string) map[string]CProxy {
	allProxies := make(map[string]CProxy)

	for _, configPath := range strings.Split(configPaths, ",") {
		body, err := readConfig(configPath, proxy)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read config from %s: %v\n", configPath, err)
			continue
		}

		proxies, err := loadProxies(body, forwardProxy)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse config from %s: %v\n", configPath, err)
			continue
		}

		for name, proxy := range proxies {
			if _, exists := allProxies[name]; !exists {
				allProxies[name] = proxy
			}
		}
	}

	return allProxies
}

func readConfig(configPath string, proxy string) ([]byte, error) {
	if strings.HasPrefix(configPath, "http") {
		// 创建 Resty 客户端
		client := resty.New()

		// 如果 proxy 不为空，设置代理
		if proxy != "" {
			client.SetProxy(proxy)
		}

		// 发起 GET 请求
		resp, err := client.R().
			SetHeader("User-Agent", "clash.meta").
			Get(configPath)

		if err != nil {
			return nil, fmt.Errorf("HTTP GET failed: %v", err)
		}

		// 返回响应体
		return resp.Body(), nil
	}

	// 如果 configPath 不是 HTTP URL，读取本地文件
	return os.ReadFile(configPath)
}

func parseProxyLink(proxyURL string) (map[string]any, error) {
	parsedURL, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}

	// Supported schemes: http, socks5
	supportedSchemes := map[string]bool{
		"socks5": true,
		"http":   true,
	}

	if _, ok := supportedSchemes[parsedURL.Scheme]; !ok {
		return nil, fmt.Errorf("unsupported proxy scheme: %s", parsedURL.Scheme)
	}

	hostParts := strings.Split(parsedURL.Host, ":")
	if len(hostParts) != 2 {
		return nil, fmt.Errorf("invalid host format: %s", parsedURL.Host)
	}

	port, err := strconv.Atoi(hostParts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid port: %s", hostParts[1])
	}

	// Create the result map with required fields
	result := map[string]any{
		"name":   "dialer",
		"type":   parsedURL.Scheme,
		"server": hostParts[0],
		"port":   port,
	}

	// Add username and password if present
	if parsedURL.User != nil {
		if username := parsedURL.User.Username(); username != "" {
			result["username"] = username
		}
		if password, hasPassword := parsedURL.User.Password(); hasPassword {
			result["password"] = password
		}
	}

	return result, nil
}

func loadProxies(data []byte, forwardProxy string) (map[string]CProxy, error) {
	rawCfg := &RawConfig{}
	err := yaml.Unmarshal(data, rawCfg)
	if err != nil {
		return nil, fmt.Errorf("YAML unmarshal failed: %v", err)
	}

	// 前置代理
	var dial_config map[string]any
	if forwardProxy != "" {
		dial_config, err = parseProxyLink(forwardProxy)
		if err != nil {
			return nil, err
		}
		rawCfg.Proxies = append(rawCfg.Proxies, dial_config)
	}

	proxies := make(map[string]CProxy)

	// Load individual proxies
	for _, config := range rawCfg.Proxies {
		if forwardProxy != "" && config["name"] != "dialer" {
			config["dialer-proxy"] = "dialer"
		}
		proxy, err := adapter.ParseProxy(config)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse proxy: %v err:%v", config, err)
		}
		if _, exists := proxies[proxy.Name()]; exists {
			return nil, fmt.Errorf("Duplicate proxy name: %s", proxy.Name())
		}
		proxies[proxy.Name()] = proxy
	}

	// Load proxies from providers
	for name, config := range rawCfg.Providers {
		if name == provider.ReservedName {
			return nil, fmt.Errorf("Provider name '%s' is reserved", provider.ReservedName)
		}
		pd, err := provider.ParseProxyProvider(name, config)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse provider %s: %v", name, err)
		}
		if err := pd.Initial(); err != nil {
			return nil, fmt.Errorf("Failed to initialize provider %s: %v", pd.Name(), err)
		}
		for _, proxy := range pd.Proxies() {
			proxyName := fmt.Sprintf("[%s] %s", name, proxy.Name())
			proxies[proxyName] = proxy
		}
	}

	tunnel.UpdateProxies(proxies, nil)

	return proxies, nil
}
