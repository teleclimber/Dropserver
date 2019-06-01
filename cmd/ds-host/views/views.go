package views

import (
	"fmt"
	"html/template"
	"net/http"
	"path"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// Views struct handles server-rendered templated views
type Views struct {
	Config *domain.RuntimeConfig
	Logger domain.LogCLientI

	base BaseData

	loginTemplate  *template.Template
	signupTemplate *template.Template
}

// BaseData is the basic data that the page needs to render
// Contains things like url prefixes.
type BaseData struct {
	PublicStaticPrefix string
}

// PrepareTemplates opens the template files and parses them for future use
func (v *Views) PrepareTemplates() {
	prefix := "//static." + v.Config.Server.Host
	port := v.Config.Server.Port
	if port != 80 && port != 443 {
		prefix += fmt.Sprintf(":%d", port)
	}
	v.base = BaseData{
		PublicStaticPrefix: prefix}

	// templates:
	templatesPath := path.Join(v.Config.ResourcesDir, "go-templates/")

	v.loginTemplate = template.Must(template.ParseFiles(path.Join(templatesPath, "login.html")))
	v.signupTemplate = template.Must(template.ParseFiles(path.Join(templatesPath, "signup.html")))
}

type loginData struct {
	BaseData
	LoginViewData domain.LoginViewData
}

// Login executes the login template and sends it down as a response?
func (v *Views) Login(res http.ResponseWriter, viewData domain.LoginViewData) {
	d := loginData{
		BaseData:      v.base,
		LoginViewData: viewData}

	err := v.loginTemplate.Execute(res, d)
	if err != nil {
		v.Logger.Log(domain.ERROR, nil, err.Error())
		// Too late to send error status. Hopefully the logger is enough.
	}
}

// Signup..
type signupData struct {
	BaseData
	SignupViewData domain.SignupViewData
}

// Signup presents the signup (account registration) page
func (v *Views) Signup(res http.ResponseWriter, viewData domain.SignupViewData) {
	d := signupData{
		BaseData:       v.base,
		SignupViewData: viewData}

	err := v.signupTemplate.Execute(res, d)
	if err != nil {
		v.Logger.Log(domain.ERROR, nil, err.Error())
		// Too late to send error status. Hopefully the logger is enough.
	}
}
