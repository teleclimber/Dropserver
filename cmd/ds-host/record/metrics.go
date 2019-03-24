package record

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// can we make metrics a struct or something
// .. so that we can use an interface to define the methods we want?

// Metrics encapsulates the metrics calls
type Metrics struct {
}

func initMetrics() {
	// prometheus metrics. Do metrics just accumulate if prometheus is not pulling them?
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(":2112", nil) //this should come from config
		if err != nil {
			fmt.Println("Error starting prometheus metrics handler", err)
			os.Exit(1)
		}
	}()
}

// maybe a var to turn on metrics,
// ..and maybe to set their detail level (less dtail for low power, non-dev)

// This probably needs to be refactored
// probably a map[string] struct {
// 	Help string
// 	Buckets []float64
// }
// ..or not. Not sure what it's buying us.

var promeReqDur = promauto.NewHistogram(prometheus.HistogramOpts{
	Name:    "dshost_requests_duration_s",
	Help:    "Duration of requests",
	Buckets: prometheus.ExponentialBuckets(0.001, 2, 12), //[]float64{0.001, 0.010, 0.100, 1.000},
})

var recycDur = promauto.NewHistogram(prometheus.HistogramOpts{
	Name:    "dshost_recycle_duration_s",
	Help:    "Duration of sandbox recycle",
	Buckets: prometheus.ExponentialBuckets(0.05, 1.5, 10),
})

var commitDur = promauto.NewHistogram(prometheus.HistogramOpts{
	Name:    "dshost_commit_duration_s",
	Help:    "Duration of sandbox commit",
	Buckets: prometheus.ExponentialBuckets(0.005, 1.5, 10),
})

// HostHandleReq measures ds-host' requets durations
func (m *Metrics) HostHandleReq(start time.Time) {
	promeReqDur.Observe(time.Since(start).Seconds())
}

// SandboxRecycleTime observes the time it takes for a sanbox to recycle
func SandboxRecycleTime(start time.Time) {
	recycDur.Observe(time.Since(start).Seconds())
}

// SandboxCommitTime observes the time it takes for a sanbox to commit
func SandboxCommitTime(start time.Time) {
	commitDur.Observe(time.Since(start).Seconds())
}

// SandboxStatuses struct records number of sandboxes at each state
type SandboxStatuses struct {
	Starting   int
	Ready      int
	Committing int
	Committed  int
	Recycling  int
}

var sandboxStatusCounts = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "dshost_sandbox_status_counts",
	Help: "Number of sandboxes at each state"},
	[]string{"status"},
)

//SandboxStatusCounts sets counts for sandbox statuses
func SandboxStatusCounts(counts *SandboxStatuses) {
	sandboxStatusCounts.WithLabelValues("starting").Set(float64(counts.Starting))
	sandboxStatusCounts.WithLabelValues("ready").Set(float64(counts.Ready))
	sandboxStatusCounts.WithLabelValues("committed").Set(float64(counts.Committed))
	sandboxStatusCounts.WithLabelValues("committing").Set(float64(counts.Committing))
	sandboxStatusCounts.WithLabelValues("recycling").Set(float64(counts.Recycling))
}
