package runtimeconfig

import (
	"bytes"
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestLoadDefault(t *testing.T) {
	rtc := loadDefault()

	if rtc.Server.TLSPort != 5050 {
		t.Error("port didn't register correctly. Expected 5050")
	}
	if rtc.Log != "" {
		t.Error("Expected empty log")
	}
	if rtc.ManageTLSCertificates.Enable {
		t.Error("expected cert management to be off")
	}
	if rtc.Prometheus.Enable {
		t.Error("Expected prometheus to not be enabled")
	}
}

func TestMergeLocal(t *testing.T) {
	rtc := loadDefault()

	var localJSON = bytes.NewReader([]byte(`{
		"server": {
			"tls-port": 3999
		}
	}`))

	mergeLocal(rtc, localJSON)

	if rtc.Server.TLSPort != 3999 {
		t.Error("port wasn't overriden by local config. Expected 3999")
	}
}

func TestValidateHost(t *testing.T) {
	rtc := getPassingDefault()
	tv(t, rtc, "default", false)

	rtc = getPassingDefault()
	cases := []struct {
		host        string
		shouldPanic bool
	}{
		{"", true},
		{"abc.def", false},
		{"//abc.def", true},
		{"abc.def/", true},
		{".abc.def", true},
		{" .abc.def", true},
		{"abc.def.", true},
		{"abc.def:3000", true},
		{"10.255.5.11", true},
	}
	for _, c := range cases {
		rtc.Server.Host = c.host
		tv(t, rtc, "server host: "+c.host, c.shouldPanic)
	}
}

func TestValidateServerTLS(t *testing.T) {
	rtc := getPassingDefault()
	tv(t, rtc, "no-tls:true and no certs", false)

	rtc.Server.NoTLS = false
	tv(t, rtc, "no-tls: false and no certs", true)

	rtc.Server.SslCert = "some.crt"
	rtc.Server.SslKey = "the.key"
	tv(t, rtc, "no-tls: false with certs", false)

	rtc.Server.NoTLS = true
	tv(t, rtc, "no-tls: true with certs", true)

	rtc.ManageTLSCertificates.Enable = true
	rtc.ManageTLSCertificates.Email = "a@b.c"
	tv(t, rtc, "no-tls: true with certs and managed", true)

	rtc.Server.SslCert = ""
	rtc.Server.SslKey = ""
	tv(t, rtc, "no-tls: true no certs and managed", true)

	rtc.Server.NoTLS = false
	tv(t, rtc, "no-tls: false no certs and managed", false)

	if rtc.Server.HTTPPort != 80 {
		t.Error("Expected HTTPPort to default to 80")
	}
}

func TestValidateCertManageEnable(t *testing.T) {
	rtc := getPassingDefault()
	rtc.Server.NoTLS = false
	rtc.ManageTLSCertificates.Enable = true
	tv(t, rtc, "no-tls:false and enabled, no email", true)

	rtc.ManageTLSCertificates.Email = "a@b.c"
	tv(t, rtc, "no-tls:false and enabled", false)

}

func TestSetExec(t *testing.T) {
	rtc := getPassingDefault()
	rtc.Server.Host = "somedomain.com"
	rtc.Subdomains.UserAccounts = "user-accounts"
	setExec(rtc)
	assertEqStr(t, "/a/b/c/sandbox-code", rtc.Exec.SandboxCodePath)
	assertEqStr(t, "/a/b/c/apps", rtc.Exec.AppsPath)
	assertEqStr(t, "/a/b/c/appspaces", rtc.Exec.AppspacesPath)
	assertEqStr(t, "/a/b/c/certificates", rtc.Exec.CertificatesPath)
	assertEqStr(t, "user-accounts.somedomain.com", rtc.Exec.UserRoutesDomain)

	rtc.Subdomains.UserAccounts = ""
	setExec(rtc)
	assertEqStr(t, "somedomain.com", rtc.Exec.UserRoutesDomain)
}

func getPassingDefault() *domain.RuntimeConfig {
	rtc := loadDefault()
	rtc.DataDir = "/a/b/c"
	rtc.Sandbox.SocketsDir = "/d/e/f"
	rtc.Server.NoTLS = true
	return rtc
}
func tv(t *testing.T, rtc *domain.RuntimeConfig, hintStr string, shouldPanic bool) {
	defer func() {
		r := recover()
		if shouldPanic && r == nil {
			t.Error(hintStr + ": should panic but didn't")
		} else if !shouldPanic && r != nil {
			t.Error(hintStr+": panic though it shouldn't have.", r)
		}
	}()
	validateConfig(rtc)
}

func assertEqStr(t *testing.T, expected, actual string) {
	if expected != actual {
		t.Errorf("expected: %s actual: %s", expected, actual)
	}
}
