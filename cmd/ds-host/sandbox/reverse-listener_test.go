package sandbox

import (
	"io/ioutil"
	"os"
	"testing"
	"net"
	"time"
	"net/http"
	"bytes"
	"context"
	"fmt"
	"encoding/json"

	"github.com/golang/mock/gomock"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestNewReverseListener(t *testing.T) {
	cfg := &domain.RuntimeConfig{}

	newReverseListener(cfg, 1)
}

func TestStartReverseListener(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	cfg := &domain.RuntimeConfig{}
	cfg.Sandbox.SocketsDir = dir

	rl, dsErr := newReverseListener(cfg, 1)
	if dsErr != nil {
		t.Error(dsErr)
	}

	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", rl.socketPath)
			},
		},
	}
	
	// client:
	go func() {
		time.Sleep(100 * time.Millisecond)
		var hiData struct {
			Port int	`json:"port"`
		}
		hiData.Port = 1234

		hiJSON, err := json.Marshal(hiData)
		if err != nil {
			t.Error(err)
		}

		fmt.Println("sending post")
		resp, err := httpc.Post("http://unix/status/hi", "application/json", bytes.NewBuffer(hiJSON))
		if err != nil {
			t.Error(err)
		}

		if resp.StatusCode != 200 {
			t.Error(resp.Status)
		}
	}()

	port := <-rl.portChan
	fmt.Println("Port", port)
}


