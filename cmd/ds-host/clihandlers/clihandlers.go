package clihandlers

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/validator"
)

// CliHandlers handles cli based factions
type CliHandlers struct {
	UserModel interface {
		Create(email, password string) (domain.User, error)
		GetFromEmail(email string) (domain.User, error)
		MakeAdmin(userID domain.UserID) error
	}
	//Validator domain.Validator
	StdInput domain.StdInput
}

// AddAdmin adds a user and sets them as administrator
func (h *CliHandlers) AddAdmin() error {
	var email, pw string

	// ask for email
	for true {
		email = h.StdInput.ReadLine("Admin Email: ")
		dsErr := validator.Email(email)
		if dsErr != nil {
			fmt.Println(dsErr)
			continue
		}

		// test if exists
		_, err := h.UserModel.GetFromEmail(email)
		if err != nil {
			if err == sql.ErrNoRows {
				break
			} else {
				return err
			}
		} else {
			fmt.Println("email exists")
			continue
		}
	}

	for true {
		pw = h.StdInput.ReadLine("Password: ")
		dsErr := validator.Password(pw)
		if dsErr != nil {
			fmt.Println(dsErr)
		} else {
			break
		}
	}

	pw2 := h.StdInput.ReadLine("Please confirm password: ")
	if pw2 != pw {
		fmt.Println("Passwords do not match :(")
		return errors.New("Passwords do not match")
	}

	user, err := h.UserModel.Create(email, pw)
	if err != nil {
		return err
	}

	fmt.Println("Created user.")

	err = h.UserModel.MakeAdmin(user.UserID)
	if err != nil {
		return err
	}

	fmt.Println("Made user an admin.")

	return nil
}
