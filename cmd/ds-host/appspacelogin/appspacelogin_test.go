package appspacelogin

import (
	"net/url"
	"testing"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestCreate(t *testing.T) {
	a := AppspaceLogin{}
	a.tokens = make(map[string]domain.AppspaceLoginToken)

	tok := a.Create(domain.AppspaceID(7), url.URL{})
	if tok.AppspaceID != domain.AppspaceID(7) {
		t.Error("wrong appspace id")
	}
	if tok.LoginToken.Token == "" {
		t.Error("expected a log in token")
	}
	if tok.LoginToken.Created.Add(time.Second).Before(time.Now()) {
		t.Error("login token created date seems wrong")
	}
	if tok.RedirectToken.Token != "" {
		t.Error("redirect token should be blank at this point")
	}
}

func TestLogIn(t *testing.T) {
	a := AppspaceLogin{}
	a.tokens = make(map[string]domain.AppspaceLoginToken)

	tok := a.Create(domain.AppspaceID(7), url.URL{})

	tok, err := a.LogIn(tok.LoginToken.Token, domain.UserID(11))
	if err != nil {
		t.Error(err)
	}
	if tok.RedirectToken.Token == "" {
		t.Error("expected redirect token to be created")
	}
	if tok.RedirectToken.Created.Add(time.Second).Before(time.Now()) {
		t.Error("redirect token created time seems off")
	}
}

func TestLogInBadTokens(t *testing.T) {
	a := AppspaceLogin{}
	a.tokens = make(map[string]domain.AppspaceLoginToken)

	tok := a.Create(domain.AppspaceID(7), url.URL{})

	_, err := a.LogIn("baz", domain.UserID(11))
	if err == nil {
		t.Error("baz is non existent token, expected error")
	}

	expiredToken := tok
	expiredToken.LoginToken.Created = time.Now().Add(-time.Hour)
	a.tokens[tok.LoginToken.Token] = expiredToken
	_, err = a.LogIn(tok.LoginToken.Token, domain.UserID(11))
	if err == nil {
		t.Error("token is expired, expected error")
	}
}

func TestLogInTwice(t *testing.T) {
	a := AppspaceLogin{}
	a.tokens = make(map[string]domain.AppspaceLoginToken)

	tok := a.Create(domain.AppspaceID(7), url.URL{})

	_, err := a.LogIn(tok.LoginToken.Token, domain.UserID(11))
	if err != nil {
		t.Error(err)
	}
	_, err = a.LogIn(tok.LoginToken.Token, domain.UserID(11))
	if err == nil {
		t.Error("expected error from second login")
	}
}

func TestCheckRedirect(t *testing.T) {
	a := AppspaceLogin{}
	a.tokens = make(map[string]domain.AppspaceLoginToken)

	tok := a.Create(domain.AppspaceID(7), url.URL{})

	tok, err := a.LogIn(tok.LoginToken.Token, domain.UserID(11))
	if err != nil {
		t.Error(err)
	}

	tok, err = a.CheckRedirectToken(tok.RedirectToken.Token)
	if err != nil {
		t.Error(err)
	}
}

func TestCheckRedirectBad(t *testing.T) {
	a := AppspaceLogin{}
	a.tokens = make(map[string]domain.AppspaceLoginToken)

	a.tokens[""] = domain.AppspaceLoginToken{}

	_, err := a.CheckRedirectToken("")
	if err == nil {
		t.Error("empty string token should be an error")
	}

	_, err = a.CheckRedirectToken("baz")
	if err == nil {
		t.Error("non-existent token should be an error")
	}
}

func TestCheckRedirectExpired(t *testing.T) {
	a := AppspaceLogin{}
	a.tokens = make(map[string]domain.AppspaceLoginToken)

	tok := a.Create(domain.AppspaceID(7), url.URL{})

	tok, err := a.LogIn(tok.LoginToken.Token, domain.UserID(11))
	if err != nil {
		t.Error(err)
	}

	expiredTok := tok
	expiredTok.RedirectToken.Created = time.Now().Add(-time.Hour)
	a.tokens[tok.LoginToken.Token] = expiredTok

	_, err = a.CheckRedirectToken(tok.RedirectToken.Token)
	if err == nil {
		t.Error("expecred error from expired token")
	}
}

func TestCheckRedirectTwice(t *testing.T) {
	a := AppspaceLogin{}
	a.tokens = make(map[string]domain.AppspaceLoginToken)

	tok := a.Create(domain.AppspaceID(7), url.URL{})

	tok, err := a.LogIn(tok.LoginToken.Token, domain.UserID(11))
	if err != nil {
		t.Error(err)
	}

	_, err = a.CheckRedirectToken(tok.RedirectToken.Token)
	if err != nil {
		t.Error(err)
	}
	_, err = a.CheckRedirectToken(tok.RedirectToken.Token)
	if err == nil {
		t.Error("expecred error from second check")
	}
}

func TestPurgeTokens(t *testing.T) {
	a := AppspaceLogin{}
	a.tokens = make(map[string]domain.AppspaceLoginToken)
	a.tokens["a"] = domain.AppspaceLoginToken{
		LoginToken: domain.TimedToken{
			Token:   "foo",
			Created: time.Now().Add(-time.Hour), // login time expired
		},
	}
	a.tokens["b"] = domain.AppspaceLoginToken{
		LoginToken: domain.TimedToken{
			Token:   "foo",
			Created: time.Now(), // valid
		},
	}
	a.tokens["c"] = domain.AppspaceLoginToken{
		LoginToken: domain.TimedToken{
			Token:   "foo",
			Created: time.Now(), // valid
		},
		RedirectToken: domain.TimedToken{
			Token:   "bar",
			Created: time.Now().Add(-time.Minute), // expired
		},
	}
	a.tokens["d"] = domain.AppspaceLoginToken{
		LoginToken: domain.TimedToken{
			Token:   "foo",
			Created: time.Now(), // valid
		},
		RedirectToken: domain.TimedToken{
			Token:   "bar",
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
	if _, ok := a.tokens["c"]; ok {
		t.Error("expected c to be removed")
	}
	if _, ok := a.tokens["d"]; !ok {
		t.Error("expected d to stay")
	}
}

func TestStartStop(t *testing.T) {
	a := AppspaceLogin{}
	a.Start()
	a.Stop()
}
