package domaincontroller

import (
	"errors"
	"regexp"
	"strings"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

// AUTO-TLS (and probably naked domains etc..)
// For each domain we should know the TLS situation:
// To use naked:
// - Is there a cert? Or will there be one generated?
// - Or is ds-host configured to generate certs?
// To use with subdomain:
// - there is a wildcard cert managed by admin or if a cert will be generated for a subdomain
//   ..in both cases, it's OK to create a subdomain
// - or is ds-host configured to genrate the cert
//   .. then create subdomain and generate the cert.
// If none of the above, then adding a [sub]domain should come with a warning that no cert will be created and you need to deal with that.

// For proxy routing:
// - Is this [sub]domain configured in reverse proxy to forward to ds-host?
// - Is ds-host configured to notify reverse proxy of [sub]domain

// Note that the main domain of the dropserver instance may need som special treatment?
// -> Probaly that you can 't change it in the UI, has to be in config.
// -> Also if cert is managed, get it first instead of queuing it with all the others
// See RuntimeConfig file
// -> "main domain" may actually be the full user panel domain: dropid.mo.com,
//   ..and the domain above that is considered a "normal" domain, which has to consider where it sits wrt to subdomains

// So for each domain name entered by admin:
// - send to proxy: [ ]naked [ ]wildcard subs [create specific subs]
// - TLS cert: ( )Manual ( )Automatic outside ds-host ( )Manage certs [if enabled in config]
// ^ these optsion make sense for an admin. What about a user?
// -> needs more configuration thoughts.

// DomainController ensures validity, uniqueness of domain names,
// this might be where we cache domains and their associations
// ..for faster lookups on requests
type DomainController struct {
	Config        *domain.RuntimeConfig `checkinject:"required"`
	AppspaceModel interface {
		GetFromDomain(dom string) (*domain.Appspace, error)
		GetAllDomains() ([]string, error)
	} `checkinject:"required"`
	CertificateManager interface {
		ResumeManaging([]string) error
		StartManaging(string) error
		StopManaging(string)
	}
}

func (d *DomainController) ResumeManagingCertificates() {
	if !d.Config.ManageTLSCertificates.Enable {
		return
	}
	// get all the domains in an array
	// Also need the user routes domain
	doms, err := d.AppspaceModel.GetAllDomains()
	if err != nil {
		d.getLogger("ResumeManagingCertificates").Error(err)
		return
	}
	// Will also have to make certs for dropids
	doms = append(doms, d.Config.Exec.UserRoutesDomain) // UserRoutesDomain should really be obtained first, then add the others.

	d.CertificateManager.ResumeManaging(doms)
}

// GetDomains for user. Includes all available domains for all use cases
func (d *DomainController) GetDomains(userID domain.UserID) ([]domain.DomainData, error) {
	return []domain.DomainData{{
		DomainName:             d.Config.Exec.UserRoutesDomain,
		UserOwned:              false,
		ForAppspace:            false,
		ForDropID:              true,
		DropIDSubdomainAllowed: false,
	}, {
		DomainName:                d.Config.ExternalAccess.Domain,
		UserOwned:                 false,
		ForAppspace:               true,
		AppspaceSubdomainRequired: true,
		ForDropID:                 false,
	}}, nil
}

// GetDropIDDomains that a user can use to create a new drop id
func (d *DomainController) GetDropIDDomains(userID domain.UserID) ([]domain.DomainData, error) {
	return []domain.DomainData{{
		DomainName:             d.Config.Exec.UserRoutesDomain,
		UserOwned:              false,
		ForAppspace:            false,
		ForDropID:              true,
		DropIDSubdomainAllowed: false,
	}}, nil
}

// CheckAppspaceDomain determines whether a suggested domain/subdomain
// can be used for an appspace.
func (d *DomainController) CheckAppspaceDomain(userID domain.UserID, dom string, subdomain string) (domain.DomainCheckResult, error) {
	ret := domain.DomainCheckResult{
		Valid:     false,
		Available: false,
		Message:   ""}

	// get domain...
	if dom != d.Config.ExternalAccess.Domain {
		ret.Message = "Base domain not found"
		return ret, nil
	}

	// user id will be used to check ownership of custom domains when we get there.

	// Currently subdomain can not be empty
	// ..so go straight to this validation.
	// In future, a user-owned naked domain could be used, and subdomain == ""
	if err := validateSubdomains(subdomain); err != nil {
		ret.Message = err.Error()
		return ret, nil
	}

	fullDomain := subdomain + "." + dom
	// check length of full domain

	ret.Valid = true

	// Here we should ensure that no domain is [grand]parent or [grand]child of this domain.
	appspace, err := d.AppspaceModel.GetFromDomain(fullDomain)
	if err != nil {
		return ret, err
	}
	if appspace != nil {
		ret.Message = "In use by appspace"
		return ret, nil
	}

	ret.Available = true

	return ret, nil
}

func (d *DomainController) StartManaging(dom string) error {
	if !d.Config.ManageTLSCertificates.Enable {
		return nil
	}
	return d.CertificateManager.StartManaging(dom)
}

func (d *DomainController) StopManaging(dom string) {
	if !d.Config.ManageTLSCertificates.Enable {
		return
	}
	d.CertificateManager.StopManaging(dom)
}

// validateSubdomains takes a string of subdomains
// like "abc.def", or just "abc"
// and returns and error with validation problem
// or nil
func validateSubdomains(sub string) error {
	// Explode on '.'
	subs := strings.Split(sub, ".")
	for _, s := range subs {
		err := validateDomainLabel(s)
		if err != nil {
			return err
		}
	}
	return nil
}

var labelChars = regexp.MustCompile(`^[a-zA-Z0-9-]*$`).MatchString

func validateDomainLabel(label string) error {
	if label == "" {
		return errors.New("domain label can not be blank")
	}
	if len(label) > 63 {
		return errors.New("domain label can not be longer than 63")
	}
	if !labelChars(label) {
		return errors.New("domain label contains invalid characters")
	}
	if strings.HasPrefix(label, "-") {
		return errors.New("domain name label can not start with - hyphen")
	}

	return nil
}

func (d *DomainController) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("DomainController")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
