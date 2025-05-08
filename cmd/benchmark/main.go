package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

type Payload struct {
	Items []Item `json:"item_list"`
}

type Item struct {
	ProductID string  `json:"product_id"`
	Price     float64 `json:"price"`
}

var (
	concurrency   = flag.Int("c", 5, "Number of concurrent workers")
	totalRequests = flag.Int("n", 1, "Total number of requests")
	timeout       = flag.Duration("t", 10*time.Second, "Request timeout")
	payloadSize   = flag.Int("s", 1000, "Number of items in payload")
	targetURL     = flag.String("u", "http://localhost:8080/data/123", "Target URL")
)

type Metrics struct {
	Total       int64
	Success     int64
	Failed      int64
	TotalTime   time.Duration
	StatusCodes map[int]int64
}

func main() {
	flag.Parse()

	sampleFile := "samples/large_payload.json"
	if err := generateSampleFile(sampleFile, *payloadSize); err != nil {
		log.Fatalf("Failed to generate sample file: %v", err)
	}
	fmt.Printf("Generated sample file: %s (%d)\n", sampleFile, *payloadSize)
	return
	rand.Seed(time.Now().UnixNano())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var metrics Metrics
	metrics.StatusCodes = make(map[int]int64)

	workChan := make(chan struct{}, *totalRequests)
	for i := 0; i < *totalRequests; i++ {
		workChan <- struct{}{}
	}
	close(workChan)

	var wg sync.WaitGroup
	start := time.Now()

	// Start workers
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go worker(ctx, &wg, workChan, &metrics, *targetURL, *payloadSize, *timeout)
	}

	// Print periodic updates
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				log.Printf("Progress: %d/%d (%.1f%%)",
					atomic.LoadInt64(&metrics.Total),
					*totalRequests,
					100*float64(atomic.LoadInt64(&metrics.Total))/float64(*totalRequests),
				)
			case <-ctx.Done():
				return
			}
		}
	}()

	wg.Wait()
	duration := time.Since(start)

	printReport(&metrics, duration)
}

func worker(ctx context.Context, wg *sync.WaitGroup, work <-chan struct{}, metrics *Metrics, url string, payloadSize int, timeout time.Duration) {
	defer wg.Done()

	client := &http.Client{
		Timeout: timeout,
	}

	for range work {
		// start := time.Now()
		payload := generatePayload(payloadSize)
		result := makeRequest(client, url, payload)
		// elapsed := time.Since(start)

		atomic.AddInt64(&metrics.Total, 1)
		// atomic.AddInt64(&metrics.TotalTime, int64(elapsed))

		if result.Error != nil {
			atomic.AddInt64(&metrics.Failed, 1)
		} else {
			atomic.AddInt64(&metrics.Success, 1)
			// atomic.AddInt64(&metrics.StatusCodes[result.StatusCode], 1)
		}
	}
}

type RequestResult struct {
	StatusCode int
	Error      error
}

func makeRequest(client *http.Client, url string, payload Payload) RequestResult {
	body, err := json.Marshal(payload)
	if err != nil {
		return RequestResult{Error: err}
	}

	req, err := http.NewRequest("GET", url, bytes.NewReader(body))
	if err != nil {
		return RequestResult{Error: err}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return RequestResult{Error: err}
	}
	defer resp.Body.Close()
	return RequestResult{
		StatusCode: resp.StatusCode,
	}
}

func generatePayload(size int) Payload {
	items := make([]Item, size)
	for i := 0; i < size; i++ {
		items[i] = Item{
			ProductID: fmt.Sprintf("prod-%d", rand.Intn(1000)),
			Price:     rand.Float64() * 100,
		}
	}
	return Payload{Items: items}
}

func generateSampleFile(filename string, itemCount int) error {
	payload := generatePayload(itemCount)

	// Create pretty-printed JSON
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sample: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write sample file: %w", err)
	}

	return nil
}

func printReport(metrics *Metrics, duration time.Duration) {
	fmt.Printf("\nLoad Test Report:\n")
	fmt.Printf("=================\n")
	fmt.Printf("Total requests:  %d\n", metrics.Total)
	fmt.Printf("Successful:      %d (%.1f%%)\n", metrics.Success, 100*float64(metrics.Success)/float64(metrics.Total))
	fmt.Printf("Failed:          %d (%.1f%%)\n", metrics.Failed, 100*float64(metrics.Failed)/float64(metrics.Total))
	fmt.Printf("Total duration:  %s\n", duration.Round(time.Millisecond))
	fmt.Printf("Requests/sec:    %.1f\n", float64(metrics.Total)/duration.Seconds())
	fmt.Printf("Status Codes:\n")
	for code, count := range metrics.StatusCodes {
		fmt.Printf("  %d: %d\n", code, count)
	}
}
