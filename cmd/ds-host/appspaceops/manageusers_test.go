package appspaceops

import (
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestFindByAuth(t *testing.T) {
	_, found := findByAuth("abc", "123", []domain.AppspaceUser{})
	if found {
		t.Error("expected found to be false")
	}

	users := []domain.AppspaceUser{
		{
			AppspaceID: 7,
			ProxyID:    "proxy123",
			Auths: []domain.AppspaceUserAuth{
				{
					Type:       "dropid",
					Identifier: "dropid456",
				},
			},
		},
	}
	_, found = findByAuth("dropid", "bogus", users)
	if found {
		t.Error("expected found to be false")
	}
	user, found := findByAuth("dropid", "dropid456", users)
	if !found {
		t.Error("expected found to be true")
	}
	if user.ProxyID != "proxy123" {
		t.Error("got wrong user", user)
	}
}

func TestGetDisplayNameFromTSNetUser(t *testing.T) {
	result := getDisplayNameFromTSNetUser(domain.TSNetPeerUser{
		DisplayName: "The Display Name",
		LoginName:   "The Login Name"})
	if result != "The Display Name" {
		t.Errorf("Got %s", result)
	}
	result = getDisplayNameFromTSNetUser(domain.TSNetPeerUser{
		DisplayName: "",
		LoginName:   "The Login Name"})
	if result != "The Login Name" {
		t.Errorf("Got %s", result)
	}
	result = getDisplayNameFromTSNetUser(domain.TSNetPeerUser{
		DisplayName: "A very Long Display Name to get past 30 characters",
		LoginName:   "The Login Name"})
	if result != "A very Long Display Name to ge" {
		t.Errorf("Got %s", result)
	}
	result = getDisplayNameFromTSNetUser(domain.TSNetPeerUser{
		DisplayName: "",
		LoginName:   ""})
	if result != "" {
		t.Errorf("Got %s", result)
	}
}

func TestPairs_Init(t *testing.T) {
	var p pairs[domain.UserID, domain.ProxyID]
	p.init()

	if p.associations == nil {
		t.Error("expected associations to be initialized")
	}
	if len(p.associations) != 0 {
		t.Error("expected associations to be empty")
	}
}

func TestPairs_AddSingle(t *testing.T) {
	var p pairs[domain.UserID, domain.ProxyID]
	p.init()

	u1 := domain.UserID(1)
	pA := domain.ProxyID("proxy-a")

	p.add(u1, pA)

	if len(p.associations) != 1 {
		t.Errorf("expected 1 key in associations, got %d", len(p.associations))
	}

	if _, ok := p.associations[u1][pA]; !ok {
		t.Error("expected proxy-a to be associated with user 1")
	}
}

func TestPairs_AddMultipleDifferentKeys(t *testing.T) {
	var p pairs[domain.UserID, domain.ProxyID]
	p.init()

	u1 := domain.UserID(1)
	u2 := domain.UserID(2)
	u3 := domain.UserID(3)
	pA := domain.ProxyID("proxy-a")
	pB := domain.ProxyID("proxy-b")
	pC := domain.ProxyID("proxy-c")

	p.add(u1, pA)
	p.add(u2, pB)
	p.add(u3, pC)

	if len(p.associations) != 3 {
		t.Errorf("expected 3 keys in associations, got %d", len(p.associations))
	}

	if _, ok := p.associations[u1][pA]; !ok {
		t.Error("expected proxy-a to be associated with user 1")
	}
	if _, ok := p.associations[u2][pB]; !ok {
		t.Error("expected proxy-b to be associated with user 2")
	}
	if _, ok := p.associations[u3][pC]; !ok {
		t.Error("expected proxy-c to be associated with user 3")
	}
}

func TestPairs_AddSameKeyMultipleValues(t *testing.T) {
	var p pairs[domain.UserID, domain.ProxyID]
	p.init()

	u1 := domain.UserID(1)
	pA := domain.ProxyID("proxy-a")
	pB := domain.ProxyID("proxy-b")
	pC := domain.ProxyID("proxy-c")

	p.add(u1, pA)
	p.add(u1, pB)
	p.add(u1, pC)

	if len(p.associations) != 1 {
		t.Errorf("expected 1 key in associations, got %d", len(p.associations))
	}

	userAssocs := p.associations[u1]
	if len(userAssocs) != 3 {
		t.Errorf("expected user 1 to have 3 associations, got %d", len(userAssocs))
	}

	if _, ok := userAssocs[pA]; !ok {
		t.Error("expected proxy-a to be associated with user 1")
	}
	if _, ok := userAssocs[pB]; !ok {
		t.Error("expected proxy-b to be associated with user 1")
	}
	if _, ok := userAssocs[pC]; !ok {
		t.Error("expected proxy-c to be associated with user 1")
	}
}

func TestPairs_AddDuplicateValue(t *testing.T) {
	var p pairs[domain.UserID, domain.ProxyID]
	p.init()

	u1 := domain.UserID(1)
	pA := domain.ProxyID("proxy-a")

	p.add(u1, pA)
	p.add(u1, pA)
	p.add(u1, pA)

	if len(p.associations) != 1 {
		t.Errorf("expected 1 key in associations, got %d", len(p.associations))
	}

	userAssocs := p.associations[u1]
	if len(userAssocs) != 1 {
		t.Errorf("expected user 1 to have 1 association (duplicates ignored), got %d", len(userAssocs))
	}

	if _, ok := userAssocs[pA]; !ok {
		t.Error("expected proxy-a to be associated with user 1")
	}
}

func TestPairs_GetSingleV_NoKey(t *testing.T) {
	var p pairs[domain.UserID, domain.ProxyID]
	p.init()

	u999 := domain.UserID(999)
	result := p.getSingleV(u999)

	if result != domain.ProxyID("") {
		t.Errorf("expected empty ProxyID for non-existent key, got %v", result)
	}
}

func TestPairs_GetSingleV_SingleValue(t *testing.T) {
	var p pairs[domain.UserID, domain.ProxyID]
	p.init()

	u1 := domain.UserID(1)
	pA := domain.ProxyID("proxy-a")

	p.add(u1, pA)
	result := p.getSingleV(u1)

	if result != pA {
		t.Errorf("expected proxy-a, got %v", result)
	}
}

func TestPairs_GetSingleV_MultipleValues(t *testing.T) {
	var p pairs[domain.UserID, domain.ProxyID]
	p.init()

	u1 := domain.UserID(1)
	pA := domain.ProxyID("proxy-a")
	pB := domain.ProxyID("proxy-b")
	pC := domain.ProxyID("proxy-c")

	p.add(u1, pA)
	p.add(u1, pB)
	p.add(u1, pC)

	result := p.getSingleV(u1)

	// Should return one of the values (arbitrary which one due to map iteration)
	if result != pA && result != pB && result != pC {
		t.Errorf("expected one of the proxy values, got %v", result)
	}
}

func TestPairs_GetAllV_NoKey(t *testing.T) {
	var p pairs[domain.UserID, domain.ProxyID]
	p.init()

	u999 := domain.UserID(999)
	result := p.getAllV(u999)

	if result == nil {
		t.Error("expected non-nil slice")
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice for non-existent key, got %d items", len(result))
	}
}

func TestPairs_GetAllV_SingleValue(t *testing.T) {
	var p pairs[domain.UserID, domain.ProxyID]
	p.init()

	u1 := domain.UserID(1)
	pA := domain.ProxyID("proxy-a")

	p.add(u1, pA)
	result := p.getAllV(u1)

	if len(result) != 1 {
		t.Errorf("expected 1 value, got %d", len(result))
	}
	if result[0] != pA {
		t.Errorf("expected proxy-a, got %v", result[0])
	}
}

func TestPairs_GetAllV_MultipleValues(t *testing.T) {
	var p pairs[domain.UserID, domain.ProxyID]
	p.init()

	u1 := domain.UserID(1)
	pA := domain.ProxyID("proxy-a")
	pB := domain.ProxyID("proxy-b")
	pC := domain.ProxyID("proxy-c")

	p.add(u1, pA)
	p.add(u1, pB)
	p.add(u1, pC)

	result := p.getAllV(u1)

	if len(result) != 3 {
		t.Errorf("expected 3 values, got %d", len(result))
	}

	// Check all expected values are present
	found := make(map[domain.ProxyID]bool)
	for _, v := range result {
		found[v] = true
	}

	if !found[pA] {
		t.Error("expected proxy-a in results")
	}
	if !found[pB] {
		t.Error("expected proxy-b in results")
	}
	if !found[pC] {
		t.Error("expected proxy-c in results")
	}
}

func TestPairs_ReverseDirection(t *testing.T) {
	// Test pairs in the reverse direction (ProxyID -> UserID)
	var p pairs[domain.ProxyID, domain.UserID]
	p.init()

	pA := domain.ProxyID("proxy-a")
	pB := domain.ProxyID("proxy-b")
	u1 := domain.UserID(1)
	u2 := domain.UserID(2)
	u3 := domain.UserID(3)

	p.add(pA, u1)
	p.add(pA, u2)
	p.add(pB, u3)

	if len(p.associations) != 2 {
		t.Errorf("expected 2 keys in associations, got %d", len(p.associations))
	}

	proxyAAssocs := p.associations[pA]
	if len(proxyAAssocs) != 2 {
		t.Errorf("expected proxy-a to have 2 user associations, got %d", len(proxyAAssocs))
	}

	users := p.getAllV(pA)
	if len(users) != 2 {
		t.Errorf("expected 2 users for proxy-a, got %d", len(users))
	}
}

func TestPairAuths_AddAuth_Single(t *testing.T) {
	pa := makePairAuths()

	u1 := domain.UserID(1)
	pA := domain.ProxyID("proxy-a")
	auth1 := domain.AppspaceUserAuthBare{Type: "dropid", Identifier: "user1@example.com"}

	pa.addAuth(pA, u1, auth1)

	auths := pa.getAuths(pA, u1)
	if len(auths) != 1 {
		t.Errorf("expected 1 auth, got %d", len(auths))
	}
	if auths[0].Type != "dropid" {
		t.Errorf("expected type 'dropid', got %s", auths[0].Type)
	}
	if auths[0].Identifier != "user1@example.com" {
		t.Errorf("expected identifier 'user1@example.com', got %s", auths[0].Identifier)
	}
}

func TestPairAuths_AddAuth_MultipleForSamePair(t *testing.T) {
	pa := makePairAuths()

	u1 := domain.UserID(1)
	pA := domain.ProxyID("proxy-a")
	auth1 := domain.AppspaceUserAuthBare{Type: "dropid", Identifier: "user1@example.com"}
	auth2 := domain.AppspaceUserAuthBare{Type: "tsnetid", Identifier: "tsnet-user1"}

	pa.addAuth(pA, u1, auth1)
	pa.addAuth(pA, u1, auth2)

	auths := pa.getAuths(pA, u1)
	if len(auths) != 2 {
		t.Errorf("expected 2 auths, got %d", len(auths))
	}

	// Check both auths are present
	foundDropID := false
	foundTSNetID := false
	for _, auth := range auths {
		if auth.Type == "dropid" && auth.Identifier == "user1@example.com" {
			foundDropID = true
		}
		if auth.Type == "tsnetid" && auth.Identifier == "tsnet-user1" {
			foundTSNetID = true
		}
	}

	if !foundDropID {
		t.Error("expected to find dropid auth")
	}
	if !foundTSNetID {
		t.Error("expected to find tsnetid auth")
	}
}

func TestPairAuths_AddAuth_DifferentPairs(t *testing.T) {
	pa := makePairAuths()

	u1 := domain.UserID(1)
	u2 := domain.UserID(2)
	pA := domain.ProxyID("proxy-a")
	pB := domain.ProxyID("proxy-b")
	auth1 := domain.AppspaceUserAuthBare{Type: "dropid", Identifier: "user1@example.com"}
	auth2 := domain.AppspaceUserAuthBare{Type: "dropid", Identifier: "user2@example.com"}

	pa.addAuth(pA, u1, auth1)
	pa.addAuth(pB, u2, auth2)

	auths1 := pa.getAuths(pA, u1)
	if len(auths1) != 1 {
		t.Errorf("expected 1 auth for pA/u1, got %d", len(auths1))
	}
	if auths1[0].Identifier != "user1@example.com" {
		t.Errorf("wrong identifier for pA/u1: %s", auths1[0].Identifier)
	}

	auths2 := pa.getAuths(pB, u2)
	if len(auths2) != 1 {
		t.Errorf("expected 1 auth for pB/u2, got %d", len(auths2))
	}
	if auths2[0].Identifier != "user2@example.com" {
		t.Errorf("wrong identifier for pB/u2: %s", auths2[0].Identifier)
	}
}

func TestPairAuths_GetAuths_NonExistent(t *testing.T) {
	pa := makePairAuths()

	u999 := domain.UserID(999)
	p999 := domain.ProxyID("proxy-999")

	auths := pa.getAuths(p999, u999)
	if auths != nil {
		t.Errorf("expected nil for non-existent pair, got %v", auths)
	}
}

func TestPairAuths_AddInstanceRelation_Single(t *testing.T) {
	pa := makePairAuths()

	u1 := domain.UserID(1)
	pA := domain.ProxyID("proxy-a")

	pa.addInstanceRelation(pA, u1)

	hasInstance := pa.getInstanceRelation(pA, u1)
	if !hasInstance {
		t.Error("expected instance relation to be true")
	}
}

func TestPairAuths_AddInstanceRelation_Multiple(t *testing.T) {
	pa := makePairAuths()

	u1 := domain.UserID(1)
	u2 := domain.UserID(2)
	pA := domain.ProxyID("proxy-a")
	pB := domain.ProxyID("proxy-b")

	pa.addInstanceRelation(pA, u1)
	pa.addInstanceRelation(pB, u2)

	if !pa.getInstanceRelation(pA, u1) {
		t.Error("expected instance relation pA/u1 to be true")
	}
	if !pa.getInstanceRelation(pB, u2) {
		t.Error("expected instance relation pB/u2 to be true")
	}
}

func TestPairAuths_GetInstanceRelation_NonExistent(t *testing.T) {
	pa := makePairAuths()

	u999 := domain.UserID(999)
	p999 := domain.ProxyID("proxy-999")

	hasInstance := pa.getInstanceRelation(p999, u999)
	if hasInstance {
		t.Error("expected instance relation to be false for non-existent pair")
	}
}

func TestPairAuths_GetMatchedOn_OnlyAuths(t *testing.T) {
	pa := makePairAuths()

	u1 := domain.UserID(1)
	pA := domain.ProxyID("proxy-a")
	auth1 := domain.AppspaceUserAuthBare{Type: "dropid", Identifier: "user1@example.com"}

	pa.addAuth(pA, u1, auth1)

	matched := pa.getMatchedOn(pA, u1)

	if matched.Instance {
		t.Error("expected Instance to be false")
	}
	if len(matched.Auths) != 1 {
		t.Errorf("expected 1 auth, got %d", len(matched.Auths))
	}
	if matched.Auths[0].Identifier != "user1@example.com" {
		t.Errorf("wrong auth identifier: %s", matched.Auths[0].Identifier)
	}
}

func TestPairAuths_GetMatchedOn_OnlyInstance(t *testing.T) {
	pa := makePairAuths()

	u1 := domain.UserID(1)
	pA := domain.ProxyID("proxy-a")

	pa.addInstanceRelation(pA, u1)

	matched := pa.getMatchedOn(pA, u1)

	if !matched.Instance {
		t.Error("expected Instance to be true")
	}
	if matched.Auths != nil {
		t.Errorf("expected Auths to be nil, got %v", matched.Auths)
	}
}

func TestPairAuths_GetMatchedOn_BothAuthsAndInstance(t *testing.T) {
	pa := makePairAuths()

	u1 := domain.UserID(1)
	pA := domain.ProxyID("proxy-a")
	auth1 := domain.AppspaceUserAuthBare{Type: "dropid", Identifier: "user1@example.com"}
	auth2 := domain.AppspaceUserAuthBare{Type: "tsnetid", Identifier: "tsnet-user1"}

	pa.addAuth(pA, u1, auth1)
	pa.addAuth(pA, u1, auth2)
	pa.addInstanceRelation(pA, u1)

	matched := pa.getMatchedOn(pA, u1)

	if !matched.Instance {
		t.Error("expected Instance to be true")
	}
	if len(matched.Auths) != 2 {
		t.Errorf("expected 2 auths, got %d", len(matched.Auths))
	}
}

func TestPairAuths_GetMatchedOn_NonExistent(t *testing.T) {
	pa := makePairAuths()

	u999 := domain.UserID(999)
	p999 := domain.ProxyID("proxy-999")

	matched := pa.getMatchedOn(p999, u999)

	if matched.Instance {
		t.Error("expected Instance to be false for non-existent pair")
	}
	if matched.Auths != nil {
		t.Errorf("expected Auths to be nil for non-existent pair, got %v", matched.Auths)
	}
}

func TestIdConflicts_GetProxyIDsConflicts_NoConflicts(t *testing.T) {
	// Test clean case: one user maps to one proxy
	var ic idConflicts
	ic.init()

	u1 := domain.UserID(1)
	u2 := domain.UserID(2)
	pA := domain.ProxyID("proxy-a")
	pB := domain.ProxyID("proxy-b")

	auth1 := domain.AppspaceUserAuthBare{Type: "dropid", Identifier: "user1@example.com"}
	auth2 := domain.AppspaceUserAuthBare{Type: "dropid", Identifier: "user2@example.com"}

	ic.addAuth(pA, u1, auth1)
	ic.addAuth(pB, u2, auth2)

	result := ic.getProxyIDsConflicts()

	if len(result) != 2 {
		t.Errorf("expected 2 proxy IDs in result, got %d", len(result))
	}

	// Check proxy-a
	if confA, ok := result[pA]; !ok {
		t.Error("expected proxy-a in result")
	} else {
		if confA.UserID != u1 {
			t.Errorf("expected UserID %v for proxy-a, got %v", u1, confA.UserID)
		}
		if confA.ProxyID != pA {
			t.Errorf("expected ProxyID %v for proxy-a, got %v", pA, confA.ProxyID)
		}
		if confA.Conflict {
			t.Error("expected Conflict to be false for proxy-a")
		}
		if len(confA.UserIDMatches) != 1 {
			t.Errorf("expected 1 user in UserIDMatches for proxy-a, got %d", len(confA.UserIDMatches))
		}
		if match, ok := confA.UserIDMatches[u1]; !ok {
			t.Error("expected u1 in UserIDMatches for proxy-a")
		} else {
			if len(match.Auths) != 1 {
				t.Errorf("expected 1 auth for u1, got %d", len(match.Auths))
			}
			if match.Instance {
				t.Error("expected Instance to be false for u1")
			}
		}
		if len(confA.ProxyIDMatches) != 1 {
			t.Errorf("expected 1 proxy in ProxyIDMatches, got %d", len(confA.ProxyIDMatches))
		}
	}

	// Check proxy-b
	if confB, ok := result[pB]; !ok {
		t.Error("expected proxy-b in result")
	} else {
		if confB.UserID != u2 {
			t.Errorf("expected UserID %v for proxy-b, got %v", u2, confB.UserID)
		}
		if confB.ProxyID != pB {
			t.Errorf("expected ProxyID %v for proxy-b, got %v", pB, confB.ProxyID)
		}
		if confB.Conflict {
			t.Error("expected Conflict to be false for proxy-b")
		}
		if len(confB.UserIDMatches) != 1 {
			t.Errorf("expected 1 user in UserIDMatches for proxy-b, got %d", len(confB.UserIDMatches))
		}
	}
}

func TestIdConflicts_GetProxyIDsConflicts_OneProxyMultipleUsers(t *testing.T) {
	// Test conflict: one proxy maps to multiple users
	var ic idConflicts
	ic.init()

	u1 := domain.UserID(1)
	u2 := domain.UserID(2)
	u3 := domain.UserID(3)
	pA := domain.ProxyID("proxy-a")

	auth1 := domain.AppspaceUserAuthBare{Type: "dropid", Identifier: "user1@example.com"}
	auth2 := domain.AppspaceUserAuthBare{Type: "dropid", Identifier: "user2@example.com"}
	auth3 := domain.AppspaceUserAuthBare{Type: "dropid", Identifier: "user3@example.com"}

	ic.addAuth(pA, u1, auth1)
	ic.addAuth(pA, u2, auth2)
	ic.addAuth(pA, u3, auth3)

	result := ic.getProxyIDsConflicts()

	if len(result) != 1 {
		t.Errorf("expected 1 proxy ID in result, got %d", len(result))
	}

	confA, ok := result[pA]
	if !ok {
		t.Fatal("expected proxy-a in result")
	}

	// Should have a conflict because one proxy maps to multiple users
	if !confA.Conflict {
		t.Error("expected Conflict to be true")
	}

	// UserIDMatches should contain all three users
	if len(confA.UserIDMatches) != 3 {
		t.Errorf("expected 3 user IDs in UserIDMatches, got %d", len(confA.UserIDMatches))
	}

	// Check all users are present
	if _, ok := confA.UserIDMatches[u1]; !ok {
		t.Error("expected u1 in UserIDMatches")
	}
	if _, ok := confA.UserIDMatches[u2]; !ok {
		t.Error("expected u2 in UserIDMatches")
	}
	if _, ok := confA.UserIDMatches[u3]; !ok {
		t.Error("expected u3 in UserIDMatches")
	}

	// Check that each user has their auth
	if len(confA.UserIDMatches[u1].Auths) != 1 {
		t.Errorf("expected 1 auth for u1, got %d", len(confA.UserIDMatches[u1].Auths))
	}
}

func TestIdConflicts_GetProxyIDsConflicts_OneUserMultipleProxies(t *testing.T) {
	// Test conflict: one user maps to multiple proxies
	var ic idConflicts
	ic.init()

	u1 := domain.UserID(1)
	pA := domain.ProxyID("proxy-a")
	pB := domain.ProxyID("proxy-b")
	pC := domain.ProxyID("proxy-c")

	authA := domain.AppspaceUserAuthBare{Type: "dropid", Identifier: "user1@example.com"}
	authB := domain.AppspaceUserAuthBare{Type: "tsnetid", Identifier: "tsnet-user1"}

	ic.addAuth(pA, u1, authA)
	ic.addAuth(pB, u1, authB)
	ic.addAuth(pC, u1, authA)

	result := ic.getProxyIDsConflicts()

	if len(result) != 3 {
		t.Errorf("expected 3 proxy IDs in result, got %d", len(result))
	}

	// All three proxies should have Conflict = true because u1 matches multiple proxies
	for _, pid := range []domain.ProxyID{pA, pB, pC} {
		conf, ok := result[pid]
		if !ok {
			t.Errorf("expected %v in result", pid)
			continue
		}

		// When there's a conflict (user matches multiple proxies), UserID and ProxyID are zero
		if !conf.Conflict {
			t.Errorf("expected Conflict to be true for %v", pid)
		}

		if len(conf.ProxyIDMatches) != 3 {
			t.Errorf("expected 3 proxies in ProxyIDMatches for %v (user matches multiple proxies), got %d", pid, len(conf.ProxyIDMatches))
		}

		if len(conf.UserIDMatches) != 1 {
			t.Errorf("expected 1 user in UserIDMatches for %v, got %d", pid, len(conf.UserIDMatches))
		}
	}
}

func TestIdConflicts_GetProxyIDsConflicts_MixedScenario(t *testing.T) {
	// Test scenario with conflicts in BOTH directions:
	// - u2 maps to both pB and pC (one user, multiple proxies)
	// - pC also maps to both u2 and u3 (one proxy, multiple users)
	// So pC has conflicts in both directions
	var ic idConflicts
	ic.init()

	u1 := domain.UserID(1)
	u2 := domain.UserID(2)
	u3 := domain.UserID(3)
	pA := domain.ProxyID("proxy-a")
	pB := domain.ProxyID("proxy-b")
	pC := domain.ProxyID("proxy-c")

	auth1 := domain.AppspaceUserAuthBare{Type: "dropid", Identifier: "user1@example.com"}
	auth2 := domain.AppspaceUserAuthBare{Type: "dropid", Identifier: "user2@example.com"}
	auth3 := domain.AppspaceUserAuthBare{Type: "dropid", Identifier: "user3@example.com"}

	// Clean: u1 -> pA
	ic.addAuth(pA, u1, auth1)

	// u2 -> pB and pC (one user, multiple proxies)
	ic.addAuth(pB, u2, auth2)
	ic.addAuth(pC, u2, auth2)

	// pC also gets u3 (so pC has both u2 and u3 - conflict in other direction)
	ic.addAuth(pC, u3, auth3)

	result := ic.getProxyIDsConflicts()

	if len(result) != 3 {
		t.Errorf("expected 3 proxy IDs in result, got %d", len(result))
	}

	// Check pA (clean)
	if confA, ok := result[pA]; !ok {
		t.Error("expected proxy-a in result")
	} else {
		if confA.UserID != u1 {
			t.Errorf("expected UserID %v for proxy-a, got %v", u1, confA.UserID)
		}
		if confA.Conflict {
			t.Error("expected Conflict to be false for proxy-a")
		}
		if len(confA.UserIDMatches) != 1 {
			t.Errorf("expected 1 user in UserIDMatches for proxy-a, got %d", len(confA.UserIDMatches))
		}
		if len(confA.ProxyIDMatches) != 1 {
			t.Errorf("expected 1 proxy in ProxyIDMatches for proxy-a, got %d", len(confA.ProxyIDMatches))
		}
	}

	// Check pB (u2 has multiple proxies, pB has only one user)
	// Since u2 maps to multiple proxies, there's a conflict
	if confB, ok := result[pB]; !ok {
		t.Error("expected proxy-b in result")
	} else {
		// UserID and ProxyID are zero when there's a conflict
		if !confB.Conflict {
			t.Error("expected Conflict to be true for proxy-b (u2 matches multiple proxies)")
		}
		if len(confB.UserIDMatches) != 1 {
			t.Errorf("expected 1 user in UserIDMatches for proxy-b, got %d", len(confB.UserIDMatches))
		}
		if len(confB.ProxyIDMatches) != 2 {
			t.Errorf("expected 2 proxies in ProxyIDMatches for proxy-b (u2 matches pB and pC), got %d", len(confB.ProxyIDMatches))
		}
	}

	// Check pC (conflict in BOTH directions)
	// - pC has multiple users (u2 and u3)
	// - u2 (one of those users) also matches multiple proxies (pB and pC)
	if confC, ok := result[pC]; !ok {
		t.Error("expected proxy-c in result")
	} else {
		// Should have a conflict
		if !confC.Conflict {
			t.Error("expected Conflict to be true for proxy-c")
		}
		if len(confC.UserIDMatches) != 2 {
			t.Errorf("expected 2 user IDs in UserIDMatches for proxy-c, got %d", len(confC.UserIDMatches))
		}
		// Verify both users are in UserIDMatches
		if _, ok := confC.UserIDMatches[u2]; !ok {
			t.Error("expected u2 in UserIDMatches for proxy-c")
		}
		if _, ok := confC.UserIDMatches[u3]; !ok {
			t.Error("expected u3 in UserIDMatches for proxy-c")
		}
		// Check that each user has their auth
		if len(confC.UserIDMatches[u2].Auths) != 1 {
			t.Errorf("expected 1 auth for u2, got %d", len(confC.UserIDMatches[u2].Auths))
		}
		if len(confC.UserIDMatches[u3].Auths) != 1 {
			t.Errorf("expected 1 auth for u3, got %d", len(confC.UserIDMatches[u3].Auths))
		}
	}
}

func TestIdConflicts_GetProxyIDsConflicts_Empty(t *testing.T) {
	// Test empty case
	var ic idConflicts
	ic.init()

	result := ic.getProxyIDsConflicts()

	if len(result) != 0 {
		t.Errorf("expected empty result, got %d items", len(result))
	}
}
