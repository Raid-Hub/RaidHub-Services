package monitoring

import (
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Track the count of each Bungie error code returned by the API
var BungieErrorCode = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "bungie_error_status",
	},
	[]string{"error_status"},
)

// Track the number of active workers
var ActiveWorkers = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "atlas_active_workers",
	},
)

var PGCRCrawlStatus = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "pgcr_crawl_summary_status",
	},
	[]string{"status", "attempts"},
)

var PGCRCrawlLag = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "pgcr_crawl_summary_lag",
		Buckets: []float64{5, 15, 25, 30, 35, 40, 45, 60, 90, 300, 1800, 14400},
	},
	[]string{"status", "attempts"},
)

var PGCRCrawlReqTime = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "pgcr_crawl_summary_req_time",
		Buckets: []float64{100, 200, 300, 400, 500, 600, 700, 800, 1000, 1200, 1500, 2000, 5000, 10000},
	},
	[]string{"status", "attempts"},
)

func RegisterPrometheus(port int) {
	prometheus.MustRegister(BungieErrorCode)
	prometheus.MustRegister(ActiveWorkers)
	prometheus.MustRegister(PGCRCrawlLag)
	prometheus.MustRegister(PGCRCrawlReqTime)
	prometheus.MustRegister(PGCRCrawlStatus)

	http.Handle("/metrics", promhttp.Handler())

	go func() {
		port := fmt.Sprintf(":%d", port)
		log.Fatal(http.ListenAndServe(port, nil))
	}()
}
