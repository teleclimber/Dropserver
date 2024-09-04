package certificatemanager

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/caddyserver/certmagic"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// There are probably some situations where the admin will have to delete all the certs and manage them again.
// Like if you change from self-signed root CA, to just using Letsencrypt.
// -> because the trusted root CA will no longer be in the configuration, therefor certs won't validate, Or something?

type CertficateManager struct {
	Config *domain.RuntimeConfig `checkinject:"required"`

	magic  *certmagic.Config
	issuer *certmagic.ACMEIssuer
}

func (c *CertficateManager) Init() {
	cfg := c.Config.ManageTLSCertificates

	certmagic.Default.Storage = &certmagic.FileStorage{
		Path: c.Config.Exec.CertificatesPath}

	if cfg.DisableOCSPStapling {
		certmagic.Default.OCSP = certmagic.OCSPConfig{DisableStapling: true}
	}

	c.magic = certmagic.NewDefault()

	issuer := certmagic.ACMEIssuer{
		Agreed:                  true,
		Email:                   cfg.Email, // email should be admin, particularly for self-hosters
		CA:                      cfg.IssuerEndpoint,
		AltHTTPPort:             int(c.Config.Server.HTTPPort),
		AltTLSALPNPort:          int(c.Config.Server.TLSPort),
		DisableTLSALPNChallenge: false,
	}

	if cfg.RootCACertificate != "" {
		rootCertPath := cfg.RootCACertificate
		if !filepath.IsAbs(rootCertPath) {
			wd, err := os.Getwd()
			if err != nil {
				panic(err)
			}
			rootCertPath = filepath.Join(wd, rootCertPath)
		}
		rootCA, err := ioutil.ReadFile(rootCertPath) // the file is inside the local directory
		if err != nil {
			panic(err)
		}
		rootCAPool := x509.NewCertPool()
		ok := rootCAPool.AppendCertsFromPEM(rootCA)
		if !ok {
			panic("unable to append supplied cert into tls.Config, are you sure it is a valid certificate")
		}
		issuer.TrustedRoots = rootCAPool
	}

	c.issuer = certmagic.NewACMEIssuer(c.magic, issuer)
	certmagic.Default.Issuers = []certmagic.Issuer{c.issuer}

	c.magic = certmagic.NewDefault()
}

func (c *CertficateManager) GetTLSConfig() *tls.Config {
	if c.magic == nil {
		panic("trying to get TLS Config before initializing Cert Manager")
	}
	tlsConfig := c.magic.TLSConfig()
	tlsConfig.NextProtos = append([]string{"h2", "http/1.1"}, tlsConfig.NextProtos...)
	return tlsConfig
}
func (c *CertficateManager) GetHTTPChallengeHandler(handler http.Handler) http.Handler {
	return c.issuer.HTTPChallengeHandler(handler)
}

// ResumeManaging manages the passed domain but returns immediately
func (c *CertficateManager) ResumeManaging(d []string) error {
	err := c.magic.ManageAsync(context.TODO(), d)
	if err != nil {
		return err
	}
	return nil
}

// StartManaging begins managing certificates for this domain
func (c *CertficateManager) StartManaging(d string) error {
	err := c.magic.ManageSync(context.TODO(), []string{d})
	if err != nil {
		return err
	}
	return nil
}

// StopManaging turns off management for the domain
// certmagic Unmamage deletes the cert from the cache but not from the storage?
func (c *CertficateManager) StopManaging(d string) {
	// TODO temporary c.magic.Unmanage([]string{d})
	// see https://github.com/caddyserver/certmagic/issues/307
	// https://github.com/teleclimber/Dropserver/issues/133
}
