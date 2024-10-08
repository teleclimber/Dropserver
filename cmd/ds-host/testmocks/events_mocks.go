// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/teleclimber/DropServer/cmd/ds-host/testmocks (interfaces: AppspaceFilesEvents,AppUrlDataEvents,AppspaceStatusEvents)

// Package testmocks is a generated GoMock package.
package testmocks

import (
	gomock "github.com/golang/mock/gomock"
	domain "github.com/teleclimber/DropServer/cmd/ds-host/domain"
	reflect "reflect"
)

// MockAppspaceFilesEvents is a mock of AppspaceFilesEvents interface
type MockAppspaceFilesEvents struct {
	ctrl     *gomock.Controller
	recorder *MockAppspaceFilesEventsMockRecorder
}

// MockAppspaceFilesEventsMockRecorder is the mock recorder for MockAppspaceFilesEvents
type MockAppspaceFilesEventsMockRecorder struct {
	mock *MockAppspaceFilesEvents
}

// NewMockAppspaceFilesEvents creates a new mock instance
func NewMockAppspaceFilesEvents(ctrl *gomock.Controller) *MockAppspaceFilesEvents {
	mock := &MockAppspaceFilesEvents{ctrl: ctrl}
	mock.recorder = &MockAppspaceFilesEventsMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockAppspaceFilesEvents) EXPECT() *MockAppspaceFilesEventsMockRecorder {
	return m.recorder
}

// Send mocks base method
func (m *MockAppspaceFilesEvents) Send(arg0 domain.AppspaceID) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Send", arg0)
}

// Send indicates an expected call of Send
func (mr *MockAppspaceFilesEventsMockRecorder) Send(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Send", reflect.TypeOf((*MockAppspaceFilesEvents)(nil).Send), arg0)
}

// Subscribe mocks base method
func (m *MockAppspaceFilesEvents) Subscribe() <-chan domain.AppspaceID {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Subscribe")
	ret0, _ := ret[0].(<-chan domain.AppspaceID)
	return ret0
}

// Subscribe indicates an expected call of Subscribe
func (mr *MockAppspaceFilesEventsMockRecorder) Subscribe() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Subscribe", reflect.TypeOf((*MockAppspaceFilesEvents)(nil).Subscribe))
}

// SubscribeApp mocks base method
func (m *MockAppspaceFilesEvents) SubscribeApp(arg0 domain.AppID) <-chan domain.AppspaceID {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SubscribeApp", arg0)
	ret0, _ := ret[0].(<-chan domain.AppspaceID)
	return ret0
}

// SubscribeApp indicates an expected call of SubscribeApp
func (mr *MockAppspaceFilesEventsMockRecorder) SubscribeApp(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SubscribeApp", reflect.TypeOf((*MockAppspaceFilesEvents)(nil).SubscribeApp), arg0)
}

// SubscribeOwner mocks base method
func (m *MockAppspaceFilesEvents) SubscribeOwner(arg0 domain.UserID) <-chan domain.AppspaceID {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SubscribeOwner", arg0)
	ret0, _ := ret[0].(<-chan domain.AppspaceID)
	return ret0
}

// SubscribeOwner indicates an expected call of SubscribeOwner
func (mr *MockAppspaceFilesEventsMockRecorder) SubscribeOwner(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SubscribeOwner", reflect.TypeOf((*MockAppspaceFilesEvents)(nil).SubscribeOwner), arg0)
}

// Unsubscribe mocks base method
func (m *MockAppspaceFilesEvents) Unsubscribe(arg0 <-chan domain.AppspaceID) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Unsubscribe", arg0)
}

// Unsubscribe indicates an expected call of Unsubscribe
func (mr *MockAppspaceFilesEventsMockRecorder) Unsubscribe(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Unsubscribe", reflect.TypeOf((*MockAppspaceFilesEvents)(nil).Unsubscribe), arg0)
}

// MockAppUrlDataEvents is a mock of AppUrlDataEvents interface
type MockAppUrlDataEvents struct {
	ctrl     *gomock.Controller
	recorder *MockAppUrlDataEventsMockRecorder
}

// MockAppUrlDataEventsMockRecorder is the mock recorder for MockAppUrlDataEvents
type MockAppUrlDataEventsMockRecorder struct {
	mock *MockAppUrlDataEvents
}

// NewMockAppUrlDataEvents creates a new mock instance
func NewMockAppUrlDataEvents(ctrl *gomock.Controller) *MockAppUrlDataEvents {
	mock := &MockAppUrlDataEvents{ctrl: ctrl}
	mock.recorder = &MockAppUrlDataEventsMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockAppUrlDataEvents) EXPECT() *MockAppUrlDataEventsMockRecorder {
	return m.recorder
}

// Send mocks base method
func (m *MockAppUrlDataEvents) Send(arg0 domain.UserID, arg1 domain.AppURLData) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Send", arg0, arg1)
}

// Send indicates an expected call of Send
func (mr *MockAppUrlDataEventsMockRecorder) Send(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Send", reflect.TypeOf((*MockAppUrlDataEvents)(nil).Send), arg0, arg1)
}

// Subscribe mocks base method
func (m *MockAppUrlDataEvents) Subscribe() <-chan domain.AppURLData {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Subscribe")
	ret0, _ := ret[0].(<-chan domain.AppURLData)
	return ret0
}

// Subscribe indicates an expected call of Subscribe
func (mr *MockAppUrlDataEventsMockRecorder) Subscribe() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Subscribe", reflect.TypeOf((*MockAppUrlDataEvents)(nil).Subscribe))
}

// SubscribeApp mocks base method
func (m *MockAppUrlDataEvents) SubscribeApp(arg0 domain.AppID) <-chan domain.AppURLData {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SubscribeApp", arg0)
	ret0, _ := ret[0].(<-chan domain.AppURLData)
	return ret0
}

// SubscribeApp indicates an expected call of SubscribeApp
func (mr *MockAppUrlDataEventsMockRecorder) SubscribeApp(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SubscribeApp", reflect.TypeOf((*MockAppUrlDataEvents)(nil).SubscribeApp), arg0)
}

// SubscribeOwner mocks base method
func (m *MockAppUrlDataEvents) SubscribeOwner(arg0 domain.UserID) <-chan domain.AppURLData {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SubscribeOwner", arg0)
	ret0, _ := ret[0].(<-chan domain.AppURLData)
	return ret0
}

// SubscribeOwner indicates an expected call of SubscribeOwner
func (mr *MockAppUrlDataEventsMockRecorder) SubscribeOwner(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SubscribeOwner", reflect.TypeOf((*MockAppUrlDataEvents)(nil).SubscribeOwner), arg0)
}

// Unsubscribe mocks base method
func (m *MockAppUrlDataEvents) Unsubscribe(arg0 <-chan domain.AppURLData) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Unsubscribe", arg0)
}

// Unsubscribe indicates an expected call of Unsubscribe
func (mr *MockAppUrlDataEventsMockRecorder) Unsubscribe(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Unsubscribe", reflect.TypeOf((*MockAppUrlDataEvents)(nil).Unsubscribe), arg0)
}

// MockAppspaceStatusEvents is a mock of AppspaceStatusEvents interface
type MockAppspaceStatusEvents struct {
	ctrl     *gomock.Controller
	recorder *MockAppspaceStatusEventsMockRecorder
}

// MockAppspaceStatusEventsMockRecorder is the mock recorder for MockAppspaceStatusEvents
type MockAppspaceStatusEventsMockRecorder struct {
	mock *MockAppspaceStatusEvents
}

// NewMockAppspaceStatusEvents creates a new mock instance
func NewMockAppspaceStatusEvents(ctrl *gomock.Controller) *MockAppspaceStatusEvents {
	mock := &MockAppspaceStatusEvents{ctrl: ctrl}
	mock.recorder = &MockAppspaceStatusEventsMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockAppspaceStatusEvents) EXPECT() *MockAppspaceStatusEventsMockRecorder {
	return m.recorder
}

// Send mocks base method
func (m *MockAppspaceStatusEvents) Send(arg0 domain.AppspaceStatusEvent) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Send", arg0)
}

// Send indicates an expected call of Send
func (mr *MockAppspaceStatusEventsMockRecorder) Send(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Send", reflect.TypeOf((*MockAppspaceStatusEvents)(nil).Send), arg0)
}

// Subscribe mocks base method
func (m *MockAppspaceStatusEvents) Subscribe() <-chan domain.AppspaceStatusEvent {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Subscribe")
	ret0, _ := ret[0].(<-chan domain.AppspaceStatusEvent)
	return ret0
}

// Subscribe indicates an expected call of Subscribe
func (mr *MockAppspaceStatusEventsMockRecorder) Subscribe() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Subscribe", reflect.TypeOf((*MockAppspaceStatusEvents)(nil).Subscribe))
}

// SubscribeApp mocks base method
func (m *MockAppspaceStatusEvents) SubscribeApp(arg0 domain.AppID) <-chan domain.AppspaceStatusEvent {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SubscribeApp", arg0)
	ret0, _ := ret[0].(<-chan domain.AppspaceStatusEvent)
	return ret0
}

// SubscribeApp indicates an expected call of SubscribeApp
func (mr *MockAppspaceStatusEventsMockRecorder) SubscribeApp(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SubscribeApp", reflect.TypeOf((*MockAppspaceStatusEvents)(nil).SubscribeApp), arg0)
}

// SubscribeOwner mocks base method
func (m *MockAppspaceStatusEvents) SubscribeOwner(arg0 domain.UserID) <-chan domain.AppspaceStatusEvent {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SubscribeOwner", arg0)
	ret0, _ := ret[0].(<-chan domain.AppspaceStatusEvent)
	return ret0
}

// SubscribeOwner indicates an expected call of SubscribeOwner
func (mr *MockAppspaceStatusEventsMockRecorder) SubscribeOwner(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SubscribeOwner", reflect.TypeOf((*MockAppspaceStatusEvents)(nil).SubscribeOwner), arg0)
}

// Unsubscribe mocks base method
func (m *MockAppspaceStatusEvents) Unsubscribe(arg0 <-chan domain.AppspaceStatusEvent) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Unsubscribe", arg0)
}

// Unsubscribe indicates an expected call of Unsubscribe
func (mr *MockAppspaceStatusEventsMockRecorder) Unsubscribe(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Unsubscribe", reflect.TypeOf((*MockAppspaceStatusEvents)(nil).Unsubscribe), arg0)
}
