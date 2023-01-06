package runtimeconfig

import (
	"fmt"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

type SetupKey struct {
	Config    *domain.RuntimeConfig `checkinject:"required"`
	DBManager interface {
		GetSetupKey() (string, error)
		DeleteSetupKey() error
	} `checkinject:"required"`
	UserModel interface {
		GetAllAdmins() ([]domain.UserID, error)
	} `checkinject:"required"`
	cached bool
	key    string
}

func (k *SetupKey) Has() (bool, error) {
	if !k.cached {
		err := k.loadKey()
		if err != nil {
			return false, err
		}
	}
	return k.key != "", nil
}

func (k *SetupKey) Get() (string, error) {
	if !k.cached {
		err := k.loadKey()
		if err != nil {
			return "", err
		}
	}
	return k.key, nil
}

func (k *SetupKey) Delete() error {
	k.cached = false
	k.key = ""
	return k.DBManager.DeleteSetupKey()
}

func (k *SetupKey) RevealKey() {
	has, err := k.Has()
	if err != nil {
		return
	}
	if has {
		k.getLogger("RevealKey()").Log("Use this link to create an administrator account: setup_key_reveal=" + k.getSecretUrl())
	}
}

func (k *SetupKey) getSecretUrl() string {
	key, err := k.Get()
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s://%s%s/%s", k.Config.ExternalAccess.Scheme, k.Config.Exec.UserRoutesDomain, k.Config.Exec.PortString, key)
}

func (k *SetupKey) loadKey() error {
	k.key = ""
	k.cached = false

	setupKey, err := k.DBManager.GetSetupKey()
	if err != nil {
		return err
	}
	if setupKey != "" {
		// Check if there are any admins on the system.
		// If there are the setup key should not be used.
		// This is off-nominal and should be logged.
		admins, err := k.UserModel.GetAllAdmins()
		if err != nil {
			return err
		}
		if len(admins) != 0 {
			setupKey = ""
			k.getLogger("loadKey()").Log("Found a setup_key while there already are admins. setup_key should have been deleted.")
		}
	}
	k.key = setupKey
	k.cached = true
	return nil
}

func (k *SetupKey) getLogger(note string) *record.DsLogger {
	return record.NewDsLogger("SetupKey", note)
}
