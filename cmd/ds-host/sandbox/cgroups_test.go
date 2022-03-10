package sandbox

import "testing"

func TestGetNewCGroups(t *testing.T) {
	c := &CGroups{
		curCGroups: []string{}}

	cg1 := c.getNewCGroup()

	if len(c.curCGroups) != 1 {
		t.Error("expected 1 cgroup")
	}

	err := c.validateCGroup(cg1)
	if err != nil {
		t.Error(err)
	}

	err = c.removeCurCGroup(cg1)
	if err != nil {
		t.Error(err)
	}

	if len(c.curCGroups) != 0 {
		t.Error("expected 0 cgroup")
	}
}

// unsure how to test the functions that hit the OS.
// The Linux cgroup virtual fs works in ways that differ from "normal" fs,
// (like echo "+cpu" > cgroup.controllers actually sets that file to "cpu", not "+cpu")
// so testing against a dummy fs is pointless.
