package main

// BaseData is the basic app and appspace meta data
type BaseData struct {
	AppPath    string `json:"app_path"`
	AppName    string `json:"app_name"`
	AppVersion string `json:"app_version"`

	// AppSchema is the highest migration directory number in the app source
	AppSchema int `json:"app_schema"`

	AppspacePath string `json:"appspace_path"`

	// Also need AppspaceSchema
	AppspaceSchema int `json:"appspace_schema"`
}
