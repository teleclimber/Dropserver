//go:generate mockgen -destination=services_mocks.go -package=testmocks -self_package=github.com/teleclimber/DropServer/cmd/ds-host/testmocks github.com/teleclimber/DropServer/cmd/ds-host/testmocks VXServices

package testmocks

import "github.com/teleclimber/DropServer/cmd/ds-host/domain"

// VXServices returns a twine message handler for sandbox services
type VXServices interface {
	Get(appspace *domain.Appspace, api domain.APIVersion) (service domain.ReverseServiceI)
}
