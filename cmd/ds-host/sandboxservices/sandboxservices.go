package sandboxservices

import (
	"fmt"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/twine-go/twine"
)

// ServiceMaker holds the structs necessary to create a service for any api version
type ServiceMaker struct {
	AppspaceUserModel interface {
		Get(appspaceID domain.AppspaceID, proxyID domain.ProxyID) (domain.AppspaceUser, error)
		GetAll(appspaceID domain.AppspaceID) ([]domain.AppspaceUser, error)
	}
}

// Get returns a reverse service for the appspace
func (x *ServiceMaker) Get(appspace *domain.Appspace) (service domain.ReverseServiceI) {
	return &AppspaceService{
		Users: &UsersService{
			AppspaceUserModel: x.AppspaceUserModel,
			appspaceID:        appspace.AppspaceID},
	}
}

// local service IDs:
const (
	sandboxServiceID = 11 // unused
	routesServiceID  = 14 // unused
	usersServiceID   = 16
)

// AppspaceService is a twine handler for reverse services with API version 0
type AppspaceService struct {
	Users domain.ReverseServiceI
}

// HandleMessage passes the message along to the relevant service
func (s *AppspaceService) HandleMessage(message twine.ReceivedMessageI) {
	switch message.ServiceID() {
	case usersServiceID:
		s.Users.HandleMessage(message) // not anymore
	default:
		s.getLogger("listenMessages()").Log(fmt.Sprintf("Service not recognized: %v, command: %v", message.ServiceID(), message.CommandID()))
		message.SendError("service not recognized")
	}
}

func (s *AppspaceService) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("AppspaceService")
	if note != "" {
		l.AddNote(note)
	}
	return l
}
