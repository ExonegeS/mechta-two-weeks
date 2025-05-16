package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

type Payload struct {
	Items []Item `json:"items"`
}

type Item struct {
	ProductID string  `json:"product_id"`
	Price     float64 `json:"price"`
}

var (
	payloadSize = flag.Int("s", 1000, "Number of items in payload")
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
}

func generatePayload(size int) Payload {
	items := make([]Item, size)
	for i := 0; i < size; i++ {
		items[i] = Item{
			ProductID: fmt.Sprintf("prod-%d", rand.Intn(100000)),
			Price:     rand.Float64() * 100,
		}
	}
	return Payload{Items: items}
}

func generateSampleFile(filename string, itemCount int) error {
	payload := generatePayload(itemCount)
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sample: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write sample file: %w", err)
	}
	return nil
}
