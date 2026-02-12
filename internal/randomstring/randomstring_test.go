package randomstring

import (
	"testing"
)

const testCharset = "abcdefghijklmnopqrstuvwxyz0123456789"

func TestLength(t *testing.T) {
	for _, length := range []int{1, 6, 10, 32} {
		s := RandomString(length, testCharset)
		if len(s) != length {
			t.Errorf("expected length %d, got %d", length, len(s))
		}
	}
}

func TestValidCharacters(t *testing.T) {
	valid := make(map[byte]bool)
	for i := range []byte(testCharset) {
		valid[testCharset[i]] = true
	}

	s := RandomString(1000, testCharset)
	for i := range []byte(s) {
		if !valid[s[i]] {
			t.Errorf("invalid character %q at position %d", s[i], i)
		}
	}
}

func TestUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for range 100 {
		s := RandomString(10, testCharset)
		if seen[s] {
			t.Errorf("duplicate string %q", s)
		}
		seen[s] = true
	}
}

func TestDistribution(t *testing.T) {
	counts := make(map[byte]int)
	total := 100_000
	s := RandomString(total, testCharset)

	for i := range []byte(s) {
		counts[s[i]]++
	}

	expected := float64(total) / float64(len(testCharset))
	for _, c := range []byte(testCharset) {
		count := float64(counts[c])
		if count < expected*0.8 || count > expected*1.2 {
			t.Errorf("character %q appeared %d times, expected ~%.0f", c, counts[c], expected)
		}
	}
}

func TestRandomStringNoCaps(t *testing.T) {
	s := RandomStringNoCaps(10)
	if len(s) != 10 {
		t.Errorf("expected length 10, got %d", len(s))
	}
	for i := range []byte(s) {
		if s[i] >= 'A' && s[i] <= 'Z' {
			t.Errorf("unexpected uppercase character %q at position %d", s[i], i)
		}
	}
}

func TestRandomStringWithCaps(t *testing.T) {
	s := RandomStringWithCaps(1000)
	if len(s) != 1000 {
		t.Errorf("expected length 1000, got %d", len(s))
	}
	hasUpper := false
	for i := range []byte(s) {
		if s[i] >= 'A' && s[i] <= 'Z' {
			hasUpper = true
			break
		}
	}
	if !hasUpper {
		t.Error("expected at least one uppercase character in 1000-char string")
	}
}
