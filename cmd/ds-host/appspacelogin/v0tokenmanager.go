package appspacelogin

import (
	"bytes"
	"encoding/json"
	"errors"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

const loginTokenDuration = time.Minute

// V0TokenManager creates appspace login tokens
// and sends them as needed
type V0TokenManager struct {
	Config        domain.RuntimeConfig
	AppspaceModel interface {
		GetFromID(domain.AppspaceID) (*domain.Appspace, error)
	}
	AppspaceUserModel interface {
		GetByDropID(domain.AppspaceID, string) (domain.AppspaceUser, error)
	}

	tokensMux sync.Mutex
	tokens    map[string]domain.V0AppspaceLoginToken
	ticker    *time.Ticker
	stop      chan struct{}
}

// Start creates data structures and fires up the token purge ticker
func (m *V0TokenManager) Start() {
	m.tokens = make(map[string]domain.V0AppspaceLoginToken)

	m.ticker = time.NewTicker(time.Minute)
	m.stop = make(chan struct{})

	go func() {
		for {
			select {
			case <-m.stop:
				return
			case <-m.ticker.C:
				m.purgeTokens()
			}
		}
	}()
}

// Stop terminates the token purge ticker
func (m *V0TokenManager) Stop() {
	m.stop <- struct{}{}
}

// create an appspace login token
func (m *V0TokenManager) create(appspaceID domain.AppspaceID, dropID string) domain.V0AppspaceLoginToken {
	token := domain.V0AppspaceLoginToken{
		AppspaceID: appspaceID,
		DropID:     dropID,
		LoginToken: domain.TimedToken{
			Token:   randomString(24),
			Created: time.Now()},
	}

	m.tokensMux.Lock()
	defer m.tokensMux.Unlock()
	// check it doesn't exist first?
	m.tokens[token.LoginToken.Token] = token

	return token
}

// CheckToken returns a valid V0AppspaceLoginToken if found
func (m *V0TokenManager) CheckToken(token string) (domain.V0AppspaceLoginToken, bool) {
	m.tokensMux.Lock()
	defer m.tokensMux.Unlock()

	var t domain.V0AppspaceLoginToken
	found := false
	for _, t = range m.tokens {
		if t.LoginToken.Token == token {
			found = true
			break
		}
	}

	if !found {
		return domain.V0AppspaceLoginToken{}, false
	}

	delete(m.tokens, t.LoginToken.Token)

	if t.LoginToken.Created.Add(loginTokenDuration).Before(time.Now()) {
		return domain.V0AppspaceLoginToken{}, false
	}

	return t, true
}

// purgeTokens iterates over all tokens and deletes those that are expired
func (m *V0TokenManager) purgeTokens() {
	m.tokensMux.Lock()
	defer m.tokensMux.Unlock()

	for key, t := range m.tokens {
		if t.LoginToken.Created.Add(loginTokenDuration).Before(time.Now()) {
			delete(m.tokens, key)
		}
	}
}

// Get a login token for an appspace owned by the user
// Truns out this is just a proxy for create token
func (m *V0TokenManager) GetForOwner(appspaceID domain.AppspaceID, dropID string) string {
	tok := m.create(appspaceID, dropID)
	return tok.LoginToken.Token
}

// SendLoginToken verifies that drop id can access appspace
// (slightly odd that this checks creds yet Create does not)
// then creates a token and sends it to remote
// future: token types (long life, lock to IP or other client attribute)
func (m *V0TokenManager) SendLoginToken(appspaceID domain.AppspaceID, dropID string, ref string) error {
	log := m.getLogger("SendLoginToken").AppspaceID(appspaceID)

	appspace, err := m.AppspaceModel.GetFromID(appspaceID)
	if err != nil {
		return err
	}

	// should we check to see if appspace is paused?

	// if dropid not in appspace user, this returns no rows, so bail because you can't log them in
	_, err = m.AppspaceUserModel.GetByDropID(appspaceID, dropID)
	if err != nil {
		return err
	}

	// if no errors then user exists for that appspace
	// Should check if user is blocked and things like that when we have those features.

	token := m.create(appspaceID, dropID)

	// Now send the token. For now we can do this here?
	data := domain.V0LoginTokenResponse{
		Appspace: appspace.DomainName,
		Token:    token.LoginToken.Token,
		Ref:      ref}

	jsonStr, err := json.Marshal(data)
	if err != nil {
		log.AddNote("JSON encode login token").Error(err)
		return err
	}

	protocol := "https"
	if m.Config.Server.NoSsl {
		protocol = "http"
	}

	resp, err := http.Post(protocol+"://"+appspace.DomainName+m.Config.Exec.PortString+"/.dropserver/v0/appspace-login-token", "application/json", bytes.NewBuffer(jsonStr))
	if err != nil {
		log.AddNote("Error posting token").Error(err)
		return err
	}
	if resp.StatusCode != http.StatusOK {
		log.Log("got unexpected status code: " + resp.Status)
		return errors.New("got unexpected status code from remote: " + resp.Status)
	}
	return nil
}

func (m *V0TokenManager) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("V0TokenManager")
	if note != "" {
		l.AddNote(note)
	}
	return l
}

////////////
// random string stuff
// TODO CRYPTO: this should be using crypto package
const chars61 = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand2 = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func randomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = chars61[seededRand2.Intn(len(chars61))]
	}
	return string(b)
}
