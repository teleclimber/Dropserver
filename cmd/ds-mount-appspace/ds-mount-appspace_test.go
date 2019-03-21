package main

import "testing"

func TestValidateInput(t *testing.T) {
	cases := []struct {
		input string
		panic bool
	}{
		{"", true},
		{".", true},
		{"..", true},
		{"/", true},
		{"abc/../def", true},
		{"abc", false},
		{"abc-DEF", false},
		{"abc123", false},
	}

	for _, c := range cases {
		defer func() {
			r := recover()
			if c.panic && r == nil {
				t.Error(c.input + ": should panic but didn't")
			} else if !c.panic && r != nil {
				t.Error(c.input + ": panic though it shouldn't have.")
			}
		}()
		validateInput(c.input)
	}
}
