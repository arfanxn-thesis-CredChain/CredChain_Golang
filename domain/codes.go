package domain

// ---- 6-Digit AABBCC Status Code Registry ----
//
// AA = Category:  10 System | 20 Auth | 30 User | 40 Credential
// BB = Feature:   01, 02, 03 ... (resets per category)
// CC = Status:    00-19 Success | 20-39 Reserved | 40-99 Error

const (
	// ── System (10) ──────────────────────────────────────────────────────────
	CodeSystemSuccess    = 100000
	CodeSystemValidation = 100040
	CodeSystemInternal   = 100050

	// ── Overview (10) ──────────────────────────────────────────────────────────
	CodeOverviewSuccess  = 100100
	CodeOverviewInternal = 100150

	// ── Meta (10) ──────────────────────────────────────────────────────────────
	CodeMetaSuccess  = 100200
	CodeMetaInternal = 100250

	// ── Auth (20) ────────────────────────────────────────────────────────────
	CodeAuthUnauthorized      = 200140
	CodeAuthInvalidToken      = 200141
	CodeAuthForbidden         = 200142
	CodeAuthRateLimitExceeded = 200143

	// ── Auth Google Login (20) ───────────────────────────────────────────────
	CodeAuthGoogleLoginSuccess        = 200200
	CodeAuthGoogleLoginInvalidToken   = 200241
	CodeAuthGoogleLoginUserNotFound   = 200242
	CodeAuthGoogleLoginAccountDeleted = 200243
	CodeAuthGoogleLoginJWTFailed      = 200250

	// ── Auth Refresh (20) ──────────────────────────────────────────────
	CodeAuthRefreshSuccess      = 200300
	CodeAuthRefreshInvalidToken = 200340
	CodeAuthRefreshTokenExpired = 200341
	CodeAuthRefreshTokenRevoked = 200342
	CodeAuthRefreshUserNotFound = 200343
	CodeAuthRefreshJWTFailed    = 200350

	// ── Auth Logout (20) ───────────────────────────────────────────────
	CodeAuthLogoutSuccess = 200400

	// ── User (30) ────────────────────────────────────────────────────────────
	CodeUserFetchSuccess                     = 300100
	CodeUserFetchNotFound                    = 300140
	CodeUserStoreSuccess                     = 300200
	CodeUserStoreEmailDuplicateInBatch       = 300241
	CodeUserStoreEmailDuplicateInDatabase    = 300242
	CodeUserStoreWalletGenerationFailed      = 300243
	CodeUserStoreBlockchainSyncFailed        = 300244
	CodeUserStoreSuperAdminForbidden         = 300245
	CodeUserStoreAdminCreateAdminForbidden   = 300246
	CodeUserProfileSuccess                   = 300300
	CodeUserEmailSuccess                     = 300400
	CodeUserEmailConflict                    = 300440
	CodeUserEmailMismatchedIdToken           = 300441
	CodeUserEmailInvalidIdToken              = 300442
	CodeUserRoleSuccess                      = 300500
	CodeUserRoleAdminUpdatePeerForbidden     = 300541
	CodeUserRoleSignerAdminRequiredForbidden = 300542
	CodeUserRoleSameRoleUpdateForbidden      = 300543
	CodeUserRoleSuperAdminBatchForbidden     = 300544
	CodeUserRoleBlockchainSyncFailed         = 300545
	CodeUserRoleSelfTargetForbidden          = 300546
	CodeUserRoleTrashedForbidden             = 300547
	CodeUserBatchDeleteSuccess               = 300700
	CodeUserDeleteAdminForbidden             = 300741
	CodeUserDeleteBlockchainSyncFailed       = 300742
	CodeUserDeleteSelfTargetForbidden        = 300743
	CodeUserUpdateSuccess                    = 300800
	CodeUserUpdateNotFound                   = 300841
	CodeUserUpdatePeerAdminForbidden         = 300842
	CodeUserUpdateSuperAdminForbidden        = 300843
	CodeUserUpdateSelfForbidden              = 300844
	CodeUserUpdateBlockchainSyncFailed       = 300845
	CodeUserUpdateTrashedForbidden           = 300846
	CodeUserUpdateSelfEmailForbidden         = 300847

	CodeUserTransferSuperAdminSuccess              = 300600
	CodeUserTransferSuperAdminSelfTargetForbidden  = 300641
	CodeUserTransferSuperAdminTargetNotFound       = 300642
	CodeUserTransferSuperAdminTrashedForbidden     = 300643
	CodeUserTransferSuperAdminBlockchainSyncFailed = 300645

	// ── Restore (09) ─────────────────────────────────────────────────────────
	CodeUserRestoreSuccess                      = 300900
	CodeUserRestoreSignerAdminRequiredForbidden = 300941
	CodeUserRestoreSelfTargetForbidden          = 300942
	CodeUserRestoreSuperAdminTargetForbidden    = 300943
	CodeUserRestoreNotTrashedForbidden          = 300944
	CodeUserRestoreBlockchainSyncFailed         = 300945

	CodeUserUnitDestroyInUse = 300650

	// ── Credential (40) ──────────────────────────────────────────────────────
	CodeCredentialFetchSuccess    = 400100
	CodeCredentialFetchNotFound   = 400140
	CodeCredentialFetchValidation = 400141

	CodeCredentialIssueSuccess              = 400200
	CodeCredentialIssueValidation           = 400241
	CodeCredentialIssueDuplicateFileHash    = 400242
	CodeCredentialIssueHolderNotFound       = 400243
	CodeCredentialIssueBlockchainSyncFailed = 400244
	CodeCredentialIssueStorageFailed        = 400245
	CodeCredentialIssueHashFailed           = 400246

	CodeCredentialRevokeSuccess              = 400300
	CodeCredentialRevokeFailed               = 400340
	CodeCredentialRevokeNotFound             = 400341
	CodeCredentialRevokeAlreadyRevoked       = 400342
	CodeCredentialRevokeBlockchainSyncFailed = 400343

	CodeCredentialVerifySuccess            = 400400
	CodeCredentialVerifyFailed             = 400440
	CodeCredentialVerifyValidation         = 400441
	CodeCredentialVerifyExtractNotReady    = 400442
	CodeCredentialVerifyExtractFailed      = 400443
	CodeCredentialVerifyAiServiceFailed    = 400444
	CodeCredentialVerifyCredentialNotFound = 400445
	CodeCredentialVerifyDocumentUnreadable = 400446

	// Verify verdict outcomes use the success sub-range CC=01-13 (all HTTP 200
	// except IntegrityWarning=409). These are deliberate verdict codes, NOT
	// errors — the verify operation succeeded; the verdict is the result data.
	// Do not reuse CC=01-13 here for unrelated codes.
	CodeCredentialVerifyAuthentic        = 400401
	CodeCredentialVerifyRevoked          = 400402
	CodeCredentialVerifyIntegrityWarning = 400403
	CodeCredentialVerifyTampered         = 400404
	CodeCredentialVerifySuspicious       = 400405
	CodeCredentialVerifyLowSimilarity    = 400406
	CodeCredentialVerifyNotSimilar       = 400407
	CodeCredentialVerifyNoIdentifiers    = 400408
	CodeCredentialVerifyNoMatch          = 400409
	CodeCredentialVerifyHolderDisabled   = 400410
	CodeCredentialVerifyIssuerDisabled   = 400411
	CodeCredentialVerifyPartyDisabled    = 400412
	CodeCredentialVerifyExpired          = 400413

	// ── Credential Re-Extract (05) ──────────────────────────────────────────
	CodeCredentialReExtractSuccess     = 400500
	CodeCredentialReExtractNotFound    = 400540
	CodeCredentialReExtractNotEligible = 400541

	// ── Credential File Download (06) ──────────────────────────────────────
	CodeCredentialFileDownloadSuccess          = 400600
	CodeCredentialFileDownloadNotFound         = 400640
	CodeCredentialFileDownloadForbidden        = 400641
	CodeCredentialFileDownloadDecryptionFailed = 400642
	CodeCredentialFileDownloadNoFile           = 400643

	// Lookup-table deletion guards: rows referenced by credentials (or, for
	// units, by users/child units) cannot be hard-deleted. For credential
	// types, active = false is the intended everyday path; hard deletion is
	// the exception. The reference pre-check lives in the step-3 services;
	// these codes are returned by the services and by the repo-level 23503
	// translation backstop.
	CodeCredentialTypeDestroyInUse               = 400741
	CodeCredentialIssuerOrganizationDestroyInUse = 400742
	CodeCompetencyDestroyInUse                   = 400743

	// ── Credential Type CRUD (08) ───────────────────────────────────────────
	CodeCredentialTypeFetchSuccess   = 400800
	CodeCredentialTypeStoreSuccess   = 400801
	CodeCredentialTypeUpdateSuccess  = 400802
	CodeCredentialTypeDestroySuccess = 400803
	CodeCredentialTypeNotFound       = 400840
	CodeCredentialTypeNameDuplicate  = 400841

	// ── Issuer Organization CRUD (09) ───────────────────────────────────────
	CodeIssuerOrganizationFetchSuccess   = 400900
	CodeIssuerOrganizationStoreSuccess   = 400901
	CodeIssuerOrganizationUpdateSuccess  = 400902
	CodeIssuerOrganizationDestroySuccess = 400903
	CodeIssuerOrganizationNotFound       = 400940
	CodeIssuerOrganizationNameDuplicate  = 400941

	// ── Competency CRUD (10) ────────────────────────────────────────────────
	CodeCompetencyFetchSuccess   = 401000
	CodeCompetencyStoreSuccess   = 401001
	CodeCompetencyUpdateSuccess  = 401002
	CodeCompetencyDestroySuccess = 401003
	CodeCompetencyNotFound       = 401040
	CodeCompetencyNameDuplicate  = 401041

	// ── User Unit CRUD (30/10) ──────────────────────────────────────────────
	CodeUserUnitFetchSuccess   = 301000
	CodeUserUnitStoreSuccess   = 301001
	CodeUserUnitUpdateSuccess  = 301002
	CodeUserUnitDestroySuccess = 301003
	CodeUserUnitNotFound       = 301040
	CodeUserUnitParentInvalid  = 301041
	CodeUserUnitParentInactive = 301042

	// ── Issue extensions (02, CC 47-53) ─────────────────────────────────────
	CodeCredentialIssueTypeNotFound         = 400247
	CodeCredentialIssueTypeInactive         = 400248
	CodeCredentialIssueOrganizationNotFound = 400249
	CodeCredentialIssueNumberDuplicate      = 400250
	CodeCredentialIssueCompetencyNotFound   = 400251
	CodeCredentialIssueOrganizationInactive = 400252
	CodeCredentialIssueCompetencyInactive   = 400253

	// ── Submission (11) ─────────────────────────────────────────────────────
	CodeCredentialSubmitSuccess       = 401100
	CodeCredentialSubmitStorageFailed = 401141

	// ── Review (12) ─────────────────────────────────────────────────────────
	CodeCredentialReviewSuccess              = 401200
	CodeCredentialReviewNotFound             = 401240
	CodeCredentialReviewAlreadyApproved      = 401241
	CodeCredentialReviewAlreadyRejected      = 401242
	CodeCredentialReviewAlreadyRevoked       = 401243
	CodeCredentialReviewBlockchainSyncFailed = 401244

	// ── Credential Competency Link (13) ─────────────────────────────────────
	CodeCredentialCompetencyLinkSuccess            = 401300
	CodeCredentialCompetencyLinkCredentialNotFound = 401340
	CodeCredentialCompetencyLinkCompetencyNotFound = 401341

	// ── Credential Update (14) ──────────────────────────────────────────────
	CodeCredentialUpdateSuccess    = 401400
	CodeCredentialUpdateNotFound   = 401440
	CodeCredentialUpdateNotPending = 401441

	// ── Metadata Resolution (15) ────────────────────────────────────────────
	// A submitted credential may carry free-text metadata names that had no
	// taxonomy row at submit time. A reviewer resolves each name — linking an
	// existing row or creating one — before the credential can be approved.
	CodeCredentialMetadataResolveSuccess         = 401500
	CodeCredentialMetadataSuggestSuccess         = 401501
	CodeCredentialMetadataResolveNotFound        = 401540
	CodeCredentialMetadataResolveNotPending      = 401541
	CodeCredentialMetadataResolveNothingStaged   = 401542
	CodeCredentialMetadataResolveTargetNotFound  = 401543
	CodeCredentialMetadataResolveTargetInactive  = 401544
	CodeCredentialMetadataResolveNumberDuplicate = 401545
	CodeCredentialApproveUnresolvedMetadata      = 401546
)
