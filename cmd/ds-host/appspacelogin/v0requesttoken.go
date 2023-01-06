package appspacelogin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

type tokenRequest struct {
	appspace  string
	sessionID string
	ref       string

	subscribers []chan requestResults
}

type requestResults struct {
	token string
	err   error
}

var timeout = 30 * time.Second

type V0RequestToken struct {
	Config domain.RuntimeConfig `checkinject:"required"`
	DS2DS  interface {
		GetClient() *http.Client
	} `checkinject:"required"`
	RemoteAppspaceModel interface {
		Get(userID domain.UserID, domainName string) (domain.RemoteAppspace, error)
	} `checkinject:"required"`

	reqsMux sync.Mutex
	reqs    []tokenRequest
}

// RequestToken for an appspace that is not owned by the user
func (r *V0RequestToken) RequestToken(ctx context.Context, userID domain.UserID, appspaceDomain string, sessionID string) (string, error) {
	ch := make(chan requestResults)
	go r.makeRequest(userID, appspaceDomain, sessionID, ch)

	select {
	case results := <-ch:
		return results.token, results.err
	case <-ctx.Done():
		r.unsubscribeRequest(appspaceDomain, sessionID, ch)
		close(ch)
		return "", errors.New("request cancelled")
	}
}

// makeRequest sends the request for a token to remote
// unless there already is an ongoing request
func (r *V0RequestToken) makeRequest(userID domain.UserID, appspaceDomain string, sessionID string, ch chan requestResults) {
	// first we register our channel, return if the request already exists
	ref, exists := r.registerRequest(appspaceDomain, sessionID, ch)
	if exists {
		return
	}

	remote, err := r.RemoteAppspaceModel.Get(userID, appspaceDomain)
	if err != nil {
		r.pushResults(ref, "", err)
		return
	}

	go func() {
		<-time.After(timeout)
		r.pushResults(ref, "", errors.New("operation get token timed out"))
	}()

	log := r.getLogger("makeRequest").UserID(userID).AddNote("remote appspace: " + appspaceDomain)

	data := domain.V0LoginTokenRequest{
		DropID: remote.UserDropID,
		Ref:    ref,
	}
	jsonStr, err := json.Marshal(data)
	if err != nil {
		log.AddNote("JSON encode login token").Error(err)
		return
	}

	client := r.DS2DS.GetClient()
	u := fmt.Sprintf("%s://%s%s/.dropserver/v0/login-token-request", r.Config.ExternalAccess.Scheme, appspaceDomain, r.Config.Exec.PortString)
	req, err := http.NewRequest(http.MethodPost, u, bytes.NewBuffer(jsonStr))
	if err != nil {
		r.pushResults(ref, "", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		r.pushResults(ref, "", err)
		return
	}
	if resp != nil {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusAccepted {
			r.pushResults(ref, "", errors.New("remote host sent status: "+resp.Status))
			return
		}
	}
	log.Debug("successfully completed")
}

// registerRequest adds the channel to subscribers of existing request
// or creates a new request with the channel as its only subscriber
func (r *V0RequestToken) registerRequest(appspaceDomain string, sessionID string, ch chan requestResults) (string, bool) {
	r.reqsMux.Lock()
	defer r.reqsMux.Unlock()

	if r.reqs == nil {
		r.reqs = make([]tokenRequest, 0, 10)
	}

	for i, req := range r.reqs {
		if req.appspace == appspaceDomain && req.sessionID == sessionID {
			r.reqs[i].subscribers = append(req.subscribers, ch)
			return req.ref, true
		}
	}

	ref := randomString(10)
	r.reqs = append(r.reqs, tokenRequest{
		appspace:    appspaceDomain,
		sessionID:   sessionID,
		ref:         ref,
		subscribers: []chan requestResults{ch}})

	return ref, false
}

// unsubscribeRequest removes the subscribed channel from requests
func (r *V0RequestToken) unsubscribeRequest(appspaceDomain string, sessionID string, ch chan requestResults) {
	r.reqsMux.Lock()
	defer r.reqsMux.Unlock()

	for i, req := range r.reqs {
		if req.appspace == appspaceDomain && req.sessionID == sessionID {
			for j, c := range req.subscribers {
				if c == ch {
					subs := r.reqs[i].subscribers
					r.reqs[i].subscribers = append(subs[:j], subs[j+1:]...)
				}
			}
			return
		}
	}
}

// pushResults pusehs the result to subscriber channels for the request
// then deletes the request
func (r *V0RequestToken) pushResults(ref string, token string, err error) {
	r.reqsMux.Lock()
	defer r.reqsMux.Unlock()

	for i, req := range r.reqs {
		if req.ref == ref {
			r.reqs = append(r.reqs[:i], r.reqs[i+1:]...) // that removes the req
			for _, c := range req.subscribers {
				c <- requestResults{token: token, err: err}
			}
			return
		}
	}

	// if we get here it means we didn't find the req.
	// This could happen if the req timed out but the remote eventually sent the token.
}

// ReceiveToken pushes the received token to all subscribers of that request
func (r *V0RequestToken) ReceiveToken(ref, token string) {
	r.pushResults(ref, token, nil)
}

// ReceiveError pushes the received error to all subscribers of that request
func (r *V0RequestToken) ReceiveError(ref string, err error) {
	r.pushResults(ref, "", err)
}

func (r *V0RequestToken) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("V0RequestToken")
	if note != "" {
		l.AddNote(note)
	}
	return l
}
