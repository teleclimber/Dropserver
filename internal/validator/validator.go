package validator

import (
	"errors"
	"strings"

	goValidator "github.com/go-playground/validator/v10"
)

var goVal = goValidator.New()

// Password validates a password for logging in or registering
func Password(pw string) error {
	return goVal.Var(pw, "min=10")
}

// Email validates an email address. Assumed to be required.
func Email(email string) error {
	return goVal.Var(email, "required,email")
}

// DomainName validates a domain name
// Domain can include subdomains
func DomainName(domainName string) error {
	return goVal.Var(domainName, "required,fqdn")
}

// DBName validates an appspace DB name
func DBName(pw string) error {
	return goVal.Var(pw, "min=1,max=30,alphanum") // super restrictive for now
}

// AppspaceUserAuthType validates auth type for appspace users
func AppspaceUserAuthType(authType string) error {
	if authType != "email" && authType != "dropid" {
		return errors.New("Auth type must be email or dropid")
	}
	return nil
}

// DropIDFull validates a full dropid
func DropIDFull(dropID string) error {
	h, d := SplitDropID(dropID)
	if h != "" { // an empty handle is valid
		err := DropIDHandle(h)
		if err != nil {
			return err
		}
	}
	err := DomainName(d)
	if err != nil {
		return err
	}
	return nil
}

// DropIDHandle validates a handle
// It assumes there is a handle (although that may not be that helpful?)
func DropIDHandle(handle string) error {
	return goVal.Var(handle, "min=1,max=30,alphanum") // super restrictive for now
}

// UserProxyID validates an appspace user proxy id
func UserProxyID(p string) error {
	// specific length?
	// specific set of characters?
	return goVal.Var(p, "min=8,max=10,alphanum")
}

// DisplayName validates a user's display name
func DisplayName(dn string) error {
	// can't start / end with spaces
	// min lenght, max lengh.
	// look at Go's TrimSpace for details on detecting space chars
	return goVal.Var(dn, "min=1,max=30")
}

// AppspacePermission validates an appspace permission identifier string
func AppspacePermission(p string) error {
	// we're doing comma-seperated string in DB, so no commas
	// But this leads me to think we should tab-separate the permissions
	// .. and disallow tabs?
	// disallow whitespace too
	if strings.Contains(p, "\t") {
		return errors.New("Permission can not contain tab")
	}
	if strings.Contains(p, ",") {
		return errors.New("Permission can not contain comma")
	}
	return goVal.Var(p, "min=1,max=20")
}

// it might be easier to force all inputs into structs, and set the validations as tags on structs.
// Reason: "email" validation here implies email is required. But that is not properlynormalized.
