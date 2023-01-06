package appspacelogin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
	Config domain.RuntimeConfig `checkinject:"required"`
	DS2DS  interface {
		GetClient() *http.Client
	} `checkinject:"required"`
	AppspaceModel interface {
		GetFromID(domain.AppspaceID) (*domain.Appspace, error)
	} `checkinject:"required"`
	AppspaceUsersModelV0 interface {
		GetByDropID(domain.AppspaceID, string) (domain.AppspaceUser, error)
	} `checkinject:"required"`

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
func (m *V0TokenManager) create(appspaceID domain.AppspaceID, dropID string, proxyID domain.ProxyID) domain.V0AppspaceLoginToken {
	token := domain.V0AppspaceLoginToken{
		AppspaceID: appspaceID,
		DropID:     dropID,
		ProxyID:    proxyID,
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
func (m *V0TokenManager) CheckToken(appspaceID domain.AppspaceID, token string) (domain.V0AppspaceLoginToken, bool) {
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

	if t.AppspaceID != appspaceID {
		m.getLogger("CheckToken").Log(fmt.Sprintf("attempt to use token with wrong appspace. token appspace: %v, check appspace: %v", t.AppspaceID, appspaceID))
		return domain.V0AppspaceLoginToken{}, false
	}

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
func (m *V0TokenManager) GetForOwner(appspaceID domain.AppspaceID, dropID string) (string, error) {
	user, err := m.AppspaceUsersModelV0.GetByDropID(appspaceID, dropID)
	if err != nil {
		// this can happen if an appspace is imported without the new owner among its users.
		m.getLogger("GetForOwner").Debug("appspace user dropid not found " + dropID)
		return "", err
	}
	tok := m.create(appspaceID, dropID, user.ProxyID)
	return tok.LoginToken.Token, nil
}

// SendLoginToken verifies that drop id can access appspace
// (slightly odd that this checks creds yet Create does not)
// then creates a token and sends it to remote
// future: token types (long life, lock to IP or other client attribute)
func (m *V0TokenManager) SendLoginToken(appspaceID domain.AppspaceID, dropID string, ref string) error {
	log := m.getLogger("SendLoginToken").AppspaceID(appspaceID)

	appspace, err := m.AppspaceModel.GetFromID(appspaceID)
	if err != nil {
		log.Debug("appspace id not found")
		return err
	}

	// should we check to see if appspace is paused?

	// if dropid not in appspace user, this returns no rows, so bail because you can't log them in
	user, err := m.AppspaceUsersModelV0.GetByDropID(appspaceID, dropID)
	if err != nil {
		log.Debug("appspace user dropid not found " + dropID)
		return err
	}

	// Should check if user is blocked and things like that when we have those features.

	token := m.create(appspaceID, dropID, user.ProxyID)

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

	client := m.DS2DS.GetClient()
	u := fmt.Sprintf("%s://%s%s/.dropserver/v0/login-token", m.Config.ExternalAccess.Scheme, appspace.DomainName, m.Config.Exec.PortString)
	req, err := http.NewRequest(http.MethodPost, u, bytes.NewBuffer(jsonStr))
	if err != nil {
		log.AddNote("Error creating request").Error(err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.AddNote("Error posting token").Error(err)
		return err
	}
	if resp.StatusCode != http.StatusOK {
		log.Log("got unexpected status code: " + resp.Status)
		return errors.New("got unexpected status code from remote: " + resp.Status)
	}

	log.Debug(fmt.Sprintf("sent token for %v to %v", appspace.DomainName, dropID))

	return nil
}

func (m *V0TokenManager) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("V0TokenManager")
	if note != "" {
		l.AddNote(note)
	}
	return l
}

// //////////
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
