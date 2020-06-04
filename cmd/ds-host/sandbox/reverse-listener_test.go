package sandbox

// import (
// 	"io/ioutil"
// 	"net"
// 	"os"
// 	"path"
// 	"testing"
// 	"time"

// 	"github.com/golang/mock/gomock"

// 	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
// )

// func TestNewReverseListener(t *testing.T) {
// 	dir, err := ioutil.TempDir("", "")
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	defer os.RemoveAll(dir)

// 	cfg := &domain.RuntimeConfig{}
// 	cfg.Sandbox.SocketsDir = dir

// 	newReverseListener(cfg, &domain.Appspace{AppspaceID: 1})
// }

// func TestStartReverseListener(t *testing.T) {
// 	mockCtrl := gomock.NewController(t)
// 	defer mockCtrl.Finish()

// 	dir, err := ioutil.TempDir("", "")
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	defer os.RemoveAll(dir)

// 	cfg := &domain.RuntimeConfig{}
// 	cfg.Sandbox.SocketsDir = dir

// 	appspaceID := domain.AppspaceID(7)

// 	sockPath, dsErr := makeSocketsDir(dir, appspaceID)
// 	if dsErr != nil {
// 		t.Fatal(dsErr)
// 	}

// 	rl, dsErr := newReverseListener(cfg, &domain.Appspace{AppspaceID: appspaceID})
// 	if dsErr != nil {
// 		t.Error(dsErr)
// 	}

// 	c, err := net.Dial("unix", path.Join(sockPath, "rev.sock"))
// 	if err != nil {
// 		t.Fatal("Dial error", err)
// 	}
// 	defer c.Close()

// 	// client:
// 	go func() {
// 		time.Sleep(100 * time.Millisecond)

// 		b := make([]byte, 1)
// 		b[0] = uint8(1) // 1 is "hi"
// 		_, err := c.Write(b)
// 		if err != nil {
// 			t.Error("Write error:", err)
// 		}

// 		time.Sleep(100 * time.Millisecond)
// 		b = make([]byte, 1)
// 		b[0] = uint8(2)
// 		_, err = c.Write(b)
// 		if err != nil {
// 			t.Error("Write error:", err)
// 		}

// 	}()

// 	startStatus := <-rl.startChan
// 	if startStatus != revReady {
// 		t.Error("expected ready status")
// 	}

// 	// then expect the conn to close from the client side.
// 	dsErr = <-rl.errorChan
// 	if dsErr != nil {
// 		t.Error(dsErr)
// 	}
// }
