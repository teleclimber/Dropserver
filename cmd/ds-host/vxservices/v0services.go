package vxservices

import (
	"fmt"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/twine-go/twine"
)

// local service IDs:
const v0sandboxService = 11
const v0routesService = 14
const v0databaseService = 15
const v0usersService = 16 //no more?

//V0Services is a twine handler for reverse services with API version 0
type V0Services struct {
	UsersModel domain.ReverseServiceI
	AppspaceDB domain.ReverseServiceI
}

// HandleMessage passes the message along to the relevant service
func (s *V0Services) HandleMessage(message twine.ReceivedMessageI) {
	switch message.ServiceID() {
	case v0databaseService:
		s.AppspaceDB.HandleMessage(message)
	case v0usersService:
		s.UsersModel.HandleMessage(message) // not anymore
	default:
		s.getLogger("listenMessages()").Log(fmt.Sprintf("Service not recognized: %v, command: %v", message.ServiceID(), message.CommandID()))
		message.SendError("service not recognized")
	}
}

func (s *V0Services) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("V0Services")
	if note != "" {
		l.AddNote(note)
	}
	return l
}
