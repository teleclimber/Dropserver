package record

import (
	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

var srv *http.Server

func ExposePromMetrics(config domain.RuntimeConfig) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	srv = &http.Server{
		Addr:    fmt.Sprintf(":%v", config.Prometheus.Port),
		Handler: mux,
	}

	getLogger("").Log("Starting Prometheus metrics server")

	go func() {
		err := srv.ListenAndServe()
		if err != http.ErrServerClosed {
			getLogger("server closed").Error(err)
		}
	}()
}

func StopPromMetrics() error {
	if srv == nil {
		return nil
	}
	getLogger("").Log("Stopping Prometheus metrics server")
	if err := srv.Shutdown(context.Background()); err != nil {
		// Error from closing listeners, or context timeout:
		getLogger("shutdown server").Error(err)
		return err
	}
	srv = nil
	return nil
}

func getLogger(note string) *DsLogger {
	l := NewDsLogger().AddNote("Prometheus metrics")
	if note != "" {
		l.AddNote(note)
	}
	return l
}
