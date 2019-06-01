// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/teleclimber/DropServer/cmd/ds-host/domain (interfaces: DBManagerI,LogCLientI,MetricsI,SandboxI,SandboxManagerI,RouteHandler,CookieModel,UserModel,AppModel,AppspaceModel,ASRoutesModel,TrustedClientI,Authenticator,Validator,Views)

// Package domain is a generated GoMock package.
package domain

import (
	gomock "github.com/golang/mock/gomock"
	http "net/http"
	reflect "reflect"
	time "time"
)

// MockDBManagerI is a mock of DBManagerI interface
type MockDBManagerI struct {
	ctrl     *gomock.Controller
	recorder *MockDBManagerIMockRecorder
}

// MockDBManagerIMockRecorder is the mock recorder for MockDBManagerI
type MockDBManagerIMockRecorder struct {
	mock *MockDBManagerI
}

// NewMockDBManagerI creates a new mock instance
func NewMockDBManagerI(ctrl *gomock.Controller) *MockDBManagerI {
	mock := &MockDBManagerI{ctrl: ctrl}
	mock.recorder = &MockDBManagerIMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockDBManagerI) EXPECT() *MockDBManagerIMockRecorder {
	return m.recorder
}

// GetHandle mocks base method
func (m *MockDBManagerI) GetHandle() *DB {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHandle")
	ret0, _ := ret[0].(*DB)
	return ret0
}

// GetHandle indicates an expected call of GetHandle
func (mr *MockDBManagerIMockRecorder) GetHandle() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHandle", reflect.TypeOf((*MockDBManagerI)(nil).GetHandle))
}

// GetSchema mocks base method
func (m *MockDBManagerI) GetSchema() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSchema")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetSchema indicates an expected call of GetSchema
func (mr *MockDBManagerIMockRecorder) GetSchema() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSchema", reflect.TypeOf((*MockDBManagerI)(nil).GetSchema))
}

// Open mocks base method
func (m *MockDBManagerI) Open() (*DB, Error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Open")
	ret0, _ := ret[0].(*DB)
	ret1, _ := ret[1].(Error)
	return ret0, ret1
}

// Open indicates an expected call of Open
func (mr *MockDBManagerIMockRecorder) Open() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Open", reflect.TypeOf((*MockDBManagerI)(nil).Open))
}

// SetSchema mocks base method
func (m *MockDBManagerI) SetSchema(arg0 string) Error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetSchema", arg0)
	ret0, _ := ret[0].(Error)
	return ret0
}

// SetSchema indicates an expected call of SetSchema
func (mr *MockDBManagerIMockRecorder) SetSchema(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetSchema", reflect.TypeOf((*MockDBManagerI)(nil).SetSchema), arg0)
}

// MockLogCLientI is a mock of LogCLientI interface
type MockLogCLientI struct {
	ctrl     *gomock.Controller
	recorder *MockLogCLientIMockRecorder
}

// MockLogCLientIMockRecorder is the mock recorder for MockLogCLientI
type MockLogCLientIMockRecorder struct {
	mock *MockLogCLientI
}

// NewMockLogCLientI creates a new mock instance
func NewMockLogCLientI(ctrl *gomock.Controller) *MockLogCLientI {
	mock := &MockLogCLientI{ctrl: ctrl}
	mock.recorder = &MockLogCLientIMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockLogCLientI) EXPECT() *MockLogCLientIMockRecorder {
	return m.recorder
}

// Log mocks base method
func (m *MockLogCLientI) Log(arg0 LogLevel, arg1 map[string]string, arg2 string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Log", arg0, arg1, arg2)
}

// Log indicates an expected call of Log
func (mr *MockLogCLientIMockRecorder) Log(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Log", reflect.TypeOf((*MockLogCLientI)(nil).Log), arg0, arg1, arg2)
}

// NewSandboxLogClient mocks base method
func (m *MockLogCLientI) NewSandboxLogClient(arg0 string) LogCLientI {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewSandboxLogClient", arg0)
	ret0, _ := ret[0].(LogCLientI)
	return ret0
}

// NewSandboxLogClient indicates an expected call of NewSandboxLogClient
func (mr *MockLogCLientIMockRecorder) NewSandboxLogClient(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewSandboxLogClient", reflect.TypeOf((*MockLogCLientI)(nil).NewSandboxLogClient), arg0)
}

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

// GetAddress mocks base method
func (m *MockSandboxI) GetAddress() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAddress")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetAddress indicates an expected call of GetAddress
func (mr *MockSandboxIMockRecorder) GetAddress() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAddress", reflect.TypeOf((*MockSandboxI)(nil).GetAddress))
}

// GetLogClient mocks base method
func (m *MockSandboxI) GetLogClient() LogCLientI {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLogClient")
	ret0, _ := ret[0].(LogCLientI)
	return ret0
}

// GetLogClient indicates an expected call of GetLogClient
func (mr *MockSandboxIMockRecorder) GetLogClient() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLogClient", reflect.TypeOf((*MockSandboxI)(nil).GetLogClient))
}

// GetName mocks base method
func (m *MockSandboxI) GetName() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetName")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetName indicates an expected call of GetName
func (mr *MockSandboxIMockRecorder) GetName() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetName", reflect.TypeOf((*MockSandboxI)(nil).GetName))
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

// TaskBegin mocks base method
func (m *MockSandboxI) TaskBegin() chan bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TaskBegin")
	ret0, _ := ret[0].(chan bool)
	return ret0
}

// TaskBegin indicates an expected call of TaskBegin
func (mr *MockSandboxIMockRecorder) TaskBegin() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TaskBegin", reflect.TypeOf((*MockSandboxI)(nil).TaskBegin))
}

// MockSandboxManagerI is a mock of SandboxManagerI interface
type MockSandboxManagerI struct {
	ctrl     *gomock.Controller
	recorder *MockSandboxManagerIMockRecorder
}

// MockSandboxManagerIMockRecorder is the mock recorder for MockSandboxManagerI
type MockSandboxManagerIMockRecorder struct {
	mock *MockSandboxManagerI
}

// NewMockSandboxManagerI creates a new mock instance
func NewMockSandboxManagerI(ctrl *gomock.Controller) *MockSandboxManagerI {
	mock := &MockSandboxManagerI{ctrl: ctrl}
	mock.recorder = &MockSandboxManagerIMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockSandboxManagerI) EXPECT() *MockSandboxManagerIMockRecorder {
	return m.recorder
}

// GetForAppSpace mocks base method
func (m *MockSandboxManagerI) GetForAppSpace(arg0, arg1 string) chan SandboxI {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetForAppSpace", arg0, arg1)
	ret0, _ := ret[0].(chan SandboxI)
	return ret0
}

// GetForAppSpace indicates an expected call of GetForAppSpace
func (mr *MockSandboxManagerIMockRecorder) GetForAppSpace(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetForAppSpace", reflect.TypeOf((*MockSandboxManagerI)(nil).GetForAppSpace), arg0, arg1)
}

// MockRouteHandler is a mock of RouteHandler interface
type MockRouteHandler struct {
	ctrl     *gomock.Controller
	recorder *MockRouteHandlerMockRecorder
}

// MockRouteHandlerMockRecorder is the mock recorder for MockRouteHandler
type MockRouteHandlerMockRecorder struct {
	mock *MockRouteHandler
}

// NewMockRouteHandler creates a new mock instance
func NewMockRouteHandler(ctrl *gomock.Controller) *MockRouteHandler {
	mock := &MockRouteHandler{ctrl: ctrl}
	mock.recorder = &MockRouteHandlerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockRouteHandler) EXPECT() *MockRouteHandlerMockRecorder {
	return m.recorder
}

// ServeHTTP mocks base method
func (m *MockRouteHandler) ServeHTTP(arg0 http.ResponseWriter, arg1 *http.Request, arg2 *AppspaceRouteData) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "ServeHTTP", arg0, arg1, arg2)
}

// ServeHTTP indicates an expected call of ServeHTTP
func (mr *MockRouteHandlerMockRecorder) ServeHTTP(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ServeHTTP", reflect.TypeOf((*MockRouteHandler)(nil).ServeHTTP), arg0, arg1, arg2)
}

// MockCookieModel is a mock of CookieModel interface
type MockCookieModel struct {
	ctrl     *gomock.Controller
	recorder *MockCookieModelMockRecorder
}

// MockCookieModelMockRecorder is the mock recorder for MockCookieModel
type MockCookieModelMockRecorder struct {
	mock *MockCookieModel
}

// NewMockCookieModel creates a new mock instance
func NewMockCookieModel(ctrl *gomock.Controller) *MockCookieModel {
	mock := &MockCookieModel{ctrl: ctrl}
	mock.recorder = &MockCookieModelMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockCookieModel) EXPECT() *MockCookieModelMockRecorder {
	return m.recorder
}

// Create mocks base method
func (m *MockCookieModel) Create(arg0 Cookie) (string, Error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(Error)
	return ret0, ret1
}

// Create indicates an expected call of Create
func (mr *MockCookieModelMockRecorder) Create(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockCookieModel)(nil).Create), arg0)
}

// Get mocks base method
func (m *MockCookieModel) Get(arg0 string) (*Cookie, Error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0)
	ret0, _ := ret[0].(*Cookie)
	ret1, _ := ret[1].(Error)
	return ret0, ret1
}

// Get indicates an expected call of Get
func (mr *MockCookieModelMockRecorder) Get(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockCookieModel)(nil).Get), arg0)
}

// PrepareStatements mocks base method
func (m *MockCookieModel) PrepareStatements() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "PrepareStatements")
}

// PrepareStatements indicates an expected call of PrepareStatements
func (mr *MockCookieModelMockRecorder) PrepareStatements() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PrepareStatements", reflect.TypeOf((*MockCookieModel)(nil).PrepareStatements))
}

// UpdateExpires mocks base method
func (m *MockCookieModel) UpdateExpires(arg0 string, arg1 time.Time) Error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateExpires", arg0, arg1)
	ret0, _ := ret[0].(Error)
	return ret0
}

// UpdateExpires indicates an expected call of UpdateExpires
func (mr *MockCookieModelMockRecorder) UpdateExpires(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateExpires", reflect.TypeOf((*MockCookieModel)(nil).UpdateExpires), arg0, arg1)
}

// MockUserModel is a mock of UserModel interface
type MockUserModel struct {
	ctrl     *gomock.Controller
	recorder *MockUserModelMockRecorder
}

// MockUserModelMockRecorder is the mock recorder for MockUserModel
type MockUserModelMockRecorder struct {
	mock *MockUserModel
}

// NewMockUserModel creates a new mock instance
func NewMockUserModel(ctrl *gomock.Controller) *MockUserModel {
	mock := &MockUserModel{ctrl: ctrl}
	mock.recorder = &MockUserModelMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockUserModel) EXPECT() *MockUserModelMockRecorder {
	return m.recorder
}

// Create mocks base method
func (m *MockUserModel) Create(arg0, arg1 string) (*User, Error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0, arg1)
	ret0, _ := ret[0].(*User)
	ret1, _ := ret[1].(Error)
	return ret0, ret1
}

// Create indicates an expected call of Create
func (mr *MockUserModelMockRecorder) Create(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockUserModel)(nil).Create), arg0, arg1)
}

// DeleteAdmin mocks base method
func (m *MockUserModel) DeleteAdmin(arg0 UserID) Error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteAdmin", arg0)
	ret0, _ := ret[0].(Error)
	return ret0
}

// DeleteAdmin indicates an expected call of DeleteAdmin
func (mr *MockUserModelMockRecorder) DeleteAdmin(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteAdmin", reflect.TypeOf((*MockUserModel)(nil).DeleteAdmin), arg0)
}

// GetFromEmail mocks base method
func (m *MockUserModel) GetFromEmail(arg0 string) (*User, Error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFromEmail", arg0)
	ret0, _ := ret[0].(*User)
	ret1, _ := ret[1].(Error)
	return ret0, ret1
}

// GetFromEmail indicates an expected call of GetFromEmail
func (mr *MockUserModelMockRecorder) GetFromEmail(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFromEmail", reflect.TypeOf((*MockUserModel)(nil).GetFromEmail), arg0)
}

// GetFromEmailPassword mocks base method
func (m *MockUserModel) GetFromEmailPassword(arg0, arg1 string) (*User, Error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFromEmailPassword", arg0, arg1)
	ret0, _ := ret[0].(*User)
	ret1, _ := ret[1].(Error)
	return ret0, ret1
}

// GetFromEmailPassword indicates an expected call of GetFromEmailPassword
func (mr *MockUserModelMockRecorder) GetFromEmailPassword(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFromEmailPassword", reflect.TypeOf((*MockUserModel)(nil).GetFromEmailPassword), arg0, arg1)
}

// GetFromID mocks base method
func (m *MockUserModel) GetFromID(arg0 UserID) (*User, Error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFromID", arg0)
	ret0, _ := ret[0].(*User)
	ret1, _ := ret[1].(Error)
	return ret0, ret1
}

// GetFromID indicates an expected call of GetFromID
func (mr *MockUserModelMockRecorder) GetFromID(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFromID", reflect.TypeOf((*MockUserModel)(nil).GetFromID), arg0)
}

// IsAdmin mocks base method
func (m *MockUserModel) IsAdmin(arg0 UserID) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsAdmin", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsAdmin indicates an expected call of IsAdmin
func (mr *MockUserModelMockRecorder) IsAdmin(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsAdmin", reflect.TypeOf((*MockUserModel)(nil).IsAdmin), arg0)
}

// MakeAdmin mocks base method
func (m *MockUserModel) MakeAdmin(arg0 UserID) Error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MakeAdmin", arg0)
	ret0, _ := ret[0].(Error)
	return ret0
}

// MakeAdmin indicates an expected call of MakeAdmin
func (mr *MockUserModelMockRecorder) MakeAdmin(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MakeAdmin", reflect.TypeOf((*MockUserModel)(nil).MakeAdmin), arg0)
}

// PrepareStatements mocks base method
func (m *MockUserModel) PrepareStatements() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "PrepareStatements")
}

// PrepareStatements indicates an expected call of PrepareStatements
func (mr *MockUserModelMockRecorder) PrepareStatements() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PrepareStatements", reflect.TypeOf((*MockUserModel)(nil).PrepareStatements))
}

// MockAppModel is a mock of AppModel interface
type MockAppModel struct {
	ctrl     *gomock.Controller
	recorder *MockAppModelMockRecorder
}

// MockAppModelMockRecorder is the mock recorder for MockAppModel
type MockAppModelMockRecorder struct {
	mock *MockAppModel
}

// NewMockAppModel creates a new mock instance
func NewMockAppModel(ctrl *gomock.Controller) *MockAppModel {
	mock := &MockAppModel{ctrl: ctrl}
	mock.recorder = &MockAppModelMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockAppModel) EXPECT() *MockAppModelMockRecorder {
	return m.recorder
}

// Create mocks base method
func (m *MockAppModel) Create(arg0 UserID, arg1 string) (*App, Error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0, arg1)
	ret0, _ := ret[0].(*App)
	ret1, _ := ret[1].(Error)
	return ret0, ret1
}

// Create indicates an expected call of Create
func (mr *MockAppModelMockRecorder) Create(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockAppModel)(nil).Create), arg0, arg1)
}

// CreateVersion mocks base method
func (m *MockAppModel) CreateVersion(arg0 AppID, arg1 Version, arg2 string) (*AppVersion, Error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateVersion", arg0, arg1, arg2)
	ret0, _ := ret[0].(*AppVersion)
	ret1, _ := ret[1].(Error)
	return ret0, ret1
}

// CreateVersion indicates an expected call of CreateVersion
func (mr *MockAppModelMockRecorder) CreateVersion(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateVersion", reflect.TypeOf((*MockAppModel)(nil).CreateVersion), arg0, arg1, arg2)
}

// GetFromID mocks base method
func (m *MockAppModel) GetFromID(arg0 AppID) (*App, Error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFromID", arg0)
	ret0, _ := ret[0].(*App)
	ret1, _ := ret[1].(Error)
	return ret0, ret1
}

// GetFromID indicates an expected call of GetFromID
func (mr *MockAppModelMockRecorder) GetFromID(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFromID", reflect.TypeOf((*MockAppModel)(nil).GetFromID), arg0)
}

// GetVersion mocks base method
func (m *MockAppModel) GetVersion(arg0 AppID, arg1 Version) (*AppVersion, Error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVersion", arg0, arg1)
	ret0, _ := ret[0].(*AppVersion)
	ret1, _ := ret[1].(Error)
	return ret0, ret1
}

// GetVersion indicates an expected call of GetVersion
func (mr *MockAppModelMockRecorder) GetVersion(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVersion", reflect.TypeOf((*MockAppModel)(nil).GetVersion), arg0, arg1)
}

// MockAppspaceModel is a mock of AppspaceModel interface
type MockAppspaceModel struct {
	ctrl     *gomock.Controller
	recorder *MockAppspaceModelMockRecorder
}

// MockAppspaceModelMockRecorder is the mock recorder for MockAppspaceModel
type MockAppspaceModelMockRecorder struct {
	mock *MockAppspaceModel
}

// NewMockAppspaceModel creates a new mock instance
func NewMockAppspaceModel(ctrl *gomock.Controller) *MockAppspaceModel {
	mock := &MockAppspaceModel{ctrl: ctrl}
	mock.recorder = &MockAppspaceModelMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockAppspaceModel) EXPECT() *MockAppspaceModelMockRecorder {
	return m.recorder
}

// Create mocks base method
func (m *MockAppspaceModel) Create(arg0 UserID, arg1 AppID, arg2 Version, arg3 string) (*Appspace, Error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(*Appspace)
	ret1, _ := ret[1].(Error)
	return ret0, ret1
}

// Create indicates an expected call of Create
func (mr *MockAppspaceModelMockRecorder) Create(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockAppspaceModel)(nil).Create), arg0, arg1, arg2, arg3)
}

// GetFromID mocks base method
func (m *MockAppspaceModel) GetFromID(arg0 AppspaceID) (*Appspace, Error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFromID", arg0)
	ret0, _ := ret[0].(*Appspace)
	ret1, _ := ret[1].(Error)
	return ret0, ret1
}

// GetFromID indicates an expected call of GetFromID
func (mr *MockAppspaceModelMockRecorder) GetFromID(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFromID", reflect.TypeOf((*MockAppspaceModel)(nil).GetFromID), arg0)
}

// GetFromSubdomain mocks base method
func (m *MockAppspaceModel) GetFromSubdomain(arg0 string) (*Appspace, Error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFromSubdomain", arg0)
	ret0, _ := ret[0].(*Appspace)
	ret1, _ := ret[1].(Error)
	return ret0, ret1
}

// GetFromSubdomain indicates an expected call of GetFromSubdomain
func (mr *MockAppspaceModelMockRecorder) GetFromSubdomain(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFromSubdomain", reflect.TypeOf((*MockAppspaceModel)(nil).GetFromSubdomain), arg0)
}

// MockASRoutesModel is a mock of ASRoutesModel interface
type MockASRoutesModel struct {
	ctrl     *gomock.Controller
	recorder *MockASRoutesModelMockRecorder
}

// MockASRoutesModelMockRecorder is the mock recorder for MockASRoutesModel
type MockASRoutesModelMockRecorder struct {
	mock *MockASRoutesModel
}

// NewMockASRoutesModel creates a new mock instance
func NewMockASRoutesModel(ctrl *gomock.Controller) *MockASRoutesModel {
	mock := &MockASRoutesModel{ctrl: ctrl}
	mock.recorder = &MockASRoutesModelMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockASRoutesModel) EXPECT() *MockASRoutesModelMockRecorder {
	return m.recorder
}

// GetRouteConfig mocks base method
func (m *MockASRoutesModel) GetRouteConfig(arg0 AppVersion, arg1, arg2 string) (*RouteConfig, Error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRouteConfig", arg0, arg1, arg2)
	ret0, _ := ret[0].(*RouteConfig)
	ret1, _ := ret[1].(Error)
	return ret0, ret1
}

// GetRouteConfig indicates an expected call of GetRouteConfig
func (mr *MockASRoutesModelMockRecorder) GetRouteConfig(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRouteConfig", reflect.TypeOf((*MockASRoutesModel)(nil).GetRouteConfig), arg0, arg1, arg2)
}

// MockTrustedClientI is a mock of TrustedClientI interface
type MockTrustedClientI struct {
	ctrl     *gomock.Controller
	recorder *MockTrustedClientIMockRecorder
}

// MockTrustedClientIMockRecorder is the mock recorder for MockTrustedClientI
type MockTrustedClientIMockRecorder struct {
	mock *MockTrustedClientI
}

// NewMockTrustedClientI creates a new mock instance
func NewMockTrustedClientI(ctrl *gomock.Controller) *MockTrustedClientI {
	mock := &MockTrustedClientI{ctrl: ctrl}
	mock.recorder = &MockTrustedClientIMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockTrustedClientI) EXPECT() *MockTrustedClientIMockRecorder {
	return m.recorder
}

// GetAppMeta mocks base method
func (m *MockTrustedClientI) GetAppMeta(arg0 *TrustedGetAppMeta) (*TrustedGetAppMetaReply, Error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAppMeta", arg0)
	ret0, _ := ret[0].(*TrustedGetAppMetaReply)
	ret1, _ := ret[1].(Error)
	return ret0, ret1
}

// GetAppMeta indicates an expected call of GetAppMeta
func (mr *MockTrustedClientIMockRecorder) GetAppMeta(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAppMeta", reflect.TypeOf((*MockTrustedClientI)(nil).GetAppMeta), arg0)
}

// Init mocks base method
func (m *MockTrustedClientI) Init(arg0 string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Init", arg0)
}

// Init indicates an expected call of Init
func (mr *MockTrustedClientIMockRecorder) Init(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Init", reflect.TypeOf((*MockTrustedClientI)(nil).Init), arg0)
}

// SaveAppFiles mocks base method
func (m *MockTrustedClientI) SaveAppFiles(arg0 *TrustedSaveAppFiles) (*TrustedSaveAppFilesReply, Error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveAppFiles", arg0)
	ret0, _ := ret[0].(*TrustedSaveAppFilesReply)
	ret1, _ := ret[1].(Error)
	return ret0, ret1
}

// SaveAppFiles indicates an expected call of SaveAppFiles
func (mr *MockTrustedClientIMockRecorder) SaveAppFiles(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveAppFiles", reflect.TypeOf((*MockTrustedClientI)(nil).SaveAppFiles), arg0)
}

// MockAuthenticator is a mock of Authenticator interface
type MockAuthenticator struct {
	ctrl     *gomock.Controller
	recorder *MockAuthenticatorMockRecorder
}

// MockAuthenticatorMockRecorder is the mock recorder for MockAuthenticator
type MockAuthenticatorMockRecorder struct {
	mock *MockAuthenticator
}

// NewMockAuthenticator creates a new mock instance
func NewMockAuthenticator(ctrl *gomock.Controller) *MockAuthenticator {
	mock := &MockAuthenticator{ctrl: ctrl}
	mock.recorder = &MockAuthenticatorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockAuthenticator) EXPECT() *MockAuthenticatorMockRecorder {
	return m.recorder
}

// GetForAccount mocks base method
func (m *MockAuthenticator) GetForAccount(arg0 http.ResponseWriter, arg1 *http.Request, arg2 *AppspaceRouteData) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetForAccount", arg0, arg1, arg2)
	ret0, _ := ret[0].(bool)
	return ret0
}

// GetForAccount indicates an expected call of GetForAccount
func (mr *MockAuthenticatorMockRecorder) GetForAccount(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetForAccount", reflect.TypeOf((*MockAuthenticator)(nil).GetForAccount), arg0, arg1, arg2)
}

// SetForAccount mocks base method
func (m *MockAuthenticator) SetForAccount(arg0 http.ResponseWriter, arg1 UserID) Error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetForAccount", arg0, arg1)
	ret0, _ := ret[0].(Error)
	return ret0
}

// SetForAccount indicates an expected call of SetForAccount
func (mr *MockAuthenticatorMockRecorder) SetForAccount(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetForAccount", reflect.TypeOf((*MockAuthenticator)(nil).SetForAccount), arg0, arg1)
}

// MockValidator is a mock of Validator interface
type MockValidator struct {
	ctrl     *gomock.Controller
	recorder *MockValidatorMockRecorder
}

// MockValidatorMockRecorder is the mock recorder for MockValidator
type MockValidatorMockRecorder struct {
	mock *MockValidator
}

// NewMockValidator creates a new mock instance
func NewMockValidator(ctrl *gomock.Controller) *MockValidator {
	mock := &MockValidator{ctrl: ctrl}
	mock.recorder = &MockValidatorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockValidator) EXPECT() *MockValidatorMockRecorder {
	return m.recorder
}

// Email mocks base method
func (m *MockValidator) Email(arg0 string) Error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Email", arg0)
	ret0, _ := ret[0].(Error)
	return ret0
}

// Email indicates an expected call of Email
func (mr *MockValidatorMockRecorder) Email(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Email", reflect.TypeOf((*MockValidator)(nil).Email), arg0)
}

// Init mocks base method
func (m *MockValidator) Init() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Init")
}

// Init indicates an expected call of Init
func (mr *MockValidatorMockRecorder) Init() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Init", reflect.TypeOf((*MockValidator)(nil).Init))
}

// Password mocks base method
func (m *MockValidator) Password(arg0 string) Error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Password", arg0)
	ret0, _ := ret[0].(Error)
	return ret0
}

// Password indicates an expected call of Password
func (mr *MockValidatorMockRecorder) Password(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Password", reflect.TypeOf((*MockValidator)(nil).Password), arg0)
}

// MockViews is a mock of Views interface
type MockViews struct {
	ctrl     *gomock.Controller
	recorder *MockViewsMockRecorder
}

// MockViewsMockRecorder is the mock recorder for MockViews
type MockViewsMockRecorder struct {
	mock *MockViews
}

// NewMockViews creates a new mock instance
func NewMockViews(ctrl *gomock.Controller) *MockViews {
	mock := &MockViews{ctrl: ctrl}
	mock.recorder = &MockViewsMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockViews) EXPECT() *MockViewsMockRecorder {
	return m.recorder
}

// Login mocks base method
func (m *MockViews) Login(arg0 http.ResponseWriter, arg1 LoginViewData) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Login", arg0, arg1)
}

// Login indicates an expected call of Login
func (mr *MockViewsMockRecorder) Login(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Login", reflect.TypeOf((*MockViews)(nil).Login), arg0, arg1)
}

// PrepareTemplates mocks base method
func (m *MockViews) PrepareTemplates() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "PrepareTemplates")
}

// PrepareTemplates indicates an expected call of PrepareTemplates
func (mr *MockViewsMockRecorder) PrepareTemplates() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PrepareTemplates", reflect.TypeOf((*MockViews)(nil).PrepareTemplates))
}
