package server

import "testing"

func TestDropserverIdentifier(t *testing.T) {
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
			result := dropserverIdentifier(c.id, c.controlURL)
			if result != c.expected {
				t.Errorf("expected: %s, got %s", c.expected, result)
			}
		})
	}
}
