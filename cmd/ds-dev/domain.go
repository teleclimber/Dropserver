package main

import "github.com/teleclimber/DropServer/cmd/ds-host/domain"

// BaseData is the basic app and appspace meta data
type BaseData struct {
	AppPath      string `json:"app_path"`
	AppspacePath string `json:"appspace_path"`
}

type SandboxStatus struct {
	Type   string               `json:"type"`
	Status domain.SandboxStatus `json:"status"`
}

// DevAppspaceUser represents a user and is intended to be independent of DS API version
// iow it might be a union of all props of the vxUsers
type DevAppspaceUser struct {
	ProxyID     domain.ProxyID `json:"proxy_id"`
	DisplayName string         `json:"display_name"`
	Permissions []string       `json:"permissions"`
}
