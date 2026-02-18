package userdisplayimagesmodel

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

type UserDisplayImagesModel struct {
	Config *domain.RuntimeConfig `checkinject:"required"`
}

func (m *UserDisplayImagesModel) Save(userID domain.UserID, img io.Reader) (string, error) {
	processedImg, err := profilepic.MakeImage(img)
	if err != nil {
		return "", err
	}

	fn := fmt.Sprintf("%s.jpg", randomstring.RandomStringNoCaps(6))

	dir := m.userDir(userID)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		m.getLogger("Save, os.MkdirAll").Error(err)
		return "", err
	}

	fp := filepath.Join(dir, fn)
	err = os.WriteFile(fp, processedImg, 0644)
	if err != nil {
		m.getLogger("Save, os.WriteFile").Error(err)
		return "", err
	}

	return fn, nil
}

func (m *UserDisplayImagesModel) Remove(userID domain.UserID, fn string) error {
	fp := filepath.Join(m.userDir(userID), fn)
	err := os.Remove(fp)
	if err != nil {
		m.getLogger("Remove").Error(err)
		return err
	}
	return nil
}

func (m *UserDisplayImagesModel) FilePath(userID domain.UserID, fn string) string {
	return filepath.Join(m.userDir(userID), fn)
}

func (m *UserDisplayImagesModel) userDir(userID domain.UserID) string {
	return filepath.Join(m.Config.DataDir, "user-display-images", fmt.Sprintf("user_%d", userID))
}

func (m *UserDisplayImagesModel) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("UserDisplayImagesModel")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
