package filter

import (
	"regexp"
	"sort"

	"github.com/0x10240/mihomo-speedtest/config"
)

func FilterProxies(filter string, proxies map[string]config.CProxy) []string {
	filterRegexp := regexp.MustCompile(filter)
	filteredProxies := make([]string, 0)
	for name := range proxies {
		if filterRegexp.MatchString(name) {
			filteredProxies = append(filteredProxies, name)
		}
	}
	sort.Strings(filteredProxies)
	return filteredProxies
}
