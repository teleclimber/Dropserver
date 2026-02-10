package appspaceops

import (
	"sync"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

// AppspaceUsersCache caches user-appspace relationships and invalidates
// based on events from InstanceUserAuthsChangeEvents and AppspaceUsersChangeEvents.
type AppspaceUsersCache struct {
	ManageUsers interface {
		ConflictsForAppspace(appspaceID domain.AppspaceID) (map[domain.UserProxyTuple]domain.UserIDProxyIDConflicts, error)
		AppspacesForUser(userID domain.UserID) (map[domain.AppspaceID]domain.UserIDProxyIDConflicts, error)
	} `checkinject:"required"`
	InstanceUserAuthsChangeEvents interface {
		Subscribe() <-chan domain.UserID
		Unsubscribe(ch <-chan domain.UserID)
	} `checkinject:"required"`
	AppspaceUsersChangeEvents interface {
		Subscribe() <-chan domain.AppspaceID
		Unsubscribe(ch <-chan domain.AppspaceID)
	} `checkinject:"required"`
	UserAppspacesEvent interface {
		Send(domain.UserID)
	} `checkinject:"required"`
	AppspaceUsersEvent interface {
		Send(domain.AppspaceID)
	} `checkinject:"required"`

	appspaceCacheMux sync.RWMutex
	appspaceCache    map[domain.AppspaceID]map[domain.UserProxyTuple]domain.UserIDProxyIDConflicts

	userCacheMux sync.RWMutex
	userCache    map[domain.UserID]map[domain.AppspaceID]domain.UserIDProxyIDConflicts
}

// Init initializes the caches and starts event handler goroutines.
func (c *AppspaceUsersCache) Init() {
	c.appspaceCache = make(map[domain.AppspaceID]map[domain.UserProxyTuple]domain.UserIDProxyIDConflicts)
	c.userCache = make(map[domain.UserID]map[domain.AppspaceID]domain.UserIDProxyIDConflicts)

	userCh := c.InstanceUserAuthsChangeEvents.Subscribe()
	go c.handleInstanceUserChanges(userCh)

	appspaceCh := c.AppspaceUsersChangeEvents.Subscribe()
	go c.handleAppspaceUserChanges(appspaceCh)
}

// UsersForAppspace returns cached users for an appspace, fetching fresh data if not cached.
func (c *AppspaceUsersCache) UsersForAppspace(appspaceID domain.AppspaceID) (map[domain.UserID]domain.UserIDProxyIDConflicts, error) {
	conflicts, err := c.conflictsForAppspace(appspaceID)
	if err != nil {
		return nil, err
	}
	ret := make(map[domain.UserID]domain.UserIDProxyIDConflicts)
	for t, c := range conflicts {
		ret[t.UserID] = c
	}
	return ret, nil
}

// ProxyIDsForAppspace returns cached users for an appspace, fetching fresh data if not cached.
func (c *AppspaceUsersCache) ProxyIDsForAppspace(appspaceID domain.AppspaceID) (map[domain.ProxyID]domain.UserIDProxyIDConflicts, error) {
	conflicts, err := c.conflictsForAppspace(appspaceID)
	if err != nil {
		return nil, err
	}
	ret := make(map[domain.ProxyID]domain.UserIDProxyIDConflicts)
	for t, c := range conflicts {
		ret[t.ProxyID] = c
	}
	return ret, nil
}

func (c *AppspaceUsersCache) conflictsForAppspace(appspaceID domain.AppspaceID) (map[domain.UserProxyTuple]domain.UserIDProxyIDConflicts, error) {
	c.appspaceCacheMux.RLock()
	cached, ok := c.appspaceCache[appspaceID]
	c.appspaceCacheMux.RUnlock()

	if ok {
		c.getLogger("conflictsForAppspace").Debug("cached")
		return cached, nil
	}

	// Cache miss - fetch fresh data
	conflicts, err := c.ManageUsers.ConflictsForAppspace(appspaceID)
	if err != nil {
		return nil, err
	}

	c.appspaceCacheMux.Lock()
	c.appspaceCache[appspaceID] = conflicts
	c.appspaceCacheMux.Unlock()

	c.getLogger("conflictsForAppspace").Debug("not cached")

	return conflicts, nil
}

// AppspacesForUser returns cached appspaces for a user, fetching fresh data if not cached.
func (c *AppspaceUsersCache) AppspacesForUser(userID domain.UserID) (map[domain.AppspaceID]domain.UserIDProxyIDConflicts, error) {
	c.userCacheMux.RLock()
	cached, ok := c.userCache[userID]
	c.userCacheMux.RUnlock()

	if ok {
		c.getLogger("AppspacesForUser").Debug("cached")
		return cached, nil
	}

	// Cache miss - fetch fresh data
	appspaces, err := c.ManageUsers.AppspacesForUser(userID)
	if err != nil {
		return nil, err
	}

	c.userCacheMux.Lock()
	c.userCache[userID] = appspaces
	c.userCacheMux.Unlock()

	c.getLogger("AppspacesForUser").Debug("not cached")

	return appspaces, nil
}

// handleInstanceUserChanges listens for instance user auth changes and invalidates caches.
func (c *AppspaceUsersCache) handleInstanceUserChanges(ch <-chan domain.UserID) {
	for userID := range ch {
		c.invalidateForUser(userID)
	}
}

// invalidateForUser invalidates caches when a user's auths change.
// We invalidate that user's cache and all appspace caches since the user
// could now match different appspace users.
func (c *AppspaceUsersCache) invalidateForUser(userID domain.UserID) {
	c.userCacheMux.Lock()
	delete(c.userCache, userID)
	c.userCacheMux.Unlock()

	c.appspaceCacheMux.Lock()
	appspaceIDs := make([]domain.AppspaceID, 0, len(c.appspaceCache))
	for appspaceID := range c.appspaceCache {
		appspaceIDs = append(appspaceIDs, appspaceID)
	}
	c.appspaceCache = make(map[domain.AppspaceID]map[domain.UserProxyTuple]domain.UserIDProxyIDConflicts)
	c.appspaceCacheMux.Unlock()

	c.UserAppspacesEvent.Send(userID)

	// then send event for all appspaces
	for _, appspaceID := range appspaceIDs {
		c.AppspaceUsersEvent.Send(appspaceID)
	}
}

// handleAppspaceUserChanges listens for appspace user changes and invalidates caches.
func (c *AppspaceUsersCache) handleAppspaceUserChanges(ch <-chan domain.AppspaceID) {
	for appspaceID := range ch {
		c.invalidateForAppspace(appspaceID)
	}
}

// invalidateForAppspace invalidates caches when appspace users change.
// We invalidate that appspace's cache and all user caches since users
// could now match different appspace users.
func (c *AppspaceUsersCache) invalidateForAppspace(appspaceID domain.AppspaceID) {
	c.appspaceCacheMux.Lock()
	delete(c.appspaceCache, appspaceID)
	c.appspaceCacheMux.Unlock()

	c.userCacheMux.Lock()
	userIDs := make([]domain.UserID, 0, len(c.userCache))
	for userID := range c.userCache {
		userIDs = append(userIDs, userID)
	}
	c.userCache = make(map[domain.UserID]map[domain.AppspaceID]domain.UserIDProxyIDConflicts)
	c.userCacheMux.Unlock()

	for _, userID := range userIDs {
		c.UserAppspacesEvent.Send(userID)
	}

	c.AppspaceUsersEvent.Send(appspaceID)
}

func (c *AppspaceUsersCache) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("AppspaceUsersCache")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
