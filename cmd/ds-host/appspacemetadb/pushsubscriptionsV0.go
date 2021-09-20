package appspacemetadb

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/sqlxprepper"

	webpush "github.com/SherClockHolmes/webpush-go"
)

// CRUD ops against the db table that stores push subscriptions

// should we involve SherClockHolmes types here?
// -> could simplify things.

type pushSubDbRow struct {
	hash     string `db:"hash"`
	json_sub []byte `db:"json_sub"`
	proxy_id string `db:"proxy_id"`
}

type PushSubscriptionsV0 struct {
	AppspaceMetaDB interface {
		GetHandle(domain.AppspaceID) (*sqlx.DB, error)
	}
}

func (p *PushSubscriptionsV0) AddSubscription(appspaceID domain.AppspaceID, sub webpush.Subscription, proxyID domain.ProxyID) (string, error) {
	db, err := p.AppspaceMetaDB.GetHandle(appspaceID)
	if err != nil {
		return "", err
	}

	hash := getHash(sub)
	jsonSub, err := json.Marshal(sub)
	if err != nil {
		return "", err
	}

	// validate subscription to some extent.
	// In particular, check that the origin matches a whitelist?
	prepper := sqlxprepper.NewPrepper(db)
	stmt := prepper.Prep(`INSERT INTO push_subscriptions 
		(hash, json_sub, proxy_id) 
		VALUES (?, ?, ?)`)
	_, err = stmt.Exec(hash, jsonSub, string(proxyID))
	if err != nil {
		// guessing that duplicate hash will be a common occurence so handle that?
		return "", err
	}
	return hash, nil
}

// func (p *PushSubscriptionsV0) GetSubscription(hash string) (domain.PushSubscriptionV0, error) {

// }

func (p *PushSubscriptionsV0) GetSubscriptionsForUser(appspaceID domain.AppspaceID, proxyID domain.ProxyID) ([]domain.PushSubscriptionV0, error) {
	db, err := p.AppspaceMetaDB.GetHandle(appspaceID)
	if err != nil {
		return nil, err
	}
	prepper := sqlxprepper.NewPrepper(db)
	stmt := prepper.Prep(`SELECT * FROM push_subscriptions WHERE proxy_id = ?`)
	var rows []pushSubDbRow
	err = stmt.Select(&rows, proxyID)
	if err != nil {
		// log it
		return nil, err
	}
	return toPushSubscriptionV0s(rows)
}

// func (p *PushSubscriptionsV0) GetAllSubscriptions() ([]domain.PushSubscriptionV0, error) {

// }

func getHash(sub webpush.Subscription) string {
	raw := sha256.Sum256([]byte(sub.Endpoint + sub.Keys.P256dh + sub.Keys.Auth))
	return string(raw[:])
}

func toPushSubscriptionV0s(rows []pushSubDbRow) ([]domain.PushSubscriptionV0, error) {
	rets := make([]domain.PushSubscriptionV0, len(rows))
	for i, row := range rows {
		ret, err := toPushSubscriptionV0(row)
		if err != nil {
			return nil, err
		}
		rets[i] = ret
	}
	return rets, nil
}
func toPushSubscriptionV0(row pushSubDbRow) (domain.PushSubscriptionV0, error) {
	sub := webpush.Subscription{}
	err := json.Unmarshal(row.json_sub, &sub)
	if err != nil {
		return domain.PushSubscriptionV0{}, fmt.Errorf("Error unmarshaling json sub for %v: %w", row.hash, err)
	}
	hasProxy := false
	if row.proxy_id != "" {
		hasProxy = true
	}
	return domain.PushSubscriptionV0{
		Hash:         row.hash,
		Subscription: sub,
		HasProxyID:   hasProxy,
		ProxyID:      domain.ProxyID(row.proxy_id),
	}, nil
}
