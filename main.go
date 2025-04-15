package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

type Endpoint struct {
	Name    string            `yaml:"name"`
	URL     string            `yaml:"url"`
	Method  string            `yaml:"method"`
	Headers map[string]string `yaml:"headers"`
	Body    string            `yaml:"body"`
}

type DomainStats struct {
	Success int
	Total   int
}

var stats = make(map[string]*DomainStats)

func extractDomain(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		
		urlSplit := strings.Split(rawURL, "//")
		domain := strings.Split(urlSplit[len(urlSplit)-1], "/")[0]
		return domain
	}
	return parsed.Hostname() 
}

func checkHealth(endpoint Endpoint) {
	domain := extractDomain(endpoint.URL)
	if stats[domain] == nil {
		stats[domain] = &DomainStats{}
	}

	var reqBody *bytes.Reader
	if endpoint.Body != "" {
		reqBody = bytes.NewReader([]byte(endpoint.Body))
	} else {
		reqBody = bytes.NewReader(nil)
	}

	req, err := http.NewRequest(endpoint.Method, endpoint.URL, reqBody)
	if err != nil {
		log.Printf("Error creating request for %s: %v\n", endpoint.Name, err)
		stats[domain].Total++
		return
	}

	for key, value := range endpoint.Headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: 500 * time.Millisecond}
	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)

	stats[domain].Total++

	if err != nil {
		log.Printf("Request error for %s: %v\n", endpoint.Name, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("Non-2xx response for %s: %d\n", endpoint.Name, resp.StatusCode)
	} else if duration > 500*time.Millisecond {
		log.Printf("Slow response for %s: %v > 500ms\n", endpoint.Name, duration)
	} else {
		stats[domain].Success++
	}
}

func logResults() {
	fmt.Println("----- AVAILABILITY REPORT -----")
	for domain, stat := range stats {
		availability := 0
		if stat.Total > 0 {
			availability = int(math.Round(100 * float64(stat.Success) / float64(stat.Total)))
		}
		fmt.Printf("%s - %d%% availability\n", domain, availability)
	}
	fmt.Println("--------------------------------")
}

func monitorEndpoints(ctx context.Context, endpoints []Endpoint) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for _, endpoint := range endpoints {
		domain := extractDomain(endpoint.URL)
		if stats[domain] == nil {
			stats[domain] = &DomainStats{}
		}
	}

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Stopping monitoring service...")
			return
		default:
			for _, endpoint := range endpoints {
				go checkHealth(endpoint)
			}
			time.Sleep(3 * time.Second) 
			logResults()
			<-ticker.C
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <sample.yaml>")
	}

	filePath := os.Args[1]
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Error reading file: %v", err)
	}

	var endpoints []Endpoint
	if err := yaml.Unmarshal(data, &endpoints); err != nil {
		log.Fatalf("Error parsing YAML: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
		<-signalChan
		cancel()
	}()

	monitorEndpoints(ctx, endpoints)
}
