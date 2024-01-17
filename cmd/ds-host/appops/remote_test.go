package appops

import (
	"net/http"
	"net/http/httptest"
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

func TestRedirect(t *testing.T) {
	cases := []struct {
		desc         string
		responseCode int
		err          bool
	}{
		{"status OK", http.StatusOK, false},
		{"permanent redirect", http.StatusPermanentRedirect, false},
		{"temporary redirect", http.StatusTemporaryRedirect, true},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if c.responseCode != http.StatusOK {
					w.Header().Add("Location", "https://abc.com")
				}
				w.WriteHeader(c.responseCode)

			}))
			defer ts.Close()

			client := &http.Client{CheckRedirect: checkRedirect}
			resp, err := client.Get(ts.URL)
			if c.err {
				if err == nil {
					t.Errorf("expected error got %v", resp.StatusCode)
				}
			} else {
				if err != nil {
					t.Error(err)
				}
				if resp.StatusCode != c.responseCode {
					t.Errorf("expected code %v got %v", c.responseCode, resp.StatusCode)
				}
			}
		})
	}
}

func TestIsFresh(t *testing.T) {
	if !isFresh(time.Now()) {
		t.Error("time.Now should be fresh")
	}
	if isFresh(time.Now().Add(-cacheDuration).Add(-time.Minute * 5)) {
		t.Error("cache should not be fresh")
	}
}
