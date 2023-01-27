package appspaceops

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/nfnt/resize"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"

	_ "image/gif"
	"image/jpeg"
	_ "image/png"
)

// !! It could be that a lot of this just ends up in appspace files.
// It seems we're closign towards a simeple read/write/delete file
// Also this could become AppspaceUserOps
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
	appspaceImg, err := a.makeImage(img)
	if err != nil {
		return "", err
	}

	fn, err := a.imageToFile(locationKey, proxyID, appspaceImg)
	if err != nil {
		return "", err
	}

	return fn, nil
}

func (a *Avatars) makeImage(img io.Reader) ([]byte, error) {
	orig, _, err := image.Decode(img)
	if err != nil {
		a.getLogger("makeImage, imageDecode").Log(err.Error()) // Log() not Error() because it's likely a bad input file
		return nil, err                                        // maybe sentinel to point out there was a problem with the image.
	}

	thumb := resize.Thumbnail(100, 100, orig, resize.Bicubic)

	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, thumb, nil)
	if err != nil {
		a.getLogger("makeImage, jpeg.Encode").Error(err)
		return nil, err
	}

	return buf.Bytes(), nil
}

func (a *Avatars) imageToFile(loc string, proxyID domain.ProxyID, img []byte) (string, error) {
	fn := fmt.Sprintf("%s-%s.jpg", proxyID, randomString(6))
	fp := filepath.Join(a.AppspaceLocation2Path.Avatar(loc, fn))

	err := ioutil.WriteFile(fp, img, 0644) // TODO omg permissions!
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

// //////////
// random string stuff
// TODO CRYPTO: this should be using crypto package
const chars61 = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand2 = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func randomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = chars61[seededRand2.Intn(len(chars61))]
	}
	return string(b)
}
