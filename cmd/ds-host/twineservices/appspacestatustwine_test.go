package twineservices

import (
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
	"github.com/teleclimber/DropServer/internal/leaktest"
	"github.com/teleclimber/DropServer/internal/twine"
)

func TestSubscribe(t *testing.T) {
	leaktest.GoroutineLeakCheck(t)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	testSeqCh := make(chan struct{}, 0)

	ownerID := domain.UserID(77)
	appspaceID := domain.AppspaceID(33)

	var eventChan chan<- domain.AppspaceStatusEvent

	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	appspaceModel.EXPECT().GetFromID(appspaceID).Return(&domain.Appspace{OwnerID: ownerID}, nil)

	appspaceStatus := testmocks.NewMockAppspaceStatus(mockCtrl)
	appspaceStatus.EXPECT().Track(appspaceID).Return(domain.AppspaceStatusEvent{})

	appspaceStatusEvents := testmocks.NewMockAppspaceStatusEvents(mockCtrl)
	appspaceStatusEvents.EXPECT().Subscribe(appspaceID, gomock.Any()).
		Do(func(a domain.AppspaceID, ch chan<- domain.AppspaceStatusEvent) {
			eventChan = ch
		})
	appspaceStatusEvents.EXPECT().Unsubscribe(appspaceID, gomock.Any())

	s := &AppspaceStatusService{
		AppspaceModel:        appspaceModel,
		AppspaceStatus:       appspaceStatus,
		AppspaceStatusEvents: appspaceStatusEvents,
	}
	s.authUser = ownerID

	// I need to create a Twine message to subscribe to event I suppose?

	subPayload, _ := json.Marshal(IncomingAppspaceID{AppspaceID: appspaceID})
	subMsg := twine.NewMockReceivedMessageI(mockCtrl)
	subMsg.EXPECT().Payload().Return(subPayload)

	// need ref req chan and EXPECT that it will be asked for.
	refCh := make(chan twine.ReceivedMessageI)
	subMsg.EXPECT().GetRefRequestsChan().Return(refCh)
	subMsg.EXPECT().SendOK()
	subMsg.EXPECT().RefSendBlock(statusEventCmd, gomock.Any()) // initial data sent when subscribing

	s.handleSubscribeMessage(subMsg)

	subMsg.EXPECT().RefSendBlock(statusEventCmd, gomock.Any()).Do(func(args ...interface{}) {
		testSeqCh <- struct{}{}
	})
	eventChan <- domain.AppspaceStatusEvent{}

	<-testSeqCh

	unsubMsg := twine.NewMockReceivedMessageI(mockCtrl)
	unsubMsg.EXPECT().CommandID().Return(unsubscribeStatus)

	unsubMsg.EXPECT().SendOK().Do(func() {
		testSeqCh <- struct{}{}
	})

	refCh <- unsubMsg

	<-testSeqCh
	//time.Sleep(time.Second)
}
