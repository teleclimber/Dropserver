package appspaceops

import (
	"errors"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/validator"
)

type CreateAppspace struct {
	AppspaceModel interface {
		Create(domain.Appspace) (*domain.Appspace, error)
		Delete(domain.AppspaceID) error
	} `checkinject:"required"`
	AppspaceFilesModel interface {
		CreateLocation() (string, error)
		DeleteLocation(string) error
	} `checkinject:"required"`
	AppspaceMetaDB interface {
		Create(domain.AppspaceID, int) error
	} `checkinject:"required"`
	AppspaceUserModel interface {
		Create(appspaceID domain.AppspaceID, authType string, authID string) (domain.ProxyID, error)
	} `checkinject:"required"`
	DomainController interface {
		CheckAppspaceDomain(userID domain.UserID, dom string, subdomain string) (domain.DomainCheckResult, error)
		StartManaging(dom string) error
	} `checkinject:"required"`
	MigrationJobModel interface {
		Create(domain.UserID, domain.AppspaceID, domain.Version, bool) (*domain.MigrationJob, error)
	} `checkinject:"required"`
	MigrationJobController interface {
		WakeUp()
	} `checkinject:"required"`
}

// Shouldn't we also have appspace status in here?

// Possible errors:
// - domain not available

// Create a new appspace
func (c *CreateAppspace) Create(dropID domain.DropID, appVersion domain.AppVersion, baseDomain, subDomain string) (domain.AppspaceID, domain.JobID, error) {

	// Possible race condition here. If you check domain is available then later actually register it.
	// It would be nice if CheckAppspaceDomain also reserved that name temporarily
	check, err := c.DomainController.CheckAppspaceDomain(dropID.UserID, baseDomain, subDomain)
	if err != nil {
		return domain.AppspaceID(0), domain.JobID(0), err
	}
	if !check.Valid || !check.Available { // validitiy should be checked at route handler?
		return domain.AppspaceID(0), domain.JobID(0), errors.New("domain invalid or unavailable") // this is unhelpful.
	}

	fullDomain := baseDomain
	if subDomain != "" {
		fullDomain = subDomain + "." + baseDomain
	}

	dropIDStr := validator.JoinDropID(dropID.Handle, dropID.Domain)

	locationKey, err := c.AppspaceFilesModel.CreateLocation()
	if err != nil {
		return domain.AppspaceID(0), domain.JobID(0), err
	}

	inAppspace := domain.Appspace{
		OwnerID:     dropID.UserID,
		AppID:       appVersion.AppID,
		AppVersion:  "",
		DomainName:  fullDomain,
		DropID:      dropIDStr,
		LocationKey: locationKey,
	}

	appspace, err := c.AppspaceModel.Create(inAppspace)
	if err != nil {
		c.AppspaceFilesModel.DeleteLocation(locationKey)
		return domain.AppspaceID(0), domain.JobID(0), err
	}

	err = c.AppspaceMetaDB.Create(appspace.AppspaceID, 0) // 0 is the ds api version
	if err != nil {
		c.AppspaceFilesModel.DeleteLocation(locationKey)
		c.AppspaceModel.Delete(appspace.AppspaceID)
		return domain.AppspaceID(0), domain.JobID(0), err
	}

	// Create owner user
	_, err = c.AppspaceUserModel.Create(appspace.AppspaceID, "dropid", dropIDStr)
	if err != nil {
		return domain.AppspaceID(0), domain.JobID(0), err
	}
	// TODO use whatver process that sets values of display name and avatar to set those for owner user
	// Or use an AppspaceUserController to do this consistently

	// migrate to whatever version was selected
	job, err := c.MigrationJobModel.Create(dropID.UserID, appspace.AppspaceID, appVersion.Version, true)
	if err != nil {
		return domain.AppspaceID(0), domain.JobID(0), err
	}

	c.MigrationJobController.WakeUp()

	err = c.DomainController.StartManaging(fullDomain)
	if err != nil {
		return domain.AppspaceID(0), domain.JobID(0), err
	}

	return appspace.AppspaceID, job.JobID, nil
}
