package server

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"tailscale.com/ipn/ipnstate"
)

func TestHasFiles(t *testing.T) {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	n := &TSNetNode{
		tsnetDir: filepath.Join(dir, "blah")}

	has, err := n.hasFiles()
	if err != nil {
		t.Error(err)
	}
	if has {
		t.Error("expected false")
	}

	n.tsnetDir = dir
	has, err = n.hasFiles()
	if err != nil {
		t.Error(err)
	}
	if has {
		t.Error("expected false")
	}

	err = os.WriteFile(filepath.Join(dir, "bar.txt"), make([]byte, 10), 0666)
	if err != nil {
		t.Fatal(err)
	}
	has, err = n.hasFiles()
	if err != nil {
		t.Error(err)
	}
	if !has {
		t.Error("expected true")
	}
}

func TestTransitory(t *testing.T) {
	n := &TSNetNode{}
	n.setTransitory(transitoryConnect)
	if n.transitoryState != transitoryConnect {
		t.Error("expected transitoryConnect")
	}
	n.unsetTransitory()
	if n.transitoryState != transitoryNone {
		t.Error("expected transitoryNone")
	}
	n.setTransitory(transitoryDisconnect)
	if n.transitoryState != transitoryDisconnect {
		t.Error("expected transitoryDisconnect")
	}
	ret := n.setTransitory(transitoryConnect)
	if ret {
		t.Error("expected false since we can't change from disconnect to connect")
	}
	if n.transitoryState != transitoryDisconnect {
		t.Error("expected transitoryDisconnect")
	}
}

func TestFullIdentifier(t *testing.T) {
	cases := []struct {
		id         string
		controlURL string
		expected   string
	}{{
		id:         "userid:123",
		controlURL: "example.com",
		expected:   "123@example.com",
	}, {
		id:         "123",
		controlURL: "example.com",
		expected:   "123@example.com",
	}, {
		id:         "userid:456",
		controlURL: "",
		expected:   "456@tailscale.com",
	}}
	for _, c := range cases {
		t.Run(c.id+c.controlURL, func(t *testing.T) {
			result := fullIdentifier(c.id, c.controlURL)
			if result != c.expected {
				t.Errorf("expected: %s, got %s", c.expected, result)
			}
		})
	}
}

func TestIngestLCStatus(t *testing.T) {
	n := tsNodeStatus{}
	if n.ingestLCStatus(nil) {
		t.Error("expected false")
	}

	lcStatus := ipnstate.Status{
		CurrentTailnet: &ipnstate.TailnetStatus{
			MagicDNSEnabled: true},
		Self: &ipnstate.PeerStatus{},
	}

	n = tsNodeStatus{}
	if !n.ingestLCStatus(&lcStatus) {
		t.Error("expected true")
	}
	if !n.magicDNS {
		t.Error("expected magic dns to be true")
	}
}

func TestIngestKeyExpiry(t *testing.T) {
	t1, _ := time.Parse(time.RFC3339, "2025-02-17T15:04:05Z")
	t1clone, _ := time.Parse(time.RFC3339, "2025-02-17T15:04:05Z")
	t2, _ := time.Parse(time.RFC3339, "2025-04-05T11:27:55Z")
	cases := []struct {
		cur      *time.Time
		incoming *time.Time
		result   bool
	}{{
		cur:      nil,
		incoming: nil,
		result:   false,
	}, {
		cur:      nil,
		incoming: &t1,
		result:   true,
	}, {
		cur:      &t1,
		incoming: &t2,
		result:   true,
	}, {
		cur:      &t1,
		incoming: nil,
		result:   true,
	}, {
		cur:      &t1,
		incoming: &t1clone,
		result:   false,
	}}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%v %v", c.cur, c.incoming), func(t *testing.T) {
			lcStatus := ipnstate.Status{
				Self: &ipnstate.PeerStatus{
					KeyExpiry: c.incoming,
				},
			}
			n := tsNodeStatus{keyExpiry: c.cur}
			result := n.ingestLCStatus(&lcStatus)
			if result != c.result {
				t.Errorf("expected result %v got %v", c.result, result)
			}
			if (n.keyExpiry == nil && c.incoming != nil) ||
				(n.keyExpiry != nil && c.incoming == nil) {
				t.Errorf("got different values for incoming and resulting: %v, %v", n.keyExpiry, c.incoming)
			}
			if n.keyExpiry != nil && c.incoming != nil && !n.keyExpiry.Equal(*c.incoming) {
				t.Errorf("expected %v to be set, got %v", c.incoming, n.keyExpiry)
			}
		})
	}
}
