//go:build !linux

package sandbox

import (
	"errors"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

var errNoSupport = errors.New("this build does not support cgroups")

type CGroups struct {
	Config *domain.RuntimeConfig `checkinject:"required"`
}

func (c *CGroups) Init() error {
	return errNoSupport
}

func (c *CGroups) CreateCGroup(domain.CGroupLimits) (string, error) {
	return "", errNoSupport
}
func (c *CGroups) AddPid(string, int) error {
	return errNoSupport
}
func (c *CGroups) SetLimits(string, domain.CGroupLimits) error {
	return errNoSupport
}
func (c *CGroups) GetMetrics(string) (domain.CGroupData, error) {
	return domain.CGroupData{}, errNoSupport
}
func (c *CGroups) RemoveCGroup(string) error {
	return errNoSupport
}
