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
		Name: "bungie_error_code",
	},
	[]string{"error_code"}, // labels
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

var PGCRCrawlLag = prometheus.NewSummaryVec(
    prometheus.SummaryOpts{
        Name: "pgcr_crawl_summary_lag",
    },
    []string{"status", "attempts"},
)

var PGCRCrawlReqTime = prometheus.NewSummaryVec(
    prometheus.SummaryOpts{
        Name: "pgcr_crawl_summary_req_time",
    },
    []string{"status", "attempts"},
)

// Port should be in range 9090-9093
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
