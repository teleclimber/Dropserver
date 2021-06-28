// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/teleclimber/DropServer/cmd/ds-host/testmocks (interfaces: SandboxMaker,SandboxManager)

// Package testmocks is a generated GoMock package.
package testmocks

import (
	gomock "github.com/golang/mock/gomock"
	domain "github.com/teleclimber/DropServer/cmd/ds-host/domain"
	reflect "reflect"
)

// MockSandboxMaker is a mock of SandboxMaker interface
type MockSandboxMaker struct {
	ctrl     *gomock.Controller
	recorder *MockSandboxMakerMockRecorder
}

// MockSandboxMakerMockRecorder is the mock recorder for MockSandboxMaker
type MockSandboxMakerMockRecorder struct {
	mock *MockSandboxMaker
}

// NewMockSandboxMaker creates a new mock instance
func NewMockSandboxMaker(ctrl *gomock.Controller) *MockSandboxMaker {
	mock := &MockSandboxMaker{ctrl: ctrl}
	mock.recorder = &MockSandboxMakerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockSandboxMaker) EXPECT() *MockSandboxMakerMockRecorder {
	return m.recorder
}

// ForApp mocks base method
func (m *MockSandboxMaker) ForApp(arg0 *domain.AppVersion) (domain.SandboxI, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ForApp", arg0)
	ret0, _ := ret[0].(domain.SandboxI)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ForApp indicates an expected call of ForApp
func (mr *MockSandboxMakerMockRecorder) ForApp(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ForApp", reflect.TypeOf((*MockSandboxMaker)(nil).ForApp), arg0)
}

// ForMigration mocks base method
func (m *MockSandboxMaker) ForMigration(arg0 *domain.AppVersion, arg1 *domain.Appspace) (domain.SandboxI, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ForMigration", arg0, arg1)
	ret0, _ := ret[0].(domain.SandboxI)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ForMigration indicates an expected call of ForMigration
func (mr *MockSandboxMakerMockRecorder) ForMigration(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ForMigration", reflect.TypeOf((*MockSandboxMaker)(nil).ForMigration), arg0, arg1)
}

// MockSandboxManager is a mock of SandboxManager interface
type MockSandboxManager struct {
	ctrl     *gomock.Controller
	recorder *MockSandboxManagerMockRecorder
}

// MockSandboxManagerMockRecorder is the mock recorder for MockSandboxManager
type MockSandboxManagerMockRecorder struct {
	mock *MockSandboxManager
}

// NewMockSandboxManager creates a new mock instance
func NewMockSandboxManager(ctrl *gomock.Controller) *MockSandboxManager {
	mock := &MockSandboxManager{ctrl: ctrl}
	mock.recorder = &MockSandboxManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockSandboxManager) EXPECT() *MockSandboxManagerMockRecorder {
	return m.recorder
}

// GetForAppspace mocks base method
func (m *MockSandboxManager) GetForAppspace(arg0 *domain.AppVersion, arg1 *domain.Appspace) chan domain.SandboxI {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetForAppspace", arg0, arg1)
	ret0, _ := ret[0].(chan domain.SandboxI)
	return ret0
}

// GetForAppspace indicates an expected call of GetForAppspace
func (mr *MockSandboxManagerMockRecorder) GetForAppspace(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetForAppspace", reflect.TypeOf((*MockSandboxManager)(nil).GetForAppspace), arg0, arg1)
}

// StopAppspace mocks base method
func (m *MockSandboxManager) StopAppspace(arg0 domain.AppspaceID) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "StopAppspace", arg0)
}

// StopAppspace indicates an expected call of StopAppspace
func (mr *MockSandboxManagerMockRecorder) StopAppspace(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StopAppspace", reflect.TypeOf((*MockSandboxManager)(nil).StopAppspace), arg0)
}