package views

import (
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

	loginTemplate    *template.Template
	signupTemplate   *template.Template
	userHomeTemplate *template.Template
}

// BaseData is the basic data that the page needs to render
// Contains things like url prefixes.
type BaseData struct {
	PublicStaticPrefix string
	JSAPIURLVar        string
}

// PrepareTemplates opens the template files and parses them for future use
func (v *Views) PrepareTemplates() {

	v.base = BaseData{
		PublicStaticPrefix: v.Config.Exec.PublicStaticAddress,
		JSAPIURLVar:        v.Config.Exec.UserRoutesAddress}

	templatePath := path.Join(v.Config.Exec.GoTemplatesDir, "login.html")
	v.loginTemplate = template.Must(template.ParseFiles(templatePath))

	templatePath = path.Join(v.Config.Exec.GoTemplatesDir, "signup.html")
	v.signupTemplate = template.Must(template.ParseFiles(templatePath))

	// Now need to do pages that are webpack generated
	templatePath = path.Join(v.Config.Exec.WebpackTemplatesDir, "user.html")
	v.userHomeTemplate = template.Must(template.ParseFiles(templatePath))

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

// user page
type userHomeData struct {
	BaseData
	//LoginViewData domain.LoginViewData
}

// UserHome executes the login template and sends it down as a response?
func (v *Views) UserHome(res http.ResponseWriter) {
	d := userHomeData{
		BaseData: v.base}

	err := v.userHomeTemplate.Execute(res, d)
	if err != nil {
		v.Logger.Log(domain.ERROR, nil, err.Error())
		// Too late to send error status. Hopefully the logger is enough.
	}
}
