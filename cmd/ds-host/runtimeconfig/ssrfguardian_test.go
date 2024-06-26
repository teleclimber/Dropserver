package runtimeconfig

import (
	"net/netip"
	"testing"

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
	cfg.LocalNetwork.AllowedIPs = make([]string, 0)

	s := GetSSRFGuardian(cfg)

	err := s.Safe("tcp4", "54.84.236.175:443", nil) // 54.84.236.175 is a public IP
	if err != nil {
		t.Error(err)
	}
	err = s.Safe("tcp4", "54.84.236.175:5000", nil)
	if err != nil {
		t.Error(err)
	}
	err = s.Safe("tcp4", "192.168.1.10:443", nil)
	if err == nil {
		t.Error("expected error (prohibited ip)")
	}

	cfg.LocalNetwork.AllowedIPs = []string{"192.168.1.10"}
	s = GetSSRFGuardian(cfg)
	err = s.Safe("tcp4", "192.168.1.10:443", nil)
	if err != nil {
		t.Error(err)
	}
}
