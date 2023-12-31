// Code generated by mockery v2.33.0. DO NOT EDIT.

package repomocks

import (
	context "context"
	model "my-project/domain/model"

	mock "github.com/stretchr/testify/mock"
)

// IUser is an autogenerated mock type for the IUser type
type IUser struct {
	mock.Mock
}

// CreateUser provides a mock function with given fields: ctx, user
func (_m *IUser) CreateUser(ctx context.Context, user model.User) error {
	ret := _m.Called(ctx, user)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, model.User) error); ok {
		r0 = rf(ctx, user)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetById provides a mock function with given fields: ctx, id
func (_m *IUser) GetById(ctx context.Context, id int) (model.User, error) {
	ret := _m.Called(ctx, id)

	var r0 model.User
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, int) (model.User, error)); ok {
		return rf(ctx, id)
	}
	if rf, ok := ret.Get(0).(func(context.Context, int) model.User); ok {
		r0 = rf(ctx, id)
	} else {
		r0 = ret.Get(0).(model.User)
	}

	if rf, ok := ret.Get(1).(func(context.Context, int) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetByUserName provides a mock function with given fields: ctx, userName
func (_m *IUser) GetByUserName(ctx context.Context, userName string) (model.User, error) {
	ret := _m.Called(ctx, userName)

	var r0 model.User
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (model.User, error)); ok {
		return rf(ctx, userName)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) model.User); ok {
		r0 = rf(ctx, userName)
	} else {
		r0 = ret.Get(0).(model.User)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, userName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewIUser creates a new instance of IUser. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewIUser(t interface {
	mock.TestingT
	Cleanup(func())
}) *IUser {
	mock := &IUser{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
