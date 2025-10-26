package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	requestsTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "game_api_requests_total",
		Help: "Total number of requests to game API",
	})
	uptimeSeconds = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "game_api_uptime_seconds",
		Help: "Uptime of game API in seconds",
	})
)

func init() {
	prometheus.MustRegister(requestsTotal)
	prometheus.MustRegister(uptimeSeconds)
}

func fetchMetrics(url string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var metrics struct {
		RequestsTotal uint64  `json:"requests_total"`
		UptimeSeconds float64 `json:"uptime_seconds"`
	}
	if err := json.Unmarshal(body, &metrics); err != nil {
		return err
	}

	requestsTotal.Set(float64(metrics.RequestsTotal))
	uptimeSeconds.Set(metrics.UptimeSeconds)

	return nil
}

func scrapeHandler(targetURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := fetchMetrics(targetURL)
		if err != nil {
			http.Error(w, "failed to fetch metrics from target: "+err.Error(), 500)
			return
		}
		promhttp.Handler().ServeHTTP(w, r)
	}
}

func main() {
	targetMetricsURL := os.Getenv("TARGET_METRICS_URL")
	if targetMetricsURL == "" {
		targetMetricsURL = "http://game-api:8080/metrics" // дефолтное значение
		log.Println("TARGET_METRICS_URL not set, using default:", targetMetricsURL)
	}

	http.HandleFunc("/metrics", scrapeHandler(targetMetricsURL))
	log.Println("Prometheus scraper running on :9101")
	if err := http.ListenAndServe(":9101", nil); err != nil {
		log.Fatal(err)
	}
}

