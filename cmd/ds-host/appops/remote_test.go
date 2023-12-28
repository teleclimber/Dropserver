package appops

import (
	"net/netip"
	"testing"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestGetPrefix4Single(t *testing.T) {
	s := "192.168.1.10"
	p := getPrefix(s)
	if !p.Contains(netip.MustParseAddr(s)) {
		t.Error("expected to contain IP: " + s)
	}
	if !p.IsSingleIP() {
		t.Error("expected single IP")
	}
}

func TestGetPrefix4Range(t *testing.T) {
	s := "23.240.71.62/30"
	p := getPrefix(s)
	if !p.Contains(netip.MustParseAddr("23.240.71.61")) {
		t.Error("expected to contain IP: 23.240.71.61")
	}
}

func TestGetPrefix6Single(t *testing.T) {
	s := "2001:db8:85a3::8a2e:370:7334"
	p := getPrefix(s)
	if !p.Contains(netip.MustParseAddr(s)) {
		t.Error("expected to contain IP: " + s)
	}
	if !p.IsSingleIP() {
		t.Error("expected single IP")
	}
}

func TestGetSSRF(t *testing.T) {
	cfg := domain.RuntimeConfig{}
	cfg.InternalNetwork.AllowedIPs = make([]string, 0)
	r := &RemoteAppGetter{
		Config: &cfg,
	}

	s := r.getSSRF()

	err := s.Safe("tcp4", "54.84.236.175:443", nil)
	if err != nil {
		t.Error(err)
	}
	err = s.Safe("tcp4", "54.84.236.175:80", nil)
	if err == nil {
		t.Error("expected error (prohibited port)")
	}
	err = s.Safe("tcp4", "192.168.1.10:443", nil)
	if err == nil {
		t.Error("expected error (prohibited ip)")
	}

	cfg.InternalNetwork.AllowedIPs = []string{"192.168.1.10"}
	s = r.getSSRF()
	err = s.Safe("tcp4", "192.168.1.10:443", nil)
	if err != nil {
		t.Error(err)
	}
}

func TestCacheFresh(t *testing.T) {
	if !cacheFresh(cachedListing{fetchDt: time.Now()}) {
		t.Error("cache for time.Now should be fresh")
	}
	if cacheFresh(cachedListing{fetchDt: time.Now().Add(-cacheDuration).Add(-time.Minute * 5)}) {
		t.Error("cache should not be fresh")
	}
}
