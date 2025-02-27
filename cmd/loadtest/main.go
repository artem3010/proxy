package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"math/rand"
	"net/http"
	"os"
	"proxy/internal/env"

	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type RequestRow struct {
	Id       string `json:"inventoryId"`
	Priority int    `json:"priority"`
}

type RequestBody struct {
	InventoryIds []RequestRow `json:"inventoryIds"`
}

func generateRandomPayload(maxRows int) ([]byte, error) {
	numRows := rand.Intn(maxRows) + 1
	rows := make([]RequestRow, numRows)
	for i := 0; i < numRows; i++ {
		id := rand.Intn(10000)
		rows[i] = RequestRow{
			Id:       "id" + strconv.Itoa(id),
			Priority: rand.Intn(50),
		}
	}
	body := RequestBody{InventoryIds: rows}
	return json.Marshal(body)
}

func main() {

	env.LoadEnv()
	if needTest := os.Getenv("NEED_TEST"); needTest != "true" {
		return
	}
	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	url := "http://localhost:8080/api/v1/measure"
	concurrency := flag.Int("concurrency", 100, "Number of concurrent workers")
	duration := flag.Duration("duration", 5*time.Second, "Duration of the load test")
	maxRows := flag.Int("maxRows", 1000, "Maximum number of rows in a payload")
	flag.Parse()

	if envURL := os.Getenv("TARGET_URL"); envURL != "" {
		url = envURL
	}

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	log.Info().
		Str("target", url).
		Int("concurrency", *concurrency).
		Dur("duration", *duration).
		Msg("Starting load test")

	startTime := time.Now()
	var wg sync.WaitGroup
	var reqCount int
	var reqMu sync.Mutex

	latencyChan := make(chan time.Duration, 100000)

	var latencies []time.Duration
	var latMu sync.Mutex
	var aggWg sync.WaitGroup
	aggWg.Add(1)
	go func() {
		defer aggWg.Done()
		for lat := range latencyChan {
			latMu.Lock()
			latencies = append(latencies, lat)
			latMu.Unlock()
			reqMu.Lock()
			reqCount++
			reqMu.Unlock()
		}

	}()

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			client := &http.Client{}
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				payload, err := generateRandomPayload(*maxRows)
				if err != nil {
					log.Error().Err(err).Int("worker", workerID).Msg("Error generating payload")
					continue
				}

				req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
				if err != nil {
					log.Error().Err(err).Int("worker", workerID).Msg("Error creating request")
					continue
				}
				req.Header.Set("Content-Type", "application/json")
				// Передаем контекст в запрос, чтобы он мог быть отменен по истечении времени теста.
				req = req.WithContext(ctx)

				reqStart := time.Now()
				resp, err := client.Do(req)
				latency := time.Since(reqStart)
				latencyChan <- latency

				if resp != nil && resp.Body != nil {
					_ = resp.Body.Close()
				}

				if err != nil {
					log.Error().Err(err).Int("worker", workerID).Msg("Error sending request or reading response")
				}
			}
		}(i)
	}

	wg.Wait()
	close(latencyChan)
	aggWg.Wait()

	if len(latencies) > 0 {
		sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
		index := int(float64(len(latencies)) * 0.99)
		if index >= len(latencies) {
			index = len(latencies) - 1
		}
		log.Info().Str("99th_percentile", latencies[index].String()).Msg("99th percentile latency")
	}

	totalTime := time.Since(startTime).Seconds()
	rps := float64(reqCount) / totalTime
	log.Info().
		Int("total_requests", reqCount).
		Float64("rps", rps).
		Msg("Load test completed")
}
