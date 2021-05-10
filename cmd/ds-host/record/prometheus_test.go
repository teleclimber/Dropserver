package record

import (
	"net/http"
	"testing"
)

func TestShutdown(t *testing.T) {
	// Just want to make sure calling shutdown doesnt result in errors
	// When no server is running.
	err := StopPromMetrics()
	if err != nil {
		t.Error(err)
	}

	srv = &http.Server{}

	err = StopPromMetrics()
	if err != nil {
		t.Error(err)
	}
}
