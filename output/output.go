package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/0x10240/mihomo-speedtest/config"
	"github.com/0x10240/mihomo-speedtest/result"
	"gopkg.in/yaml.v3"
)

func WriteResultsToFile(format string, filePath string, results []result.Result, proxies map[string]config.CProxy) error {
	switch format {
	case "json":
		return writeResultsToJson(filePath, results)
	case "yaml":
		return writeResultsToYAML(filePath, results, proxies)
	case "csv":
		return writeResultsToCSV(filePath, results)
	default:
		return fmt.Errorf("Unsupported output: %v format: %s", filePath, format)
	}
}

// writeResultsToJson writes a slice of Result structs to a JSON file at the specified file path.
func writeResultsToJson(filePath string, results []result.Result) error {
	// Create or overwrite the specified JSON file
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Convert the results to JSON format with indentation
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}

	// Write the JSON data to the file
	_, err = file.Write(data)
	return err
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
