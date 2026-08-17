package domain

import "context"

// UnitOfWork defines the transactional boundary contract
// Implementations manage transactions across multiple repositories
type UnitOfWork interface {
	// Execute runs a function within a transaction
	// All repository operations within the function share the same transaction
	Execute(ctx context.Context, fn func(uow UnitOfWork) error) error

	// Repository accessors (explicit)
	User() UserRepository
	Credential() CredentialRepository
	UserToken() UserTokenRepository
	CompetencyCredential() CompetencyCredentialRepository
}

// TransactionType defines types of multi-repository transactions
type TransactionType string

const (
	TransactionTypeUserDelete       TransactionType = "user_delete"
	TransactionTypeCredentialIssue  TransactionType = "credential_issue"
	TransactionTypeCredentialRevoke TransactionType = "credential_revoke"
)

// TransactionMetadata holds metadata about a transaction for auditing/logging
type TransactionMetadata struct {
	Type        TransactionType
	Description string
	Entities    []string
}
