package responder

import (
	"net/http"
	"testing"

	"CredChain_Golang/domain"

	"github.com/stretchr/testify/assert"
)

// allDomainCodes is the canonical list of every code constant in domain/codes.go.
// Adding a new code requires adding it here AND to both maps. This catches drift.
var allDomainCodes = []int{
	// System
	domain.CodeSystemSuccess,
	domain.CodeSystemValidation,
	domain.CodeSystemInternal,
	// Overview
	domain.CodeOverviewSuccess,
	domain.CodeOverviewInternal,
	// Meta
	domain.CodeMetaSuccess,
	domain.CodeMetaInternal,
	// Auth
	domain.CodeAuthUnauthorized,
	domain.CodeAuthInvalidToken,
	domain.CodeAuthForbidden,
	domain.CodeAuthRateLimitExceeded,
	// Auth Google Login
	domain.CodeAuthGoogleLoginSuccess,
	domain.CodeAuthGoogleLoginInvalidToken,
	domain.CodeAuthGoogleLoginUserNotFound,
	domain.CodeAuthGoogleLoginAccountDeleted,
	domain.CodeAuthGoogleLoginJWTFailed,
	// Auth Refresh
	domain.CodeAuthRefreshSuccess,
	domain.CodeAuthRefreshInvalidToken,
	domain.CodeAuthRefreshTokenExpired,
	domain.CodeAuthRefreshTokenRevoked,
	domain.CodeAuthRefreshUserNotFound,
	domain.CodeAuthRefreshJWTFailed,
	// Auth Logout
	domain.CodeAuthLogoutSuccess,
	// User
	domain.CodeUserFetchSuccess,
	domain.CodeUserFetchNotFound,
	domain.CodeUserStoreSuccess,
	domain.CodeUserStoreEmailDuplicateInBatch,
	domain.CodeUserStoreEmailDuplicateInDatabase,
	domain.CodeUserStoreWalletGenerationFailed,
	domain.CodeUserStoreBlockchainSyncFailed,
	domain.CodeUserStoreSuperAdminForbidden,
	domain.CodeUserStoreAdminCreateAdminForbidden,
	domain.CodeUserProfileSuccess,
	domain.CodeUserEmailSuccess,
	domain.CodeUserEmailConflict,
	domain.CodeUserEmailMismatchedIdToken,
	domain.CodeUserEmailInvalidIdToken,
	domain.CodeUserRoleSuccess,
	domain.CodeUserRoleAdminUpdatePeerForbidden,
	domain.CodeUserRoleSignerAdminRequiredForbidden,
	domain.CodeUserRoleSameRoleUpdateForbidden,
	domain.CodeUserRoleSuperAdminBatchForbidden,
	domain.CodeUserRoleBlockchainSyncFailed,
	domain.CodeUserRoleSelfTargetForbidden,
	domain.CodeUserRoleTrashedForbidden,
	domain.CodeUserBatchDeleteSuccess,
	domain.CodeUserDeleteAdminForbidden,
	domain.CodeUserDeleteBlockchainSyncFailed,
	domain.CodeUserDeleteSelfTargetForbidden,
	domain.CodeUserUpdateSuccess,
	domain.CodeUserUpdateNotFound,
	domain.CodeUserUpdatePeerAdminForbidden,
	domain.CodeUserUpdateSuperAdminForbidden,
	domain.CodeUserUpdateSelfForbidden,
	domain.CodeUserUpdateBlockchainSyncFailed,
	domain.CodeUserUpdateTrashedForbidden,
	domain.CodeUserUpdateSelfEmailForbidden,
	domain.CodeUserTransferSuperAdminSuccess,
	domain.CodeUserTransferSuperAdminSelfTargetForbidden,
	domain.CodeUserTransferSuperAdminTargetNotFound,
	domain.CodeUserTransferSuperAdminTrashedForbidden,
	domain.CodeUserTransferSuperAdminBlockchainSyncFailed,
	domain.CodeUserRestoreSuccess,
	domain.CodeUserRestoreSignerAdminRequiredForbidden,
	domain.CodeUserRestoreSelfTargetForbidden,
	domain.CodeUserRestoreSuperAdminTargetForbidden,
	domain.CodeUserRestoreNotTrashedForbidden,
	domain.CodeUserRestoreBlockchainSyncFailed,
	// Credential
	domain.CodeCredentialFetchSuccess,
	domain.CodeCredentialFetchNotFound,
	domain.CodeCredentialFetchValidation,
	domain.CodeCredentialIssueSuccess,
	domain.CodeCredentialIssueValidation,
	domain.CodeCredentialIssueDuplicateFileHash,
	domain.CodeCredentialIssueHolderNotFound,
	domain.CodeCredentialIssueBlockchainSyncFailed,
	domain.CodeCredentialIssueStorageFailed,
	domain.CodeCredentialIssueHashFailed,
	domain.CodeCredentialRevokeSuccess,
	domain.CodeCredentialRevokeFailed,
	domain.CodeCredentialRevokeNotFound,
	domain.CodeCredentialRevokeAlreadyRevoked,
	domain.CodeCredentialRevokeBlockchainSyncFailed,
	domain.CodeCredentialVerifySuccess,
	domain.CodeCredentialVerifyFailed,
	domain.CodeCredentialVerifyValidation,
	domain.CodeCredentialVerifyExtractNotReady,
	domain.CodeCredentialVerifyExtractFailed,
	domain.CodeCredentialVerifyAiServiceFailed,
	domain.CodeCredentialVerifyCredentialNotFound,
	domain.CodeCredentialVerifyAuthentic,
	domain.CodeCredentialVerifyRevoked,
	domain.CodeCredentialVerifyIntegrityWarning,
	domain.CodeCredentialVerifyTampered,
	domain.CodeCredentialVerifySuspicious,
	domain.CodeCredentialVerifyLowSimilarity,
	domain.CodeCredentialVerifyNotSimilar,
	domain.CodeCredentialVerifyNoIdentifiers,
	domain.CodeCredentialVerifyNoMatch,
	domain.CodeCredentialVerifyHolderDisabled,
	domain.CodeCredentialVerifyIssuerDisabled,
	domain.CodeCredentialVerifyPartyDisabled,
	domain.CodeCredentialReExtractSuccess,
	domain.CodeCredentialReExtractNotFound,
	domain.CodeCredentialReExtractNotEligible,
	domain.CodeCredentialFileDownloadSuccess,
	domain.CodeCredentialFileDownloadNotFound,
	domain.CodeCredentialFileDownloadForbidden,
	domain.CodeCredentialFileDownloadDecryptionFailed,
	domain.CodeCredentialFileDownloadNoFile,
	domain.CodeCredentialTypeDestroyInUse,
	domain.CodeCredentialIssuerOrganizationDestroyInUse,
	domain.CodeCompetencyDestroyInUse,
	domain.CodeUserUnitDestroyInUse,
	// Credential Type CRUD
	domain.CodeCredentialTypeFetchSuccess,
	domain.CodeCredentialTypeStoreSuccess,
	domain.CodeCredentialTypeUpdateSuccess,
	domain.CodeCredentialTypeDestroySuccess,
	domain.CodeCredentialTypeNotFound,
	domain.CodeCredentialTypeNameDuplicate,
	// Issuer Organization CRUD
	domain.CodeIssuerOrganizationFetchSuccess,
	domain.CodeIssuerOrganizationStoreSuccess,
	domain.CodeIssuerOrganizationUpdateSuccess,
	domain.CodeIssuerOrganizationDestroySuccess,
	domain.CodeIssuerOrganizationNotFound,
	domain.CodeIssuerOrganizationNameDuplicate,
	// Competency CRUD
	domain.CodeCompetencyFetchSuccess,
	domain.CodeCompetencyStoreSuccess,
	domain.CodeCompetencyUpdateSuccess,
	domain.CodeCompetencyDestroySuccess,
	domain.CodeCompetencyNotFound,
	domain.CodeCompetencyNameDuplicate,
	// User Unit CRUD
	domain.CodeUserUnitFetchSuccess,
	domain.CodeUserUnitStoreSuccess,
	domain.CodeUserUnitUpdateSuccess,
	domain.CodeUserUnitDestroySuccess,
	domain.CodeUserUnitNotFound,
	domain.CodeUserUnitParentInvalid,
	// Credential Issue extensions
	domain.CodeCredentialIssueTypeNotFound,
	domain.CodeCredentialIssueTypeInactive,
	domain.CodeCredentialIssueOrganizationNotFound,
	domain.CodeCredentialIssueNumberDuplicate,
	domain.CodeCredentialIssueCompetencyNotFound,
	// Credential Submission
	domain.CodeCredentialSubmitSuccess,
	domain.CodeCredentialSubmitStorageFailed,
	// Credential Review
	domain.CodeCredentialReviewSuccess,
	domain.CodeCredentialReviewNotFound,
	domain.CodeCredentialReviewAlreadyApproved,
	domain.CodeCredentialReviewAlreadyRejected,
	domain.CodeCredentialReviewAlreadyRevoked,
	domain.CodeCredentialReviewBlockchainSyncFailed,
	// Credential Update
	domain.CodeCredentialUpdateSuccess,
	domain.CodeCredentialUpdateNotFound,
	domain.CodeCredentialUpdateNotPending,
}

func TestHttpCodeFromCode_KnownCodes(t *testing.T) {
	tests := map[int]int{
		domain.CodeSystemSuccess:         http.StatusOK,
		domain.CodeAuthUnauthorized:      http.StatusUnauthorized,
		domain.CodeAuthForbidden:         http.StatusForbidden,
		domain.CodeUserFetchNotFound:     http.StatusNotFound,
		domain.CodeUserStoreSuccess:      http.StatusCreated,
		domain.CodeAuthRateLimitExceeded: http.StatusTooManyRequests,
	}
	for code, want := range tests {
		assert.Equal(t, want, HttpCodeFromCode(code), "code %d", code)
	}
}

func TestHttpCodeFromCode_UnknownReturns500(t *testing.T) {
	assert.Equal(t, http.StatusInternalServerError, HttpCodeFromCode(999999))
}

func TestRegistry_EveryCodeHasMessageKey(t *testing.T) {
	for _, code := range allDomainCodes {
		_, ok := CodeToMessageKey[code]
		assert.True(t, ok, "code %d missing from CodeToMessageKey", code)
	}
}

func TestRegistry_EveryCodeHasHttpStatus(t *testing.T) {
	for _, code := range allDomainCodes {
		_, ok := HttpCodes[code]
		assert.True(t, ok, "code %d missing from HttpCodes", code)
	}
}
