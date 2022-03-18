package domain

import "testing"

func TestNewNullAppspaceID(t *testing.T) {
	a := NewNullAppspaceID()
	if a.Valid {
		t.Error("should not be valid")
	}
	_, ok := a.Get()
	if ok {
		t.Error("expected ok false")
	}

	orig := AppspaceID(123)
	a.Set(orig)
	if !a.Valid {
		t.Error("should be valid")
	}

	ret, ok := a.Get()
	if !ok {
		t.Error("expected ok true")
	}
	if ret != orig {
		t.Error("expected return to be same as original")
	}

	b := NewNullAppspaceID(orig)
	ret, ok = b.Get()
	if !ok {
		t.Error("expected ok")
	}
	if ret != orig {
		t.Error("expcted b ret same as orig")
	}

	b.Unset()
	_, ok = b.Get()
	if ok {
		t.Error("expected ok false after unset")
	}
}

func TestNullAppspaceIDEqual(t *testing.T) {
	a := NewNullAppspaceID()
	b := NewNullAppspaceID()
	if !a.Equal(b) {
		t.Error("expected equal")
	}

	id1 := AppspaceID(123)
	a.Set(id1)
	if a.Equal(b) {
		t.Error("should be different")
	}

	id2 := AppspaceID(456)
	b.Set(id2)
	if a.Equal(b) {
		t.Error("should be different")
	}

	b.Set(id1)
	if !a.Equal(b) {
		t.Error("expected equal")
	}
}
