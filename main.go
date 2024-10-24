package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/0x10240/mihomo-speedtest/config"
	"github.com/0x10240/mihomo-speedtest/filter"
	"github.com/0x10240/mihomo-speedtest/output"
	"github.com/0x10240/mihomo-speedtest/result"
	"github.com/0x10240/mihomo-speedtest/tester"
)

var (
	livenessObject     = flag.String("l", "https://speed.cloudflare.com/__down?bytes=%d", "URL of the target to test, supports custom size")
	configPathConfig   = flag.String("c", "", "Configuration file path or URL")
	filterRegexConfig  = flag.String("f", ".*", "Filter node names using regular expressions")
	downloadSizeConfig = flag.Int("size", 100, "Download size for testing (in MB)")
	timeoutConfig      = flag.Duration("timeout", 5*time.Second, "Timeout duration for testing")
	sortField          = flag.String("sort", "b", "Sort field: 'b' for bandwidth, 't' for latency")
	outputFormat       = flag.String("output", "", "Output results to 'csv' or 'yaml' file")
	concurrent         = flag.Int("concurrent", 4, "Number of concurrent downloads")
	proxy              = flag.String("proxy", "", "proxy to get resource")
	forwardProxy       = flag.String("forward-proxy", "", "Forward proxy, supporting SOCKS5 and HTTP proxy.")
	delayTest          = flag.Bool("delay", false, "only delay testing")
	delayTestUrl       = flag.String("delayurl", "https://www.gstatic.com/generate_204", "delay test url")
)

func main() {
	flag.Parse()

	if *configPathConfig == "" {
		fmt.Fprintln(os.Stderr, "Please specify the configuration file using the -c flag")
		os.Exit(1)
	}

	// Load all proxies
	allProxies := config.LoadAllProxies(*configPathConfig, *proxy, *forwardProxy)
	if len(allProxies) == 0 {
		fmt.Fprintln(os.Stderr, "No proxies found, please check the configuration file")
		os.Exit(1)
	}

	// Filter proxies
	filteredProxies := filter.FilterProxies(*filterRegexConfig, allProxies)
	if len(filteredProxies) == 0 {
		fmt.Fprintln(os.Stderr, "No matching proxies found")
		os.Exit(1)
	}

	// Test proxies
	var results []result.Result
	if *delayTest {
		results = tester.TestProxiesDelay(allProxies, *delayTestUrl, *timeoutConfig)
		result.DisplayDelayResult(results)
		return
	}

	results = tester.TestProxies(filteredProxies, allProxies, *downloadSizeConfig, *timeoutConfig, *concurrent, *livenessObject)

	// Sort results
	if *sortField != "" {
		result.SortResults(results, *sortField)
	}

	// Display results
	result.DisplayResults(results, *sortField)

	// Output to file
	if *outputFormat != "" {
		if err := output.WriteResultsToFile(*outputFormat, results, allProxies); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write results to file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Results have been written to the %s file\n", *outputFormat)
	}
}
