package views

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestPrepareTemplates(t *testing.T) {
	v := &Views{
		Config: &domain.RuntimeConfig{
			ResourcesDir: "../../../resources"},
	}

	v.PrepareTemplates()
}

func TestLogin(t *testing.T) {
	v := &Views{
		Config: &domain.RuntimeConfig{
			ResourcesDir: "../../../resources"},
	}

	v.PrepareTemplates()

	rr := httptest.NewRecorder()

	v.Login(rr, domain.LoginViewData{})

	bodyStr := rr.Body.String()

	if !strings.Contains(bodyStr, "</html>") {
		t.Error("End of template disappeared from html")
	}
}

func TestLoginMessage(t *testing.T) {
	v := &Views{
		Config: &domain.RuntimeConfig{
			ResourcesDir: "../../../resources"},
	}

	v.PrepareTemplates()

	msgStr := "valkjhavonasv"
	emailStr := "foo@bar.social"

	viewData := domain.LoginViewData{
		Message: msgStr,
		Email:   emailStr}

	rr := httptest.NewRecorder()

	v.Login(rr, viewData)

	bodyStr := rr.Body.String()
	if !strings.Contains(bodyStr, msgStr) {
		t.Error("message didn't make it into html")
	}
	if !strings.Contains(bodyStr, emailStr) {
		t.Error("email didn't make it into html")
	}
	if !strings.Contains(bodyStr, "</html>") {
		t.Error("End of template disappeared from html")
	}
}
