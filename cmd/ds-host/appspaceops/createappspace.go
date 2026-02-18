package appspaceops

import (
	"errors"
	"io"
	"os"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
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
		Create(domain.AppspaceID) error
	} `checkinject:"required"`
	AppspaceUserModel interface {
		Create(appspaceID domain.AppspaceID, displayName string, avatar string, auths []domain.EditAppspaceUserAuth) (domain.ProxyID, error)
		UpdateAvatar(appspaceID domain.AppspaceID, proxyID domain.ProxyID, avatar string) error
	} `checkinject:"required"`
	Avatars interface {
		Save(locationKey string, proxyID domain.ProxyID, img io.Reader) (string, error)
	} `checkinject:"required"`
	UserDisplayImagesModel interface {
		FilePath(userID domain.UserID, fn string) string
	} `checkinject:"required"`
	UserModel interface {
		GetFromID(domain.UserID) (domain.User, error)
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
func (c *CreateAppspace) Create(ownerID domain.UserID, appVersion domain.AppVersion, baseDomain, subDomain string) (domain.AppspaceID, domain.JobID, error) {

	// Possible race condition here. If you check domain is available then later actually register it.
	// It would be nice if CheckAppspaceDomain also reserved that name temporarily
	check, err := c.DomainController.CheckAppspaceDomain(ownerID, baseDomain, subDomain)
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

	locationKey, err := c.AppspaceFilesModel.CreateLocation()
	if err != nil {
		return domain.AppspaceID(0), domain.JobID(0), err
	}

	user, err := c.UserModel.GetFromID(ownerID)
	if err != nil {
		return domain.AppspaceID(0), domain.JobID(0), err
	}

	inAppspace := domain.Appspace{
		OwnerID:     ownerID,
		AppID:       appVersion.AppID,
		AppVersion:  "",
		DomainName:  fullDomain,
		LocationKey: locationKey,
	}

	appspace, err := c.AppspaceModel.Create(inAppspace)
	if err != nil {
		c.AppspaceFilesModel.DeleteLocation(locationKey)
		return domain.AppspaceID(0), domain.JobID(0), err
	}

	err = c.AppspaceMetaDB.Create(appspace.AppspaceID)
	if err != nil {
		c.AppspaceFilesModel.DeleteLocation(locationKey)
		c.AppspaceModel.Delete(appspace.AppspaceID)
		return domain.AppspaceID(0), domain.JobID(0), err
	}

	// Create owner user
	auths := make([]domain.EditAppspaceUserAuth, 0)
	if user.TSNetIdentifier != "" {
		auths = append(auths, domain.EditAppspaceUserAuth{
			Type:       "tsnetid",
			Identifier: user.TSNetIdentifier,
			ExtraName:  user.TSNetExtraName,
			Operation:  domain.EditOperationAdd})
	}
	// here leverage "instance user" to give the owner access even if they have no auths.

	displayName := "The Owner"
	if user.DisplayName != "" {
		displayName = user.DisplayName
	}

	ownerProxyID, err := c.AppspaceUserModel.Create(appspace.AppspaceID, displayName, "", auths)
	if err != nil {
		return domain.AppspaceID(0), domain.JobID(0), err
	}

	if user.DisplayImage != "" {
		f, err := os.Open(c.UserDisplayImagesModel.FilePath(ownerID, user.DisplayImage))
		if err == nil {
			avatar, err := c.Avatars.Save(appspace.LocationKey, ownerProxyID, f)
			f.Close()
			if err == nil {
				c.AppspaceUserModel.UpdateAvatar(appspace.AppspaceID, ownerProxyID, avatar)
			}
		}
	}
	// Or use an AppspaceUserController to do this consistently

	// migrate to whatever version was selected
	job, err := c.MigrationJobModel.Create(ownerID, appspace.AppspaceID, appVersion.Version, true)
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
