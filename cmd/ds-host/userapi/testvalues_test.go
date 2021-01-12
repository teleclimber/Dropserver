package userapi

import "github.com/teleclimber/DropServer/cmd/ds-host/domain"

// Create values for the api tests to use.
// This alleviates the tedious task of creating reasonable values for all tests.

// Remember values are shared across tests in userapi, so don't change them willy nilly.
// Create a new value if you need it. Make sure strings and ids are entirely unique

// ownerID 7, the main character
var ownerID = domain.UserID(7)

var appID1 = domain.AppID(11)
var app1 = domain.App{
	AppID:   appID1,
	Name:    "app-one",
	OwnerID: ownerID,
}
var appID2 = domain.AppID(12)
var app2 = domain.App{
	AppID:   appID2,
	Name:    "app-two",
	OwnerID: ownerID,
}

var appVersion1 = domain.AppVersion{
	AppID:   appID1,
	AppName: "app-version-one",
	Version: "1.1.1",
}
var appVersion2 = domain.AppVersion{
	AppID:   appID1,
	AppName: "app-version-two",
	Version: "2.2.2",
}
var appVersion3 = domain.AppVersion{
	AppID:   appID2,
	AppName: "app-version-three",
	Version: "3.3.3",
}

var appspace1 = domain.Appspace{
	OwnerID:    ownerID,
	AppspaceID: 21,
	Subdomain:  "subdomain-one",
	AppID:      appID1,
	AppVersion: appVersion1.Version,
}
var appspace2 = domain.Appspace{
	OwnerID:    ownerID,
	AppspaceID: 22,
	Subdomain:  "subdomain-two",
	AppID:      appID1,
	AppVersion: appVersion1.Version,
}

var appspace3 = domain.Appspace{
	OwnerID:    ownerID,
	AppspaceID: 23,
	Subdomain:  "subdomain-three",
	AppID:      appID2,
	AppVersion: appVersion3.Version,
}
