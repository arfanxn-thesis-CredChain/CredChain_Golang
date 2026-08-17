package credential

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"CredChain_Golang/domain"
	"CredChain_Golang/tests/fixtures"
	"CredChain_Golang/tests/gintest"

	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	assert.Equal(t, "type-1", it.TypeID)
	assert.Equal(t, "org-1", it.IssuerOrganizationID)
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
