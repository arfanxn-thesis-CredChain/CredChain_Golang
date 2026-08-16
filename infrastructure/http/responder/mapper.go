package responder

import (
	"CredChain_Golang/domain"
	"net/http"
)

// CodeToMessageKey maps every status code to its i18n message key.
var CodeToMessageKey = map[int]string{
	// System codes
	domain.CodeSystemSuccess:    "success_generic",
	domain.CodeSystemValidation: "error_invalid_payload",
	domain.CodeSystemInternal:   "error_internal",

	// Overview codes
	domain.CodeOverviewSuccess:  "success_overview",
	domain.CodeOverviewInternal: "error_overview_internal",

	// Meta codes
	domain.CodeMetaSuccess:  "success_meta",
	domain.CodeMetaInternal: "error_meta_internal",

	// Auth codes
	domain.CodeAuthUnauthorized:      "error_unauthorized",
	domain.CodeAuthInvalidToken:      "error_invalid_token",
	domain.CodeAuthForbidden:         "error_unauthorized_email",
	domain.CodeAuthRateLimitExceeded: "error_rate_limit_exceeded",

	// Auth Google Login codes
	domain.CodeAuthGoogleLoginSuccess:        "success_login",
	domain.CodeAuthGoogleLoginInvalidToken:   "error_google_token_invalid",
	domain.CodeAuthGoogleLoginUserNotFound:   "error_unauthorized_email",
	domain.CodeAuthGoogleLoginAccountDeleted: "error_account_deleted",
	domain.CodeAuthGoogleLoginJWTFailed:      "error_token_issue_failed",

	// Auth Refresh codes
	domain.CodeAuthRefreshSuccess:      "success_login",
	domain.CodeAuthRefreshInvalidToken: "error_invalid_token",
	domain.CodeAuthRefreshTokenExpired: "error_refresh_token_expired",
	domain.CodeAuthRefreshTokenRevoked: "error_refresh_token_revoked",
	domain.CodeAuthRefreshUserNotFound: "error_user_not_found",
	domain.CodeAuthRefreshJWTFailed:    "error_token_issue_failed",

	// Auth Logout codes
	domain.CodeAuthLogoutSuccess: "success_logout",

	// User codes
	domain.CodeUserFetchSuccess:                           "success_user_fetched",
	domain.CodeUserFetchNotFound:                          "error_user_not_found",
	domain.CodeUserStoreSuccess:                           "success_users_created",
	domain.CodeUserStoreEmailDuplicateInBatch:             "error_email_duplicate_in_batch",
	domain.CodeUserStoreEmailDuplicateInDatabase:          "error_email_duplicate_in_database",
	domain.CodeUserStoreWalletGenerationFailed:            "error_store_wallet_generation_failed",
	domain.CodeUserStoreBlockchainSyncFailed:              "error_store_blockchain_sync_failed",
	domain.CodeUserStoreSuperAdminForbidden:               "error_store_super_admin_forbidden",
	domain.CodeUserStoreAdminCreateAdminForbidden:         "error_store_admin_create_admin_forbidden",
	domain.CodeUserProfileSuccess:                         "success_profile_updated",
	domain.CodeUserEmailSuccess:                           "success_email_updated",
	domain.CodeUserEmailConflict:                          "error_update_email_failed",
	domain.CodeUserEmailMismatchedIdToken:                 "error_email_mismatched_id_token",
	domain.CodeUserEmailInvalidIdToken:                    "error_email_invalid_id_token",
	domain.CodeUserRoleSuccess:                            "success_role_updated",
	domain.CodeUserRoleAdminUpdatePeerForbidden:           "error_admin_update_peer_role_forbidden",
	domain.CodeUserRoleSignerAdminRequiredForbidden:       "error_signer_admin_required_forbidden",
	domain.CodeUserRoleSameRoleUpdateForbidden:            "error_same_role_update_forbidden",
	domain.CodeUserRoleSuperAdminBatchForbidden:           "error_super_admin_batch_forbidden",
	domain.CodeUserRoleBlockchainSyncFailed:               "error_role_blockchain_sync_failed",
	domain.CodeUserRoleSelfTargetForbidden:                "error_role_self_target_forbidden",
	domain.CodeUserRoleTrashedForbidden:                   "error_role_trashed_forbidden",
	domain.CodeUserBatchDeleteSuccess:                     "success_users_deleted",
	domain.CodeUserDeleteAdminForbidden:                   "error_user_delete_admin_forbidden",
	domain.CodeUserDeleteBlockchainSyncFailed:             "error_delete_blockchain_sync_failed",
	domain.CodeUserDeleteSelfTargetForbidden:              "error_delete_self_target_forbidden",
	domain.CodeUserUpdateSuccess:                          "success_users_updated",
	domain.CodeUserUpdateNotFound:                         "error_users_update_not_found",
	domain.CodeUserUpdatePeerAdminForbidden:               "error_users_update_peer_admin_forbidden",
	domain.CodeUserUpdateSuperAdminForbidden:              "error_users_update_super_admin_forbidden",
	domain.CodeUserUpdateSelfForbidden:                    "error_users_update_self_forbidden",
	domain.CodeUserUpdateBlockchainSyncFailed:             "error_users_update_blockchain_sync_failed",
	domain.CodeUserUpdateTrashedForbidden:                 "error_users_update_trashed_forbidden",
	domain.CodeUserUpdateSelfEmailForbidden:               "error_users_update_self_email_forbidden",
	domain.CodeUserTransferSuperAdminSuccess:              "success_super_admin_transferred",
	domain.CodeUserTransferSuperAdminSelfTargetForbidden:  "error_transfer_super_admin_self_target_forbidden",
	domain.CodeUserTransferSuperAdminTargetNotFound:       "error_transfer_super_admin_target_not_found",
	domain.CodeUserTransferSuperAdminTrashedForbidden:     "error_transfer_super_admin_trashed_forbidden",
	domain.CodeUserTransferSuperAdminBlockchainSyncFailed: "error_transfer_super_admin_blockchain_sync_failed",
	domain.CodeUserRestoreSuccess:                         "success_users_restore",
	domain.CodeUserRestoreSignerAdminRequiredForbidden:    "error_users_restore_signer_admin_required",
	domain.CodeUserRestoreSelfTargetForbidden:             "error_users_restore_self_target_forbidden",
	domain.CodeUserRestoreSuperAdminTargetForbidden:       "error_users_restore_super_admin_target_forbidden",
	domain.CodeUserRestoreNotTrashedForbidden:             "error_users_restore_not_trashed_forbidden",
	domain.CodeUserRestoreBlockchainSyncFailed:            "error_users_restore_blockchain_sync_failed",

	// Credential codes
	domain.CodeCredentialFetchSuccess:               "success_credential_fetched",
	domain.CodeCredentialFetchNotFound:              "error_credential_not_found",
	domain.CodeCredentialFetchValidation:            "error_credential_fetch_validation",
	domain.CodeCredentialIssueSuccess:               "success_credential_issued",
	domain.CodeCredentialIssueValidation:            "error_credential_issue_validation",
	domain.CodeCredentialIssueDuplicateFileHash:     "error_credential_issue_duplicate_file_hash",
	domain.CodeCredentialIssueHolderNotFound:        "error_credential_issue_holder_not_found",
	domain.CodeCredentialIssueBlockchainSyncFailed:  "error_credential_issue_blockchain_sync_failed",
	domain.CodeCredentialIssueStorageFailed:         "error_credential_issue_storage_failed",
	domain.CodeCredentialIssueHashFailed:            "error_credential_issue_hash_failed",
	domain.CodeCredentialRevokeSuccess:              "success_credential_revoked",
	domain.CodeCredentialRevokeFailed:               "error_credential_revoke_failed",
	domain.CodeCredentialRevokeNotFound:             "error_credential_revoke_not_found",
	domain.CodeCredentialRevokeAlreadyRevoked:       "error_credential_revoke_already_revoked",
	domain.CodeCredentialRevokeBlockchainSyncFailed: "error_credential_revoke_blockchain_sync_failed",
	domain.CodeCredentialVerifySuccess:              "success_credential_verified",
	domain.CodeCredentialVerifyFailed:               "error_credential_verify_failed",
	domain.CodeCredentialVerifyValidation:           "error_credential_verify_validation",
	domain.CodeCredentialVerifyExtractNotReady:      "error_credential_verify_extract_not_ready",
	domain.CodeCredentialVerifyExtractFailed:        "error_credential_verify_extract_failed",
	domain.CodeCredentialVerifyAiServiceFailed:      "error_credential_verify_ai_service_failed",
	domain.CodeCredentialVerifyCredentialNotFound:   "error_credential_verify_credential_not_found",
	domain.CodeCredentialVerifyDocumentUnreadable:   "error_credential_verify_document_unreadable",
	domain.CodeCredentialVerifyAuthentic:            "success_credential_verify_authentic",
	domain.CodeCredentialVerifyRevoked:              "success_credential_verify_revoked",
	domain.CodeCredentialVerifyIntegrityWarning:     "warning_credential_verify_integrity",
	domain.CodeCredentialVerifyTampered:             "success_credential_verify_tampered",
	domain.CodeCredentialVerifySuspicious:           "success_credential_verify_suspicious",
	domain.CodeCredentialVerifyLowSimilarity:        "success_credential_verify_low_similarity",
	domain.CodeCredentialVerifyNotSimilar:           "success_credential_verify_not_similar",
	domain.CodeCredentialVerifyNoIdentifiers:        "success_credential_verify_no_identifiers",
	domain.CodeCredentialVerifyNoMatch:              "success_credential_verify_no_match",
	domain.CodeCredentialVerifyHolderDisabled:       "success_credential_verify_holder_disabled",
	domain.CodeCredentialVerifyIssuerDisabled:       "success_credential_verify_issuer_disabled",
	domain.CodeCredentialVerifyPartyDisabled:        "success_credential_verify_party_disabled",

	// Credential Re-Extract codes
	domain.CodeCredentialReExtractSuccess:     "success_credential_reextract",
	domain.CodeCredentialReExtractNotFound:    "error_credential_reextract_not_found",
	domain.CodeCredentialReExtractNotEligible: "error_credential_reextract_not_eligible",

	// Credential File Download codes
	domain.CodeCredentialFileDownloadSuccess:          "success_credential_file_download",
	domain.CodeCredentialFileDownloadNotFound:         "error_credential_file_download_not_found",
	domain.CodeCredentialFileDownloadForbidden:        "error_credential_file_download_forbidden",
	domain.CodeCredentialFileDownloadDecryptionFailed: "error_credential_file_download_decryption_failed",
	domain.CodeCredentialFileDownloadNoFile:           "error_credential_file_download_no_file",

	// Lookup-table deletion guards
	domain.CodeCredentialTypeDeleteInUse:               "error_credential_type_delete_in_use",
	domain.CodeCredentialIssuerOrganizationDeleteInUse: "error_issuer_organization_delete_in_use",
	domain.CodeCompetencyDeleteInUse:                   "error_competency_delete_in_use",
	domain.CodeUserUnitDeleteInUse:                     "error_user_unit_delete_in_use",
}

// HttpCodes maps every domain status code to its exact HTTP status code.
var HttpCodes = map[int]int{
	domain.CodeSystemSuccess:    http.StatusOK,
	domain.CodeSystemValidation: http.StatusBadRequest,
	domain.CodeSystemInternal:   http.StatusInternalServerError,

	domain.CodeOverviewSuccess:  http.StatusOK,
	domain.CodeOverviewInternal: http.StatusInternalServerError,

	domain.CodeMetaSuccess:  http.StatusOK,
	domain.CodeMetaInternal: http.StatusInternalServerError,

	domain.CodeAuthUnauthorized:      http.StatusUnauthorized,
	domain.CodeAuthInvalidToken:      http.StatusUnauthorized,
	domain.CodeAuthForbidden:         http.StatusForbidden,
	domain.CodeAuthRateLimitExceeded: http.StatusTooManyRequests,

	domain.CodeAuthGoogleLoginSuccess:        http.StatusOK,
	domain.CodeAuthGoogleLoginInvalidToken:   http.StatusUnauthorized,
	domain.CodeAuthGoogleLoginUserNotFound:   http.StatusForbidden,
	domain.CodeAuthGoogleLoginAccountDeleted: http.StatusForbidden,
	domain.CodeAuthGoogleLoginJWTFailed:      http.StatusInternalServerError,

	domain.CodeAuthRefreshSuccess:      http.StatusOK,
	domain.CodeAuthRefreshInvalidToken: http.StatusUnauthorized,
	domain.CodeAuthRefreshTokenExpired: http.StatusUnauthorized,
	domain.CodeAuthRefreshTokenRevoked: http.StatusUnauthorized,
	domain.CodeAuthRefreshUserNotFound: http.StatusNotFound,
	domain.CodeAuthRefreshJWTFailed:    http.StatusInternalServerError,

	domain.CodeAuthLogoutSuccess: http.StatusOK,

	domain.CodeUserFetchSuccess:                           http.StatusOK,
	domain.CodeUserFetchNotFound:                          http.StatusNotFound,
	domain.CodeUserStoreSuccess:                           http.StatusCreated,
	domain.CodeUserStoreEmailDuplicateInBatch:             http.StatusBadRequest,
	domain.CodeUserStoreEmailDuplicateInDatabase:          http.StatusBadRequest,
	domain.CodeUserStoreWalletGenerationFailed:            http.StatusInternalServerError,
	domain.CodeUserStoreBlockchainSyncFailed:              http.StatusInternalServerError,
	domain.CodeUserStoreSuperAdminForbidden:               http.StatusForbidden,
	domain.CodeUserStoreAdminCreateAdminForbidden:         http.StatusForbidden,
	domain.CodeUserProfileSuccess:                         http.StatusOK,
	domain.CodeUserEmailSuccess:                           http.StatusOK,
	domain.CodeUserEmailConflict:                          http.StatusConflict,
	domain.CodeUserEmailMismatchedIdToken:                 http.StatusUnprocessableEntity,
	domain.CodeUserEmailInvalidIdToken:                    http.StatusUnauthorized,
	domain.CodeUserRoleSuccess:                            http.StatusOK,
	domain.CodeUserRoleAdminUpdatePeerForbidden:           http.StatusForbidden,
	domain.CodeUserRoleSignerAdminRequiredForbidden:       http.StatusForbidden,
	domain.CodeUserRoleSameRoleUpdateForbidden:            http.StatusForbidden,
	domain.CodeUserRoleSuperAdminBatchForbidden:           http.StatusForbidden,
	domain.CodeUserRoleBlockchainSyncFailed:               http.StatusInternalServerError,
	domain.CodeUserRoleSelfTargetForbidden:                http.StatusForbidden,
	domain.CodeUserRoleTrashedForbidden:                   http.StatusForbidden,
	domain.CodeUserBatchDeleteSuccess:                     http.StatusOK,
	domain.CodeUserDeleteAdminForbidden:                   http.StatusForbidden,
	domain.CodeUserDeleteBlockchainSyncFailed:             http.StatusInternalServerError,
	domain.CodeUserDeleteSelfTargetForbidden:              http.StatusForbidden,
	domain.CodeUserUpdateSuccess:                          http.StatusOK,
	domain.CodeUserUpdateNotFound:                         http.StatusNotFound,
	domain.CodeUserUpdatePeerAdminForbidden:               http.StatusForbidden,
	domain.CodeUserUpdateSuperAdminForbidden:              http.StatusForbidden,
	domain.CodeUserUpdateSelfForbidden:                    http.StatusForbidden,
	domain.CodeUserUpdateBlockchainSyncFailed:             http.StatusInternalServerError,
	domain.CodeUserUpdateTrashedForbidden:                 http.StatusForbidden,
	domain.CodeUserUpdateSelfEmailForbidden:               http.StatusForbidden,
	domain.CodeUserTransferSuperAdminSuccess:              http.StatusOK,
	domain.CodeUserTransferSuperAdminSelfTargetForbidden:  http.StatusForbidden,
	domain.CodeUserTransferSuperAdminTargetNotFound:       http.StatusNotFound,
	domain.CodeUserTransferSuperAdminTrashedForbidden:     http.StatusForbidden,
	domain.CodeUserTransferSuperAdminBlockchainSyncFailed: http.StatusInternalServerError,
	domain.CodeUserRestoreSuccess:                         http.StatusOK,
	domain.CodeUserRestoreSignerAdminRequiredForbidden:    http.StatusForbidden,
	domain.CodeUserRestoreSelfTargetForbidden:             http.StatusForbidden,
	domain.CodeUserRestoreSuperAdminTargetForbidden:       http.StatusForbidden,
	domain.CodeUserRestoreNotTrashedForbidden:             http.StatusForbidden,
	domain.CodeUserRestoreBlockchainSyncFailed:            http.StatusInternalServerError,

	domain.CodeCredentialFetchSuccess:               http.StatusOK,
	domain.CodeCredentialFetchNotFound:              http.StatusNotFound,
	domain.CodeCredentialFetchValidation:            http.StatusBadRequest,
	domain.CodeCredentialIssueSuccess:               http.StatusCreated,
	domain.CodeCredentialIssueValidation:            http.StatusBadRequest,
	domain.CodeCredentialIssueDuplicateFileHash:     http.StatusConflict,
	domain.CodeCredentialIssueHolderNotFound:        http.StatusNotFound,
	domain.CodeCredentialIssueBlockchainSyncFailed:  http.StatusInternalServerError,
	domain.CodeCredentialIssueStorageFailed:         http.StatusInternalServerError,
	domain.CodeCredentialIssueHashFailed:            http.StatusInternalServerError,
	domain.CodeCredentialRevokeSuccess:              http.StatusOK,
	domain.CodeCredentialRevokeFailed:               http.StatusInternalServerError,
	domain.CodeCredentialRevokeNotFound:             http.StatusNotFound,
	domain.CodeCredentialRevokeAlreadyRevoked:       http.StatusConflict,
	domain.CodeCredentialRevokeBlockchainSyncFailed: http.StatusInternalServerError,
	domain.CodeCredentialVerifySuccess:              http.StatusOK,
	domain.CodeCredentialVerifyFailed:               http.StatusUnprocessableEntity,
	domain.CodeCredentialVerifyValidation:           http.StatusBadRequest,
	domain.CodeCredentialVerifyExtractNotReady:      http.StatusConflict,
	domain.CodeCredentialVerifyExtractFailed:        http.StatusUnprocessableEntity,
	domain.CodeCredentialVerifyAiServiceFailed:      http.StatusServiceUnavailable,
	domain.CodeCredentialVerifyCredentialNotFound:   http.StatusNotFound,
	domain.CodeCredentialVerifyDocumentUnreadable:   http.StatusUnprocessableEntity,
	domain.CodeCredentialVerifyAuthentic:            http.StatusOK,
	domain.CodeCredentialVerifyRevoked:              http.StatusOK,
	domain.CodeCredentialVerifyIntegrityWarning:     http.StatusConflict,
	domain.CodeCredentialVerifyTampered:             http.StatusOK,
	domain.CodeCredentialVerifySuspicious:           http.StatusOK,
	domain.CodeCredentialVerifyLowSimilarity:        http.StatusOK,
	domain.CodeCredentialVerifyNotSimilar:           http.StatusOK,
	domain.CodeCredentialVerifyNoIdentifiers:        http.StatusOK,
	domain.CodeCredentialVerifyNoMatch:              http.StatusOK,
	domain.CodeCredentialVerifyHolderDisabled:       http.StatusOK,
	domain.CodeCredentialVerifyIssuerDisabled:       http.StatusOK,
	domain.CodeCredentialVerifyPartyDisabled:        http.StatusOK,

	domain.CodeCredentialReExtractSuccess:     http.StatusOK,
	domain.CodeCredentialReExtractNotFound:    http.StatusNotFound,
	domain.CodeCredentialReExtractNotEligible: http.StatusConflict,

	domain.CodeCredentialFileDownloadSuccess:          http.StatusOK,
	domain.CodeCredentialFileDownloadNotFound:         http.StatusNotFound,
	domain.CodeCredentialFileDownloadForbidden:        http.StatusForbidden,
	domain.CodeCredentialFileDownloadDecryptionFailed: http.StatusInternalServerError,
	domain.CodeCredentialFileDownloadNoFile:           http.StatusNotFound,

	// Lookup-table deletion guards (all 409 Conflict)
	domain.CodeCredentialTypeDeleteInUse:               http.StatusConflict,
	domain.CodeCredentialIssuerOrganizationDeleteInUse: http.StatusConflict,
	domain.CodeCompetencyDeleteInUse:                   http.StatusConflict,
	domain.CodeUserUnitDeleteInUse:                     http.StatusConflict,
}

// HttpCodeFromCode looks up the HTTP status for a given domain status code.
func HttpCodeFromCode(code int) int {
	if httpCode, ok := HttpCodes[code]; ok {
		return httpCode
	}
	return http.StatusInternalServerError
}
