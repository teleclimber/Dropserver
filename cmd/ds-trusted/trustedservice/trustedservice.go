package trustedservice

import (
	"fmt"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-trusted/trusteddomain"
)

// put stuff in dirs
// get stuff from dirs
// list stuff in dirs...

// TrustedAPI is just a type
type TrustedAPI struct {
	AppFiles trusteddomain.AppFilesI
}

// SaveAppFiles saves application files to ds-trusted
func (t *TrustedAPI) SaveAppFiles(args *domain.TrustedSaveAppFiles, reply *domain.TrustedSaveAppFilesReply) error {
	locationKey, err := t.AppFiles.Save(args)
	if err != nil {
		return err.ToStandard()
	}

	reply.LocationKey = locationKey

	return nil
}

// GetAppMeta returns metadata gleaned from examining files at specified location
func (t *TrustedAPI) GetAppMeta(args *domain.TrustedGetAppMeta, reply *domain.TrustedGetAppMetaReply) (err error) {
	// args.LocationKey should map to directory
	// read application.json
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Panic in GetAppMeta: %v", r)
		}
	}()

	metadata, dsErr := t.AppFiles.ReadMeta(args.LocationKey)
	if dsErr != nil {
		err = dsErr.ToStandard()
		return
	}

	reply.AppFilesMetadata = *metadata

	return
}



