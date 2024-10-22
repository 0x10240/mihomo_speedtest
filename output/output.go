package output

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"

	"github.com/0x10240/mihomo-speedtest/config"
	"github.com/0x10240/mihomo-speedtest/result"
	"gopkg.in/yaml.v3"
)

func WriteResultsToFile(format string, results []result.Result, proxies map[string]config.CProxy) error {
	switch format {
	case "yaml":
		return writeResultsToYAML("result.yaml", results, proxies)
	case "csv":
		return writeResultsToCSV("result.csv", results)
	default:
		return fmt.Errorf("Unsupported output format: %s", format)
	}
}

func writeResultsToYAML(filePath string, results []result.Result, proxies map[string]config.CProxy) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var sortedProxies []any
	for _, res := range results {
		if proxy, exists := proxies[res.Name]; exists {
			sortedProxies = append(sortedProxies, proxy)
		}
	}

	data, err := yaml.Marshal(sortedProxies)
	if err != nil {
		return err
	}

	_, err = file.Write(data)
	return err
}

func writeResultsToCSV(filePath string, results []result.Result) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write UTF-8 BOM for Excel compatibility
	file.WriteString("\xEF\xBB\xBF")

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"Node", "Bandwidth (MB/s)", "Latency (ms)"})

	for _, res := range results {
		line := []string{
			res.Name,
			fmt.Sprintf("%.2f", res.Bandwidth/1024/1024),
			strconv.FormatInt(res.TTFB.Milliseconds(), 10),
		}
		writer.Write(line)
	}

	return nil
}
