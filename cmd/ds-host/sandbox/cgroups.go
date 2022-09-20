package sandbox

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

var hostCGroup = "host"
var sandboxesCGroup = "sandboxes"

type CGroups struct {
	Config *domain.RuntimeConfig `checkinject:"required"`

	rootCGroupPath string

	idMux      sync.Mutex
	nextID     int
	curCGroups []string
}

// Init creates a CGroup for the host and prepares to create cgroups
// for the sandboxes
func (c *CGroups) Init() error {
	c.curCGroups = []string{}

	// try to figure out what cgroup we're in.
	err := c.initRootCgroupPath()
	if err != nil {
		return err
	}
	c.getLogger("Init").Log("Root CGroup path: " + c.rootCGroupPath)

	// remove all subgroups (may be left over from previous run)
	files, err := ioutil.ReadDir(c.rootCGroupPath)
	if err != nil {
		c.getLogger("Init").Error(err)
		return err
	}

	for _, f := range files {
		if f.IsDir() && f.Name() != "." && f.Name() != ".." {
			err = os.RemoveAll(filepath.Join(c.rootCGroupPath, f.Name()))
			if err != nil {
				c.getLogger("Init").AddNote("remove existing cgroups").Error(err)
				return err // log it fist
			}
		}
	}

	// then reset subtree_control to allow creation of subdir.
	err = c.setSubtreeControl("", []string{})
	if err != nil {
		return err
	}

	// then need to create a new subgroup
	// is this even possible? bc ds is running in the cgroup that we are putting a subgroup under
	// ->yes it's possible. proc can be in branch while no controllers are in subtree_control
	err = os.MkdirAll(filepath.Join(c.rootCGroupPath, hostCGroup), 0755)
	if err != nil {
		c.getLogger("Init").AddNote("make host cgroup dir").Error(err)
		return err
	}

	// then move self over.
	err = c.addPidToSubGroup(os.Getpid(), hostCGroup)
	if err != nil {
		return err
	}

	// then create the sandboxes cgroup
	err = os.MkdirAll(filepath.Join(c.rootCGroupPath, sandboxesCGroup), 0755)
	if err != nil {
		c.getLogger("Init").AddNote("make sandboxes cgroup dir").Error(err)
		return err
	}

	// then populate subtree_control at root and sandboxes
	for _, p := range []string{"", sandboxesCGroup} {
		err = c.setSubtreeControl(p, []string{"memory", "io"})
		if err != nil {
			return err
		}
	}

	err = c.SetSandboxesMemoryHigh(c.Config.Sandbox.MemoryHighMb * 1024 * 1024)
	if err != nil {
		return err
	}

	// Here maybe monitor memory pressure on sandbox cgroup
	// for now just log it when you get an event.
	go c.epollMemoryPressure()

	return nil
}

//CreateCGroup creates a cgroup for a sandbox
func (c *CGroups) CreateCGroup(limits domain.CGroupLimits) (string, error) {
	cGroup := c.getNewCGroup()
	p := filepath.Join(c.rootCGroupPath, sandboxesCGroup, cGroup)

	l := c.getLogger(fmt.Sprintf("CreateCGroup: %v", cGroup))
	l.Log("creating cgroup")

	err := os.MkdirAll(p, 0755)
	if err != nil {
		l.AddNote("make cgroup dir at " + p).Error(err)
		return "", err
	}

	ctls := []struct {
		controller string
		value      string
	}{
		//{controller: "cpu.weight", value: "100"},
		{controller: "memory.high", value: fmt.Sprintf("%v", limits.MemoryHigh)},
	}
	for _, ctl := range ctls {
		err = c.setController(filepath.Join(sandboxesCGroup, cGroup, ctl.controller), ctl.value)
		if err != nil {
			return "", err
		}
	}
	return cGroup, nil
}

func (c *CGroups) SetLimits(cGroup string, limits domain.CGroupLimits) error {
	err := c.validateCGroup(cGroup)
	if err != nil {
		return err
	}
	ctls := []struct {
		controller string
		value      string
	}{
		{controller: "memory.high", value: fmt.Sprintf("%v", limits.MemoryHigh)},
	}
	for _, ctl := range ctls {
		err = c.setController(filepath.Join(sandboxesCGroup, cGroup, ctl.controller), ctl.value)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *CGroups) setController(subPath string, val string) error {
	l := c.getLogger(fmt.Sprintf("setController: %v -> %v", subPath, val))
	file, err := os.OpenFile(filepath.Join(c.rootCGroupPath, subPath), os.O_WRONLY, 0644)
	if err != nil {
		l.AddNote("os.OpenFile").Error(err)
		return err
	}
	_, err = file.WriteString(val)
	file.Close()
	if err != nil {
		l.AddNote("file.WriteString").Error(err)
		return err
	}
	return nil
}

func (c *CGroups) AddPid(cGroup string, pid int) error {
	err := c.validateCGroup(cGroup)
	if err != nil {
		return err
	}
	return c.addPidToSubGroup(pid, filepath.Join(sandboxesCGroup, cGroup))
}

func (c *CGroups) addPidToSubGroup(pid int, subPath string) error {
	l := c.getLogger(fmt.Sprintf("addPidToSubGroup cgroup: %v", subPath))
	l.Log("adding pid")

	p := filepath.Join(c.rootCGroupPath, subPath, "cgroup.procs")
	file, err := os.OpenFile(p, os.O_WRONLY, 0644)
	if err != nil {
		l.AddNote("OpenFile").Error(err)
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("%v\n", pid))
	file.Close()
	if err != nil {
		l.AddNote("WriteString").Error(err)
		return err
	}
	return nil
}

func (c *CGroups) setSubtreeControl(subPath string, controllers []string) error {
	ctrl := ""
	for _, c := range controllers {
		ctrl += "+" + c + " "
	}
	l := c.getLogger(fmt.Sprintf("setSubtreeControl at: %v, num controllers: %v", subPath, ctrl))
	l.Log("setting subtree_control")

	p := filepath.Join(c.rootCGroupPath, subPath, "cgroup.subtree_control")
	file, err := os.OpenFile(p, os.O_WRONLY, 0644)
	if err != nil {
		l.AddNote("OpenFile: " + p).Error(err)
		return err
	}
	_, err = file.WriteString(ctrl)
	file.Close()
	if err != nil {
		l.AddNote("WriteString " + p).Error(err)
		return err
	}
	return nil
}

func (c *CGroups) getSandboxMemoryHigh() int {
	return 128 * 1024 * 1024
}

func (c *CGroups) GetMetrics(cGroup string) (data domain.CGroupData, err error) {
	data.MemoryBytes = c.getSandboxMemoryHigh()

	err = c.validateCGroup(cGroup)
	if err != nil {
		return
	}

	cpuStr, err := c.readFile(filepath.Join(cGroup, "cpu.stat"))
	if err != nil {
		return
	}
	cpuTime, err := c.parseCpuTime(cpuStr)
	if err != nil {
		return
	}
	data.CpuUsec = cpuTime

	ioStats, err := c.readFile(filepath.Join(cGroup, "io.stat"))
	if err != nil {
		return
	}
	ioBytes, ioOps, err := c.readIOStats(ioStats)
	if err != nil {
		return
	}
	data.IOBytes = ioBytes
	data.IOs = ioOps

	return
}

func (c *CGroups) parseCpuTime(cpuStr string) (int, error) {
	cpuLines := strings.Split(cpuStr, "\n")
	// if len(cpuLines) != 3 {
	// 	err = errors.New("expected cpu.stat to be 3 lines long, got: " + cpuStr)
	// 	c.getLogger("GetMetrics").Error(err)
	// 	return 0, err
	// }
	if !strings.HasPrefix(cpuLines[0], "usage_usec ") {
		err := errors.New("cpu.stat start of first line is not 'usage_usec ': " + cpuStr)
		c.getLogger("parseCpuTime").Error(err)
		return 0, err
	}
	microSec, err := strconv.Atoi(strings.TrimPrefix(cpuLines[0], "usage_usec "))
	if err != nil {
		c.getLogger("parseCpuTime() strconv.Atoi").Error(err)
		return 0, err
	}
	return microSec, nil
}
func (c *CGroups) readIOStats(ioStats string) (ioBytes int, ioOps int, err error) {
	lines := strings.Split(ioStats, "\n")
	for _, line := range lines {
		pairs := strings.Split(line, " ")
		for _, p := range pairs[1:] {
			parts := strings.Split(p, "=")
			if len(parts) != 2 {
				err = fmt.Errorf("invalid pair %v", p)
				c.getLogger("readIOStats()").Error(err)
				return
			}
			var val int
			val, err = strconv.Atoi(parts[1])
			if err != nil {
				c.getLogger("readIOStats() strconv.Atoi").AddNote(p).Error(err)
				return
			}
			switch parts[0] {
			case "rbytes", "wbytes":
				ioBytes += val
			case "rios", "wios":
				ioOps += val
			}
		}
	}
	return
}

func (c *CGroups) readFile(subPath string) (string, error) {
	p := filepath.Join(c.rootCGroupPath, sandboxesCGroup, subPath)
	cpuBytes, err := ioutil.ReadFile(p)
	if err != nil {
		c.getLogger("readFile() ReadFile() error").AddNote(fmt.Sprintf("subPath: %v", subPath)).Error(err)
		return "", err
	}
	return string(cpuBytes), nil
}

func (c *CGroups) RemoveCGroup(cGroup string) error {
	err := c.validateCGroup(cGroup)
	if err != nil {
		return err
	}

	p := filepath.Join(c.rootCGroupPath, sandboxesCGroup, cGroup)

	l := c.getLogger(fmt.Sprintf("RemoveCGroup: %v", cGroup))
	l.Log("removing cgroup")

	err = os.Remove(p)
	if err != nil {
		l.AddNote("os.Remove() dir: " + p).Error(err)
		return err
	}

	l.Log("cgroup dir removed")

	// remove cgroup
	err = c.removeCurCGroup(cGroup)
	if err != nil {
		return err
	}

	return nil
}

// Borrowed from https://github.com/elastic/gosigar/blob/v0.14.2/cgroup/util.go#L230 (Apache 2.0 license)
// Modified by Olivier Forget
// getRootCgroupPath returns the path of the cgroup to which this process belongs
func (c *CGroups) initRootCgroupPath() error {
	p := ""

	cgroup, err := os.Open(filepath.Join("/proc", strconv.Itoa(os.Getpid()), "cgroup"))
	if err != nil {
		return err
	}
	defer cgroup.Close()

	sc := bufio.NewScanner(cgroup)
	for sc.Scan() {
		// http://man7.org/linux/man-pages/man7/cgroups.7.html
		// Format: hierarchy-ID:subsystem-list:cgroup-path
		// Example:
		// 2:cpu:/docker/b29faf21b7eff959f64b4192c34d5d67a707fe8561e9eaa608cb27693fba4242
		// in v2 hiearchy id is always 0 and meaningless.
		// in v2 subsystem-list is empty and meaningless

		line := sc.Text()

		fields := strings.Split(line, ":")
		if len(fields) != 3 {
			continue
		}

		if p != "" {
			// we alreay found a path. Why is there more than one on cgroup v2?
			c.getLogger("getRootCgroupPath").Log(fmt.Sprintf("Already have a path (%v), found another: %v", p, fields[2]))
			return errors.New("found more than one cgroup path")
		}

		p = fields[2]
	}
	p = strings.TrimPrefix(p, "/")

	c.rootCGroupPath = filepath.Join(c.Config.Sandbox.CGroupMount, p)

	return sc.Err()
}

type pressure struct {
	some map[string]int
	full map[string]int
}

func (c *CGroups) readMemoryPressure() {
	memData, err := ioutil.ReadFile(filepath.Join(c.rootCGroupPath, sandboxesCGroup, "memory.pressure"))
	if err != nil {
		c.getLogger("monitorMemoryPressure() ReadFile memory pressure").Error(err)
		return
	}

	lines := strings.Split(string(memData), "\n")
	c.getLogger("monitorMemoryPressure()").Log(lines[0] + " " + lines[1])

	// memData:
	// some avg10=0.00 avg60=0.00 avg300=0.00 total=0
	// full avg10=0.00 avg60=0.00 avg300=0.00 total=0
	// lines :=  strings.Split(string(memData), "\n");
	// if len(lines) != 2 {
	// 	c.getLogger("monitorMemoryPressure()").Log(fmt.Sprintf("Split returned %v lines", len(lines)))
	// 	return
	// }

	// for _, l := range lines {
	// 	head, values, err := parsePressureLine(l)
	// 	if err != nil {
	// 		c.getLogger("monitorMemoryPressure()").AddNote("line: "+l).Error(err)
	// 	}
	// 	var different bool
	// 	if head == "some" {
	// 		different, err = comparePressures(c.pressure.some, values)
	// 	} else if head == "full" {
	// 		different, err = comparePressures(c.pressure.full, values)
	// 	} else {
	// 		c.getLogger("monitorMemoryPressure()").Log("head not recognized: "+head+" "+l)
	// 		return
	// 	}
	// 	if err != nil {
	// 		c.getLogger("monitorMemoryPressure()").AddNote("line: "+l).Error(err)
	// 	}
	// 	if different {

	// 	}
	// }

}

const (
	EPOLLET        = 1 << 31
	MaxEpollEvents = 32
)

func (c *CGroups) epollMemoryPressure() {
	var event syscall.EpollEvent
	var events [MaxEpollEvents]syscall.EpollEvent

	pFile, err := os.OpenFile(filepath.Join(c.rootCGroupPath, sandboxesCGroup, "memory.pressure"), os.O_RDWR, 0000)
	if err != nil {
		c.getLogger("epollMemoryPressure() os.Open memory pressure").Error(err)
		return
	}
	defer pFile.Close()

	fd := int(pFile.Fd())

	sub := []byte("some 1000 500000")

	_, err = syscall.Write(fd, sub)
	if err != nil {
		c.getLogger("epollMemoryPressure() syscall.Write sub").Error(err)
		return
	}

	epfd, err := syscall.EpollCreate1(0)
	if err != nil {
		c.getLogger("epollMemoryPressure() EpollCreate1").Error(err)
		return
	}
	defer syscall.Close(epfd)

	event.Events = syscall.EPOLLPRI
	event.Fd = int32(fd)
	err = syscall.EpollCtl(epfd, syscall.EPOLL_CTL_ADD, fd, &event)
	if err != nil {
		c.getLogger("epollMemoryPressure() EpollCtl").Error(err)
		return
	}

	//increased := false

	for {
		nevents, err := syscall.EpollWait(epfd, events[:], -1)
		if err != nil {
			c.getLogger("epollMemoryPressure() EpollWaits").Error(err)
			break
			// Got this error once: interrupted system call
			// Apparently you just have to keep reading EPollWaits
			// So maybe if it's EINTR, continue instead of break
		}

		//c.getLogger("epollMemoryPressure()").Log()
		//c.getLogger("epollMemoryPressure()").Log(fmt.Sprintf("Got events: %v", nevents))

		for ev := 0; ev < nevents; ev++ {
			c.getLogger("epollMemoryPressure()").Log(fmt.Sprintf("Event: %+v", events[ev]))
			c.readMemoryPressure()

			// Old experiment to squeeze and relieve pressure on all sandboxes:
			// if !increased {
			// 	c.SetSandboxesMemoryHigh(memHigh)
			// 	increased = true
			// 	go func() {
			// 		time.Sleep(time.Second)
			// 		c.SetSandboxesMemoryHigh(memSqueeze)
			// 		increased = false
			// 	}()
			// }

			//if int(events[ev].Fd) == fd && {
			// 	connFd, _, err := syscall.Accept(fd)
			// 	if err != nil {
			// 		fmt.Println("accept: ", err)
			// 		continue
			// 	}
			// 	syscall.SetNonblock(fd, true)
			// 	event.Events = syscall.EPOLLIN | EPOLLET
			// 	event.Fd = int32(connFd)
			// 	if err := syscall.EpollCtl(epfd, syscall.EPOLL_CTL_ADD, connFd, &event); err != nil {
			// 		fmt.Print("epoll_ctl: ", connFd, err)
			// 		os.Exit(1)
			// 	}
			// } else {
			// 	go echo(int(events[ev].Fd))
			// }
		}

	}
}

// SetSandboxesMemoryHigh sets memory.high for all sandbox parent cgroup
// memHigh is in bytes
func (c *CGroups) SetSandboxesMemoryHigh(memHigh int) error {
	ctls := []struct {
		controller string
		value      string
	}{
		{controller: "memory.high", value: fmt.Sprintf("%v", memHigh)},
	}
	for _, ctl := range ctls {
		err := c.setController(filepath.Join(sandboxesCGroup, ctl.controller), ctl.value)
		if err != nil {
			c.getLogger("setSandboxesMemoryHigh() setController").Error(err)
			return err
		}
	}
	return nil
}

func parsePressureLine(str string) (head string, values map[string]int, err error) {
	values = make(map[string]int)
	pieces := strings.Split(str, " ")
	for i, p := range pieces {
		if i == 0 {
			head = p
			continue
		}
		pair := strings.Split(p, "=")
		intVal, e := strconv.Atoi(pair[1])
		if e != nil {
			err = e
		}
		values[pair[0]] = intVal
	}
	return
}
func comparePressures(a map[string]int, b map[string]int) (bool, error) {
	for k, aVal := range a {
		bVal, ok := b[k]
		if !ok {
			return true, errors.New("key not found in b: " + k)
		}
		if aVal != bVal {
			return true, nil
		}
	}
	return false, nil
}

func (c *CGroups) getNewCGroup() string {
	c.idMux.Lock()
	defer c.idMux.Unlock()
	c.nextID++
	cGroup := fmt.Sprintf("sandbox-%v", c.nextID)
	c.curCGroups = append(c.curCGroups, cGroup)
	return cGroup
}
func (c *CGroups) validateCGroup(cGroup string) error {
	c.idMux.Lock()
	defer c.idMux.Unlock()
	for _, g := range c.curCGroups {
		if g == cGroup {
			return nil
		}
	}
	err := errors.New("cgroup not found in current cgroups: " + cGroup)
	c.getLogger("validateCGroup").Error(err)
	return err
}
func (c *CGroups) removeCurCGroup(cGroup string) error {
	c.idMux.Lock()
	defer c.idMux.Unlock()
	for i, g := range c.curCGroups {
		if g == cGroup {
			c.curCGroups = append(c.curCGroups[:i], c.curCGroups[i+1:]...)
			c.getLogger("removeCurCGroup").Log("cgroup removed")
			return nil
		}
	}
	err := errors.New("cgroup not found in current cgroups: " + cGroup)
	c.getLogger("removeCurCGroup").Error(err)
	return err
}
func (c *CGroups) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("CGroups")
	if note != "" {
		l.AddNote(note)
	}
	return l
}
