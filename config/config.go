package config

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/metacubex/mihomo/adapter"
	"github.com/metacubex/mihomo/adapter/provider"
	"github.com/metacubex/mihomo/constant"
	"gopkg.in/yaml.v3"
)

type CProxy struct {
	constant.Proxy
	SecretConfig any
}

type RawConfig struct {
	Providers map[string]map[string]any `yaml:"proxy-providers"`
	Proxies   []map[string]any          `yaml:"proxies"`
}

func LoadAllProxies(configPaths string) map[string]CProxy {
	allProxies := make(map[string]CProxy)

	for _, configPath := range strings.Split(configPaths, ",") {
		body, err := readConfig(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read config from %s: %v\n", configPath, err)
			continue
		}

		proxies, err := loadProxies(body)
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

func readConfig(configPath string) ([]byte, error) {
	if strings.HasPrefix(configPath, "http") {
		resp, err := http.Get(configPath)
		if err != nil {
			return nil, fmt.Errorf("HTTP GET failed: %v", err)
		}
		defer resp.Body.Close()
		return io.ReadAll(resp.Body)
	}
	return os.ReadFile(configPath)
}

func loadProxies(data []byte) (map[string]CProxy, error) {
	rawCfg := &RawConfig{}
	if err := yaml.Unmarshal(data, rawCfg); err != nil {
		return nil, fmt.Errorf("YAML unmarshal failed: %v", err)
	}

	proxies := make(map[string]CProxy)

	// Load individual proxies
	for _, config := range rawCfg.Proxies {
		proxy, err := adapter.ParseProxy(config)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse proxy: %v", err)
		}
		if _, exists := proxies[proxy.Name()]; exists {
			return nil, fmt.Errorf("Duplicate proxy name: %s", proxy.Name())
		}
		proxies[proxy.Name()] = CProxy{Proxy: proxy, SecretConfig: config}
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
			proxies[proxyName] = CProxy{Proxy: proxy}
		}
	}

	return proxies, nil
}
