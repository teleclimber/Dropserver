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

func TestParseCpuTime(t *testing.T) {
	c := &CGroups{}

	str := `usage_usec 1447992
user_usec 1338740
system_usec 109252`
	cpu, err := c.parseCpuTime(str)
	if err != nil {
		t.Error(err)
	}
	if cpu != 1447992 {
		t.Errorf("cpu time does not match: %v", cpu)
	}
}

func TestReadIOStats(t *testing.T) {
	c := &CGroups{}

	str := "8:0 rbytes=4096 wbytes=745472 rios=1 wios=72 dbytes=0 dios=0"
	ioBytes, ioOps, err := c.readIOStats(str)
	if err != nil {
		t.Error(err)
	}
	if ioBytes != 4096+745472 {
		t.Error("Got wrong ioBytes")
	}
	if ioOps != 1+72 {
		t.Error("Got Wrong io ops")
	}
}
