package userroutes

import (
	"context"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

type ctxKey string

// URL tail
const urlTailCtxKey = ctxKey("url tail")

func ctxWithURLTail(ctx context.Context, urlTail string) context.Context {
	return context.WithValue(ctx, urlTailCtxKey, urlTail)
}
func ctxURLTail(ctx context.Context) string {
	t, ok := ctx.Value(urlTailCtxKey).(string)
	if !ok {
		return ""
	}
	return t
}

// Authenticated User ID
const authUserIDCtxKey = ctxKey("authenticated user ID")

func ctxWithAuthUserID(ctx context.Context, userID domain.UserID) context.Context {
	return context.WithValue(ctx, authUserIDCtxKey, userID)
}
func ctxAuthUserID(ctx context.Context) (domain.UserID, bool) {
	t, ok := ctx.Value(authUserIDCtxKey).(domain.UserID)
	return t, ok
}

// Appspace Data
const appspaceDataCtxKey = ctxKey("appspace data")

func ctxWithAppspaceData(ctx context.Context, appspace domain.Appspace) context.Context {
	return context.WithValue(ctx, appspaceDataCtxKey, appspace)
}

func ctxAppspaceData(ctx context.Context) (domain.Appspace, bool) {
	t, ok := ctx.Value(appspaceDataCtxKey).(domain.Appspace)
	return t, ok
}
