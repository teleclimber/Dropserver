package validator

import "testing"

func TestNormalizeEmail(t *testing.T) {
	cases := []struct {
		in  string
		out string
	}{
		{"abc@def.com", "abc@def.com"},
		{"abc@dÉf.com", "abc@déf.com"},
	}

	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			norm := NormalizeEmail(c.in)
			if norm != c.out {
				t.Errorf("%v != %v", norm, c.out)
			}
		})
	}
}
