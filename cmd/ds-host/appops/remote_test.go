package appops

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

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
