package sandbox

import (
	"io/ioutil"
	"os"
	"testing"
	"net"
	"time"

	"github.com/golang/mock/gomock"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestNewReverseListener(t *testing.T) {
	cfg := &domain.RuntimeConfig{}
	msgCb := func(m string) {
		//?
	}

	newReverseListener(cfg, 1, msgCb)
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

	msgCb := func(m string) {
		//?
	}

	rl := newReverseListener(cfg, 1, msgCb)

	c, err := net.Dial("unix", rl.socketPath)
	if err != nil {
		t.Error(err)
	}
	defer c.Close()

	// client:
	go func() {
		time.Sleep(100 * time.Millisecond)
		_, err = c.Write([]byte("hi"))
		if err != nil {
			t.Error(err)
		}
	}()

	rl.waitFor("hi")
}


