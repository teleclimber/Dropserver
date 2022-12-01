package appspacelogin

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

func TestRegisterToken(t *testing.T) {
	v0requestToken := V0RequestToken{}

	ch1 := make(chan requestResults)
	_, exists := v0requestToken.registerRequest("abc.def", "hjkl", ch1)
	if exists {
		t.Error("expected exists flag to be false")
	}
	if len(v0requestToken.reqs) != 1 {
		t.Error("expecte reqs of length 1")
	}
	req := v0requestToken.reqs[0]
	if req.appspace != "abc.def" {
		t.Error("wrong domain")
	}
	if req.sessionID != "hjkl" {
		t.Error("wrong session id")
	}
	if len(req.ref) < 8 {
		t.Error("ref should be at least 8 long")
	}
	if len(req.subscribers) != 1 || req.subscribers[0] != ch1 {
		t.Error("channel not in subscribers")
	}
}

func TestRegisterTwice(t *testing.T) {
	v0requestToken := V0RequestToken{}

	ch1 := make(chan requestResults)
	ref1, _ := v0requestToken.registerRequest("abc.def", "hjkl", ch1)

	ch2 := make(chan requestResults)
	ref2, exists := v0requestToken.registerRequest("abc.def", "hjkl", ch2)
	if !exists {
		t.Error("expected exists flag to be true")
	}
	if ref1 != ref2 {
		t.Error("expected the same ref")
	}

	if len(v0requestToken.reqs) != 1 {
		t.Error("expecte reqs of length 1")
	}
	req := v0requestToken.reqs[0]
	if len(req.subscribers) != 2 {
		t.Error("expected two channels in subscribers")
	}
}

func TestUnsubscribeRequests(t *testing.T) {
	v0requestToken := V0RequestToken{}

	dom := "abc.def"
	sess := "hjkl"

	ch1 := make(chan requestResults)
	v0requestToken.registerRequest(dom, sess, ch1)

	ch2 := make(chan requestResults)
	v0requestToken.registerRequest(dom, sess, ch2)

	v0requestToken.unsubscribeRequest(dom, sess, ch1)
	req := v0requestToken.reqs[0]
	if len(req.subscribers) != 1 {
		t.Error("expected 1 channel in subscribers")
	}
	if req.subscribers[0] != ch2 {
		t.Error("expected ch2 in remaining subscribers")
	}

	v0requestToken.unsubscribeRequest(dom, sess, ch2)
	req = v0requestToken.reqs[0]
	if len(req.subscribers) != 0 {
		t.Error("expected 0 channels in subscribers")
	}
}

func TestPushResults(t *testing.T) {
	v0requestToken := V0RequestToken{}

	ch1 := make(chan requestResults)
	ref, _ := v0requestToken.registerRequest("abc.def", "hjkl", ch1)

	go v0requestToken.pushResults(ref, "tokenabc", nil)

	results := <-ch1
	if results.token != "tokenabc" {
		t.Error("did not get expected token")
	}
	if results.err != nil {
		t.Error("expected nil error")
	}

	if len(v0requestToken.reqs) != 0 {
		t.Error("expected zero reqs after push result")
	}

	// push results again to ensure it is inconsequential
	v0requestToken.pushResults(ref, "tokendef", nil)
}

func TestMakeRequest(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	config := domain.RuntimeConfig{}
	config.Server.NoTLS = true

	sess := "hjkl"
	tok := "tokenabc"
	userID := domain.UserID(7)
	userDropID := "dropid.domain.com/alice"

	ds2ds := testmocks.NewMockDS2DS(mockCtrl)
	ds2ds.EXPECT().GetClient().Return(http.DefaultClient)

	remoteModel := testmocks.NewMockRemoteAppspaceModel(mockCtrl)
	remoteModel.EXPECT().Get(userID, gomock.Any()).Return(domain.RemoteAppspace{UserDropID: userDropID}, nil)

	v0requestToken := V0RequestToken{
		Config:              config,
		DS2DS:               ds2ds,
		RemoteAppspaceModel: remoteModel}

	server := httptest.NewServer(http.HandlerFunc(
		func(res http.ResponseWriter, req *http.Request) {
			// check the request then async call push results
			var data domain.V0LoginTokenRequest
			err := readJSON(req, &data)
			if err != nil {
				v0requestToken.ReceiveError(sess, err)
				return
			}
			if data.DropID != userDropID {
				v0requestToken.ReceiveError(sess, errors.New("did not get dropid "+data.DropID))
				return
			}
			res.WriteHeader(http.StatusAccepted)
			go v0requestToken.ReceiveToken(data.Ref, tok)
		}))

	domPort := strings.TrimPrefix(server.URL, "http://")

	ch := make(chan requestResults)

	go v0requestToken.makeRequest(domain.UserID(7), domPort, sess, ch)

	results := <-ch
	if results.err != nil {
		t.Error(results.err)
	}
	if results.token != tok {
		t.Error("token incorrect")
	}
}

func TestRequestToken(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	config := domain.RuntimeConfig{}
	config.Server.NoTLS = true

	sess := "hjkl"
	tok := "tokenabc"
	userID := domain.UserID(7)
	userDropID := "dropid.domain.com/alice"

	ds2ds := testmocks.NewMockDS2DS(mockCtrl)
	ds2ds.EXPECT().GetClient().Return(http.DefaultClient)

	remoteModel := testmocks.NewMockRemoteAppspaceModel(mockCtrl)
	remoteModel.EXPECT().Get(userID, gomock.Any()).Return(domain.RemoteAppspace{UserDropID: userDropID}, nil)

	v0requestToken := V0RequestToken{
		Config:              config,
		DS2DS:               ds2ds,
		RemoteAppspaceModel: remoteModel}

	server := httptest.NewServer(http.HandlerFunc(
		func(res http.ResponseWriter, req *http.Request) {
			// check the request then async call push results
			var data domain.V0LoginTokenRequest
			err := readJSON(req, &data)
			if err != nil {
				v0requestToken.ReceiveError(sess, err)
				return
			}
			if data.DropID != userDropID {
				v0requestToken.ReceiveError(sess, errors.New("did not get dropid "+data.DropID))
				return
			}
			res.WriteHeader(http.StatusAccepted)
			go v0requestToken.ReceiveToken(data.Ref, tok)
		}))

	domPort := strings.TrimPrefix(server.URL, "http://")

	ctx := context.Background()

	token, err := v0requestToken.RequestToken(ctx, domain.UserID(7), domPort, sess)
	if err != nil {
		t.Error(err)
	}
	if token != tok {
		t.Error("token incorrect")
	}
}

func TestRequestTokenCancel(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	config := domain.RuntimeConfig{}
	config.Server.NoTLS = true

	sess := "hjkl"
	tok := "tokenabc"
	userID := domain.UserID(7)
	userDropID := "dropid.domain.com/alice"

	ds2ds := testmocks.NewMockDS2DS(mockCtrl)
	ds2ds.EXPECT().GetClient().Return(http.DefaultClient)

	remoteModel := testmocks.NewMockRemoteAppspaceModel(mockCtrl)
	remoteModel.EXPECT().Get(userID, gomock.Any()).Return(domain.RemoteAppspace{UserDropID: userDropID}, nil)

	v0requestToken := V0RequestToken{
		Config:              config,
		DS2DS:               ds2ds,
		RemoteAppspaceModel: remoteModel}

	server := httptest.NewServer(http.HandlerFunc(
		func(res http.ResponseWriter, req *http.Request) {
			// here we just sleep a bit
			var data domain.V0LoginTokenRequest
			readJSON(req, &data)
			<-time.After(time.Second)
			go v0requestToken.ReceiveToken(data.Ref, tok)
		}))

	domPort := strings.TrimPrefix(server.URL, "http://")

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-time.After(100 * time.Millisecond)
		cancel()
	}()

	token, err := v0requestToken.RequestToken(ctx, domain.UserID(7), domPort, sess)
	if err == nil {
		t.Error("expected an error because request was canceled")
	}
	if token != "" {
		t.Error("token should be empty")
	}
}

func readJSON(req *http.Request, data interface{}) error {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, data)
	if err != nil {
		return err
	}

	return nil
}
