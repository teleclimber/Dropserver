package appspaceops

import (
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

type PauseAppspace struct {
	AppspaceModel interface {
		Pause(appspaceID domain.AppspaceID, pause bool) error
	} `checkinject:"required"`
	AppspaceStatus interface {
		PauseAppspace(domain.AppspaceID, bool)
	} `checkinject:"required"`
	SandboxManager interface {
		StopAppspace(domain.AppspaceID)
	} `checkinject:"required"`
	AppspaceLogger interface {
		Log(appspaceID domain.AppspaceID, source string, message string)
		Close(appspaceID domain.AppspaceID)
	} `checkinject:"required"`
}

// Pause the appspace
func (p *PauseAppspace) Pause(appspaceID domain.AppspaceID, pause bool) error {
	err := p.AppspaceModel.Pause(appspaceID, pause)
	if err != nil {
		return err
	}

	p.AppspaceStatus.PauseAppspace(appspaceID, pause)

	p.SandboxManager.StopAppspace(appspaceID)

	logStr := "Pausing"
	if !pause {
		logStr = "Unpausing"
	}
	p.AppspaceLogger.Log(appspaceID, "ds-host", logStr+" appspace")

	return nil
}
