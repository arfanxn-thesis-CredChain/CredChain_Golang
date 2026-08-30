package credential

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"strings"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	pyai "CredChain_Golang/infrastructure/ai/pyai"
	queryRequest "CredChain_Golang/infrastructure/http/request/query"
	"CredChain_Golang/infrastructure/http/responder"
	"CredChain_Golang/infrastructure/http/response"

	"github.com/gin-gonic/gin"
	"github.com/go-ozzo/ozzo-validation/v4"
	"github.com/samber/lo"
	"go.uber.org/fx"
)

// ── Interface ─────────────────────────────────────────────────────────────

// CredentialHandler is the HTTP layer for credential operations. Method names
// follow the user feature pattern: Paginate, Find, Issue, Revoke, Verify.
// SelfPaginate / SelfFind serve the Holder dashboard at /api/users/self/credentials.
type CredentialHandler interface {
	Paginate(c *gin.Context)
	Find(c *gin.Context)
	Issue(c *gin.Context)
	Submit(c *gin.Context)
	Approve(c *gin.Context)
	Reject(c *gin.Context)
	Update(c *gin.Context)
	Revoke(c *gin.Context)
	Verify(c *gin.Context)
	ReExtract(c *gin.Context)
	SelfPaginate(c *gin.Context)
	SelfFind(c *gin.Context)
	DownloadFile(c *gin.Context)
	LinkCompetencies(c *gin.Context)
}

// ── Implementation & constructor ──────────────────────────────────────────

type credentialHandler struct {
	credSvc CredentialService
}

type CredentialHandlerParams struct {
	fx.In
	CredSvc CredentialService
}

// NewCredentialHandler is the exported factory for FX injection.
func NewCredentialHandler(p CredentialHandlerParams) CredentialHandler {
	return &credentialHandler{credSvc: p.CredSvc}
}

// ── Paginate ──────────────────────────────────────────────────────────────

// Paginate returns a paginated credential list with search, filters, sorts,
// and optional holder/issuer/revoker user expansions via the includes query
// parameter (e.g. ?includes=holder,issuer).
func (h *credentialHandler) Paginate(c *gin.Context) {
	var req queryRequest.QueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	query, err := req.ToDomain()
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	credentials, total, err := h.credSvc.Paginate(c.Request.Context(), query)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	out := mapCredentialsToResponse(credentials)
	responder.SendPagination(c, domain.CodeCredentialFetchSuccess, out, total)
}

// ── Find ──────────────────────────────────────────────────────────────────

// Find returns a single credential by ID with optional user expansions.
// The includes query parameter controls which relations (holder, issuer,
// revoker) are preloaded by the repository.
func (h *credentialHandler) Find(c *gin.Context) {
	id := c.Param("id")

	var req queryRequest.QueryRequest
	c.ShouldBindQuery(&req)
	query, _ := req.ToDomain()
	if query == nil {
		query = &domainQuery.Query{}
	}

	cred, err := h.credSvc.Find(c.Request.Context(), id, query)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	out := mapCredentialsToResponse([]domain.Credential{*cred})
	if len(out) == 0 {
		responder.Send(c, domain.CodeCredentialFetchSuccess, gin.H{})
		return
	}
	responder.Send(c, domain.CodeCredentialFetchSuccess, out[0])
}

// ── Self (Holder) ─────────────────────────────────────────────────────────

// SelfPaginate returns a paginated list of credentials owned by the
// authenticated user (holder_user_id == auth_user.id). Same query DSL as
// Paginate; the holder filter is injected by the service.
func (h *credentialHandler) SelfPaginate(c *gin.Context) {
	var req queryRequest.QueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	query, err := req.ToDomain()
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	credentials, total, err := h.credSvc.SelfPaginate(c.Request.Context(), query)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	out := mapCredentialsToResponse(credentials)
	responder.SendPagination(c, domain.CodeCredentialFetchSuccess, out, total)
}

// SelfFind returns one credential by ID, scoped to the authenticated user.
// Returns 404 (CodeCredentialFetchNotFound) when the credential does not
// exist OR when it exists but is owned by another user — never leaks which
// IDs exist.
func (h *credentialHandler) SelfFind(c *gin.Context) {
	id := c.Param("id")

	var req queryRequest.QueryRequest
	c.ShouldBindQuery(&req)
	query, _ := req.ToDomain()
	if query == nil {
		query = &domainQuery.Query{}
	}

	cred, err := h.credSvc.SelfFind(c.Request.Context(), id, query)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	out := mapCredentialsToResponse([]domain.Credential{*cred})
	if len(out) == 0 {
		responder.Send(c, domain.CodeCredentialFetchSuccess, gin.H{})
		return
	}
	responder.Send(c, domain.CodeCredentialFetchSuccess, out[0])
}

// ── Issue ─────────────────────────────────────────────────────────────────

// Issue parses a multipart form into batch credential issue items and
// delegates to the service layer.
//
// Expected form structure (one set per item, zero-indexed):
//
//	credentials[0][holder_user_id]
//	credentials[0][type_id]
//	credentials[0][issuer_organization_id]
//	credentials[0][number]            (optional)
//	credentials[0][issued_at]         (optional, "2006-01-02")
//	credentials[0][expires_at]        (optional, "2006-01-02")
//	credentials[0][competency_ids]    (optional, comma-separated)
//	credentials[0][name]
//	credentials[0][meta]            (JSON string, optional)
//	credentials[0][file]            (binary upload)
//	credentials[1][...]
func (h *credentialHandler) Issue(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	items, err := buildIssueItems(form)
	if err != nil {
		c.Error(err)
		responder.SendValidationError(c, err)
		return
	}

	req := CredentialIssueRequest{Credentials: items}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}

	serviceItems := make([]CredentialIssuance, len(items))
	for i, it := range items {
		fileBytes, mime, filename, err := readUploadedFile(it.File)
		if err != nil {
			c.Error(err)
			responder.SendError(c, err)
			return
		}
		if !allowedMIMETypes[mime] {
			verrs := validation.Errors{
				fmt.Sprintf("credentials.%d.file", i): validation.NewError("validation_file_type_invalid", ""),
			}
			responder.SendValidationError(c, verrs)
			return
		}
		if int64(len(fileBytes)) > maxFileBytes {
			verrs := validation.Errors{
				fmt.Sprintf("credentials.%d.file", i): validation.NewError("validation_file_max_size", ""),
			}
			responder.SendValidationError(c, verrs)
			return
		}
		serviceItems[i] = CredentialIssuance{
			HolderUserID:         it.HolderUserID,
			TypeID:               it.TypeID,
			IssuerOrganizationID: it.IssuerOrganizationID,
			Number:               it.Number,
			IssuedAt:             parseDatePtr(it.IssuedAt),
			ExpiresAt:            parseDatePtr(it.ExpiresAt),
			Name:                 it.Name,
			Meta:                 it.Meta,
			CompetencyIDs:        it.CompetencyIDs,
			Filename:             filename,
			MIMEType:             mime,
			FileBytes:            fileBytes,
		}
	}

	created, err := h.credSvc.Issue(c.Request.Context(), serviceItems)
	if err != nil {
		c.Error(err)
		if verrs, ok := err.(validation.Errors); ok {
			responder.SendValidationError(c, verrs)
			return
		}
		responder.SendError(c, err)
		return
	}
	out := mapCredentialsToResponse(created)
	responder.Send(c, domain.CodeCredentialIssueSuccess, out)
}

// ── Submit ────────────────────────────────────────────────────────────────

// Submit parses a multipart form into batch self-submission items and
// delegates to the service layer. Mirrors Issue's multipart flow with one
// difference: there is NO holder_user_id key — the holder is the authenticated
// user.
//
// Expected form structure (one set per item, zero-indexed):
//
//	credentials[0][name]
//	credentials[0][type_id]
//	credentials[0][issuer_organization_id]
//	credentials[0][number]            (optional)
//	credentials[0][issued_at]         (required, "2006-01-02")
//	credentials[0][expires_at]        (optional, "2006-01-02")
//	credentials[0][competency_ids]    (optional, comma-separated)
//	credentials[0][meta]            (JSON string, optional)
//	credentials[0][file]            (binary upload)
//	credentials[1][...]
func (h *credentialHandler) Submit(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	items, err := buildSubmitItems(form)
	if err != nil {
		c.Error(err)
		responder.SendValidationError(c, err)
		return
	}

	req := CredentialSubmitRequest{Credentials: items}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}

	serviceItems := make([]CredentialSubmission, len(items))
	for i, it := range items {
		fileBytes, mime, filename, err := readUploadedFile(it.File)
		if err != nil {
			c.Error(err)
			responder.SendError(c, err)
			return
		}
		if !allowedMIMETypes[mime] {
			verrs := validation.Errors{
				fmt.Sprintf("credentials.%d.file", i): validation.NewError("validation_file_type_invalid", ""),
			}
			responder.SendValidationError(c, verrs)
			return
		}
		if int64(len(fileBytes)) > maxFileBytes {
			verrs := validation.Errors{
				fmt.Sprintf("credentials.%d.file", i): validation.NewError("validation_file_max_size", ""),
			}
			responder.SendValidationError(c, verrs)
			return
		}
		serviceItems[i] = CredentialSubmission{
			Name:                 it.Name,
			TypeID:               lo.ToPtr(it.TypeID),
			IssuerOrganizationID: lo.ToPtr(it.IssuerOrganizationID),
			Number:               it.Number,
			IssuedAt:             parseDatePtr(it.IssuedAt),
			ExpiresAt:            parseDatePtr(it.ExpiresAt),
			CompetencyIDs:        it.CompetencyIDs,
			Meta:                 it.Meta,
			Filename:             filename,
			MIMEType:             mime,
			FileBytes:            fileBytes,
		}
	}

	created, err := h.credSvc.Submit(c.Request.Context(), serviceItems)
	if err != nil {
		c.Error(err)
		if verrs, ok := err.(validation.Errors); ok {
			responder.SendValidationError(c, verrs)
			return
		}
		responder.SendError(c, err)
		return
	}
	out := mapCredentialsToResponse(created)
	responder.Send(c, domain.CodeCredentialSubmitSuccess, out)
}

// ── Approve ───────────────────────────────────────────────────────────────

// Approve batch-approves pending self-submissions by ID (JSON body). Approval
// mints the credentials on chain inside the same unit of work — a failed mint
// rolls the approval back and the rows stay pending.
func (h *credentialHandler) Approve(c *gin.Context) {
	var req CredentialApproveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	approved, err := h.credSvc.Approve(c.Request.Context(), req.Ids...)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	out := mapCredentialsToResponse(approved)
	responder.Send(c, domain.CodeCredentialReviewSuccess, out)
}

// ── Reject ────────────────────────────────────────────────────────────────

// Reject batch-rejects pending self-submissions with per-credential reasons
// (JSON body).
func (h *credentialHandler) Reject(c *gin.Context) {
	var req CredentialRejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	rejections := make([]CredentialRejection, len(req.Rejections))
	for i, in := range req.Rejections {
		rejections[i] = CredentialRejection{ID: in.ID, Reason: in.Reason}
	}
	rejected, err := h.credSvc.Reject(c.Request.Context(), rejections)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	out := mapCredentialsToResponse(rejected)
	responder.Send(c, domain.CodeCredentialReviewSuccess, out)
}

// ── Update ────────────────────────────────────────────────────────────────

// Update batch-updates pending credentials (JSON body). Mirrors userHandler.
func (h *credentialHandler) Update(c *gin.Context) {
	var req CredentialUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	updated, err := h.credSvc.Update(c.Request.Context(), req.ToDomain()...)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	out := mapCredentialsToResponse(updated)
	responder.Send(c, domain.CodeCredentialUpdateSuccess, out)
}

// ── Revoke ────────────────────────────────────────────────────────────────

// Revoke batch-revokes credentials by ID (JSON body).
func (h *credentialHandler) Revoke(c *gin.Context) {
	var req CredentialRevokeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	revoked, err := h.credSvc.Revoke(c.Request.Context(), req.Ids...)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	out := mapCredentialsToResponse(revoked)
	responder.Send(c, domain.CodeCredentialRevokeSuccess, out)
}

// ── Verify ────────────────────────────────────────────────────────────────

// Verify accepts a multipart file upload and returns a similarity verdict
// against all known credentials (cache → exact hash → fuzzy pipeline).
func (h *credentialHandler) Verify(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		responder.SendError(c, domain.NewError(domain.CodeCredentialVerifyValidation,
			domain.WithError(err)))
		return
	}
	fileBytes, mime, filename, err := readUploadedFile(fileHeader)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if !allowedMIMETypes[mime] {
		responder.SendError(c, domain.NewError(domain.CodeCredentialVerifyValidation,
			domain.WithMetadata("file_mime", mime)))
		return
	}
	if int64(len(fileBytes)) > maxFileBytes {
		responder.SendError(c, domain.NewError(domain.CodeCredentialVerifyValidation,
			domain.WithMetadata("file_size", len(fileBytes))))
		return
	}
	code, cred, score, percent, err := h.credSvc.Verify(c.Request.Context(), pyai.ExtractFile{
		Filename: filename,
		MIMEType: mime,
		Data:     fileBytes,
	})
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	desc := responder.ResolveMessage(c, code)
	out := response.CredentialVerify{
		VerdictCode:       code,
		SimilarityScore:   score,
		SimilarityPercent: percent,
		Description:       desc,
	}
	if cred != nil {
		dto := response.FromDomainCredential(*cred)
		out.Credential = &dto
	}
	responder.Send(c, code, out)
}

// ── ReExtract ─────────────────────────────────────────────────────────────

// ReExtract resets failed credential extractions to pending and enqueues new
// extract jobs via the River worker.
func (h *credentialHandler) ReExtract(c *gin.Context) {
	var req CredentialReExtractRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	updated, err := h.credSvc.ReExtract(c.Request.Context(), req.Ids...)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeCredentialReExtractSuccess, mapCredentialsToResponse(updated))
}

// ── DownloadFile ──────────────────────────────────────────────────────────

// DownloadFile returns a single credential file with correct Content-Type for
// browser preview. Uses domain.CodeCredentialFileDownloadSuccess (200) on
// success. Authorization via policy (holder owns OR Issuer+).
func (h *credentialHandler) DownloadFile(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		responder.SendError(c, domain.NewError(domain.CodeCredentialFileDownloadNotFound))
		return
	}
	data, filename, mimeType, err := h.credSvc.DownloadFile(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	c.Header("Content-Type", mimeType)
	c.Header("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filepath.Base(filename)))
	c.Data(200, mimeType, data)
}

// ── LinkCompetencies ───────────────────────────────────────────────────────

// LinkCompetencies replaces the credential's competency set (JSON body).
func (h *credentialHandler) LinkCompetencies(c *gin.Context) {
	id := c.Param("id")
	var req CredentialLinkCompetenciesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	if err := h.credSvc.LinkCompetencies(c.Request.Context(), id, req.CompetencyIDs); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeCredentialCompetencyLinkSuccess,
		gin.H{"credential_id": id, "competency_ids": req.CompetencyIDs})
}

// ── Helpers ───────────────────────────────────────────────────────────────

// mapCredentialsToResponse converts domain credentials to response DTOs.
// Preloaded user relations (Holder, Issuer, Revoker) are mapped directly
// from the domain entity — no separate user lookup needed.
func mapCredentialsToResponse(credentials []domain.Credential) []response.Credential {
	out := make([]response.Credential, len(credentials))
	for i, c := range credentials {
		out[i] = response.FromDomainCredential(c)
	}
	return out
}

// readUploadedFile reads a multipart upload into memory and returns
// (bytes, MIME type, filename, error). MIME is taken from the multipart
// header; falls back to extension-based detection.
func readUploadedFile(fh *multipart.FileHeader) ([]byte, string, string, error) {
	src, err := fh.Open()
	if err != nil {
		return nil, "", "", err
	}
	defer src.Close()
	buf, err := io.ReadAll(src)
	if err != nil {
		return nil, "", "", err
	}
	mimeType := fh.Header.Get("Content-Type")
	if mimeType == "" {
		if t := mime.TypeByExtension(strings.ToLower(filepath.Ext(fh.Filename))); t != "" {
			mimeType = t
		} else {
			mimeType = "application/octet-stream"
		}
	}
	return buf, mimeType, fh.Filename, nil
}

// ── Multipart parsing ─────────────────────────────────────────────────────

// buildIssueItems extracts CredentialIssueInput slices from a parsed multipart
// form. Keys follow the pattern:
//
//	credentials[N][holder_user_id] = "ulid"
//	credentials[N][type_id] = "ulid"
//	credentials[N][issuer_organization_id] = "ulid"
//	credentials[N][number] = "N-001"
//	credentials[N][issued_at] = "2026-08-01"
//	credentials[N][expires_at] = "2026-09-01"
//	credentials[N][competency_ids] = "comp-a,comp-b"  (comma-separated)
//	credentials[N][name] = "Bachelor's Degree"
//	credentials[N][meta] = `{"institution":"UI"}`
//	credentials[N][file] = <binary>
func buildIssueItems(form *multipart.Form) ([]CredentialIssueInput, error) {
	values := form.Value
	files := form.File

	idxSet := make(map[int]bool)
	for k := range values {
		if idx, ok := parseItemIndex(k); ok {
			idxSet[idx] = true
		}
	}
	for k := range files {
		if idx, ok := parseItemIndex(k); ok {
			idxSet[idx] = true
		}
	}
	if len(idxSet) == 0 {
		return nil, nil
	}

	maxIdx := lo.Max(lo.Keys(idxSet))

	items := make([]CredentialIssueInput, maxIdx+1)
	for i := 0; i <= maxIdx; i++ {
		key := "credentials[" + strconv.Itoa(i) + "][holder_user_id]"
		if v, ok := values[key]; ok && len(v) > 0 {
			items[i].HolderUserID = v[0]
		}
		key = "credentials[" + strconv.Itoa(i) + "][type_id]"
		if v, ok := values[key]; ok && len(v) > 0 {
			items[i].TypeID = v[0]
		}
		key = "credentials[" + strconv.Itoa(i) + "][issuer_organization_id]"
		if v, ok := values[key]; ok && len(v) > 0 {
			items[i].IssuerOrganizationID = v[0]
		}
		key = "credentials[" + strconv.Itoa(i) + "][number]"
		if v, ok := values[key]; ok && len(v) > 0 && v[0] != "" {
			num := v[0]
			items[i].Number = &num
		}
		key = "credentials[" + strconv.Itoa(i) + "][issued_at]"
		if v, ok := values[key]; ok && len(v) > 0 && v[0] != "" {
			issuedAt := v[0]
			items[i].IssuedAt = &issuedAt
		}
		key = "credentials[" + strconv.Itoa(i) + "][expires_at]"
		if v, ok := values[key]; ok && len(v) > 0 && v[0] != "" {
			expiresAt := v[0]
			items[i].ExpiresAt = &expiresAt
		}
		key = "credentials[" + strconv.Itoa(i) + "][competency_ids]"
		if v, ok := values[key]; ok && len(v) > 0 && v[0] != "" {
			parts := strings.Split(v[0], ",")
			ids := make([]string, 0, len(parts))
			for _, p := range parts {
				if p = strings.TrimSpace(p); p != "" {
					ids = append(ids, p)
				}
			}
			items[i].CompetencyIDs = ids
		}
		key = "credentials[" + strconv.Itoa(i) + "][name]"
		if v, ok := values[key]; ok && len(v) > 0 {
			items[i].Name = v[0]
		}
		key = "credentials[" + strconv.Itoa(i) + "][meta]"
		if v, ok := values[key]; ok && len(v) > 0 && v[0] != "" {
			var m map[string]any
			if err := json.Unmarshal([]byte(v[0]), &m); err == nil {
				items[i].Meta = m
			}
		}
		key = "credentials[" + strconv.Itoa(i) + "][file]"
		if fh, ok := files[key]; ok && len(fh) > 0 {
			items[i].File = fh[0]
		}
	}

	out := lo.Filter(items, func(it CredentialIssueInput, _ int) bool {
		return it.HolderUserID != "" || it.Name != "" || it.File != nil
	})
	return out, nil
}

func parseItemIndex(key string) (int, bool) {
	if !strings.HasPrefix(key, "credentials[") {
		return 0, false
	}
	closeBracket := strings.Index(key, "]")
	if closeBracket == -1 {
		return 0, false
	}
	idxStr := key[12:closeBracket]
	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		return 0, false
	}
	return idx, true
}

// buildSubmitItems extracts CredentialSubmitInput slices from a parsed
// multipart form. Mirrors buildIssueItems with NO holder_user_id key — the
// holder is the authenticated user. Keys follow the pattern:
//
//	credentials[N][name] = "Bachelor's Degree"
//	credentials[N][type_id] = "ulid"
//	credentials[N][issuer_organization_id] = "ulid"
//	credentials[N][number] = "N-001"
//	credentials[N][issued_at] = "2026-08-01"
//	credentials[N][expires_at] = "2026-09-01"
//	credentials[N][competency_ids] = "comp-a,comp-b"  (comma-separated)
//	credentials[N][meta] = `{"institution":"UI"}`
//	credentials[N][file] = <binary>
func buildSubmitItems(form *multipart.Form) ([]CredentialSubmitInput, error) {
	values := form.Value
	files := form.File

	idxSet := make(map[int]bool)
	for k := range values {
		if idx, ok := parseItemIndex(k); ok {
			idxSet[idx] = true
		}
	}
	for k := range files {
		if idx, ok := parseItemIndex(k); ok {
			idxSet[idx] = true
		}
	}
	if len(idxSet) == 0 {
		return nil, nil
	}

	maxIdx := lo.Max(lo.Keys(idxSet))

	items := make([]CredentialSubmitInput, maxIdx+1)
	for i := 0; i <= maxIdx; i++ {
		key := "credentials[" + strconv.Itoa(i) + "][name]"
		if v, ok := values[key]; ok && len(v) > 0 {
			items[i].Name = v[0]
		}
		key = "credentials[" + strconv.Itoa(i) + "][type_id]"
		if v, ok := values[key]; ok && len(v) > 0 {
			items[i].TypeID = v[0]
		}
		key = "credentials[" + strconv.Itoa(i) + "][issuer_organization_id]"
		if v, ok := values[key]; ok && len(v) > 0 {
			items[i].IssuerOrganizationID = v[0]
		}
		key = "credentials[" + strconv.Itoa(i) + "][number]"
		if v, ok := values[key]; ok && len(v) > 0 && v[0] != "" {
			num := v[0]
			items[i].Number = &num
		}
		key = "credentials[" + strconv.Itoa(i) + "][issued_at]"
		if v, ok := values[key]; ok && len(v) > 0 && v[0] != "" {
			issuedAt := v[0]
			items[i].IssuedAt = &issuedAt
		}
		key = "credentials[" + strconv.Itoa(i) + "][expires_at]"
		if v, ok := values[key]; ok && len(v) > 0 && v[0] != "" {
			expiresAt := v[0]
			items[i].ExpiresAt = &expiresAt
		}
		key = "credentials[" + strconv.Itoa(i) + "][competency_ids]"
		if v, ok := values[key]; ok && len(v) > 0 && v[0] != "" {
			parts := strings.Split(v[0], ",")
			ids := make([]string, 0, len(parts))
			for _, p := range parts {
				if p = strings.TrimSpace(p); p != "" {
					ids = append(ids, p)
				}
			}
			items[i].CompetencyIDs = ids
		}
		key = "credentials[" + strconv.Itoa(i) + "][meta]"
		if v, ok := values[key]; ok && len(v) > 0 && v[0] != "" {
			var m map[string]any
			if err := json.Unmarshal([]byte(v[0]), &m); err == nil {
				items[i].Meta = m
			}
		}
		key = "credentials[" + strconv.Itoa(i) + "][file]"
		if fh, ok := files[key]; ok && len(fh) > 0 {
			items[i].File = fh[0]
		}
	}

	out := lo.Filter(items, func(it CredentialSubmitInput, _ int) bool {
		return it.Name != "" || it.File != nil
	})
	return out, nil
}
