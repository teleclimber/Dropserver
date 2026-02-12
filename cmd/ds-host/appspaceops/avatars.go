package appspaceops

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/profilepic"
	"github.com/teleclimber/DropServer/internal/randomstring"
)

// This could become AppspaceUserOps
// And it could handle all changes to appspace user
// .. so that it could come from different places
// And it would presumably take care of the cascade of overrides to set the right stuff in appspace
// Also will trigger notifications of changes so appspace app can regenerate whatever it needs.

type Avatars struct {
	Config                *domain.RuntimeConfig `checkinject:"required"`
	AppspaceLocation2Path interface {
		Avatar(string, string) string
	}
}

// Let's say user uploads an avatar for thier dropid:
// - cut it down to size
// - store it in ds-data/avatars
// - register its existence in drop id table
//   (probably need to delete old one first, or after)
// - propagate the avatar change out to appspaces:
// - find appspaces owned by that user with that drop id.
//   - create avatar file for appspace (maybe smaller...)
//   - save in appspace/avatars with generated name
//   - update appspace user table with new avatar
//   - delete old avatar
//   - [trigger event for appspace (later)]
// - find remote appspaces where that drop id is used
//   - use Ds2DS to tell remote that drop id info has changed
//   - likely remote does the request for new data?
//   - remote proceeeds as above

// Simpler: owner sets and avatar for a user in appspace management:
// - create avatar file for that appspace
// - save in sppspace/avatars with generated name
// - update appspace user table with new avatar
//   - delete old avatar
//   - [trigger event for appspace (later)]

// Save cuts the image down to size and saves it in appspace data dir
// It returns the filename of the image as a string
func (a *Avatars) Save(locationKey string, proxyID domain.ProxyID, img io.Reader) (string, error) {
	appspaceImg, err := profilepic.MakeImage(img)
	if err != nil {
		return "", err
	}

	fn, err := a.imageToFile(locationKey, proxyID, appspaceImg)
	if err != nil {
		return "", err
	}

	return fn, nil
}

func (a *Avatars) imageToFile(loc string, proxyID domain.ProxyID, img []byte) (string, error) {
	fn := fmt.Sprintf("%s-%s.jpg", proxyID, randomstring.RandomStringNoCaps(6))
	fp := filepath.Join(a.AppspaceLocation2Path.Avatar(loc, fn))

	err := os.WriteFile(fp, img, 0644)
	if err != nil {
		a.getLogger("imageToFile, ioutil.WriteFile").Error(err)
		return "", err
	}

	return fn, nil
}

func (a *Avatars) Remove(locationKey string, fn string) error {
	err := os.Remove(a.AppspaceLocation2Path.Avatar(locationKey, fn))
	if err != nil {
		a.getLogger("removeFile").Error(err)
		return err
	}
	return nil
}

func (a *Avatars) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("Avatars")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
