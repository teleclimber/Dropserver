package vxservices

import (
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/twine"
)

// local service IDs:
const v0sandboxService = 11
const v0routesService = 14
const v0databaseService = 15

//V0Services is a twine handler for reverse services with API version 0
type V0Services struct {
	RouteModel domain.ReverseServiceI
	AppspaceDB domain.ReverseServiceI
}

// HandleMessage passes the message along to the relevant service
func (s *V0Services) HandleMessage(message twine.ReceivedMessageI) {
	switch message.ServiceID() {
	case v0routesService:
		s.RouteModel.HandleMessage(message)
	case v0databaseService:
		s.AppspaceDB.HandleMessage(message)
	default:
		//s.getLogger("listenMessages()").Log(fmt.Sprintf("Service not recognized: %v", message.ServiceID()))
		message.SendError("service not recognized")
	}
}
