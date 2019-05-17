// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/teleclimber/DropServer/cmd/ds-trusted/trusteddomain (interfaces: AppFilesI)

// Package trusteddomain is a generated GoMock package.
package trusteddomain

import (
	gomock "github.com/golang/mock/gomock"
	domain "github.com/teleclimber/DropServer/cmd/ds-host/domain"
	reflect "reflect"
)

// MockAppFilesI is a mock of AppFilesI interface
type MockAppFilesI struct {
	ctrl     *gomock.Controller
	recorder *MockAppFilesIMockRecorder
}

// MockAppFilesIMockRecorder is the mock recorder for MockAppFilesI
type MockAppFilesIMockRecorder struct {
	mock *MockAppFilesI
}

// NewMockAppFilesI creates a new mock instance
func NewMockAppFilesI(ctrl *gomock.Controller) *MockAppFilesI {
	mock := &MockAppFilesI{ctrl: ctrl}
	mock.recorder = &MockAppFilesIMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockAppFilesI) EXPECT() *MockAppFilesIMockRecorder {
	return m.recorder
}

// ReadMeta mocks base method
func (m *MockAppFilesI) ReadMeta(arg0 string) (*domain.AppFilesMetadata, domain.Error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadMeta", arg0)
	ret0, _ := ret[0].(*domain.AppFilesMetadata)
	ret1, _ := ret[1].(domain.Error)
	return ret0, ret1
}

// ReadMeta indicates an expected call of ReadMeta
func (mr *MockAppFilesIMockRecorder) ReadMeta(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadMeta", reflect.TypeOf((*MockAppFilesI)(nil).ReadMeta), arg0)
}

// Save mocks base method
func (m *MockAppFilesI) Save(arg0 *domain.TrustedSaveAppFiles) (string, domain.Error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Save", arg0)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(domain.Error)
	return ret0, ret1
}

// Save indicates an expected call of Save
func (mr *MockAppFilesIMockRecorder) Save(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Save", reflect.TypeOf((*MockAppFilesI)(nil).Save), arg0)
}