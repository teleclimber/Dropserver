package clihandlers

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
)

func TestAddAdmin(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	userModel := domain.NewMockUserModel(mockCtrl)
	userModel.EXPECT().GetFromEmail("foo@bar.com").Return(nil, dserror.New(dserror.NoRowsInResultSet))
	userModel.EXPECT().Create("foo@bar.com", "secretsauce").Return(&domain.User{}, nil)
	userModel.EXPECT().MakeAdmin(gomock.Any())

	validator := domain.NewMockValidator(mockCtrl)
	validator.EXPECT().Email("foo@bar.com").Return(nil)
	validator.EXPECT().Password("secretsauce").Return(nil)

	stdInput := domain.NewMockStdInput(mockCtrl)
	stdInput.EXPECT().ReadLine("Admin Email:").Return("foo@bar.com")
	stdInput.EXPECT().ReadLine("Password:").Return("secretsauce")
	stdInput.EXPECT().ReadLine("Please confirm password:").Return("secretsauce")

	h := &CliHandlers{
		UserModel: userModel,
		Validator: validator,
		StdInput:  stdInput,
	}

	user, dsErr := h.AddAdmin()
	if dsErr != nil {
		t.Fatal(dsErr)
	}

	if user == nil {
		t.Error("user should not be nil")
	}
}
