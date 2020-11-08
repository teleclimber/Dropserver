package sandbox

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// things to test:
// - killPool
// - startSandbox
// - GetForAppspace

func TestKillPool(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	cfg := &domain.RuntimeConfig{}
	cfg.Sandbox.Num = 1

	m := &Manager{
		sandboxes: make(map[domain.AppspaceID]domain.SandboxI),
		Config:    cfg}

	s1 := domain.NewMockSandboxI(mockCtrl)
	s1.EXPECT().Status().Return(domain.SandboxReady)
	s1.EXPECT().TiedUp().Return(false)
	s1.EXPECT().LastActive().Return(time.Now().Add(-1 * time.Second))
	s1.EXPECT().SetStatus(domain.SandboxKilling)
	s1.EXPECT().Graceful()
	m.sandboxes[domain.AppspaceID(1)] = s1

	s2 := domain.NewMockSandboxI(mockCtrl)
	s2.EXPECT().Status().Return(domain.SandboxReady)
	s2.EXPECT().TiedUp().Return(false)
	s2.EXPECT().LastActive().Return(time.Now())
	m.sandboxes[domain.AppspaceID(2)] = s2

	s3 := domain.NewMockSandboxI(mockCtrl)
	s3.EXPECT().Status().Return(domain.SandboxReady)
	s3.EXPECT().TiedUp().Return(false)
	s3.EXPECT().LastActive().Return(time.Now().Add(-2 * time.Second))
	s3.EXPECT().SetStatus(domain.SandboxKilling)
	s3.EXPECT().Graceful()
	m.sandboxes[domain.AppspaceID(3)] = s3

	// not clear what the behavior of killPool will be when we have sandboxes in transition (starting / killing)
	// go to that level of detail later.

	m.killPool()

	time.Sleep(100 * time.Millisecond)

}
func TestStartSandbox(t *testing.T) {
	// this is an integration test to a degree.
	// execution of JS runtime unavoidable
}
