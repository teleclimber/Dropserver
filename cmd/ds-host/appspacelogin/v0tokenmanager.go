package appspacelogin

import (
	"fmt"
	"sync"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/randomstring"
)

const loginTokenDuration = time.Minute

// V0TokenManager creates appspace login tokens
// and sends them as needed
type V0TokenManager struct {
	Config domain.RuntimeConfig `checkinject:"required"`

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
func (m *V0TokenManager) create(appspaceID domain.AppspaceID, proxyID domain.ProxyID) domain.V0AppspaceLoginToken {
	token := domain.V0AppspaceLoginToken{
		AppspaceID: appspaceID,
		ProxyID:    proxyID,
		LoginToken: domain.TimedToken{
			Token:   randomstring.RandomStringWithCaps(24),
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

// GetForProxyID returns login token for a proxy id for an appspace on the instance
func (m *V0TokenManager) GetForProxyID(appspaceID domain.AppspaceID, proxyID domain.ProxyID) string {
	tok := m.create(appspaceID, proxyID)
	return tok.LoginToken.Token
}

func (m *V0TokenManager) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("V0TokenManager")
	if note != "" {
		l.AddNote(note)
	}
	return l
}
