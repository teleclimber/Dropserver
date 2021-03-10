package domaincontroller

import "testing"

func TestValidateLabel(t *testing.T) {
	cases := []struct {
		label string
		pass  bool
	}{
		{"abc", true},
		{"abc.def", false},
		{"abcdé", false},
		{"", false},
		{"abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijkl", false},
		{"abc-def", true},
		{"-def", false},
	}

	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			err := validateDomainLabel(c.label)
			if err != nil && c.pass {
				t.Error(err)
			} else if err == nil && !c.pass {
				t.Error("expected fail")
			}
		})
	}
}

func TestValidateSubdmains(t *testing.T) {
	cases := []struct {
		label string
		pass  bool
	}{
		{"abc", true},
		{"abc.def", true},
		{"abcdé.ghi", false},
		{"", false},
		{".", false},
		{".abc", false},
		{"abc.", false},
		{"abc.-.def", false},
	}

	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			err := validateSubdomains(c.label)
			if err != nil && c.pass {
				t.Error(err)
			} else if err == nil && !c.pass {
				t.Error("expected fail")
			}
		})
	}
}
