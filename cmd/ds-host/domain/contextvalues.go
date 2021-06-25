package domain

import (
	"context"
)

type ctxKey string

// Authenticated User ID
const sessionIDCtxKey = ctxKey("session ID")

// CtxWithAuthUserID sets the authenticated user id on the context
func CtxWithSessionID(ctx context.Context, sessionId string) context.Context {
	return context.WithValue(ctx, sessionIDCtxKey, sessionId)
}

// CtxAuthUserID gets the authenticated user id from the context
// Second value is false if no auth user id
func CtxSessionID(ctx context.Context) (string, bool) {
	t, ok := ctx.Value(sessionIDCtxKey).(string)
	return t, ok
}

// Authenticated User ID
const authUserIDCtxKey = ctxKey("authenticated user ID")

// CtxWithAuthUserID sets the authenticated user id on the context
func CtxWithAuthUserID(ctx context.Context, userID UserID) context.Context {
	return context.WithValue(ctx, authUserIDCtxKey, userID)
}

// CtxAuthUserID gets the authenticated user id from the context
// Second value is false if no auth user id
func CtxAuthUserID(ctx context.Context) (UserID, bool) {
	t, ok := ctx.Value(authUserIDCtxKey).(UserID)
	return t, ok
}

// App Data
const appDataCtxKey = ctxKey("application data")

// CtxWithAppData sets the app data that is solely relevant
// to the request
func CtxWithAppData(ctx context.Context, app App) context.Context {
	return context.WithValue(ctx, appDataCtxKey, app)
}

// CtxAppData gets the app data that is solely relevant
// to the request
func CtxAppData(ctx context.Context) (App, bool) {
	t, ok := ctx.Value(appDataCtxKey).(App)
	return t, ok
}

// App Version Data
const appVersionDataCtxKey = ctxKey("application version data")

// CtxWithAppVersionData sets the app data that is solely relevant
// to the request
func CtxWithAppVersionData(ctx context.Context, appVersion AppVersion) context.Context {
	return context.WithValue(ctx, appVersionDataCtxKey, appVersion)
}

// CtxAppVersionData gets the app data that is solely relevant
// to the request
func CtxAppVersionData(ctx context.Context) (AppVersion, bool) {
	t, ok := ctx.Value(appVersionDataCtxKey).(AppVersion)
	return t, ok
}

// Appspace Data
const appspaceDataCtxKey = ctxKey("appspace data")

// CtxWithAppspaceData sets the appspace data that is solely relevant
// to the request
func CtxWithAppspaceData(ctx context.Context, appspace Appspace) context.Context {
	return context.WithValue(ctx, appspaceDataCtxKey, appspace)
}

// CtxAppspaceData gets the appspace data that is solely relevant
// to the request
func CtxAppspaceData(ctx context.Context) (Appspace, bool) {
	t, ok := ctx.Value(appspaceDataCtxKey).(Appspace)
	return t, ok
}

// Appspace User Proxy ID
const appspaceUserProxyIDCtxKey = ctxKey("appspace user proxy id")

// CtxWithAppspaceUserProxyID sets the appspace data that is solely relevant
// to the request
func CtxWithAppspaceUserProxyID(ctx context.Context, proxyID ProxyID) context.Context {
	return context.WithValue(ctx, appspaceUserProxyIDCtxKey, proxyID)
}

// CtxAppspaceUserProxyID gets the appspace data that is solely relevant
// to the request
func CtxAppspaceUserProxyID(ctx context.Context) (ProxyID, bool) {
	t, ok := ctx.Value(appspaceUserProxyIDCtxKey).(ProxyID)
	return t, ok
}

// Appspace Data
const appspaceUserDataCtxKey = ctxKey("appspace user data")

// CtxWithAppspaceUserData sets the appspace data that is solely relevant
// to the request
func CtxWithAppspaceUserData(ctx context.Context, user AppspaceUser) context.Context {
	return context.WithValue(ctx, appspaceUserDataCtxKey, user)
}

// CtxAppspaceUserData gets the appspace data that is solely relevant
// to the request
func CtxAppspaceUserData(ctx context.Context) (AppspaceUser, bool) {
	t, ok := ctx.Value(appspaceUserDataCtxKey).(AppspaceUser)
	return t, ok
}

// App Route Config Data
const v0routeConfigDataCtxKey = ctxKey("V0 appspace route config user data")

// CtxWithRouteConfig sets the appspace route data for the request
func CtxWithV0RouteConfig(ctx context.Context, routeConfig V0AppRoute) context.Context {
	return context.WithValue(ctx, v0routeConfigDataCtxKey, routeConfig)
}

// CtxRouteConfig gets the appspace route config data for the request
func CtxV0RouteConfig(ctx context.Context) (V0AppRoute, bool) {
	t, ok := ctx.Value(v0routeConfigDataCtxKey).(V0AppRoute)
	return t, ok
}
