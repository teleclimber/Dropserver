package main

import "github.com/teleclimber/DropServer/cmd/ds-host/domain"

// DevAppspaceStatus is a non op dud for now
type DevAppspaceStatus struct {
}

// Ready always returns true for now
func (s *DevAppspaceStatus) Ready(appspaceID domain.AppspaceID) bool {
	// temporary. Useful to return false while sandbox is restarting
	// Also look at schema versions to ensure the app can run the apspace?
	return true
}
