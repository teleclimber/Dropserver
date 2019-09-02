package clihandlers

import (
	"fmt"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
)

// CliHandlers handles cli based factions
type CliHandlers struct {
	UserModel domain.UserModel
	Validator domain.Validator
	StdInput  domain.StdInput
}

// AddAdmin adds a user and sets them as administrator
func (h *CliHandlers) AddAdmin() (*domain.User, domain.Error) {
	var email, pw string

	// ask for email
	for true {
		email = h.StdInput.ReadLine("Admin Email:")
		dsErr := h.Validator.Email(email)
		if dsErr != nil {
			fmt.Println(dsErr)
			continue
		}

		// test if exists
		_, dsErr = h.UserModel.GetFromEmail(email)
		if dsErr != nil {
			if dsErr.Code() == dserror.NoRowsInResultSet {
				break
			} else {
				return nil, dsErr
			}
		} else {
			fmt.Println("email exists")
			continue
		}
	}

	for true {
		pw = h.StdInput.ReadLine("Password:")
		dsErr := h.Validator.Password(pw)
		if dsErr != nil {
			fmt.Println(dsErr)
		} else {
			break
		}
	}

	pw2 := h.StdInput.ReadLine("Please confirm password:")
	if pw2 != pw {
		fmt.Println("Passwords do not match :(")
		return nil, dserror.New(dserror.PasswordsDoNotMatch)
	}

	user, dsErr := h.UserModel.Create(email, pw)
	if dsErr != nil {
		return nil, dsErr
	}

	fmt.Println("Created user.")

	dsErr = h.UserModel.MakeAdmin(user.UserID)
	if dsErr != nil {
		return nil, dsErr
	}

	fmt.Println("Made user an admin.")

	return user, nil
}
