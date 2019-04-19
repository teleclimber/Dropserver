package record

import "github.com/teleclimber/DropServer/cmd/ds-host/domain"

// Init starts up the metrics and Logging
func Init(cfg *domain.RuntimeConfig) {
	initMetrics(cfg)
	//initLogging()
}
