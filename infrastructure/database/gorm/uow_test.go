package gorm_test

import (
	"context"
	"errors"
	"testing"

	"CredChain_Golang/domain"
	"CredChain_Golang/feature/credential"
	"CredChain_Golang/feature/user"
	"CredChain_Golang/infrastructure/database/gorm"
	"CredChain_Golang/tests/db"
	"CredChain_Golang/tests/fixtures"

	"github.com/stretchr/testify/assert"
)

func TestGormUnitOfWork_Execute_CommitsOnSuccess(t *testing.T) {
	gdb := db.OpenInMemorySQLite(t)
	uow := gorm.NewGormUnitOfWork(gdb,
		user.NewGormUserRepository,
		credential.NewGormCredentialRepository,
		user.NewGormUserTokenRepository,
		credential.NewGormCompetencyCredentialRepository)

	u := fixtures.NewDomainUser(fixtures.WithEmail("commit@x.com"))
	err := uow.Execute(context.Background(), func(tx domain.UnitOfWork) error {
		_, err := tx.User().Store(context.Background(), u)
		return err
	})
	assert.NoError(t, err)

	repo := user.NewGormUserRepository(gdb)
	users, err := repo.FindByEmails(context.Background(), "commit@x.com")
	assert.NoError(t, err)
	assert.Len(t, users, 1)
}

func TestGormUnitOfWork_Execute_RollsBackOnError(t *testing.T) {
	gdb := db.OpenInMemorySQLite(t)
	uow := gorm.NewGormUnitOfWork(gdb,
		user.NewGormUserRepository,
		credential.NewGormCredentialRepository,
		user.NewGormUserTokenRepository,
		credential.NewGormCompetencyCredentialRepository)

	u := fixtures.NewDomainUser(fixtures.WithEmail("rollback@x.com"))
	err := uow.Execute(context.Background(), func(tx domain.UnitOfWork) error {
		_, err := tx.User().Store(context.Background(), u)
		assert.NoError(t, err)
		return errors.New("simulated failure")
	})
	assert.Error(t, err)

	repo := user.NewGormUserRepository(gdb)
	users, err := repo.FindByEmails(context.Background(), "rollback@x.com")
	assert.NoError(t, err)
	assert.Empty(t, users, "transaction should have been rolled back")
}
