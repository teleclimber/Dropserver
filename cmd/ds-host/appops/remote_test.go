package appops

import (
	"fmt"
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
	cfg.InternalNetwork.AllowedIPs = make([]string, 0)
	r := &RemoteAppGetter{
		Config: &cfg,
	}

	s := r.getSSRF()

	err := s.Safe("tcp4", "54.84.236.175:80", nil)
	if err != nil {
		t.Error(err)
	}
	err = s.Safe("tcp4", "192.168.1.10:80", nil)
	if err == nil {
		t.Error("expected error")
	}

	cfg.InternalNetwork.AllowedIPs = []string{"192.168.1.10"}
	s = r.getSSRF()
	err = s.Safe("tcp4", "192.168.1.10:80", nil)
	if err != nil {
		t.Error(err)
	}
}

func TestURLFromListing(t *testing.T) {
	cases := []struct {
		listingURL string
		base       string
		rel        string
		out        string
	}{{
		listingURL: "site.com/listing.json",
		rel:        "package.tar.gz",
		out:        "site.com/package.tar.gz",
	}, {
		listingURL: "site.com/deep/path/listing.json",
		rel:        "package.tar.gz",
		out:        "site.com/deep/path/package.tar.gz",
	}, {
		listingURL: "site.com/",
		rel:        "package.tar.gz",
		out:        "site.com/package.tar.gz",
	}, {
		listingURL: "site.com",
		base:       "site.com",
		rel:        "package.tar.gz",
		out:        "site.com/package.tar.gz",
	}, {
		listingURL: "https://site.com/listing.json",
		rel:        "package.tar.gz",
		out:        "https://site.com/package.tar.gz",
	}}

	for i, c := range cases {
		t.Run(fmt.Sprintf("case %v", i), func(t *testing.T) {
			result, err := URLFromListing(c.listingURL, c.base, c.rel)
			if err != nil {
				t.Error(err)
			}
			if result != c.out {
				t.Errorf("got %v expected %v", result, c.out)
			}
		})
	}
}

func TestGetLatestVersion(t *testing.T) {
	versions := map[domain.Version]domain.AppListingVersion{}
	versions["0.1.0"] = domain.AppListingVersion{}
	versions["1.2.3"] = domain.AppListingVersion{}
	versions["0.1.1"] = domain.AppListingVersion{}

	v, err := GetLatestVersion(versions)
	if err != nil {
		t.Fatal(err)
	}
	if v != domain.Version("1.2.3") {
		t.Errorf("expected 1.2.3, got %v", v)
	}
}
