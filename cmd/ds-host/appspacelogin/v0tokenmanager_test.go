package appspacelogin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

func TestCreate(t *testing.T) {
	a := V0TokenManager{}
	a.tokens = make(map[string]domain.V0AppspaceLoginToken)
	proxyID := domain.ProxyID("proxy-abc")

	tok := a.create(domain.AppspaceID(7), "abc.com/def", proxyID)
	if tok.AppspaceID != domain.AppspaceID(7) {
		t.Error("wrong appspace id")
	}
	if tok.LoginToken.Token == "" {
		t.Error("expected a log in token")
	}
	if tok.LoginToken.Created.Add(time.Second).Before(time.Now()) {
		t.Error("login token created date seems wrong")
	}
}

func TestCheckToken(t *testing.T) {
	a := V0TokenManager{}
	a.tokens = make(map[string]domain.V0AppspaceLoginToken)
	proxyID := domain.ProxyID("proxy-abc")
	appspaceID := domain.AppspaceID(7)

	tok := a.create(appspaceID, "abc.com/def", proxyID)

	tok, ok := a.CheckToken(appspaceID, tok.LoginToken.Token)
	if !ok {
		t.Error("expected token found ok")
	}
	if tok.LoginToken.Token == "" {
		t.Error("expected LoginToken token to be created")
	}
	if tok.LoginToken.Created.Add(time.Second).Before(time.Now()) {
		t.Error("LoginToken token created time seems off")
	}
}

func TestLogInBadTokens(t *testing.T) {
	a := V0TokenManager{}
	a.tokens = make(map[string]domain.V0AppspaceLoginToken)
	proxyID := domain.ProxyID("proxy-abc")
	appspaceID := domain.AppspaceID(7)

	tok := a.create(appspaceID, "abc.com/def", proxyID)

	_, ok := a.CheckToken(appspaceID, "baz")
	if ok {
		t.Error("baz is non existent token, expected false")
	}

	_, ok = a.CheckToken(domain.AppspaceID(13), tok.LoginToken.Token)
	if ok {
		t.Error("wrong appspace id, expected false")
	}

	expiredToken := tok
	expiredToken.LoginToken.Created = time.Now().Add(-time.Hour)
	a.tokens[tok.LoginToken.Token] = expiredToken
	_, ok = a.CheckToken(appspaceID, tok.LoginToken.Token)
	if ok {
		t.Error("token is expired, expected false")
	}
}

func TestLogInTwice(t *testing.T) {
	a := V0TokenManager{}
	a.tokens = make(map[string]domain.V0AppspaceLoginToken)
	proxyID := domain.ProxyID("proxy-abc")
	appspaceID := domain.AppspaceID(7)

	tok := a.create(appspaceID, "abc.com/def", proxyID)

	_, ok := a.CheckToken(appspaceID, tok.LoginToken.Token)
	if !ok {
		t.Error("expected token found ok")
	}
	_, ok = a.CheckToken(appspaceID, tok.LoginToken.Token)
	if ok {
		t.Error("expected not ok from second login")
	}
}

func TestPurgeTokens(t *testing.T) {
	a := V0TokenManager{}
	a.tokens = make(map[string]domain.V0AppspaceLoginToken)
	a.tokens["a"] = domain.V0AppspaceLoginToken{
		LoginToken: domain.TimedToken{
			Token:   "foo",
			Created: time.Now().Add(-time.Hour), // login time expired
		},
	}
	a.tokens["b"] = domain.V0AppspaceLoginToken{
		LoginToken: domain.TimedToken{
			Token:   "foo",
			Created: time.Now(), // valid
		},
	}
	a.tokens["c"] = domain.V0AppspaceLoginToken{
		LoginToken: domain.TimedToken{
			Token:   "foo",
			Created: time.Now(), // valid
		},
	}

	a.purgeTokens()

	if _, ok := a.tokens["a"]; ok {
		t.Error("expected a to be removed")
	}
	if _, ok := a.tokens["b"]; !ok {
		t.Error("expected b to stay")
	}
	if _, ok := a.tokens["c"]; !ok {
		t.Error("expected c to be removed")
	}
}

func TestStartStop(t *testing.T) {
	a := V0TokenManager{}
	a.Start()
	a.Stop()
}

func TestSendToken(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appspaceID := domain.AppspaceID(7)
	dropID := "dropid.example.com/alice"
	ref := "ref123"

	config := domain.RuntimeConfig{}
	config.Server.NoSsl = true

	ds2ds := testmocks.NewMockDS2DS(mockCtrl)
	ds2ds.EXPECT().GetClient().Return(http.DefaultClient)

	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)

	appspaceUserModel := testmocks.NewMockAppspaceUserModel(mockCtrl)
	appspaceUserModel.EXPECT().GetByDropID(appspaceID, dropID).Return(domain.AppspaceUser{}, nil)

	tokenManager := V0TokenManager{
		Config:            config,
		DS2DS:             ds2ds,
		AppspaceModel:     appspaceModel,
		AppspaceUserModel: appspaceUserModel}
	tokenManager.Start()

	server := httptest.NewServer(http.HandlerFunc(
		func(res http.ResponseWriter, req *http.Request) {
			var data domain.V0LoginTokenResponse
			err := readJSON(req, &data)
			if err != nil {
				http.Error(res, err.Error(), 999)
				return
			}
			if data.Ref != ref {
				http.Error(res, "received drop id deos not match", 999)
				return
			}
			res.WriteHeader(http.StatusOK)
		}))

	domPort := strings.TrimPrefix(server.URL, "http://")

	appspaceModel.EXPECT().GetFromID(appspaceID).Return(&domain.Appspace{DomainName: domPort}, nil)

	err := tokenManager.SendLoginToken(appspaceID, dropID, ref)
	if err != nil {
		t.Error(err)
	}
	tokenManager.Stop()
}
