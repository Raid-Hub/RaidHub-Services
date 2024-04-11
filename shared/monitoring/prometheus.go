package monitoring

import (
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
		Buckets: []float64{5, 15, 25, 30, 35, 40, 45, 60, 90, 300, 1800, 14400, 86400},
	},
	[]string{"status", "attempts"},
)

var PGCRCrawlTime = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "pgcr_crawl_summary_req_time",
		Buckets: []float64{100, 200, 300, 400, 500, 600, 700, 800, 1000, 1200, 1500, 2000, 5000, 10000},
	},
	[]string{"status", "attempts"},
)

var GetPostGameCarnageReportRequest = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "get_pgcr_req",
		Buckets: []float64{10, 20, 50, 100, 150, 200, 250, 300, 500, 750, 1000, 1500, 2000, 5000},
	},
	[]string{"status"},
)

// Track the count of each Bungie error code returned by the API
func RegisterPrometheus(port int) {
	prometheus.MustRegister(ActiveWorkers)
	prometheus.MustRegister(PGCRCrawlLag)
	prometheus.MustRegister(PGCRCrawlTime)
	prometheus.MustRegister(PGCRCrawlStatus)
	prometheus.MustRegister(GetPostGameCarnageReportRequest)

	http.Handle("/metrics", promhttp.Handler())

	go func() {
		port := fmt.Sprintf(":%d", port)
		log.Fatal(http.ListenAndServe(port, nil))
	}()
}
