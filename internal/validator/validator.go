package validator

import (
	"fmt"

	goValidator "gopkg.in/go-playground/validator.v9"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
)

// I'd like to create a thing we can inject into other packages to validate as needed

// Validator is an injectable Validation component
type Validator struct {
	// probably some internal ref to 3rd party validator.
	v *goValidator.Validate
}

// Init sets up the validator
func (v *Validator) Init() {
	v.v = goValidator.New()
}

// Q here is how we structure the API?
// do we have validator.Email(string) ?
// ..-> that's slick, but makes the API large (remember we have to add each method to domain.Validator too)
// But we could also have Struct(struct) which maps to go-playgoround/validator's API.
// ..and that would cover quite a bit of cases too.

// Password validates a password for logging in or registering
func (v *Validator) Password(pw string) domain.Error {
	err := v.v.Var(pw, "min=10")
	if err != nil {
		validationErrors, ok := err.(goValidator.ValidationErrors)
		if ok {
			// we have validation errors
			fmt.Println(validationErrors)
			return dserror.New(dserror.InputValidationError)
		}

		return dserror.New(dserror.InternalError) // ? I suppose?

	}

	return nil
}

// Email validates an email address. Assumed to be required.
func (v *Validator) Email(email string) domain.Error {
	err := v.v.Var(email, "required,email")
	if err != nil {
		validationErrors, ok := err.(goValidator.ValidationErrors)
		if ok {
			// we have validation errors
			fmt.Println(validationErrors)
			return dserror.New(dserror.InputValidationError)
		}

		return dserror.New(dserror.InternalError) // ? I suppose?

	}

	return nil
}

// it might be easier to force all inputs into structs, and set the validations as tags on structs.
// Reason: "email" validation here implies email is required. But that is not properlynormalized.
