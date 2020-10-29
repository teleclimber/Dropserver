package main

// BaseData is the basic app and appspace meta data
type BaseData struct {
	AppPath      string `json:"app_path"`
	AppspacePath string `json:"appspace_path"`

	// AppspaceSchema is from the appspace meta db
	AppspaceSchema int `json:"appspace_schema"`
}
