package ds2ds

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

// dropserver to dropserver comms.
// encapsulates:
// - mTLS verification (if enabled)
// - send a new request as a response

// Comms:
// - remote appspace login
// - user added to appspace (or change)
// - versioning of API (maybe through version hashes in header?)
// - ...

// Regarding the request/response paradigm
// First, not convinced we should implement it on this first pass.
// Second, it's not so straightforward:
// - If requesting a token, it's the token that is salient so it should be sent as a request.
// - If claiming a user is added to an appspace, what is salient?
//   .. maybe a verification has to be sent directly to appspace?
// --> worry about having requests result in more requests, because that is an avenue for DDOS
//   --> yes, but mTLS will largely defeat that

// This might be a good time to introduce go Context to our code , which is so far incredibly devoid of them.

// What are the incoming requests?
// [to: appspace domain] please send a login token for user
// [drop id domain] here is a login token?
// ...

// What are all the different request paths, and where are they coming from, and where are they going
// (in addition to ds2ds paths)
// - avatars: fetch from app frontend to a different domain so access can be limited (although is there any point to that?)
// -

// Note that inside appspacreouterpackage we have a dropserverroutes http handler
// ..that is not implemented but is said to serve login token requests.

type DS2DS struct {
	Config *domain.RuntimeConfig
	client *http.Client
}

func (d *DS2DS) Init() {
	d.client = http.DefaultClient
	d.trustCert()
}

func (d *DS2DS) trustCert() {
	// Here we need to only do this if config calls for insecure trust of certs.
	// Also I'm not sure any of this covers the case where there are different certs for different domains?
	// -> probably not.
	// ..but it could if we trusted the root CA cert?

	// See https://forfuncsake.github.io/post/2017/08/trust-extra-ca-cert-in-go-app/

	if d.Config.TrustCert == "" {
		return
	}

	// Get the SystemCertPool, continue with an empty pool on error
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	// Read in the cert file
	certs, err := ioutil.ReadFile(d.Config.TrustCert)
	if err != nil {
		log.Fatalf("Error reading cert file %v: %v", d.Config.TrustCert, err)
	}

	// Append our cert to the system pool
	if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
		d.getLogger("Init").Log("No root CA certs appended")
	}

	// Trust the augmented cert pool in our client
	config := &tls.Config{
		//InsecureSkipVerify: *insecure,	// don't need this if root ca added apparently
		RootCAs: rootCAs,
	}
	tr := &http.Transport{TLSClientConfig: config}
	d.client = &http.Client{Transport: tr}
}

func (d *DS2DS) GetRemoteAPIVersion(domainName string) (int, error) {
	// check if we stored the last used API version for this address in DB
	// If so return that
	// If not hit the remote's /.dropserver/api-version-check or whatever end point and find a match
	// (and stash the result)

	// for now fake it til you make it
	return 0, nil
}

func (d *DS2DS) GetClient() *http.Client {
	return d.client
}

func (d *DS2DS) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("DS2DS")
	if note != "" {
		l.AddNote(note)
	}
	return l
}
