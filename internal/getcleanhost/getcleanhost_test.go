package getcleanhost

import "testing"

func TestGetCleanHost(t *testing.T) {
	cases := []struct {
		hostPort string
		host     string
		er       bool
	}{
		{"abc.def:3000", "abc.def", false},
		{"abc.def:", "abc.def", false},
		{"abc.def", "abc.def", false},
		{"", "", false},
		{"abc.def:xyz", "abc.def", false},
		{"abc.[ def", "abc.[ def", false}, // In the end I'm not sure how to trigger an error in SplitHostPort.
	}

	for _, c := range cases {
		t.Run(c.hostPort, func(t *testing.T) {
			result, err := GetCleanHost(c.hostPort)
			if err != nil && !c.er {
				t.Error(err)
			} else if err == nil && c.er {
				t.Error("expected an error")
			}
			if result != c.host {
				t.Errorf("Expected %v, got %v", c.host, result)
			}
		})
	}
}
