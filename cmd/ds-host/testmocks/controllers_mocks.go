// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/teleclimber/DropServer/cmd/ds-host/testmocks (interfaces: MigrationJobController,AppspaceStatus,AppspaceRouter)

// Package testmocks is a generated GoMock package.
package testmocks

import (
	gomock "github.com/golang/mock/gomock"
	domain "github.com/teleclimber/DropServer/cmd/ds-host/domain"
	http "net/http"
	reflect "reflect"
)

// MockMigrationJobController is a mock of MigrationJobController interface
type MockMigrationJobController struct {
	ctrl     *gomock.Controller
	recorder *MockMigrationJobControllerMockRecorder
}

// MockMigrationJobControllerMockRecorder is the mock recorder for MockMigrationJobController
type MockMigrationJobControllerMockRecorder struct {
	mock *MockMigrationJobController
}

// NewMockMigrationJobController creates a new mock instance
func NewMockMigrationJobController(ctrl *gomock.Controller) *MockMigrationJobController {
	mock := &MockMigrationJobController{ctrl: ctrl}
	mock.recorder = &MockMigrationJobControllerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockMigrationJobController) EXPECT() *MockMigrationJobControllerMockRecorder {
	return m.recorder
}

// GetRunningJobs mocks base method
func (m *MockMigrationJobController) GetRunningJobs() []domain.MigrationStatusData {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRunningJobs")
	ret0, _ := ret[0].([]domain.MigrationStatusData)
	return ret0
}

// GetRunningJobs indicates an expected call of GetRunningJobs
func (mr *MockMigrationJobControllerMockRecorder) GetRunningJobs() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRunningJobs", reflect.TypeOf((*MockMigrationJobController)(nil).GetRunningJobs))
}

// Start mocks base method
func (m *MockMigrationJobController) Start() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Start")
}

// Start indicates an expected call of Start
func (mr *MockMigrationJobControllerMockRecorder) Start() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockMigrationJobController)(nil).Start))
}

// Stop mocks base method
func (m *MockMigrationJobController) Stop() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Stop")
}

// Stop indicates an expected call of Stop
func (mr *MockMigrationJobControllerMockRecorder) Stop() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockMigrationJobController)(nil).Stop))
}

// WakeUp mocks base method
func (m *MockMigrationJobController) WakeUp() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "WakeUp")
}

// WakeUp indicates an expected call of WakeUp
func (mr *MockMigrationJobControllerMockRecorder) WakeUp() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WakeUp", reflect.TypeOf((*MockMigrationJobController)(nil).WakeUp))
}

// MockAppspaceStatus is a mock of AppspaceStatus interface
type MockAppspaceStatus struct {
	ctrl     *gomock.Controller
	recorder *MockAppspaceStatusMockRecorder
}

// MockAppspaceStatusMockRecorder is the mock recorder for MockAppspaceStatus
type MockAppspaceStatusMockRecorder struct {
	mock *MockAppspaceStatus
}

// NewMockAppspaceStatus creates a new mock instance
func NewMockAppspaceStatus(ctrl *gomock.Controller) *MockAppspaceStatus {
	mock := &MockAppspaceStatus{ctrl: ctrl}
	mock.recorder = &MockAppspaceStatusMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockAppspaceStatus) EXPECT() *MockAppspaceStatusMockRecorder {
	return m.recorder
}

// Ready mocks base method
func (m *MockAppspaceStatus) Ready(arg0 domain.AppspaceID) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Ready", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// Ready indicates an expected call of Ready
func (mr *MockAppspaceStatusMockRecorder) Ready(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Ready", reflect.TypeOf((*MockAppspaceStatus)(nil).Ready), arg0)
}

// SetHostStop mocks base method
func (m *MockAppspaceStatus) SetHostStop(arg0 bool) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetHostStop", arg0)
}

// SetHostStop indicates an expected call of SetHostStop
func (mr *MockAppspaceStatusMockRecorder) SetHostStop(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetHostStop", reflect.TypeOf((*MockAppspaceStatus)(nil).SetHostStop), arg0)
}

// WaitStopped mocks base method
func (m *MockAppspaceStatus) WaitStopped(arg0 domain.AppspaceID) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "WaitStopped", arg0)
}

// WaitStopped indicates an expected call of WaitStopped
func (mr *MockAppspaceStatusMockRecorder) WaitStopped(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WaitStopped", reflect.TypeOf((*MockAppspaceStatus)(nil).WaitStopped), arg0)
}

// MockAppspaceRouter is a mock of AppspaceRouter interface
type MockAppspaceRouter struct {
	ctrl     *gomock.Controller
	recorder *MockAppspaceRouterMockRecorder
}

// MockAppspaceRouterMockRecorder is the mock recorder for MockAppspaceRouter
type MockAppspaceRouterMockRecorder struct {
	mock *MockAppspaceRouter
}

// NewMockAppspaceRouter creates a new mock instance
func NewMockAppspaceRouter(ctrl *gomock.Controller) *MockAppspaceRouter {
	mock := &MockAppspaceRouter{ctrl: ctrl}
	mock.recorder = &MockAppspaceRouterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockAppspaceRouter) EXPECT() *MockAppspaceRouterMockRecorder {
	return m.recorder
}

// ServeHTTP mocks base method
func (m *MockAppspaceRouter) ServeHTTP(arg0 http.ResponseWriter, arg1 *http.Request, arg2 *domain.AppspaceRouteData) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "ServeHTTP", arg0, arg1, arg2)
}

// ServeHTTP indicates an expected call of ServeHTTP
func (mr *MockAppspaceRouterMockRecorder) ServeHTTP(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ServeHTTP", reflect.TypeOf((*MockAppspaceRouter)(nil).ServeHTTP), arg0, arg1, arg2)
}

// SubscribeLiveCount mocks base method
func (m *MockAppspaceRouter) SubscribeLiveCount(arg0 domain.AppspaceID, arg1 chan<- int) int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SubscribeLiveCount", arg0, arg1)
	ret0, _ := ret[0].(int)
	return ret0
}

// SubscribeLiveCount indicates an expected call of SubscribeLiveCount
func (mr *MockAppspaceRouterMockRecorder) SubscribeLiveCount(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SubscribeLiveCount", reflect.TypeOf((*MockAppspaceRouter)(nil).SubscribeLiveCount), arg0, arg1)
}

// UnsubscribeLiveCount mocks base method
func (m *MockAppspaceRouter) UnsubscribeLiveCount(arg0 domain.AppspaceID, arg1 chan<- int) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "UnsubscribeLiveCount", arg0, arg1)
}

// UnsubscribeLiveCount indicates an expected call of UnsubscribeLiveCount
func (mr *MockAppspaceRouterMockRecorder) UnsubscribeLiveCount(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnsubscribeLiveCount", reflect.TypeOf((*MockAppspaceRouter)(nil).UnsubscribeLiveCount), arg0, arg1)
}
