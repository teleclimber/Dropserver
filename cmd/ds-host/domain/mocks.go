// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/teleclimber/DropServer/cmd/ds-host/domain (interfaces: MetricsI,SandboxI,V0RouteModel,AppspaceRouteModels,StdInput)

// Package domain is a generated GoMock package.
package domain

import (
	gomock "github.com/golang/mock/gomock"
	twine "github.com/teleclimber/twine-go/twine"
	http "net/http"
	reflect "reflect"
	time "time"
)

// MockMetricsI is a mock of MetricsI interface
type MockMetricsI struct {
	ctrl     *gomock.Controller
	recorder *MockMetricsIMockRecorder
}

// MockMetricsIMockRecorder is the mock recorder for MockMetricsI
type MockMetricsIMockRecorder struct {
	mock *MockMetricsI
}

// NewMockMetricsI creates a new mock instance
func NewMockMetricsI(ctrl *gomock.Controller) *MockMetricsI {
	mock := &MockMetricsI{ctrl: ctrl}
	mock.recorder = &MockMetricsIMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockMetricsI) EXPECT() *MockMetricsIMockRecorder {
	return m.recorder
}

// HostHandleReq mocks base method
func (m *MockMetricsI) HostHandleReq(arg0 time.Time) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "HostHandleReq", arg0)
}

// HostHandleReq indicates an expected call of HostHandleReq
func (mr *MockMetricsIMockRecorder) HostHandleReq(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HostHandleReq", reflect.TypeOf((*MockMetricsI)(nil).HostHandleReq), arg0)
}

// MockSandboxI is a mock of SandboxI interface
type MockSandboxI struct {
	ctrl     *gomock.Controller
	recorder *MockSandboxIMockRecorder
}

// MockSandboxIMockRecorder is the mock recorder for MockSandboxI
type MockSandboxIMockRecorder struct {
	mock *MockSandboxI
}

// NewMockSandboxI creates a new mock instance
func NewMockSandboxI(ctrl *gomock.Controller) *MockSandboxI {
	mock := &MockSandboxI{ctrl: ctrl}
	mock.recorder = &MockSandboxIMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockSandboxI) EXPECT() *MockSandboxIMockRecorder {
	return m.recorder
}

// AppVersion mocks base method
func (m *MockSandboxI) AppVersion() *AppVersion {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AppVersion")
	ret0, _ := ret[0].(*AppVersion)
	return ret0
}

// AppVersion indicates an expected call of AppVersion
func (mr *MockSandboxIMockRecorder) AppVersion() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AppVersion", reflect.TypeOf((*MockSandboxI)(nil).AppVersion))
}

// AppspaceID mocks base method
func (m *MockSandboxI) AppspaceID() NullAppspaceID {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AppspaceID")
	ret0, _ := ret[0].(NullAppspaceID)
	return ret0
}

// AppspaceID indicates an expected call of AppspaceID
func (mr *MockSandboxIMockRecorder) AppspaceID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AppspaceID", reflect.TypeOf((*MockSandboxI)(nil).AppspaceID))
}

// ExecFn mocks base method
func (m *MockSandboxI) ExecFn(arg0 AppspaceRouteHandler) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ExecFn", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// ExecFn indicates an expected call of ExecFn
func (mr *MockSandboxIMockRecorder) ExecFn(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ExecFn", reflect.TypeOf((*MockSandboxI)(nil).ExecFn), arg0)
}

// GetTransport mocks base method
func (m *MockSandboxI) GetTransport() http.RoundTripper {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTransport")
	ret0, _ := ret[0].(http.RoundTripper)
	return ret0
}

// GetTransport indicates an expected call of GetTransport
func (mr *MockSandboxIMockRecorder) GetTransport() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTransport", reflect.TypeOf((*MockSandboxI)(nil).GetTransport))
}

// Graceful mocks base method
func (m *MockSandboxI) Graceful() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Graceful")
}

// Graceful indicates an expected call of Graceful
func (mr *MockSandboxIMockRecorder) Graceful() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Graceful", reflect.TypeOf((*MockSandboxI)(nil).Graceful))
}

// Kill mocks base method
func (m *MockSandboxI) Kill() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Kill")
}

// Kill indicates an expected call of Kill
func (mr *MockSandboxIMockRecorder) Kill() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Kill", reflect.TypeOf((*MockSandboxI)(nil).Kill))
}

// LastActive mocks base method
func (m *MockSandboxI) LastActive() time.Time {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LastActive")
	ret0, _ := ret[0].(time.Time)
	return ret0
}

// LastActive indicates an expected call of LastActive
func (mr *MockSandboxIMockRecorder) LastActive() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LastActive", reflect.TypeOf((*MockSandboxI)(nil).LastActive))
}

// NewTask mocks base method
func (m *MockSandboxI) NewTask() chan struct{} {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewTask")
	ret0, _ := ret[0].(chan struct{})
	return ret0
}

// NewTask indicates an expected call of NewTask
func (mr *MockSandboxIMockRecorder) NewTask() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewTask", reflect.TypeOf((*MockSandboxI)(nil).NewTask))
}

// Operation mocks base method
func (m *MockSandboxI) Operation() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Operation")
	ret0, _ := ret[0].(string)
	return ret0
}

// Operation indicates an expected call of Operation
func (mr *MockSandboxIMockRecorder) Operation() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Operation", reflect.TypeOf((*MockSandboxI)(nil).Operation))
}

// OwnerID mocks base method
func (m *MockSandboxI) OwnerID() UserID {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "OwnerID")
	ret0, _ := ret[0].(UserID)
	return ret0
}

// OwnerID indicates an expected call of OwnerID
func (mr *MockSandboxIMockRecorder) OwnerID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OwnerID", reflect.TypeOf((*MockSandboxI)(nil).OwnerID))
}

// SendMessage mocks base method
func (m *MockSandboxI) SendMessage(arg0, arg1 int, arg2 []byte) (twine.SentMessageI, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendMessage", arg0, arg1, arg2)
	ret0, _ := ret[0].(twine.SentMessageI)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SendMessage indicates an expected call of SendMessage
func (mr *MockSandboxIMockRecorder) SendMessage(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMessage", reflect.TypeOf((*MockSandboxI)(nil).SendMessage), arg0, arg1, arg2)
}

// Start mocks base method
func (m *MockSandboxI) Start() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Start")
}

// Start indicates an expected call of Start
func (mr *MockSandboxIMockRecorder) Start() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockSandboxI)(nil).Start))
}

// Status mocks base method
func (m *MockSandboxI) Status() SandboxStatus {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Status")
	ret0, _ := ret[0].(SandboxStatus)
	return ret0
}

// Status indicates an expected call of Status
func (mr *MockSandboxIMockRecorder) Status() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Status", reflect.TypeOf((*MockSandboxI)(nil).Status))
}

// TiedUp mocks base method
func (m *MockSandboxI) TiedUp() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TiedUp")
	ret0, _ := ret[0].(bool)
	return ret0
}

// TiedUp indicates an expected call of TiedUp
func (mr *MockSandboxIMockRecorder) TiedUp() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TiedUp", reflect.TypeOf((*MockSandboxI)(nil).TiedUp))
}

// WaitFor mocks base method
func (m *MockSandboxI) WaitFor(arg0 SandboxStatus) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "WaitFor", arg0)
}

// WaitFor indicates an expected call of WaitFor
func (mr *MockSandboxIMockRecorder) WaitFor(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WaitFor", reflect.TypeOf((*MockSandboxI)(nil).WaitFor), arg0)
}

// MockV0RouteModel is a mock of V0RouteModel interface
type MockV0RouteModel struct {
	ctrl     *gomock.Controller
	recorder *MockV0RouteModelMockRecorder
}

// MockV0RouteModelMockRecorder is the mock recorder for MockV0RouteModel
type MockV0RouteModelMockRecorder struct {
	mock *MockV0RouteModel
}

// NewMockV0RouteModel creates a new mock instance
func NewMockV0RouteModel(ctrl *gomock.Controller) *MockV0RouteModel {
	mock := &MockV0RouteModel{ctrl: ctrl}
	mock.recorder = &MockV0RouteModelMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockV0RouteModel) EXPECT() *MockV0RouteModelMockRecorder {
	return m.recorder
}

// Create mocks base method
func (m *MockV0RouteModel) Create(arg0 []string, arg1 string, arg2 AppspaceRouteAuth, arg3 AppspaceRouteHandler) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// Create indicates an expected call of Create
func (mr *MockV0RouteModelMockRecorder) Create(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockV0RouteModel)(nil).Create), arg0, arg1, arg2, arg3)
}

// Delete mocks base method
func (m *MockV0RouteModel) Delete(arg0 []string, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete
func (mr *MockV0RouteModelMockRecorder) Delete(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockV0RouteModel)(nil).Delete), arg0, arg1)
}

// Get mocks base method
func (m *MockV0RouteModel) Get(arg0 []string, arg1 string) (*[]AppspaceRouteConfig, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0, arg1)
	ret0, _ := ret[0].(*[]AppspaceRouteConfig)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get
func (mr *MockV0RouteModelMockRecorder) Get(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockV0RouteModel)(nil).Get), arg0, arg1)
}

// GetAll mocks base method
func (m *MockV0RouteModel) GetAll() (*[]AppspaceRouteConfig, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAll")
	ret0, _ := ret[0].(*[]AppspaceRouteConfig)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAll indicates an expected call of GetAll
func (mr *MockV0RouteModelMockRecorder) GetAll() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAll", reflect.TypeOf((*MockV0RouteModel)(nil).GetAll))
}

// GetPath mocks base method
func (m *MockV0RouteModel) GetPath(arg0 string) (*[]AppspaceRouteConfig, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPath", arg0)
	ret0, _ := ret[0].(*[]AppspaceRouteConfig)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPath indicates an expected call of GetPath
func (mr *MockV0RouteModelMockRecorder) GetPath(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPath", reflect.TypeOf((*MockV0RouteModel)(nil).GetPath), arg0)
}

// HandleMessage mocks base method
func (m *MockV0RouteModel) HandleMessage(arg0 twine.ReceivedMessageI) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "HandleMessage", arg0)
}

// HandleMessage indicates an expected call of HandleMessage
func (mr *MockV0RouteModelMockRecorder) HandleMessage(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HandleMessage", reflect.TypeOf((*MockV0RouteModel)(nil).HandleMessage), arg0)
}

// Match mocks base method
func (m *MockV0RouteModel) Match(arg0, arg1 string) (*AppspaceRouteConfig, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Match", arg0, arg1)
	ret0, _ := ret[0].(*AppspaceRouteConfig)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Match indicates an expected call of Match
func (mr *MockV0RouteModelMockRecorder) Match(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Match", reflect.TypeOf((*MockV0RouteModel)(nil).Match), arg0, arg1)
}

// MockAppspaceRouteModels is a mock of AppspaceRouteModels interface
type MockAppspaceRouteModels struct {
	ctrl     *gomock.Controller
	recorder *MockAppspaceRouteModelsMockRecorder
}

// MockAppspaceRouteModelsMockRecorder is the mock recorder for MockAppspaceRouteModels
type MockAppspaceRouteModelsMockRecorder struct {
	mock *MockAppspaceRouteModels
}

// NewMockAppspaceRouteModels creates a new mock instance
func NewMockAppspaceRouteModels(ctrl *gomock.Controller) *MockAppspaceRouteModels {
	mock := &MockAppspaceRouteModels{ctrl: ctrl}
	mock.recorder = &MockAppspaceRouteModelsMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockAppspaceRouteModels) EXPECT() *MockAppspaceRouteModelsMockRecorder {
	return m.recorder
}

// GetV0 mocks base method
func (m *MockAppspaceRouteModels) GetV0(arg0 AppspaceID) V0RouteModel {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetV0", arg0)
	ret0, _ := ret[0].(V0RouteModel)
	return ret0
}

// GetV0 indicates an expected call of GetV0
func (mr *MockAppspaceRouteModelsMockRecorder) GetV0(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetV0", reflect.TypeOf((*MockAppspaceRouteModels)(nil).GetV0), arg0)
}

// MockStdInput is a mock of StdInput interface
type MockStdInput struct {
	ctrl     *gomock.Controller
	recorder *MockStdInputMockRecorder
}

// MockStdInputMockRecorder is the mock recorder for MockStdInput
type MockStdInputMockRecorder struct {
	mock *MockStdInput
}

// NewMockStdInput creates a new mock instance
func NewMockStdInput(ctrl *gomock.Controller) *MockStdInput {
	mock := &MockStdInput{ctrl: ctrl}
	mock.recorder = &MockStdInputMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockStdInput) EXPECT() *MockStdInputMockRecorder {
	return m.recorder
}

// ReadLine mocks base method
func (m *MockStdInput) ReadLine(arg0 string) string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadLine", arg0)
	ret0, _ := ret[0].(string)
	return ret0
}

// ReadLine indicates an expected call of ReadLine
func (mr *MockStdInputMockRecorder) ReadLine(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadLine", reflect.TypeOf((*MockStdInput)(nil).ReadLine), arg0)
}
