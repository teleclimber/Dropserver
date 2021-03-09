package validator

import (
	goValidator "github.com/go-playground/validator/v10"
)

// I'd like to create a thing we can inject into other packages to validate as needed

var goVal = goValidator.New()

// Password validates a password for logging in or registering
func Password(pw string) error {
	return goVal.Var(pw, "min=10")
}

// Email validates an email address. Assumed to be required.
func Email(email string) error {
	return goVal.Var(email, "required,email")
}

// DBName validates an appspace DB name
func DBName(pw string) error {
	return goVal.Var(pw, "min=1,max=30,alphanum") // super restrictive for now
}

// it might be easier to force all inputs into structs, and set the validations as tags on structs.
// Reason: "email" validation here implies email is required. But that is not properlynormalized.
