package testmocks

import (
	"io/fs"
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

//go:generate mockgen -destination=views_mocks.go -package=testmocks github.com/teleclimber/DropServer/cmd/ds-host/testmocks Views

type Views interface {
	PrepareTemplates()
	GetStaticFS() fs.FS
	Login(http.ResponseWriter, domain.LoginViewData)
	Signup(http.ResponseWriter, domain.SignupViewData)
}
