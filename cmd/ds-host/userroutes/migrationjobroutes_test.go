package userroutes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

func TestGetJobsQuery(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ownerID := domain.UserID(11)
	appspaceID := domain.AppspaceID(7)

	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	appspaceModel.EXPECT().GetFromID(appspaceID).Return(&domain.Appspace{OwnerID: ownerID}, nil)

	migrationJobModel := testmocks.NewMockMigrationJobModel(mockCtrl)
	migrationJobModel.EXPECT().GetForAppspace(appspaceID).Return([]*domain.MigrationJob{{ToVersion: "1.2.3"}}, nil)

	r := &MigrationJobRoutes{
		AppspaceModel:     appspaceModel,
		MigrationJobModel: migrationJobModel,
	}

	req, err := http.NewRequest(http.MethodPost, "/?appspace_id=7", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	routeData := &domain.AppspaceRouteData{
		Authentication: &domain.Authentication{
			UserID: ownerID}}

	r.getJobsQuery(rr, req, routeData)

	bodyStr := rr.Body.String()

	if !strings.Contains(bodyStr, `1.2.3`) {
		t.Fatal("body not as expected" + bodyStr)
	}

}
