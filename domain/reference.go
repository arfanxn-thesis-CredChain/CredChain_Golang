package domain

// ReferenceStatus represents the submission and review lifecycle for
// taxonomy references (CredentialType, CredentialIssuerOrganization, Competency).
// Derived from timestamps:
// - pending: activated_at IS NULL AND rejected_at IS NULL
// - approved: activated_at IS NOT NULL AND rejected_at IS NULL
// - rejected: rejected_at IS NOT NULL
type ReferenceStatus string

const (
	ReferenceStatusPending  ReferenceStatus = "pending"
	ReferenceStatusApproved ReferenceStatus = "approved"
	ReferenceStatusRejected ReferenceStatus = "rejected"
)
