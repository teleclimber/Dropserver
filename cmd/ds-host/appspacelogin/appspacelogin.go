package appspacelogin

import (
	"errors"
	"math/rand"
	"net/url"
	"sync"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

const loginTokenDuration = time.Minute
const redirectTokenDuration = 5 * time.Second

// AppspaceLogin registers tokens tha can be exchanged for cookies in appspaces
type AppspaceLogin struct {
	tokensMux sync.Mutex
	tokens    map[string]domain.AppspaceLoginToken
	ticker    *time.Ticker
	stop      chan struct{}
}

// Start creates data structures and fires up the token purge ticker
func (a *AppspaceLogin) Start() {
	// make the map

	a.tokens = make(map[string]domain.AppspaceLoginToken)

	// start a ticker to routinely purge the map of outdated tokens
	a.ticker = time.NewTicker(time.Minute)
	a.stop = make(chan struct{})

	go func() {
		for {
			select {
			case <-a.stop:
				return
			case <-a.ticker.C:
				a.purgeTokens()
			}
		}
	}()
}

// Stop terminates the token purge ticker
func (a *AppspaceLogin) Stop() {
	a.stop <- struct{}{}
}

// Create an appspace login token
func (a *AppspaceLogin) Create(appspaceID domain.AppspaceID, appspaceURL url.URL) domain.AppspaceLoginToken {
	t := domain.AppspaceLoginToken{
		AppspaceID:  appspaceID,
		AppspaceURL: appspaceURL,
		LoginToken: domain.TimedToken{
			Token:   randomString(),
			Created: time.Now()},
	}

	record.Debug("Appspace Login Create, host: " + appspaceURL.Host + " url: " + appspaceURL.String())

	a.tokensMux.Lock()
	defer a.tokensMux.Unlock()
	// check it doesn't exist first?
	a.tokens[t.LoginToken.Token] = t

	return t
}

// LogIn creates a redirect token if the login token is valid
func (a *AppspaceLogin) LogIn(loginToken string, userID domain.UserID) (domain.AppspaceLoginToken, error) {
	a.tokensMux.Lock()
	defer a.tokensMux.Unlock()

	t, ok := a.tokens[loginToken]
	if !ok {
		return domain.AppspaceLoginToken{}, errors.New("token not found")
	}
	if t.LoginToken.Created.Add(loginTokenDuration).Before(time.Now()) {
		return domain.AppspaceLoginToken{}, errors.New("token expired")
	}
	if t.RedirectToken.Token != "" {
		return domain.AppspaceLoginToken{}, errors.New("token already used")
	}

	t.UserID = userID
	t.RedirectToken = domain.TimedToken{
		Token:   randomString(),
		Created: time.Now()}

	a.tokens[loginToken] = t

	return t, nil
}

// CheckRedirectToken returns a valid AppspaceLoginToken if found
func (a *AppspaceLogin) CheckRedirectToken(redirectToken string) (domain.AppspaceLoginToken, error) {
	if redirectToken == "" {
		return domain.AppspaceLoginToken{}, errors.New("Invalid token string")
	}

	a.tokensMux.Lock()
	defer a.tokensMux.Unlock()

	var t domain.AppspaceLoginToken
	found := false
	for _, t = range a.tokens {
		if t.RedirectToken.Token == redirectToken {
			found = true
			break
		}
	}

	if !found {
		return domain.AppspaceLoginToken{}, errors.New("No valid token")
	}

	delete(a.tokens, t.LoginToken.Token)

	if t.RedirectToken.Created.Add(redirectTokenDuration).Before(time.Now()) {
		return domain.AppspaceLoginToken{}, errors.New("No valid token")
	}

	return t, nil
}

// purgeTokens iterates over all tokens and deletes those that are expired
func (a *AppspaceLogin) purgeTokens() {
	a.tokensMux.Lock()
	defer a.tokensMux.Unlock()

	for key, t := range a.tokens {
		if t.RedirectToken.Token != "" && t.RedirectToken.Created.Add(redirectTokenDuration).Before(time.Now()) {
			delete(a.tokens, key)
		} else if t.LoginToken.Created.Add(loginTokenDuration).Before(time.Now()) {
			delete(a.tokens, key)
		}
	}
}

////////////
// random string stuff
// TODO CRYPTO: this should be using crypto package
const chars61 = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand2 = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func randomString() string {
	b := make([]byte, 24)
	for i := range b {
		b[i] = chars61[seededRand2.Intn(len(chars61))]
	}
	return string(b)
}
