package record

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/afiskon/promtail-client/promtail"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-trusted/trusteddomain"
)

// DsLogClient is used to log messages in that special ds-host way
type DsLogClient struct {
	loki   promtail.Client
	Config *domain.TrustedConfig
}

// NewLogClient returns a generic log client
func NewLogClient(config *domain.TrustedConfig) trusteddomain.LogCLientI {
	pushURL := fmt.Sprintf("http://%s:%d/api/prom/push", config.Loki.Address, config.Loki.Port)
	// ^^ isolate this and test it.
	lokiConf := promtail.ClientConfig{
		PushURL:            pushURL,
		Labels:             "{cmd=\"ds-trusted\"}",
		BatchWait:          time.Second,
		BatchEntriesNumber: 1000,
		SendLevel:          promtail.DEBUG,
		PrintLevel:         promtail.DEBUG,
	}
	// ^^ most of this has to come from config <<<

	loki, err := promtail.NewClientJson(lokiConf)
	if err != nil {
		fmt.Println("error creating loki client", err)
	}

	return &DsLogClient{loki: loki, Config: config}
}

// Log logs a message to Loki
// Data values (and keys) and have unbounded number of values
// like client-ip, app-id, etc...
// This is unlike the Lolki Labels which must be kept to a relatively small number
func (c *DsLogClient) Log(severity domain.LogLevel, data map[string]string, msg string) {

	// turn hash to json?
	if data != nil {
		j, err := json.Marshal(data)
		if err != nil {
			fmt.Println(err)
			return
		}
		//fmt.Println(string(b))
		msg = msg + " " + string(j)
	}

	switch severity {
	case domain.DEBUG:
		c.loki.Debugf(msg)
	case domain.INFO:
		c.loki.Infof(msg)
	case domain.WARN:
		c.loki.Warnf(msg)
	case domain.ERROR:
		c.loki.Errorf(msg)
	}
}
