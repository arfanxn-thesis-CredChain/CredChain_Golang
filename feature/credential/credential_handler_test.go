package credential

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"CredChain_Golang/domain"
	httpContext "CredChain_Golang/infrastructure/http/context"
	"CredChain_Golang/infrastructure/http/middleware"
	"CredChain_Golang/tests/fixtures"
	"CredChain_Golang/tests/gintest"
	"CredChain_Golang/tests/mocks"

	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestParseItemIndex(t *testing.T) {
	tests := []struct {
		key     string
		wantIdx int
		wantOK  bool
	}{
		{"credentials[0][holder_user_id]", 0, true},
		{"credentials[99][name]", 99, true},
		{"credentials[abc][x]", 0, false},
		{"not_credentials[0][x]", 0, false},
	}
	for _, tt := range tests {
		got, ok := parseItemIndex(tt.key)
		assert.Equal(t, tt.wantOK, ok, "key=%s", tt.key)
		if tt.wantOK {
			assert.Equal(t, tt.wantIdx, got, "key=%s", tt.key)
		}
	}
}

func TestMapCredentialsToResponse(t *testing.T) {
	creds := []domain.Credential{
		{ID: "c1", Name: "n1"},
		{ID: "c2", Name: "n2"},
	}
	out := mapCredentialsToResponse(creds)
	assert.Len(t, out, 2)
	assert.Equal(t, "c1", out[0].ID)
	assert.Equal(t, "n2", out[1].Name)
}

func TestMapCredentialsToResponse_Empty(t *testing.T) {
	out := mapCredentialsToResponse([]domain.Credential{})
	assert.Len(t, out, 0)
}

func TestBuildIssueItems(t *testing.T) {
	form := &multipart.Form{
		Value: map[string][]string{
			"credentials[0][holder_user_id]": {"holder-1"},
			"credentials[0][name]":           {"Degree"},
			"credentials[1][holder_user_id]": {"holder-2"},
			"credentials[1][name]":           {"Diploma"},
		},
		File: map[string][]*multipart.FileHeader{},
	}
	items, err := buildIssueItems(form)
	assert.NoError(t, err)
	assert.Len(t, items, 2)
	assert.Equal(t, "holder-1", items[0].HolderUserID)
	assert.Equal(t, "Degree", items[0].Name)
	assert.Equal(t, "holder-2", items[1].HolderUserID)
}

func TestBuildIssueItems_ParsesExtendedFields(t *testing.T) {
	number := "N-001"
	form := &multipart.Form{
		Value: map[string][]string{
			"credentials[0][holder_user_id]":         {"holder-1"},
			"credentials[0][name]":                   {"Degree"},
			"credentials[0][type_id]":                {"type-1"},
			"credentials[0][issuer_organization_id]": {"org-1"},
			"credentials[0][number]":                 {"N-001"},
			"credentials[0][issued_at]":              {"2026-08-01"},
			"credentials[0][expires_at]":             {"2026-09-01"},
			"credentials[0][competency_ids]":         {"comp-a, comp-b ,,comp-c"},
		},
		File: map[string][]*multipart.FileHeader{},
	}
	items, err := buildIssueItems(form)
	assert.NoError(t, err)
	assert.Len(t, items, 1)
	it := items[0]
	assert.Equal(t, "type-1", it.TypeID)
	assert.Equal(t, "org-1", it.IssuerOrganizationID)
	assert.Equal(t, &number, it.Number)
	assert.Equal(t, "2026-08-01", *it.IssuedAt)
	assert.Equal(t, "2026-09-01", *it.ExpiresAt)
	assert.Equal(t, []string{"comp-a", "comp-b", "comp-c"}, it.CompetencyIDs)
}

func TestBuildSubmitItems(t *testing.T) {
	form := &multipart.Form{
		Value: map[string][]string{
			"credentials[0][name]": {"Degree"},
			"credentials[1][name]": {"Diploma"},
		},
		File: map[string][]*multipart.FileHeader{},
	}
	items, err := buildSubmitItems(form)
	assert.NoError(t, err)
	assert.Len(t, items, 2)
	assert.Equal(t, "Degree", items[0].Name)
	assert.Equal(t, "Diploma", items[1].Name)
}

func TestBuildSubmitItems_ParsesExtendedFields_NoHolderKey(t *testing.T) {
	number := "N-001"
	form := &multipart.Form{
		Value: map[string][]string{
			"credentials[0][name]":                   {"Degree"},
			"credentials[0][type_id]":                {"type-1"},
			"credentials[0][issuer_organization_id]": {"org-1"},
			"credentials[0][number]":                 {"N-001"},
			"credentials[0][issued_at]":              {"2026-08-01"},
			"credentials[0][expires_at]":             {"2026-09-01"},
			"credentials[0][competency_ids]":         {"comp-a, comp-b ,,comp-c"},
			"credentials[0][meta]":                   {`{"institution":"UI"}`},
		},
		File: map[string][]*multipart.FileHeader{},
	}
	items, err := buildSubmitItems(form)
	assert.NoError(t, err)
	assert.Len(t, items, 1)
	it := items[0]
	assert.Equal(t, "type-1", *it.TypeID)
	assert.Equal(t, "org-1", *it.IssuerOrganizationID)
	assert.Equal(t, &number, it.Number)
	assert.Equal(t, "2026-08-01", *it.IssuedAt)
	assert.Equal(t, "2026-09-01", *it.ExpiresAt)
	assert.Equal(t, []string{"comp-a", "comp-b", "comp-c"}, it.CompetencyIDs)
	assert.Equal(t, map[string]any{"institution": "UI"}, it.Meta)
}

func TestHandler_Submit_Success(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("credentials[0][name]", "Degree")
	_ = writer.WriteField("credentials[0][type_id]", "t1")
	_ = writer.WriteField("credentials[0][issuer_organization_id]", "o1")
	_ = writer.WriteField("credentials[0][issued_at]", "2026-08-01")
	mimeHdr := make(textproto.MIMEHeader)
	mimeHdr.Set("Content-Disposition", `form-data; name="credentials[0][file]"; filename="test.pdf"`)
	mimeHdr.Set("Content-Type", "application/pdf")
	part, _ := writer.CreatePart(mimeHdr)
	part.Write([]byte("test"))
	writer.Close()

	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req
	c.Set("user", user)
	bundle := gintest.LoadTestI18nBundle(t)
	c.Set("i18n_localizer", i18n.NewLocalizer(bundle, "en"))

	svc := &mockCredentialService{}
	svc.On("Submit", mock.Anything, mock.Anything).Return([]domain.Credential{{ID: "c1"}}, nil)
	h := &credentialHandler{credSvc: svc}
	h.Submit(c)
	assert.Equal(t, http.StatusOK, c.Writer.Status())
	svc.AssertCalled(t, "Submit", mock.Anything, mock.Anything)
}

func TestHandler_Submit_MissingIssuedAt(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("credentials[0][name]", "Degree")
	_ = writer.WriteField("credentials[0][type_id]", "t1")
	_ = writer.WriteField("credentials[0][issuer_organization_id]", "o1")
	writer.Close()

	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req
	c.Set("user", user)
	bundle := gintest.LoadTestI18nBundle(t)
	c.Set("i18n_localizer", i18n.NewLocalizer(bundle, "en"))

	svc := &mockCredentialService{}
	h := &credentialHandler{credSvc: svc}
	h.Submit(c)
	assert.Equal(t, http.StatusBadRequest, c.Writer.Status())
	svc.AssertNotCalled(t, "Submit", mock.Anything, mock.Anything)
}

func TestHandler_Submit_InvalidFileType(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("credentials[0][name]", "Degree")
	_ = writer.WriteField("credentials[0][type_id]", "t1")
	_ = writer.WriteField("credentials[0][issuer_organization_id]", "o1")
	_ = writer.WriteField("credentials[0][issued_at]", "2026-08-01")
	mimeHdr := make(textproto.MIMEHeader)
	mimeHdr.Set("Content-Disposition", `form-data; name="credentials[0][file]"; filename="test.txt"`)
	mimeHdr.Set("Content-Type", "text/plain")
	part, _ := writer.CreatePart(mimeHdr)
	part.Write([]byte("test"))
	writer.Close()

	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req
	c.Set("user", user)
	bundle := gintest.LoadTestI18nBundle(t)
	c.Set("i18n_localizer", i18n.NewLocalizer(bundle, "en"))

	svc := &mockCredentialService{}
	h := &credentialHandler{credSvc: svc}
	h.Submit(c)
	assert.Equal(t, http.StatusBadRequest, c.Writer.Status())
	svc.AssertNotCalled(t, "Submit", mock.Anything, mock.Anything)
}

func TestHandler_Paginate_Success(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	c, rr := gintest.NewContext(t,
		gintest.WithUser(&user),
		gintest.WithMethod(http.MethodGet),
		gintest.WithPath("/"),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	svc := &mockCredentialService{}
	svc.On("Paginate", mock.Anything, mock.Anything).Return([]domain.Credential{{ID: "c1", Name: "test"}}, 1, nil)
	h := &credentialHandler{credSvc: svc}
	h.Paginate(c)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandler_Paginate_ServiceError(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	c, rr := gintest.NewContext(t,
		gintest.WithUser(&user),
		gintest.WithMethod(http.MethodGet),
		gintest.WithPath("/"),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	svc := &mockCredentialService{}
	svc.On("Paginate", mock.Anything, mock.Anything).Return([]domain.Credential(nil), 0, assert.AnError)
	h := &credentialHandler{credSvc: svc}
	h.Paginate(c)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestHandler_Find_Success(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	c, rr := gintest.NewContext(t,
		gintest.WithUser(&user),
		gintest.WithMethod(http.MethodGet),
		gintest.WithPath("/c1"),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	c.Params = gin.Params{{Key: "id", Value: "c1"}}
	svc := &mockCredentialService{}
	svc.On("Find", mock.Anything, "c1", mock.Anything).Return(&domain.Credential{ID: "c1"}, nil)
	h := &credentialHandler{credSvc: svc}
	h.Find(c)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandler_Find_NotFound(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	c, rr := gintest.NewContext(t,
		gintest.WithUser(&user),
		gintest.WithMethod(http.MethodGet),
		gintest.WithPath("/missing"),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	c.Params = gin.Params{{Key: "id", Value: "missing"}}
	svc := &mockCredentialService{}
	svc.On("Find", mock.Anything, "missing", mock.Anything).Return(nil, domain.NewError(domain.CodeCredentialFetchNotFound))
	h := &credentialHandler{credSvc: svc}
	h.Find(c)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandler_SelfPaginate_Success(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	c, rr := gintest.NewContext(t,
		gintest.WithUser(&user),
		gintest.WithMethod(http.MethodGet),
		gintest.WithPath("/"),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	svc := &mockCredentialService{}
	svc.On("SelfPaginate", mock.Anything, mock.Anything).Return([]domain.Credential{{ID: "c1", HolderUserID: user.Id}}, 1, nil)
	h := &credentialHandler{credSvc: svc}
	h.SelfPaginate(c)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandler_SelfFind_Success(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	c, rr := gintest.NewContext(t,
		gintest.WithUser(&user),
		gintest.WithMethod(http.MethodGet),
		gintest.WithPath("/c1"),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	c.Params = gin.Params{{Key: "id", Value: "c1"}}
	svc := &mockCredentialService{}
	svc.On("SelfFind", mock.Anything, "c1", mock.Anything).Return(&domain.Credential{ID: "c1"}, nil)
	h := &credentialHandler{credSvc: svc}
	h.SelfFind(c)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandler_Revoke_Success(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	c, rr := gintest.NewContext(t,
		gintest.WithUser(&user),
		gintest.WithMethod("POST"),
		gintest.WithPath("/"),
		gintest.WithBody(map[string]any{"ids": []string{"c1"}}),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	svc := &mockCredentialService{}
	svc.On("Revoke", mock.Anything, mock.Anything).Return([]domain.Credential{{ID: "c1"}}, nil)
	h := &credentialHandler{credSvc: svc}
	h.Revoke(c)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandler_ReExtract_Success(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	c, rr := gintest.NewContext(t,
		gintest.WithUser(&user),
		gintest.WithMethod("POST"),
		gintest.WithPath("/"),
		gintest.WithBody(map[string]any{"ids": []string{"c1"}}),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	svc := &mockCredentialService{}
	svc.On("ReExtract", mock.Anything, mock.Anything).Return([]domain.Credential{{ID: "c1"}}, nil)
	h := &credentialHandler{credSvc: svc}
	h.ReExtract(c)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandler_Approve_Success(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	c, rr := gintest.NewContext(t,
		gintest.WithUser(&user),
		gintest.WithMethod("POST"),
		gintest.WithPath("/"),
		gintest.WithBody(map[string]any{"ids": []string{"c1"}}),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	svc := &mockCredentialService{}
	svc.On("Approve", mock.Anything, mock.Anything).Return([]domain.Credential{{ID: "c1"}}, nil)
	h := &credentialHandler{credSvc: svc}
	h.Approve(c)
	assert.Equal(t, http.StatusOK, rr.Code)
	svc.AssertCalled(t, "Approve", mock.Anything, mock.Anything)
}

func TestHandler_Approve_ValidationError(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	c, rr := gintest.NewContext(t,
		gintest.WithUser(&user),
		gintest.WithMethod("POST"),
		gintest.WithPath("/"),
		gintest.WithBody(map[string]any{"ids": []string{}}),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	svc := &mockCredentialService{}
	h := &credentialHandler{credSvc: svc}
	h.Approve(c)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	svc.AssertNotCalled(t, "Approve", mock.Anything, mock.Anything)
}

func TestHandler_Reject_Success(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	c, rr := gintest.NewContext(t,
		gintest.WithUser(&user),
		gintest.WithMethod("POST"),
		gintest.WithPath("/"),
		gintest.WithBody(map[string]any{"rejections": []map[string]any{{"id": "c1", "reason": "bad scan"}}}),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	svc := &mockCredentialService{}
	svc.On("Reject", mock.Anything, mock.Anything).Return([]domain.Credential{{ID: "c1"}}, nil)
	h := &credentialHandler{credSvc: svc}
	h.Reject(c)
	assert.Equal(t, http.StatusOK, rr.Code)
	svc.AssertCalled(t, "Reject", mock.Anything, mock.Anything)
}

func TestHandler_Reject_ValidationError(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	c, rr := gintest.NewContext(t,
		gintest.WithUser(&user),
		gintest.WithMethod("POST"),
		gintest.WithPath("/"),
		gintest.WithBody(map[string]any{"rejections": []map[string]any{{"id": "c1", "reason": ""}}}),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	svc := &mockCredentialService{}
	h := &credentialHandler{credSvc: svc}
	h.Reject(c)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	svc.AssertNotCalled(t, "Reject", mock.Anything, mock.Anything)
}

func TestHandler_Update_Success(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	c, rr := gintest.NewContext(t,
		gintest.WithUser(&user),
		gintest.WithMethod(http.MethodPut),
		gintest.WithPath("/"),
		gintest.WithBody(map[string]any{"credentials": []map[string]any{{"id": "c1", "name": "new"}}}),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	svc := &mockCredentialService{}
	svc.On("Update", mock.Anything, mock.Anything).Return([]domain.Credential{{ID: "c1", Name: "new"}}, nil)
	h := &credentialHandler{credSvc: svc}
	h.Update(c)
	assert.Equal(t, http.StatusOK, rr.Code)
	svc.AssertCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestHandler_Update_ValidationError(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	c, rr := gintest.NewContext(t,
		gintest.WithUser(&user),
		gintest.WithMethod(http.MethodPut),
		gintest.WithPath("/"),
		gintest.WithBody(map[string]any{"credentials": []map[string]any{{"name": "missing id"}}}),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	svc := &mockCredentialService{}
	h := &credentialHandler{credSvc: svc}
	h.Update(c)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	svc.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestHandler_Verify_NoFile(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	c, rr := gintest.NewContext(t,
		gintest.WithUser(&user),
		gintest.WithMethod("POST"),
		gintest.WithPath("/"),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	h := &credentialHandler{credSvc: &mockCredentialService{}}
	h.Verify(c)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandler_Verify_PublicEndpoint_Succeeds(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("POST", "/", nil)
	c.Request = req
	bundle := gintest.LoadTestI18nBundle(t)
	c.Set("i18n_localizer", i18n.NewLocalizer(bundle, "en"))

	svc := &mockCredentialService{}
	score := 0.95
	pct := "95.0%"
	svc.On("Verify", mock.Anything, mock.Anything).Return(domain.CodeCredentialVerifyAuthentic, &domain.Credential{ID: "c1"}, &score, &pct, nil)

	h := &credentialHandler{credSvc: svc}
	h.Verify(c)
	assert.NotEqual(t, http.StatusForbidden, c.Writer.Status())
}

func TestHandler_Verify_ServiceReturnsVerdict(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	mimeHdr := make(textproto.MIMEHeader)
	mimeHdr.Set("Content-Disposition", `form-data; name="file"; filename="test.pdf"`)
	part, _ := writer.CreatePart(mimeHdr)
	part.Write([]byte("test"))
	writer.Close()

	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req
	c.Set("user", user)
	bundle := gintest.LoadTestI18nBundle(t)
	c.Set("i18n_localizer", i18n.NewLocalizer(bundle, "en"))

	svc := &mockCredentialService{}
	score := 0.95
	pct := "95.0%"
	svc.On("Verify", mock.Anything, mock.Anything).Return(domain.CodeCredentialVerifyAuthentic, &domain.Credential{ID: "c1"}, &score, &pct, nil)

	h := &credentialHandler{credSvc: svc}
	h.Verify(c)
	assert.Equal(t, http.StatusOK, c.Writer.Status())
}

func TestCredentialHandler_LinkCompetencies_Success(t *testing.T) {
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodPut),
		gintest.WithPath("/api/credentials/cred-1/competencies"),
		gintest.WithBody(CredentialLinkCompetenciesRequest{CompetencyIDs: []string{"comp-a", "comp-b"}}),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	c.Params = gin.Params{{Key: "id", Value: "cred-1"}}

	svc := &mockCredentialService{}
	svc.On("LinkCompetencies", mock.Anything, "cred-1", []string{"comp-a", "comp-b"}).Return(nil)

	h := &credentialHandler{credSvc: svc}
	h.LinkCompetencies(c)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			CredentialID  string   `json:"credential_id"`
			CompetencyIDs []string `json:"competency_ids"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, domain.CodeCredentialCompetencyLinkSuccess, resp.Code)
	assert.Equal(t, "cred-1", resp.Data.CredentialID)
	assert.Equal(t, []string{"comp-a", "comp-b"}, resp.Data.CompetencyIDs)
	svc.AssertExpectations(t)
}

func TestCredentialHandler_LinkCompetencies_ValidationError(t *testing.T) {
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ids := make([]string, 101)
	for i := range ids {
		ids[i] = "x"
	}
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodPut),
		gintest.WithPath("/api/credentials/cred-1/competencies"),
		gintest.WithBody(CredentialLinkCompetenciesRequest{CompetencyIDs: ids}),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	c.Params = gin.Params{{Key: "id", Value: "cred-1"}}

	svc := &mockCredentialService{}
	h := &credentialHandler{credSvc: svc}
	h.LinkCompetencies(c)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	svc.AssertNotCalled(t, "LinkCompetencies", mock.Anything, mock.Anything, mock.Anything)
}

func TestCredentialHandler_LinkCompetencies_CredentialNotFound(t *testing.T) {
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodPut),
		gintest.WithPath("/api/credentials/cred-missing/competencies"),
		gintest.WithBody(CredentialLinkCompetenciesRequest{CompetencyIDs: []string{"comp-a"}}),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	c.Params = gin.Params{{Key: "id", Value: "cred-missing"}}

	svc := &mockCredentialService{}
	svc.On("LinkCompetencies", mock.Anything, "cred-missing", []string{"comp-a"}).
		Return(domain.NewError(domain.CodeCredentialCompetencyLinkCredentialNotFound,
			domain.WithMetadata("credential_id", "cred-missing")))

	h := &credentialHandler{credSvc: svc}
	h.LinkCompetencies(c)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	var resp struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, domain.CodeCredentialCompetencyLinkCredentialNotFound, resp.Code)
	svc.AssertExpectations(t)
}

// newTestRouter mirrors the router.go wiring for the routes under test:
// taxonomy GET is open, taxonomy writes and the credential metadata routes are
// guarded by IssuerRoleMiddleware. AuthMiddleware (JWT + DB) is stood in for by
// withHolderAuth, which injects the holder user straight into the context.
func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	auth := &mocks.MockAuthorityService{}
	auth.On("HasRoleOrAbove", mock.Anything, mock.Anything, domain.RoleIssuer).Return(false)
	issuer := gin.HandlerFunc(middleware.NewIssuerRoleMiddleware(middleware.RoleMiddlewareParams{AuthorityService: auth}))
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }

	r := gin.New()
	api := r.Group("/api")
	{
		creds := api.Group("/credentials")
		{
			creds.PUT("/:id/metadata", issuer, ok)
		}
		credentialTypes := api.Group("/credential-types")
		{
			credentialTypes.GET("", ok)
			credentialTypes.POST("", issuer, ok)
		}
		issuerOrganizations := api.Group("/issuer-organizations")
		{
			issuerOrganizations.GET("", ok)
			issuerOrganizations.POST("", issuer, ok)
		}
		competencies := api.Group("/competencies")
		{
			competencies.GET("", ok)
			competencies.POST("", issuer, ok)
		}
	}
	return r
}

// withHolderAuth injects a holder user into the request context, mirroring what
// AuthMiddleware does after validating a JWT. Mutates req in place.
func withHolderAuth(t *testing.T, req *http.Request) {
	t.Helper()
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	*req = *req.WithContext(context.WithValue(req.Context(), httpContext.UserKey, &user))
}

func TestResolveMetadataRouteRequiresIssuer(t *testing.T) {
	r := newTestRouter(t)

	req := httptest.NewRequest(http.MethodPut,
		"/api/credentials/01JCRED000000000000000000/metadata",
		strings.NewReader(`{"type_id":"01JTYPE00000000000000000"}`))
	req.Header.Set("Content-Type", "application/json")
	withHolderAuth(t, req)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("holder got %d, want 403", w.Code)
	}
}

func TestTaxonomyStoreRequiresIssuer(t *testing.T) {
	r := newTestRouter(t)

	for _, path := range []string{"/api/competencies", "/api/credential-types", "/api/issuer-organizations"} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"name":"Sneaky"}`))
		req.Header.Set("Content-Type", "application/json")
		withHolderAuth(t, req)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("%s: holder got %d, want 403", path, w.Code)
		}
	}
}

func TestTaxonomyListStaysOpenToHolders(t *testing.T) {
	r := newTestRouter(t)

	for _, path := range []string{"/api/competencies", "/api/credential-types", "/api/issuer-organizations"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		withHolderAuth(t, req)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("%s: holder got %d, want 200", path, w.Code)
		}
	}
}
