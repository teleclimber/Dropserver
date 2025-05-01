package validator

import (
	"errors"
	"regexp"
	"strings"

	goValidator "github.com/go-playground/validator/v10"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

var goVal = func() *goValidator.Validate {
	goVal := goValidator.New()
	goVal.RegisterValidation("alphanumdash", ValidateAlphaNumDash)
	goVal.RegisterValidation("startalpha", ValidateStartAlpha)
	goVal.RegisterValidation("startalphanum", ValidateStartAlphaNum)
	goVal.RegisterValidation("endalphanum", ValidateEndAlphaNum)
	goVal.RegisterAlias("tsnetcontrolurl", "max=500") // allow anything, if it doesn't connect it doesn't connect.
	goVal.RegisterAlias("tsnetmachinename", "max=63,alphanumdash,startalphanum,endalphanum")
	goVal.RegisterAlias("tsnettag", "max=50,alphanumdash,startalpha")
	goVal.RegisterAlias("tsnetauthkey", "max=500")
	return goVal
}()

var alphaNumDashRegex = regexp.MustCompile("^[a-zA-Z0-9-]*$")

// ValidateAlphaNumDash for strings that can contain alphanum and dashes
func ValidateAlphaNumDash(fl goValidator.FieldLevel) bool {
	return alphaNumDashRegex.MatchString(fl.Field().String())
}

var startAlphaRegex = regexp.MustCompile("^[a-zA-Z]")

// ValidateStartAlpha verifies a string starts with an alpha character
func ValidateStartAlpha(fl goValidator.FieldLevel) bool {
	return startAlphaRegex.MatchString(fl.Field().String())
}

var startAlphaNumRegex = regexp.MustCompile("^[a-zA-Z0-9]")

// ValidateStartAlphaNum verifies a string starts with an alphanum character
func ValidateStartAlphaNum(fl goValidator.FieldLevel) bool {
	return startAlphaNumRegex.MatchString(fl.Field().String())
}

var endAlphaNumRegex = regexp.MustCompile("[a-zA-Z0-9]$")

// ValidateEndAlphaNum verifies a string ends with an alphanum character
func ValidateEndAlphaNum(fl goValidator.FieldLevel) bool {
	return endAlphaNumRegex.MatchString(fl.Field().String())
}

//////////////////////////////////////////////////////

// Password validates a password for logging in or registering
func Password(pw string) error {
	return goVal.Var(pw, "min=10")
}

// Email validates an email address. Assumed to be required.
func Email(email string) error {
	return goVal.Var(email, "required,email")
}

func HttpURL(url string) error {
	return goVal.Var(url, "required,http_url")
}

// DomainName validates a domain name
// Domain can include subdomains
func DomainName(domainName string) error {
	return goVal.Var(domainName, "required,fqdn")
}

func AppGetKey(key string) error {
	return goVal.Var(key, "min=8,max=10,alphanum")
}

func LocationKey(loc string) error {
	return goVal.Var(loc, "min=8,max=16,alphanum") //as531411051
}

// V0AppspaceLoginToken is used to obtain a session cookie for an appspace
func V0AppspaceLoginToken(token string) error {
	return goVal.Var(token, "min=20,max=30,alphanum")
}

// V0AppspaceLoginRef is a reference used to identify requests for tokens to remote host
func V0AppspaceLoginRef(ref string) error {
	return goVal.Var(ref, "min=8,max=12,alphanum")
}

// DBName validates an appspace DB name
func DBName(pw string) error {
	return goVal.Var(pw, "min=1,max=30,alphanum") // super restrictive for now
}

// AppspaceUserAuthType validates auth type for appspace users
func AppspaceUserAuthType(authType string) error {
	if authType != "email" && authType != "dropid" && authType != "tsnetid" {
		return errors.New("auth type must be email or dropid or tsnetid")
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
func DropIDHandle(handle string) error {
	if handle == "" {
		return nil
	}
	return goVal.Var(handle, "min=0,max=30,alphanum")
}

func TSNetIDFull(tsnetid string) error {
	id, controlURL := SplitTSNetID(tsnetid)
	err := TSNetIdentifier(id)
	if err != nil {
		return err
	}
	err = DomainName(controlURL)
	if err != nil {
		return err
	}
	return nil
}

func TSNetIdentifier(handle string) error {
	return goVal.Var(handle, "min=1,max=30,alphanum")
}

func TSNetCreateConfig(c domain.TSNetCreateConfig) error {
	return goVal.Struct(c)
}

// UserProxyID validates an appspace user proxy id
func UserProxyID(p string) error {
	// specific length?
	// specific set of characters?
	return goVal.Var(p, "min=8,max=10,alphanum")
}

// DisplayName validates an aappspace user's or DropID display name
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
		return errors.New("permission can not contain tab")
	}
	if strings.Contains(p, ",") {
		return errors.New("permission can not contain comma")
	}
	return goVal.Var(p, "min=1,max=20")
}

var validAppspaceAvatarFilename = regexp.MustCompile(`^[0-9a-zA-Z]+-[0-9a-zA-Z]+\.jpg$`)

func AppspaceAvatarFilename(f string) error {
	if !validAppspaceAvatarFilename.MatchString(f) {
		return errors.New("invalid format for appspace avatar file name")
	}
	return nil
}

var validBackupFile = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}_[0-9]{4}(?:_[1-9])?\.zip$`)

// AppspaceBackupFile validates names of appspace backup files (sans .zip extension)
func AppspaceBackupFile(b string) error {
	if !validBackupFile.MatchString(b) {
		return errors.New("invalid format for appspace backup file name")
	}
	return nil
}

// it might be easier to force all inputs into structs, and set the validations as tags on structs.
// Reason: "email" validation here implies email is required. But that is not properlynormalized.
