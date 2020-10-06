package main

// BaseData is the basic app and appspace meta data
type BaseData struct {
	AppPath      string `json:"app_path"`
	AppspacePath string `json:"appspace_path"`

	AppName       string `json:"app_name"`
	AppVersion    string `json:"app_version"`
	AppMigrations []int  `json:"app_migrations"`

	// AppSchema is the highest migration directory number in the app source
	AppSchema int `json:"app_version_schema"`

	// AppspaceSchema is from the appspace meta db
	AppspaceSchema int `json:"appspace_schema"`
}
