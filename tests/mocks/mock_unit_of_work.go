package mocks

import (
	"context"

	"CredChain_Golang/domain"

	"github.com/stretchr/testify/mock"
)

type MockUnitOfWork struct {
	mock.Mock
}

func (m *MockUnitOfWork) Execute(ctx context.Context, fn func(domain.UnitOfWork) error) error {
	args := m.Called(ctx, fn)
	return args.Error(0)
}

func (m *MockUnitOfWork) User() domain.UserRepository {
	args := m.Called()
	return args.Get(0).(domain.UserRepository)
}

func (m *MockUnitOfWork) Credential() domain.CredentialRepository {
	args := m.Called()
	return args.Get(0).(domain.CredentialRepository)
}

func (m *MockUnitOfWork) UserToken() domain.UserTokenRepository {
	args := m.Called()
	return args.Get(0).(domain.UserTokenRepository)
}

func (m *MockUnitOfWork) CompetencyCredential() domain.CompetencyCredentialRepository {
	args := m.Called()
	return args.Get(0).(domain.CompetencyCredentialRepository)
}

// RunUnitOfWorkFn configures the mock to invoke the function passed to Execute.
func RunUnitOfWorkFn(m *MockUnitOfWork, innerUoW domain.UnitOfWork) {
	m.On("Execute", mock.Anything, mock.AnythingOfType("func(domain.UnitOfWork) error")).
		Return(nil).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(domain.UnitOfWork) error)
			_ = fn(innerUoW)
		})
}

// PropagatingUnitOfWork is a UnitOfWork that actually executes the function passed
// to Execute and propagates its error. Use this for tests that need to assert on
// errors returned from inside transactional code paths.
type PropagatingUnitOfWork struct {
	*MockUnitOfWork
}

func (p *PropagatingUnitOfWork) Execute(ctx context.Context, fn func(domain.UnitOfWork) error) error {
	return fn(p)
}

func NewPropagatingUnitOfWork() *PropagatingUnitOfWork {
	return &PropagatingUnitOfWork{MockUnitOfWork: &MockUnitOfWork{}}
}

var _ domain.UnitOfWork = (*MockUnitOfWork)(nil)
var _ domain.UnitOfWork = (*PropagatingUnitOfWork)(nil)
