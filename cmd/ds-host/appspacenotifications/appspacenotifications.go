package appspacenotifications

import (
	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

type WebPush struct {
}

func (w *WebPush) EnsureVapidKeys(appspaceID domain.AppspaceID) {
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		// TODO: Handle error
	}

}

// PushToAll pushes msg to every subscription in the appspace
func (w *WebPush) PushToAll(appspaceID domain.AppspaceID, msg []byte) {
	// figure out if appspace is allowed to send pushes?
	// get all subscriptions for that appspace
	// loop over subs and send

}

// PushToUser pushs the msg to all subscriptions associated with the proxyID
func (w *WebPush) PushToUser(appspaceID domain.AppspaceID, proxyID domain.ProxyID, msg []byte) {

}

func (w *WebPush) PushToSub(appspaceID domain.AppspaceID, pushSub string, msg []byte) {

}

func (w *WebPush) doPush(s webpush.Subscription) {

}
