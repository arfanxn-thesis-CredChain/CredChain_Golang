package gorm

import (
	"CredChain_Golang/domain"
	"context"
	"gorm.io/gorm"
)

// RepositoryFactory defines the signature for creating repositories from a GORM DB connection
type RepositoryFactory[T any] func(db *gorm.DB) T

// UserRepositoryFactory creates a UserRepository from a GORM DB connection
type UserRepositoryFactory = RepositoryFactory[domain.UserRepository]

// CredentialRepositoryFactory creates a CredentialRepository from a GORM DB connection
type CredentialRepositoryFactory = RepositoryFactory[domain.CredentialRepository]

// UserTokenRepositoryFactory creates a UserTokenRepository from a GORM DB connection
type UserTokenRepositoryFactory = RepositoryFactory[domain.UserTokenRepository]

// CompetencyCredentialRepositoryFactory creates a CompetencyCredentialRepository
// from a GORM DB connection
type CompetencyCredentialRepositoryFactory = RepositoryFactory[domain.CompetencyCredentialRepository]

// GormUnitOfWork implements domain.UnitOfWork using GORM transactions
type GormUnitOfWork struct {
	db                                *gorm.DB
	notificationRepository            domain.NotificationRepository
	credentialTypeRepository          domain.CredentialTypeRepository
	issuerOrganizationRepository      domain.CredentialIssuerOrganizationRepository
	competencyRepository              domain.CompetencyRepository
	newNotificationRepository         RepositoryFactory[domain.NotificationRepository]
	newCredentialTypeRepository       RepositoryFactory[domain.CredentialTypeRepository]
	newIssuerOrganizationRepository   RepositoryFactory[domain.CredentialIssuerOrganizationRepository]
	newCompetencyRepository           RepositoryFactory[domain.CompetencyRepository]
	userRepository                    domain.UserRepository
	credentialRepository              domain.CredentialRepository
	userTokenRepository               domain.UserTokenRepository
	competencyCredentialRepository    domain.CompetencyCredentialRepository
	newUserRepository                 UserRepositoryFactory
	newCredentialRepository           CredentialRepositoryFactory
	newUserTokenRepository            UserTokenRepositoryFactory
	newCompetencyCredentialRepository CompetencyCredentialRepositoryFactory
}

// NewGormUnitOfWork creates a new UnitOfWork instance with repository factories
func NewGormUnitOfWork(
	db *gorm.DB,
	newUserRepository UserRepositoryFactory,
	newCredentialRepository CredentialRepositoryFactory,
	newUserTokenRepository UserTokenRepositoryFactory,
	newCompetencyCredentialRepository CompetencyCredentialRepositoryFactory,
	newNotificationRepository RepositoryFactory[domain.NotificationRepository],
	newCredentialTypeRepository RepositoryFactory[domain.CredentialTypeRepository],
	newIssuerOrganizationRepository RepositoryFactory[domain.CredentialIssuerOrganizationRepository],
	newCompetencyRepository RepositoryFactory[domain.CompetencyRepository],
) domain.UnitOfWork {
	return &GormUnitOfWork{
		db:                                db,
		newUserRepository:                 newUserRepository,
		newCredentialRepository:           newCredentialRepository,
		newUserTokenRepository:            newUserTokenRepository,
		newCompetencyCredentialRepository: newCompetencyCredentialRepository,
		newNotificationRepository:         newNotificationRepository,
		newCredentialTypeRepository:       newCredentialTypeRepository,
		newIssuerOrganizationRepository:   newIssuerOrganizationRepository,
		newCompetencyRepository:           newCompetencyRepository,
	}
}

// Execute runs a function within a transaction
// All repository operations within the function share the same transaction
func (uow *GormUnitOfWork) Execute(ctx context.Context, fn func(uow domain.UnitOfWork) error) error {
	return uow.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := uow.newUserRepository(tx)
		txCredRepo := uow.newCredentialRepository(tx)
		txTokenRepo := uow.newUserTokenRepository(tx)
		txCompCredRepo := uow.newCompetencyCredentialRepository(tx)

		txUow := &GormUnitOfWork{
			db:                             tx,
			userRepository:                 txUserRepo,
			credentialRepository:           txCredRepo,
			userTokenRepository:            txTokenRepo,
			competencyCredentialRepository: txCompCredRepo,
			notificationRepository:         uow.newNotificationRepository(tx),
			credentialTypeRepository:       uow.newCredentialTypeRepository(tx),
			issuerOrganizationRepository:   uow.newIssuerOrganizationRepository(tx),
			competencyRepository:           uow.newCompetencyRepository(tx),
		}

		return fn(txUow)
	})
}

func (uow *GormUnitOfWork) Notification() domain.NotificationRepository {
	return uow.notificationRepository
}
func (uow *GormUnitOfWork) CredentialType() domain.CredentialTypeRepository {
	return uow.credentialTypeRepository
}
func (uow *GormUnitOfWork) CredentialIssuerOrganization() domain.CredentialIssuerOrganizationRepository {
	return uow.issuerOrganizationRepository
}
func (uow *GormUnitOfWork) Competency() domain.CompetencyRepository { return uow.competencyRepository }

// User returns the UserRepository for this transaction
func (uow *GormUnitOfWork) User() domain.UserRepository {
	return uow.userRepository
}

// Credential returns the CredentialRepository for this transaction
func (uow *GormUnitOfWork) Credential() domain.CredentialRepository {
	return uow.credentialRepository
}

// UserToken returns the UserTokenRepository for this transaction
func (uow *GormUnitOfWork) UserToken() domain.UserTokenRepository {
	return uow.userTokenRepository
}

// CompetencyCredential returns the CompetencyCredentialRepository for this transaction
func (uow *GormUnitOfWork) CompetencyCredential() domain.CompetencyCredentialRepository {
	return uow.competencyCredentialRepository
}
