package domaincontroller

import (
	"errors"
	"regexp"
	"strings"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// DomainController ensures validity, uniqueness of domain names,
// this might be where we cache domains and their associations
// ..for faster lookups on requests
type DomainController struct {
	Config        *domain.RuntimeConfig `checkinject:"required"`
	AppspaceModel interface {
		GetFromDomain(dom string) (*domain.Appspace, error)
	} `checkinject:"required"`
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
		DomainName:                d.Config.Server.Host,
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
	if dom != d.Config.Server.Host {
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

var labelChars = regexp.MustCompile(`^[a-z0-9-]*$`).MatchString

func validateDomainLabel(label string) error {
	if label == "" {
		return errors.New("Domain label can not be blank")
	}
	if len(label) > 63 {
		return errors.New("Domain label can not be longer than 63")
	}
	if !labelChars(label) {
		return errors.New("Domain label contains invalid characters")
	}
	if strings.HasPrefix(label, "-") {
		return errors.New("Domain name label can not start with - hyphen")
	}

	return nil
}
