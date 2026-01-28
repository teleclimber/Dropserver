package appspaceops

import (
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

func TestUsersForAppspaceCacheMiss(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockManageUsers := testmocks.NewMockManageUsers(ctrl)

	testData := map[domain.UserID]domain.UserIDProxyIDConflicts{
		domain.UserID(1): {UserID: domain.UserID(1), ProxyID: "proxy1"},
	}
	mockManageUsers.EXPECT().UsersForAppspace(domain.AppspaceID(1)).Return(testData, nil)

	cache := &AppspaceUsersCache{
		ManageUsers:   mockManageUsers,
		appspaceCache: make(map[domain.AppspaceID]map[domain.UserID]domain.UserIDProxyIDConflicts),
	}

	result, err := cache.UsersForAppspace(domain.AppspaceID(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 user, got %d", len(result))
	}
}

func TestUsersForAppspaceCacheHit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockManageUsers := testmocks.NewMockManageUsers(ctrl)

	// Set up expectations - should only be called once due to cache hit
	testData := map[domain.UserID]domain.UserIDProxyIDConflicts{
		domain.UserID(1): {UserID: domain.UserID(1), ProxyID: "proxy1"},
	}
	mockManageUsers.EXPECT().UsersForAppspace(domain.AppspaceID(1)).Return(testData, nil)

	cache := &AppspaceUsersCache{
		ManageUsers:   mockManageUsers,
		appspaceCache: make(map[domain.AppspaceID]map[domain.UserID]domain.UserIDProxyIDConflicts),
	}

	// First call - cache miss
	_, err := cache.UsersForAppspace(domain.AppspaceID(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second call - should be cache hit
	result, err := cache.UsersForAppspace(domain.AppspaceID(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 user, got %d", len(result))
	}
}

func TestAppspacesForUserCacheMiss(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockManageUsers := testmocks.NewMockManageUsers(ctrl)

	// Set up expectations
	testData := map[domain.AppspaceID]domain.UserIDProxyIDConflicts{
		domain.AppspaceID(1): {UserID: domain.UserID(1), ProxyID: "proxy1"},
	}
	mockManageUsers.EXPECT().AppspacesForUser(domain.UserID(1)).Return(testData, nil)

	cache := &AppspaceUsersCache{
		ManageUsers: mockManageUsers,
		userCache:   make(map[domain.UserID]map[domain.AppspaceID]domain.UserIDProxyIDConflicts),
	}

	result, err := cache.AppspacesForUser(domain.UserID(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 appspace, got %d", len(result))
	}
}

func TestAppspacesForUserCacheHit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockManageUsers := testmocks.NewMockManageUsers(ctrl)

	// Set up expectations - should only be called once due to cache hit
	testData := map[domain.AppspaceID]domain.UserIDProxyIDConflicts{
		domain.AppspaceID(1): {UserID: domain.UserID(1), ProxyID: "proxy1"},
	}
	mockManageUsers.EXPECT().AppspacesForUser(domain.UserID(1)).Return(testData, nil)

	cache := &AppspaceUsersCache{
		ManageUsers: mockManageUsers,
		userCache:   make(map[domain.UserID]map[domain.AppspaceID]domain.UserIDProxyIDConflicts),
	}

	// First call - cache miss
	_, err := cache.AppspacesForUser(domain.UserID(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second call - cache hit
	result, err := cache.AppspacesForUser(domain.UserID(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 appspace, got %d", len(result))
	}
}

func TestInvalidateForUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockManageUsers := testmocks.NewMockManageUsers(ctrl)

	// Set up expectations - each method should be called twice (before and after invalidation)
	usersData := map[domain.UserID]domain.UserIDProxyIDConflicts{
		domain.UserID(1): {UserID: domain.UserID(1), ProxyID: "proxy1"},
	}
	appspacesData := map[domain.AppspaceID]domain.UserIDProxyIDConflicts{
		domain.AppspaceID(1): {UserID: domain.UserID(1), ProxyID: "proxy1"},
	}
	mockManageUsers.EXPECT().UsersForAppspace(domain.AppspaceID(1)).Return(usersData, nil).Times(2)
	mockManageUsers.EXPECT().AppspacesForUser(domain.UserID(1)).Return(appspacesData, nil).Times(2)

	mockUserAppspacesEvent := testmocks.NewMockUserAppspacesEvent(ctrl)
	mockUserAppspacesEvent.EXPECT().Send(domain.UserID(1)).Times(1)

	cache := &AppspaceUsersCache{
		ManageUsers:        mockManageUsers,
		UserAppspacesEvent: mockUserAppspacesEvent,
		appspaceCache:      make(map[domain.AppspaceID]map[domain.UserID]domain.UserIDProxyIDConflicts),
		userCache:          make(map[domain.UserID]map[domain.AppspaceID]domain.UserIDProxyIDConflicts),
	}

	// Populate caches
	_, _ = cache.UsersForAppspace(domain.AppspaceID(1))
	_, _ = cache.AppspacesForUser(domain.UserID(1))

	// Invalidate for user
	cache.invalidateForUser(domain.UserID(1))

	// Both caches should be invalidated, causing new calls
	_, _ = cache.UsersForAppspace(domain.AppspaceID(1))
	_, _ = cache.AppspacesForUser(domain.UserID(1))
}

func TestInvalidateForAppspace(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockManageUsers := testmocks.NewMockManageUsers(ctrl)

	// Set up expectations - each method should be called twice (before and after invalidation)
	usersData := map[domain.UserID]domain.UserIDProxyIDConflicts{
		domain.UserID(1): {UserID: domain.UserID(1), ProxyID: "proxy1"},
	}
	appspacesData := map[domain.AppspaceID]domain.UserIDProxyIDConflicts{
		domain.AppspaceID(1): {UserID: domain.UserID(1), ProxyID: "proxy1"},
	}
	mockManageUsers.EXPECT().UsersForAppspace(domain.AppspaceID(1)).Return(usersData, nil).Times(2)
	mockManageUsers.EXPECT().AppspacesForUser(domain.UserID(1)).Return(appspacesData, nil).Times(2)

	mockUserAppspacesEvent := testmocks.NewMockUserAppspacesEvent(ctrl)
	mockUserAppspacesEvent.EXPECT().Send(domain.UserID(1)).Times(1)

	cache := &AppspaceUsersCache{
		ManageUsers:        mockManageUsers,
		UserAppspacesEvent: mockUserAppspacesEvent,
		appspaceCache:      make(map[domain.AppspaceID]map[domain.UserID]domain.UserIDProxyIDConflicts),
		userCache:          make(map[domain.UserID]map[domain.AppspaceID]domain.UserIDProxyIDConflicts),
	}

	// Populate caches
	_, _ = cache.UsersForAppspace(domain.AppspaceID(1))
	_, _ = cache.AppspacesForUser(domain.UserID(1))

	// Invalidate for appspace
	cache.invalidateForAppspace(domain.AppspaceID(1))

	// Both caches should be invalidated, causing new calls
	_, _ = cache.UsersForAppspace(domain.AppspaceID(1))
	_, _ = cache.AppspacesForUser(domain.UserID(1))
}
