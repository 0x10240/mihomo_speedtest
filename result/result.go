package result

import (
	"fmt"
	"github.com/olekukonko/tablewriter"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

type Result struct {
	Name       string
	OutBoundIp string
	Country    string
	Bandwidth  float64
	TTFB       time.Duration
	Delay      uint16
}

func (r *Result) Print() {
	fmt.Printf("%-42s\t%-12s\t%-12s\n", formatName(r.Name), formatBandwidth(r.Bandwidth), formatMilliseconds(r.TTFB))
}
func formatName(name string) string {
	// 使用过滤函数来移除 emoji 或符号字符
	noEmoji := removeEmoji(name)

	// 匹配多个空格的正则表达式
	spaceRegex := regexp.MustCompile(`\s{2,}`)

	// 替换连续空格为单个空格
	mergedSpaces := spaceRegex.ReplaceAllString(noEmoji, " ")

	// 返回去掉首尾空格的结果
	return strings.TrimSpace(mergedSpaces)
}

// 定义一个函数来过滤掉字符串中的 emoji
func removeEmoji(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.Is(unicode.S, r) || unicode.Is(unicode.Sk, r) || unicode.Is(unicode.So, r) {
			// 符号 (S) 包含了一些 emoji，所以这里可以过滤掉符号类型字符
			return -1
		}
		return r
	}, s)
}

func formatBandwidth(v float64) string {
	if v <= 0 {
		return "N/A"
	}
	units := []string{"B/s", "KB/s", "MB/s", "GB/s", "TB/s"}
	i := 0
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	return fmt.Sprintf("%.2f%s", v, units[i])
}

func formatMilliseconds(d time.Duration) string {
	if d <= 0 {
		return "N/A"
	}
	return fmt.Sprintf("%.2fms", float64(d.Milliseconds()))
}

func formatDelay(d uint16) string {
	if d == 9999 {
		return "N/A"
	}
	return fmt.Sprintf("%d", d)
}

func SortResults(results []Result, sortBy string) {
	switch sortBy {
	case "b", "bandwidth":
		sort.Slice(results, func(i, j int) bool {
			return results[i].Bandwidth > results[j].Bandwidth
		})
	case "t", "ttfb":
		sort.Slice(results, func(i, j int) bool {
			return results[i].TTFB < results[j].TTFB
		})
	case "d", "delay":
		sort.Slice(results, func(i, j int) bool {
			return results[i].Delay < results[j].Delay
		})
	}
}

func DisplayDelayResult(results []Result) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Node", "Delay(ms)", "IP", "Country"})

	SortResults(results, "delay")

	for _, res := range results {
		data := []string{
			formatName(res.Name),
			formatDelay(res.Delay),
			fmt.Sprintf("%v", res.OutBoundIp),
			fmt.Sprintf("%v", res.Country),
		}
		table.Append(data)
	}

	table.Render()
}

func DisplayResults(results []Result, sortedBy string) {
	if sortedBy != "" {
		fmt.Printf("\nResults sorted by %s:\n", sortedBy)
	} else {
		fmt.Printf("\nResults:\n")
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Node", "Bandwidth", "Latency", "IP", "Country"})

	for _, res := range results {
		data := []string{
			formatName(res.Name),
			fmt.Sprintf("%v", formatBandwidth(res.Bandwidth)),
			fmt.Sprintf("%v", formatMilliseconds(res.TTFB)),
			fmt.Sprintf("%v", res.OutBoundIp),
			fmt.Sprintf("%v", res.Country),
		}
		table.Append(data)
	}

	table.Render()
}
