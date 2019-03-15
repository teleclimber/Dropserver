package record

import (
	"encoding/json"
	"fmt"
	"github.com/afiskon/promtail-client/promtail"
	"time"
)

// DsLogClient is used to log messages in that special ds-host way
type DsLogClient struct {
	loki promtail.Client
}

var defaultClient *DsLogClient //no let's make that a custom client with custom methods.

// LogLevel expresses the severity of a log entry as an int
type LogLevel int

// DEBUG is for debug
const (
	DEBUG LogLevel = iota
	INFO  LogLevel = iota
	WARN  LogLevel = iota
	ERROR LogLevel = iota
	// DISABLE Maximum level, disables sending or printing
	DISABLE LogLevel = iota
)

// LogDataHash are transcribed as json in the log message
type LogDataHash struct {
	AppSpace  string
	App       string
	RequestID string
}

func initLogging() { //maybe pass a config
	lokiConf := promtail.ClientConfig{
		PushURL:            "http://localhost:3100/api/prom/push",
		Labels:             "{cmd=\"ds-host\"}",
		BatchWait:          time.Second,
		BatchEntriesNumber: 1000,
		SendLevel:          promtail.DEBUG,
		PrintLevel:         promtail.DEBUG,
	}
	// ^^ most of this has to come from config

	loki, err := promtail.NewClientJson(lokiConf)
	if err != nil {
		fmt.Println("error creating loki client", err)
	}

	defaultClient = &DsLogClient{loki: loki}
}

// NewSandboxLogClient creates a logging client with sandbox name as a label
func NewSandboxLogClient(sandboxName string) *DsLogClient {
	lokiConf := promtail.ClientConfig{
		PushURL:            "http://localhost:3100/api/prom/push",
		Labels:             "{cmd=\"ds-host\", sandbox=\"" + sandboxName + "\"}", //hmm
		BatchWait:          time.Second,
		BatchEntriesNumber: 1000,
		SendLevel:          promtail.DEBUG,
		PrintLevel:         promtail.DEBUG,
	}
	// ^^ most of this has to come from config

	loki, err := promtail.NewClientJson(lokiConf)
	if err != nil {
		fmt.Println("error creating loki client", err)
	}

	return &DsLogClient{loki: loki}
}

// Log logs a message to Loki
func (c *DsLogClient) Log(severity LogLevel, data map[string]string, msg string) {

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
	case DEBUG:
		c.loki.Debugf(msg)
	case INFO:
		c.loki.Infof(msg)
	case WARN:
		c.loki.Warnf(msg)
	case ERROR:
		c.loki.Errorf(msg)
	}
}

// Log sends log entry to default logging client
func Log(severity LogLevel, data map[string]string, msg string) {
	(*defaultClient).Log(severity, data, msg)
}

// how to structure the rest of this?
// Ideally it would use go's built in logging facility?
// .. however does it support labels?
// Kind of seems like we will need to create multiple clients
// ..and have those be reused
// Question is what is the interface?

// first what are the labels?
// - cmd: ds-host, ds-mounter, ds-sandbox-d
// - ~section: app-space, user, admin?
// - package?
// - sandbox-id: 1, 2, 3, 4...
// - origin: net, cron // maybe innerweb vs innerweb for as to as requests?

// Then there are things we would like to make repeatable, ..like
// - request id,
// - appspace,
// - app, ...
// - request method
// - response code? (not open-ended so could be a label, however maybe not necessary? -> rate of these responses will be recorded in metrics

// would be nice to be able to change labels arrangement solely from this file?
// ...
