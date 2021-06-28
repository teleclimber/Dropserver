package views

import (
	"embed"
	_ "embed"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

// Views struct handles server-rendered templated views
type Views struct {
	Config *domain.RuntimeConfig `checkinject:"required"`

	base BaseData

	loginTemplate  *template.Template
	signupTemplate *template.Template
}

// BaseData is the basic data that the page needs to render
// Contains things like url prefixes.
// (not currently used)
type BaseData struct{}

//go:embed login.html
var loginTemplateStr string

//go:embed signup.html
var signupTemplateStr string

//go:embed static
var StaticFiles embed.FS

func (v *Views) GetStaticFS() fs.FS {
	staticFS, err := fs.Sub(StaticFiles, "static")
	if err != nil {
		panic(err)
	}
	return staticFS
}

// PrepareTemplates opens the template files and parses them for future use
func (v *Views) PrepareTemplates() {
	v.base = BaseData{}

	v.loginTemplate = template.Must(template.New("login").Parse(loginTemplateStr))
	v.signupTemplate = template.Must(template.New("signup").Parse(signupTemplateStr))
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
		record.NewDsLogger().AddNote("")
		v.getLogger("Login()").Error(err)
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
		v.getLogger("Signup()").Error(err)
		// Too late to send error status. Hopefully the logger is enough.
	}
}

func (v *Views) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("Views")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
