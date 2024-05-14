//go:generate mockgen -destination=services_mocks.go -package=testmocks -self_package=github.com/teleclimber/DropServer/cmd/ds-host/testmocks github.com/teleclimber/DropServer/cmd/ds-host/testmocks ServiceMaker

package testmocks

import "github.com/teleclimber/DropServer/cmd/ds-host/domain"

// VXServices returns a twine message handler for sandbox services
type ServiceMaker interface {
	Get(appspace *domain.Appspace) (service domain.ReverseServiceI)
}
