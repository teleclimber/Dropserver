// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/teleclimber/DropServer/cmd/ds-host/testmocks (interfaces: AppspaceMetaDB,AppspaceInfoModel,AppspaceUserModel)

// Package testmocks is a generated GoMock package.
package testmocks

import (
	gomock "github.com/golang/mock/gomock"
	sqlx "github.com/jmoiron/sqlx"
	domain "github.com/teleclimber/DropServer/cmd/ds-host/domain"
	reflect "reflect"
)

// MockAppspaceMetaDB is a mock of AppspaceMetaDB interface
type MockAppspaceMetaDB struct {
	ctrl     *gomock.Controller
	recorder *MockAppspaceMetaDBMockRecorder
}

// MockAppspaceMetaDBMockRecorder is the mock recorder for MockAppspaceMetaDB
type MockAppspaceMetaDBMockRecorder struct {
	mock *MockAppspaceMetaDB
}

// NewMockAppspaceMetaDB creates a new mock instance
func NewMockAppspaceMetaDB(ctrl *gomock.Controller) *MockAppspaceMetaDB {
	mock := &MockAppspaceMetaDB{ctrl: ctrl}
	mock.recorder = &MockAppspaceMetaDBMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockAppspaceMetaDB) EXPECT() *MockAppspaceMetaDBMockRecorder {
	return m.recorder
}

// CloseConn mocks base method
func (m *MockAppspaceMetaDB) CloseConn(arg0 domain.AppspaceID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CloseConn", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// CloseConn indicates an expected call of CloseConn
func (mr *MockAppspaceMetaDBMockRecorder) CloseConn(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CloseConn", reflect.TypeOf((*MockAppspaceMetaDB)(nil).CloseConn), arg0)
}

// Create mocks base method
func (m *MockAppspaceMetaDB) Create(arg0 domain.AppspaceID, arg1 int) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Create indicates an expected call of Create
func (mr *MockAppspaceMetaDBMockRecorder) Create(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockAppspaceMetaDB)(nil).Create), arg0, arg1)
}

// GetHandle mocks base method
func (m *MockAppspaceMetaDB) GetHandle(arg0 domain.AppspaceID) (*sqlx.DB, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHandle", arg0)
	ret0, _ := ret[0].(*sqlx.DB)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetHandle indicates an expected call of GetHandle
func (mr *MockAppspaceMetaDBMockRecorder) GetHandle(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHandle", reflect.TypeOf((*MockAppspaceMetaDB)(nil).GetHandle), arg0)
}

// MockAppspaceInfoModel is a mock of AppspaceInfoModel interface
type MockAppspaceInfoModel struct {
	ctrl     *gomock.Controller
	recorder *MockAppspaceInfoModelMockRecorder
}

// MockAppspaceInfoModelMockRecorder is the mock recorder for MockAppspaceInfoModel
type MockAppspaceInfoModelMockRecorder struct {
	mock *MockAppspaceInfoModel
}

// NewMockAppspaceInfoModel creates a new mock instance
func NewMockAppspaceInfoModel(ctrl *gomock.Controller) *MockAppspaceInfoModel {
	mock := &MockAppspaceInfoModel{ctrl: ctrl}
	mock.recorder = &MockAppspaceInfoModelMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockAppspaceInfoModel) EXPECT() *MockAppspaceInfoModelMockRecorder {
	return m.recorder
}

// GetSchema mocks base method
func (m *MockAppspaceInfoModel) GetSchema(arg0 domain.AppspaceID) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSchema", arg0)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSchema indicates an expected call of GetSchema
func (mr *MockAppspaceInfoModelMockRecorder) GetSchema(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSchema", reflect.TypeOf((*MockAppspaceInfoModel)(nil).GetSchema), arg0)
}

// SetSchema mocks base method
func (m *MockAppspaceInfoModel) SetSchema(arg0 domain.AppspaceID, arg1 int) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetSchema", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetSchema indicates an expected call of SetSchema
func (mr *MockAppspaceInfoModelMockRecorder) SetSchema(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetSchema", reflect.TypeOf((*MockAppspaceInfoModel)(nil).SetSchema), arg0, arg1)
}

// MockAppspaceUserModel is a mock of AppspaceUserModel interface
type MockAppspaceUserModel struct {
	ctrl     *gomock.Controller
	recorder *MockAppspaceUserModelMockRecorder
}

// MockAppspaceUserModelMockRecorder is the mock recorder for MockAppspaceUserModel
type MockAppspaceUserModelMockRecorder struct {
	mock *MockAppspaceUserModel
}

// NewMockAppspaceUserModel creates a new mock instance
func NewMockAppspaceUserModel(ctrl *gomock.Controller) *MockAppspaceUserModel {
	mock := &MockAppspaceUserModel{ctrl: ctrl}
	mock.recorder = &MockAppspaceUserModelMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockAppspaceUserModel) EXPECT() *MockAppspaceUserModelMockRecorder {
	return m.recorder
}

// Create mocks base method
func (m *MockAppspaceUserModel) Create(arg0 domain.AppspaceID, arg1, arg2 string) (domain.ProxyID, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0, arg1, arg2)
	ret0, _ := ret[0].(domain.ProxyID)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create
func (mr *MockAppspaceUserModelMockRecorder) Create(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockAppspaceUserModel)(nil).Create), arg0, arg1, arg2)
}

// Delete mocks base method
func (m *MockAppspaceUserModel) Delete(arg0 domain.AppspaceID, arg1 domain.ProxyID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete
func (mr *MockAppspaceUserModelMockRecorder) Delete(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockAppspaceUserModel)(nil).Delete), arg0, arg1)
}

// Get mocks base method
func (m *MockAppspaceUserModel) Get(arg0 domain.AppspaceID, arg1 domain.ProxyID) (domain.AppspaceUser, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0, arg1)
	ret0, _ := ret[0].(domain.AppspaceUser)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get
func (mr *MockAppspaceUserModelMockRecorder) Get(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockAppspaceUserModel)(nil).Get), arg0, arg1)
}

// GetAll mocks base method
func (m *MockAppspaceUserModel) GetAll(arg0 domain.AppspaceID) ([]domain.AppspaceUser, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAll", arg0)
	ret0, _ := ret[0].([]domain.AppspaceUser)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAll indicates an expected call of GetAll
func (mr *MockAppspaceUserModelMockRecorder) GetAll(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAll", reflect.TypeOf((*MockAppspaceUserModel)(nil).GetAll), arg0)
}

// GetByAuth mocks base method
func (m *MockAppspaceUserModel) GetByAuth(arg0 domain.AppspaceID, arg1, arg2 string) (domain.AppspaceUser, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByAuth", arg0, arg1, arg2)
	ret0, _ := ret[0].(domain.AppspaceUser)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByAuth indicates an expected call of GetByAuth
func (mr *MockAppspaceUserModelMockRecorder) GetByAuth(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByAuth", reflect.TypeOf((*MockAppspaceUserModel)(nil).GetByAuth), arg0, arg1, arg2)
}

// UpdateMeta mocks base method
func (m *MockAppspaceUserModel) UpdateMeta(arg0 domain.AppspaceID, arg1 domain.ProxyID, arg2, arg3 string, arg4 []string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateMeta", arg0, arg1, arg2, arg3, arg4)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateMeta indicates an expected call of UpdateMeta
func (mr *MockAppspaceUserModelMockRecorder) UpdateMeta(arg0, arg1, arg2, arg3, arg4 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateMeta", reflect.TypeOf((*MockAppspaceUserModel)(nil).UpdateMeta), arg0, arg1, arg2, arg3, arg4)
}
