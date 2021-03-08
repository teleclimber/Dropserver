package clihandlers

import (
	"database/sql"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

func TestAddAdmin(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	userModel := testmocks.NewMockUserModel(mockCtrl)
	userModel.EXPECT().GetFromEmail("foo@bar.com").Return(domain.User{}, sql.ErrNoRows)
	userModel.EXPECT().Create("foo@bar.com", "secretsauce").Return(domain.User{}, nil)
	userModel.EXPECT().MakeAdmin(gomock.Any())

	validator := domain.NewMockValidator(mockCtrl)
	validator.EXPECT().Email("foo@bar.com").Return(nil)
	validator.EXPECT().Password("secretsauce").Return(nil)

	stdInput := domain.NewMockStdInput(mockCtrl)
	stdInput.EXPECT().ReadLine("Admin Email: ").Return("foo@bar.com")
	stdInput.EXPECT().ReadLine("Password: ").Return("secretsauce")
	stdInput.EXPECT().ReadLine("Please confirm password: ").Return("secretsauce")

	h := &CliHandlers{
		UserModel: userModel,
		Validator: validator,
		StdInput:  stdInput,
	}

	err := h.AddAdmin()
	if err != nil {
		t.Fatal(err)
	}
}
