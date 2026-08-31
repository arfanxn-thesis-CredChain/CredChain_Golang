package credential

import (
	"context"
	"strconv"
	"testing"
	"time"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/infrastructure/database/gorm/model"
	"CredChain_Golang/tests/db"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func openCredRepo(t *testing.T) *gormCredentialRepository {
	t.Helper()
	return &gormCredentialRepository{db: db.OpenInMemorySQLite(t)}
}

func strPtr(s string) *string { return &s }

func timePtr(t time.Time) *time.Time { return &t }

func TestGormCredentialRepository_CountByTypeAndOrganizationIds(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "i1", IssuerOrganizationID: strPtr("org-a"), TypeID: strPtr("type-a"), Name: "A", FileHash: "0x1"},
		domain.Credential{ID: "c2", HolderUserID: "h1", IssuerUserID: "i1", IssuerOrganizationID: strPtr("org-a"), TypeID: strPtr("type-b"), Name: "B", FileHash: "0x2"},
		domain.Credential{ID: "c3", HolderUserID: "h1", IssuerUserID: "i1", IssuerOrganizationID: strPtr("org-b"), TypeID: strPtr("type-b"), Name: "C", FileHash: "0x3"},
	)
	require.NoError(t, err)

	typeCount, err := repo.CountByTypeIds(ctx, "type-a", "type-b")
	require.NoError(t, err)
	assert.Equal(t, int64(3), typeCount)

	typeCountSingle, err := repo.CountByTypeIds(ctx, "type-a")
	require.NoError(t, err)
	assert.Equal(t, int64(1), typeCountSingle)

	orgCount, err := repo.CountByIssuerOrganizationIds(ctx, "org-a")
	require.NoError(t, err)
	assert.Equal(t, int64(2), orgCount)

	none, err := repo.CountByTypeIds(ctx, "missing")
	require.NoError(t, err)
	assert.Equal(t, int64(0), none)
}

func TestGormCredentialRepository_CountActiveByFileHashes(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()
	now := time.Now()

	// One pending (active) + one revoked + one rejected row sharing a hash.
	// The partial unique index only guards active rows, so all three rows are
	// storable; CountActiveByFileHashes must count only the active one.
	_, err := repo.Store(ctx,
		domain.Credential{ID: "pending", HolderUserID: "h1", IssuerUserID: "i1", Name: "P", FileHash: "0xshared"},
		domain.Credential{ID: "revoked", HolderUserID: "h1", IssuerUserID: "i1", Name: "R", FileHash: "0xshared", RevokedAt: &now},
		domain.Credential{ID: "rejected", HolderUserID: "h1", IssuerUserID: "i1", Name: "J", FileHash: "0xshared", RejectedAt: &now},
	)
	require.NoError(t, err)

	count, err := repo.CountActiveByFileHashes(ctx, "0xshared")
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	multiple, err := repo.CountActiveByFileHashes(ctx, "0xshared", "0xmissing")
	require.NoError(t, err)
	assert.Equal(t, int64(1), multiple)

	none, err := repo.CountActiveByFileHashes(ctx, "0xmissing")
	require.NoError(t, err)
	assert.Equal(t, int64(0), none)

	empty, err := repo.CountActiveByFileHashes(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), empty)
}

func TestGormCredentialVerificationPathExcludesPending(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		domain.Credential{ID: "approved", HolderUserID: "h1", IssuerUserID: "i1", Name: "Approved", FileHash: "0xapp", ApprovedAt: timePtr(time.Now())},
		domain.Credential{ID: "pending", HolderUserID: "h1", IssuerUserID: "i1", Name: "Pending", FileHash: "0xpen"},
	)
	require.NoError(t, err)

	byID, err := repo.FindVerifiableById(ctx, "pending", nil)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	assert.Nil(t, byID)

	byIDs, err := repo.FindVerifiableByIds(ctx, []string{"pending", "approved"}, nil)
	require.NoError(t, err)
	require.Len(t, byIDs, 1)
	assert.Equal(t, "approved", byIDs[0].ID)

	byHash, err := repo.FindByFileHashes(ctx, []string{"0xpen", "0xapp"}, nil)
	require.NoError(t, err)
	require.Len(t, byHash, 1)
	assert.Equal(t, "approved", byHash[0].ID)
}

func TestGormCredentialStore_FindByIds(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "iss", Name: "a", FileHash: "0xaa"},
		domain.Credential{ID: "c2", HolderUserID: "h2", IssuerUserID: "iss", Name: "b", FileHash: "0xbb"},
	)
	require.NoError(t, err)

	both, err := repo.FindByIds(ctx, []string{"c1", "c2"}, nil)
	require.NoError(t, err)
	assert.Len(t, both, 2)

	one, err := repo.FindByIds(ctx, []string{"c1", "missing"}, nil)
	require.NoError(t, err)
	assert.Len(t, one, 1)
	assert.Equal(t, "c1", one[0].ID)
}

func TestGormCredentialFindByFileHashes(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "iss", Name: "a", FileHash: "0xaa", ApprovedAt: timePtr(time.Now())},
	)
	require.NoError(t, err)

	found, err := repo.FindByFileHashes(ctx, []string{"0xaa", "0xzz"}, nil)
	require.NoError(t, err)
	assert.Len(t, found, 1)
	assert.Equal(t, "0xaa", found[0].FileHash)
}

func TestGormCredentialFindByHolderId(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "iss", Name: "a", FileHash: "0xaa"},
		domain.Credential{ID: "c2", HolderUserID: "h2", IssuerUserID: "iss", Name: "b", FileHash: "0xbb"},
	)
	require.NoError(t, err)

	found, err := repo.FindByHolderId(ctx, "h1", nil)
	require.NoError(t, err)
	assert.Len(t, found, 1)
	assert.Equal(t, "c1", found[0].ID)
}

func TestGormCredentialFind(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "iss", Name: "a", FileHash: "0xaa"},
	)
	require.NoError(t, err)

	got, err := repo.Find(ctx, "c1", &domainQuery.Query{})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "c1", got.ID)

	missing, err := repo.Find(ctx, "missing", &domainQuery.Query{})
	assert.Error(t, err)
	assert.Nil(t, missing)
}

func TestGormCredentialUpdate(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "iss", Name: "a", FileHash: "0xaa"},
	)
	require.NoError(t, err)

	revokedAt := time.Now().UTC().Truncate(time.Second)
	updated, err := repo.Update(ctx,
		domain.Credential{ID: "c1", Name: "renamed", RevokedAt: &revokedAt},
	)
	require.NoError(t, err)
	require.Len(t, updated, 1)
	assert.Equal(t, "renamed", updated[0].Name)
	require.NotNil(t, updated[0].RevokedAt)
	assert.WithinDuration(t, revokedAt, *updated[0].RevokedAt, time.Second)
}

func TestGormCredentialUpdate_IssuedAt(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "iss", Name: "a", FileHash: "0xaa"},
	)
	require.NoError(t, err)

	issuedAt := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	updated, err := repo.Update(ctx, domain.Credential{ID: "c1", IssuedAt: issuedAt})
	require.NoError(t, err)
	require.Len(t, updated, 1)
	assert.WithinDuration(t, issuedAt, updated[0].IssuedAt, time.Second)
}

func TestGormCredentialUpdate_IssuedAtZeroSkipsColumn(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "iss", Name: "a", FileHash: "0xaa"},
	)
	require.NoError(t, err)

	_, err = repo.Update(ctx, domain.Credential{ID: "c1", IssuedAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)

	// A partial row with zero IssuedAt must not clobber the stored value.
	_, err = repo.Update(ctx, domain.Credential{ID: "c1", Name: "b"})
	require.NoError(t, err)

	rows, _, err := repo.Get(ctx, &domainQuery.Query{})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), rows[0].IssuedAt)
}

func TestGormCredentialGet(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		id := "c" + strconv.Itoa(i+1)
		_, err := repo.Store(ctx, domain.Credential{ID: id, HolderUserID: "h1", IssuerUserID: "iss", Name: id, FileHash: "0x" + id})
		require.NoError(t, err)
	}

	creds, total, err := repo.Get(ctx, &domainQuery.Query{})
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, creds, 5)
}

func TestGormCredentialGet_FilterByName(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()
	for _, n := range []string{"Alpha", "Beta"} {
		_, err := repo.Store(ctx, domain.Credential{ID: "c" + n, HolderUserID: "h1", IssuerUserID: "iss", Name: n, FileHash: "0x" + n})
		require.NoError(t, err)
	}
	q := &domainQuery.Query{
		Filters: []domainQuery.Filter{
			{Column: "name", Operator: domainQuery.OperatorLike, Values: []string{"Alpha"}},
		},
	}
	results, total, err := repo.Get(ctx, q)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, results, 1)
	assert.Equal(t, "Alpha", results[0].Name)
}

func TestGormCredentialGet_SortByName(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()
	for _, n := range []string{"Gamma", "Alpha", "Beta"} {
		_, err := repo.Store(ctx, domain.Credential{ID: "c" + n, HolderUserID: "h1", IssuerUserID: "iss", Name: n, FileHash: "0x" + n})
		require.NoError(t, err)
	}
	q := &domainQuery.Query{
		Sorts: []domainQuery.Sort{
			{Column: "name", Order: domainQuery.SortAsc},
		},
	}
	results, _, err := repo.Get(ctx, q)
	require.NoError(t, err)
	assert.Len(t, results, 3)
	assert.Equal(t, "Alpha", results[0].Name)
	assert.Equal(t, "Beta", results[1].Name)
	assert.Equal(t, "Gamma", results[2].Name)
}

func TestGormCredentialGet_Pagination(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		id := "c" + strconv.Itoa(i+1)
		_, err := repo.Store(ctx, domain.Credential{ID: id, HolderUserID: "h1", IssuerUserID: "iss", Name: id, FileHash: "0x" + id})
		require.NoError(t, err)
	}
	q := domainQuery.NewQuery()
	q.Page = 1
	q.Limit = 2
	results, total, err := repo.Get(ctx, &q)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, results, 2)
}

func TestGormCredentialGet_FilterDisallowedColumnIsIgnored(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()
	for _, n := range []string{"Alpha", "Beta"} {
		_, err := repo.Store(ctx, domain.Credential{ID: "c" + n, HolderUserID: "h1", IssuerUserID: "iss", Name: n, FileHash: "0x" + n})
		require.NoError(t, err)
	}
	q := &domainQuery.Query{
		Filters: []domainQuery.Filter{
			{Column: "secret_column", Operator: domainQuery.OperatorEqual, Values: []string{"x"}},
		},
	}
	results, total, err := repo.Get(ctx, q)
	require.NoError(t, err)
	assert.Equal(t, 2, total, "disallowed filter column should be silently dropped")
	assert.Len(t, results, 2)
}

func TestGormCredentialGet_SortDisallowedColumnIsIgnored(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()
	for _, n := range []string{"Alpha", "Beta"} {
		_, err := repo.Store(ctx, domain.Credential{ID: "c" + n, HolderUserID: "h1", IssuerUserID: "iss", Name: n, FileHash: "0x" + n})
		require.NoError(t, err)
	}
	q := &domainQuery.Query{
		Sorts: []domainQuery.Sort{
			{Column: "secret_column", Order: domainQuery.SortDesc},
		},
	}
	results, total, err := repo.Get(ctx, q)
	require.NoError(t, err)
	assert.Equal(t, 2, total, "disallowed sort column should fall back to default sort")
	assert.Len(t, results, 2)
}

func TestGormCredentialGet_PaginationPage2(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		id := "c" + strconv.Itoa(i+1)
		_, err := repo.Store(ctx, domain.Credential{ID: id, HolderUserID: "h1", IssuerUserID: "iss", Name: id, FileHash: "0x" + id})
		require.NoError(t, err)
	}
	q := &domainQuery.Query{Page: 2, Limit: 2}
	results, total, err := repo.Get(ctx, q)
	require.NoError(t, err)
	assert.Equal(t, 5, total, "total must reflect all matching rows, not page slice")
	assert.Len(t, results, 2, "page 2 with limit 2 should return 2 items")
}

func TestGormCredentialGet_FilterCombinedWithSort(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()
	for _, n := range []string{"Gamma", "Alpha", "Beta"} {
		_, err := repo.Store(ctx, domain.Credential{ID: "c" + n, HolderUserID: "h1", IssuerUserID: "iss", Name: n, FileHash: "0x" + n})
		require.NoError(t, err)
	}
	q := &domainQuery.Query{
		Filters: []domainQuery.Filter{
			{Column: "name", Operator: domainQuery.OperatorNotEqual, Values: []string{"Alpha"}},
		},
		Sorts: []domainQuery.Sort{
			{Column: "name", Order: domainQuery.SortAsc},
		},
	}
	results, total, err := repo.Get(ctx, q)
	require.NoError(t, err)
	assert.Equal(t, 2, total, "filter !Alpha should exclude Alpha, leaving 2")
	assert.Len(t, results, 2)
	assert.Equal(t, "Beta", results[0].Name)
	assert.Equal(t, "Gamma", results[1].Name)
}

func TestGormCredentialGet_SearchCredentialColumns(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		domain.Credential{ID: "c001", HolderUserID: "h1", IssuerUserID: "iss1", Name: "Alpha", FileHash: "0xaa", TokenID: strPtr("tok-1"), Meta: map[string]any{"key": "val1"}},
		domain.Credential{ID: "c002", HolderUserID: "h2", IssuerUserID: "iss2", Name: "Beta", FileHash: "0xbb", TokenID: strPtr("tok-2"), Meta: map[string]any{"key": "val2"}},
		domain.Credential{ID: "c003", HolderUserID: "h3", IssuerUserID: "iss3", Name: "Gamma", FileHash: "0xcc", TokenID: strPtr("tok-3"), Meta: nil},
	)
	require.NoError(t, err)

	t.Run("search_by_name", func(t *testing.T) {
		q := &domainQuery.Query{Search: "Alpha"}
		results, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, results, 1)
		assert.Equal(t, "Alpha", results[0].Name)
	})

	t.Run("search_by_id", func(t *testing.T) {
		q := &domainQuery.Query{Search: "c001"}
		results, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, "c001", results[0].ID)
	})

	t.Run("search_by_token_id", func(t *testing.T) {
		q := &domainQuery.Query{Search: "tok-2"}
		results, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, "tok-2", *results[0].TokenID)
	})

	t.Run("search_by_file_hash", func(t *testing.T) {
		q := &domainQuery.Query{Search: "0xcc"}
		results, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, "0xcc", results[0].FileHash)
	})

	t.Run("search_by_meta", func(t *testing.T) {
		q := &domainQuery.Query{Search: "val1"}
		results, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, "Alpha", results[0].Name)
	})

	t.Run("search_partial_case_insensitive", func(t *testing.T) {
		q := &domainQuery.Query{Search: "LPHA"}
		results, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, "Alpha", results[0].Name)
	})

	t.Run("search_no_match", func(t *testing.T) {
		q := &domainQuery.Query{Search: "zzzzz"}
		results, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Len(t, results, 0)
	})
}

func TestGormCredentialGet_SearchByRelatedUsers(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	holder := model.User{Id: "holder1", Email: "holder@test.com", Name: strPtr("Holder Name"), Number: strPtr("H100"), Role: "holder", WalletAddress: "0x1", EncryptedWalletPrivateKey: "sk1"}
	issuer := model.User{Id: "issuer1", Email: "issuer@test.com", Name: strPtr("Issuer Name"), Number: strPtr("I200"), Role: "issuer", WalletAddress: "0x2", EncryptedWalletPrivateKey: "sk2"}
	revoker := model.User{Id: "revoker1", Email: "revoker@test.com", Name: strPtr("Revoker Name"), Number: strPtr("R300"), Role: "admin", WalletAddress: "0x3", EncryptedWalletPrivateKey: "sk3"}
	require.NoError(t, repo.db.Create(&holder).Error)
	require.NoError(t, repo.db.Create(&issuer).Error)
	require.NoError(t, repo.db.Create(&revoker).Error)

	revokerID := "revoker1"
	_, err := repo.Store(ctx, domain.Credential{
		ID: "cred01", HolderUserID: "holder1", IssuerUserID: "issuer1", RevokerUserID: &revokerID,
		Name: "TestCred", FileHash: "0xff", TokenID: strPtr("tok-main"),
	})
	require.NoError(t, err)

	t.Run("search_by_holder_name", func(t *testing.T) {
		q := &domainQuery.Query{Search: "Holder Name"}
		results, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, "TestCred", results[0].Name)
	})

	t.Run("search_by_holder_email", func(t *testing.T) {
		q := &domainQuery.Query{Search: "holder@test.com"}
		_, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
	})

	t.Run("search_by_holder_number", func(t *testing.T) {
		q := &domainQuery.Query{Search: "H100"}
		_, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
	})

	t.Run("search_by_issuer_name", func(t *testing.T) {
		q := &domainQuery.Query{Search: "Issuer Name"}
		results, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, "TestCred", results[0].Name)
	})

	t.Run("search_by_issuer_email", func(t *testing.T) {
		q := &domainQuery.Query{Search: "issuer@test.com"}
		_, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
	})

	t.Run("search_by_issuer_number", func(t *testing.T) {
		q := &domainQuery.Query{Search: "I200"}
		_, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
	})

	t.Run("search_by_revoker_name", func(t *testing.T) {
		q := &domainQuery.Query{Search: "Revoker Name"}
		r, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		if assert.Len(t, r, 1) {
			assert.Equal(t, "TestCred", r[0].Name)
		}
	})

	t.Run("search_by_revoker_email", func(t *testing.T) {
		q := &domainQuery.Query{Search: "revoker@test.com"}
		_, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
	})

	t.Run("search_by_revoker_number", func(t *testing.T) {
		q := &domainQuery.Query{Search: "R300"}
		_, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
	})
}

func TestGormCredentialGet_FilterByExtractStatus(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		domain.Credential{ID: "c100", HolderUserID: "h1", IssuerUserID: "iss1", Name: "Pending1", FileHash: "0xaa", ExtractStatus: domain.ExtractStatusPending},
		domain.Credential{ID: "c101", HolderUserID: "h2", IssuerUserID: "iss2", Name: "Pending2", FileHash: "0xbb", ExtractStatus: domain.ExtractStatusPending},
		domain.Credential{ID: "c102", HolderUserID: "h3", IssuerUserID: "iss3", Name: "Failed1", FileHash: "0xcc", ExtractStatus: domain.ExtractStatusFailed},
	)
	require.NoError(t, err)

	t.Run("filter_by_failed", func(t *testing.T) {
		q := &domainQuery.Query{
			Filters: []domainQuery.Filter{
				{Column: "extract_status", Operator: domainQuery.OperatorEqual, Values: []string{"failed"}},
			},
		}
		results, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, results, 1)
		assert.Equal(t, "Failed1", results[0].Name)
	})

	t.Run("filter_by_pending", func(t *testing.T) {
		q := &domainQuery.Query{
			Filters: []domainQuery.Filter{
				{Column: "extract_status", Operator: domainQuery.OperatorEqual, Values: []string{"pending"}},
			},
		}
		results, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, results, 2)
	})

	t.Run("filter_by_succeeded_no_match", func(t *testing.T) {
		q := &domainQuery.Query{
			Filters: []domainQuery.Filter{
				{Column: "extract_status", Operator: domainQuery.OperatorEqual, Values: []string{"succeeded"}},
			},
		}
		results, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Len(t, results, 0)
	})

	t.Run("filter_extract_status_combined_with_not_equal", func(t *testing.T) {
		q := &domainQuery.Query{
			Filters: []domainQuery.Filter{
				{Column: "extract_status", Operator: domainQuery.OperatorNotEqual, Values: []string{"pending"}},
			},
		}
		results, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, "Failed1", results[0].Name)
	})

	t.Run("filter_extract_status_in", func(t *testing.T) {
		q := &domainQuery.Query{
			Filters: []domainQuery.Filter{
				{Column: "extract_status", Operator: domainQuery.OperatorIn, Values: []string{"failed", "succeeded"}},
			},
		}
		results, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, "Failed1", results[0].Name)
	})

	t.Run("filter_extract_status_not_in", func(t *testing.T) {
		q := &domainQuery.Query{
			Filters: []domainQuery.Filter{
				{Column: "extract_status", Operator: domainQuery.OperatorNotIn, Values: []string{"pending"}},
			},
		}
		results, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, "Failed1", results[0].Name)
	})
}

func TestGormCredentialGet_FilterByNameComparison(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "iss", Name: "Alpha", FileHash: "0xa"},
		domain.Credential{ID: "c2", HolderUserID: "h2", IssuerUserID: "iss", Name: "Beta", FileHash: "0xb"},
		domain.Credential{ID: "c3", HolderUserID: "h3", IssuerUserID: "iss", Name: "Gamma", FileHash: "0xc"},
	)
	require.NoError(t, err)

	t.Run("greater_than", func(t *testing.T) {
		q := &domainQuery.Query{
			Filters: []domainQuery.Filter{
				{Column: "name", Operator: domainQuery.OperatorGreaterThan, Values: []string{"Beta"}},
			},
		}
		results, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, "Gamma", results[0].Name)
	})

	t.Run("less_than", func(t *testing.T) {
		q := &domainQuery.Query{
			Filters: []domainQuery.Filter{
				{Column: "name", Operator: domainQuery.OperatorLessThan, Values: []string{"Beta"}},
			},
		}
		_, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
	})

	t.Run("greater_than_or_equal", func(t *testing.T) {
		q := &domainQuery.Query{
			Filters: []domainQuery.Filter{
				{Column: "name", Operator: domainQuery.OperatorGreaterThanOrEqual, Values: []string{"Beta"}},
			},
		}
		_, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 2, total)
	})

	t.Run("less_than_or_equal", func(t *testing.T) {
		q := &domainQuery.Query{
			Filters: []domainQuery.Filter{
				{Column: "name", Operator: domainQuery.OperatorLessThanOrEqual, Values: []string{"Beta"}},
			},
		}
		_, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 2, total)
	})

	t.Run("between", func(t *testing.T) {
		q := &domainQuery.Query{
			Filters: []domainQuery.Filter{
				{Column: "name", Operator: domainQuery.OperatorBetween, Values: []string{"Alpha", "Beta"}},
			},
		}
		_, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 2, total)
	})

	t.Run("not_between", func(t *testing.T) {
		q := &domainQuery.Query{
			Filters: []domainQuery.Filter{
				{Column: "name", Operator: domainQuery.OperatorNotBetween, Values: []string{"Alpha", "Beta"}},
			},
		}
		_, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
	})
}

func TestGormCredentialGet_FilterByRevokedAt(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	revokedAt := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	_, err := repo.Store(ctx,
		domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "iss", Name: "Active", FileHash: "0xa"},
		domain.Credential{ID: "c2", HolderUserID: "h2", IssuerUserID: "iss", Name: "Revoked", FileHash: "0xb", RevokedAt: &revokedAt},
	)
	require.NoError(t, err)

	t.Run("is_null", func(t *testing.T) {
		q := &domainQuery.Query{
			Filters: []domainQuery.Filter{
				{Column: "revoked_at", Operator: domainQuery.OperatorNull},
			},
		}
		results, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, "Active", results[0].Name)
	})

	t.Run("is_not_null", func(t *testing.T) {
		q := &domainQuery.Query{
			Filters: []domainQuery.Filter{
				{Column: "revoked_at", Operator: domainQuery.OperatorNotNull},
			},
		}
		results, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, "Revoked", results[0].Name)
	})
}

func TestGormCredentialGet_FilterByHolderUserId(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "iss", Name: "A", FileHash: "0xa"},
		domain.Credential{ID: "c2", HolderUserID: "h2", IssuerUserID: "iss", Name: "B", FileHash: "0xb"},
		domain.Credential{ID: "c3", HolderUserID: "h3", IssuerUserID: "iss", Name: "C", FileHash: "0xc"},
	)
	require.NoError(t, err)

	t.Run("equal", func(t *testing.T) {
		q := &domainQuery.Query{
			Filters: []domainQuery.Filter{
				{Column: "holder_user_id", Operator: domainQuery.OperatorEqual, Values: []string{"h2"}},
			},
		}
		results, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, "B", results[0].Name)
	})

	t.Run("in", func(t *testing.T) {
		q := &domainQuery.Query{
			Filters: []domainQuery.Filter{
				{Column: "holder_user_id", Operator: domainQuery.OperatorIn, Values: []string{"h1", "h3"}},
			},
		}
		_, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 2, total)
	})

	t.Run("not_in", func(t *testing.T) {
		q := &domainQuery.Query{
			Filters: []domainQuery.Filter{
				{Column: "holder_user_id", Operator: domainQuery.OperatorNotIn, Values: []string{"h1", "h3"}},
			},
		}
		results, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, "B", results[0].Name)
	})
}

func TestGormCredentialGet_FilterByNameNotLike(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "iss", Name: "Alpha", FileHash: "0xa"},
		domain.Credential{ID: "c2", HolderUserID: "h2", IssuerUserID: "iss", Name: "Beta", FileHash: "0xb"},
		domain.Credential{ID: "c3", HolderUserID: "h3", IssuerUserID: "iss", Name: "Alpine", FileHash: "0xc"},
	)
	require.NoError(t, err)

	t.Run("not_like", func(t *testing.T) {
		q := &domainQuery.Query{
			Filters: []domainQuery.Filter{
				{Column: "name", Operator: domainQuery.OperatorNotLike, Values: []string{"Alpha"}},
			},
		}
		_, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 2, total)
	})

	t.Run("not_ilike", func(t *testing.T) {
		q := &domainQuery.Query{
			Filters: []domainQuery.Filter{
				{Column: "name", Operator: domainQuery.OperatorNotILike, Values: []string{"alpha"}},
			},
		}
		_, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 2, total)
	})
}

func TestGormCredentialGet_SortByIssuedAt(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)

	_, err := repo.Store(ctx,
		domain.Credential{ID: "c2", HolderUserID: "h1", IssuerUserID: "iss", Name: "Mid", FileHash: "0xb", IssuedAt: t2},
		domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "iss", Name: "Early", FileHash: "0xa", IssuedAt: t1},
		domain.Credential{ID: "c3", HolderUserID: "h1", IssuerUserID: "iss", Name: "Late", FileHash: "0xc", IssuedAt: t3},
	)
	require.NoError(t, err)

	t.Run("asc", func(t *testing.T) {
		q := &domainQuery.Query{
			Sorts: []domainQuery.Sort{
				{Column: "issued_at", Order: domainQuery.SortAsc},
			},
		}
		results, _, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Len(t, results, 3)
		assert.Equal(t, "Early", results[0].Name)
		assert.Equal(t, "Mid", results[1].Name)
		assert.Equal(t, "Late", results[2].Name)
	})

	t.Run("desc", func(t *testing.T) {
		q := &domainQuery.Query{
			Sorts: []domainQuery.Sort{
				{Column: "issued_at", Order: domainQuery.SortDesc},
			},
		}
		results, _, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Len(t, results, 3)
		assert.Equal(t, "Late", results[0].Name)
		assert.Equal(t, "Mid", results[1].Name)
		assert.Equal(t, "Early", results[2].Name)
	})
}

func TestGormCredentialGet_SortByRevokedAt(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	_, err := repo.Store(ctx,
		domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "iss", Name: "Active", FileHash: "0xa"},
		domain.Credential{ID: "c2", HolderUserID: "h2", IssuerUserID: "iss", Name: "OldRevoked", FileHash: "0xb", RevokedAt: &t1},
		domain.Credential{ID: "c3", HolderUserID: "h3", IssuerUserID: "iss", Name: "RecentRevoked", FileHash: "0xc", RevokedAt: &t2},
	)
	require.NoError(t, err)

	t.Run("asc", func(t *testing.T) {
		q := &domainQuery.Query{
			Sorts: []domainQuery.Sort{
				{Column: "revoked_at", Order: domainQuery.SortAsc},
			},
		}
		results, _, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Len(t, results, 3)
		assert.Equal(t, "Active", results[0].Name, "ASC: SQLite puts NULLs first")
		assert.Equal(t, "OldRevoked", results[1].Name)
		assert.Equal(t, "RecentRevoked", results[2].Name)
	})

	t.Run("desc", func(t *testing.T) {
		q := &domainQuery.Query{
			Sorts: []domainQuery.Sort{
				{Column: "revoked_at", Order: domainQuery.SortDesc},
			},
		}
		results, _, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Len(t, results, 3)
		assert.Equal(t, "RecentRevoked", results[0].Name)
		assert.Equal(t, "OldRevoked", results[1].Name)
		assert.Equal(t, "Active", results[2].Name, "DESC: SQLite puts NULLs last")
	})
}

func TestGormCredentialGet_SortByHolderFields(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	for _, u := range []model.User{
		{Id: "h-alpha", Email: "alpha@test.com", Name: strPtr("Alpha Holder"), Number: strPtr("N001"), Role: "holder", WalletAddress: "0xa", EncryptedWalletPrivateKey: "ska"},
		{Id: "h-beta", Email: "beta@test.com", Name: strPtr("Beta Holder"), Number: strPtr("N002"), Role: "holder", WalletAddress: "0xb", EncryptedWalletPrivateKey: "skb"},
		{Id: "h-gamma", Email: "gamma@test.com", Name: strPtr("Gamma Holder"), Number: strPtr("N003"), Role: "holder", WalletAddress: "0xc", EncryptedWalletPrivateKey: "skc"},
	} {
		require.NoError(t, repo.db.Create(&u).Error)
	}

	_, err := repo.Store(ctx,
		domain.Credential{ID: "c1", HolderUserID: "h-beta", IssuerUserID: "iss", Name: "Beta Cred", FileHash: "0x1"},
		domain.Credential{ID: "c2", HolderUserID: "h-alpha", IssuerUserID: "iss", Name: "Alpha Cred", FileHash: "0x2"},
		domain.Credential{ID: "c3", HolderUserID: "h-gamma", IssuerUserID: "iss", Name: "Gamma Cred", FileHash: "0x3"},
	)
	require.NoError(t, err)

	t.Run("holder_name_asc", func(t *testing.T) {
		q := &domainQuery.Query{
			Sorts: []domainQuery.Sort{
				{Column: "holder_name", Order: domainQuery.SortAsc},
			},
		}
		results, _, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Len(t, results, 3)
		assert.Equal(t, "Alpha Cred", results[0].Name)
		assert.Equal(t, "Beta Cred", results[1].Name)
		assert.Equal(t, "Gamma Cred", results[2].Name)
	})

	t.Run("holder_name_desc", func(t *testing.T) {
		q := &domainQuery.Query{
			Sorts: []domainQuery.Sort{
				{Column: "holder_name", Order: domainQuery.SortDesc},
			},
		}
		results, _, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Len(t, results, 3)
		assert.Equal(t, "Gamma Cred", results[0].Name)
		assert.Equal(t, "Beta Cred", results[1].Name)
		assert.Equal(t, "Alpha Cred", results[2].Name)
	})

	t.Run("holder_email_asc", func(t *testing.T) {
		q := &domainQuery.Query{
			Sorts: []domainQuery.Sort{
				{Column: "holder_email", Order: domainQuery.SortAsc},
			},
		}
		results, _, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Len(t, results, 3)
		assert.Equal(t, "Alpha Cred", results[0].Name)
		assert.Equal(t, "Beta Cred", results[1].Name)
	})

	t.Run("holder_number_desc", func(t *testing.T) {
		q := &domainQuery.Query{
			Sorts: []domainQuery.Sort{
				{Column: "holder_number", Order: domainQuery.SortDesc},
			},
		}
		results, _, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Len(t, results, 3)
		assert.Equal(t, "Gamma Cred", results[0].Name)
	})
}

func TestGormCredentialGet_SortMultiple(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "iss", Name: "Z", FileHash: "0xa", IssuedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		domain.Credential{ID: "c2", HolderUserID: "h1", IssuerUserID: "iss", Name: "Y", FileHash: "0xb", IssuedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		domain.Credential{ID: "c3", HolderUserID: "h1", IssuerUserID: "iss", Name: "Z", FileHash: "0xc", IssuedAt: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)},
	)
	require.NoError(t, err)

	q := &domainQuery.Query{
		Sorts: []domainQuery.Sort{
			{Column: "name", Order: domainQuery.SortAsc},
			{Column: "issued_at", Order: domainQuery.SortAsc},
		},
	}
	results, _, err := repo.Get(ctx, q)
	require.NoError(t, err)
	assert.Len(t, results, 3)
	assert.Equal(t, "Y", results[0].Name, "name=Y issued_at=Jan")
	assert.Equal(t, "Z", results[1].Name, "name=Z issued_at=Jan (first Z)")
	assert.Equal(t, "Z", results[2].Name, "name=Z issued_at=Jun (second Z, later date)")
}

func TestGormCredentialGet_SearchFilterSortCombined(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "iss", Name: "Alpha Corp", FileHash: "0xa", ExtractStatus: domain.ExtractStatusPending, IssuedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		domain.Credential{ID: "c2", HolderUserID: "h1", IssuerUserID: "iss", Name: "Alpha Inc", FileHash: "0xb", ExtractStatus: domain.ExtractStatusFailed, IssuedAt: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)},
		domain.Credential{ID: "c3", HolderUserID: "h1", IssuerUserID: "iss", Name: "Beta Corp", FileHash: "0xc", ExtractStatus: domain.ExtractStatusPending, IssuedAt: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)},
	)
	require.NoError(t, err)

	q := &domainQuery.Query{
		Search: "Alpha",
		Filters: []domainQuery.Filter{
			{Column: "extract_status", Operator: domainQuery.OperatorEqual, Values: []string{"pending"}},
		},
		Sorts: []domainQuery.Sort{
			{Column: "issued_at", Order: domainQuery.SortAsc},
		},
	}
	results, total, err := repo.Get(ctx, q)
	require.NoError(t, err)
	assert.Equal(t, 1, total, "search 'Alpha' + filter pending = only Alpha Corp")
	assert.Len(t, results, 1)
	assert.Equal(t, "Alpha Corp", results[0].Name)
}

func TestGormCredentialGet_SearchEmptyString(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "iss", Name: "Alpha", FileHash: "0xa"},
		domain.Credential{ID: "c2", HolderUserID: "h2", IssuerUserID: "iss", Name: "Beta", FileHash: "0xb"},
	)
	require.NoError(t, err)

	q := &domainQuery.Query{Search: ""}
	results, total, err := repo.Get(ctx, q)
	require.NoError(t, err)
	assert.Equal(t, 2, total, "empty search should return all rows")
	assert.Len(t, results, 2)
}

func TestGormCredentialGet_FilterByIssuerUserId(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "i1", Name: "A", FileHash: "0xa"},
		domain.Credential{ID: "c2", HolderUserID: "h2", IssuerUserID: "i2", Name: "B", FileHash: "0xb"},
		domain.Credential{ID: "c3", HolderUserID: "h3", IssuerUserID: "i3", Name: "C", FileHash: "0xc"},
	)
	require.NoError(t, err)

	t.Run("equal", func(t *testing.T) {
		q := &domainQuery.Query{
			Filters: []domainQuery.Filter{
				{Column: "issuer_user_id", Operator: domainQuery.OperatorEqual, Values: []string{"i2"}},
			},
		}
		results, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, "B", results[0].Name)
	})

	t.Run("in", func(t *testing.T) {
		q := &domainQuery.Query{
			Filters: []domainQuery.Filter{
				{Column: "issuer_user_id", Operator: domainQuery.OperatorIn, Values: []string{"i1", "i3"}},
			},
		}
		_, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 2, total)
	})

	t.Run("not_in", func(t *testing.T) {
		q := &domainQuery.Query{
			Filters: []domainQuery.Filter{
				{Column: "issuer_user_id", Operator: domainQuery.OperatorNotIn, Values: []string{"i1", "i3"}},
			},
		}
		results, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, "B", results[0].Name)
	})
}

func TestGormCredentialRepository_Get_VirtualFilters(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	// reference data
	require.NoError(t, repo.db.Create(&model.UserUnit{Id: "unit-fac", Name: "Faculty"}).Error)
	require.NoError(t, repo.db.Create(&model.Competency{Id: "comp-ai", Name: "Artificial Intelligence"}).Error)
	require.NoError(t, repo.db.Create(&model.Competency{Id: "comp-db", Name: "Databases"}).Error)
	holder := model.User{Id: "h1", Email: "h1@x.com", Name: lo.ToPtr("H"), Role: "holder", WalletAddress: "0xa", EncryptedWalletPrivateKey: "k", UnitID: lo.ToPtr("unit-fac")}
	require.NoError(t, repo.db.Create(&holder).Error)

	_, err := repo.Store(ctx,
		domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "i1", IssuerOrganizationID: strPtr("o1"), TypeID: strPtr("t1"), Name: "A", FileHash: "0x1"},
		domain.Credential{ID: "c2", HolderUserID: "h1", IssuerUserID: "i1", IssuerOrganizationID: strPtr("o1"), TypeID: strPtr("t1"), Name: "B", FileHash: "0x2"},
	)
	require.NoError(t, err)
	require.NoError(t, repo.db.Create(&[]model.CompetencyCredential{
		{CompetencyId: "comp-ai", CredentialId: "c1"},
		{CompetencyId: "comp-db", CredentialId: "c1"},
		{CompetencyId: "comp-db", CredentialId: "c2"},
	}).Error)

	t.Run("competency_name LIKE", func(t *testing.T) {
		q := &domainQuery.Query{Filters: []domainQuery.Filter{
			domainQuery.NewFilter("competency_name", domainQuery.OperatorLike, "data"),
		}}
		got, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		require.Len(t, got, 2, "c1 (databases + ai) and c2 (databases); no duplicate rows from the join")
		assert.Equal(t, 2, total)
	})

	t.Run("competency_id equal", func(t *testing.T) {
		q := &domainQuery.Query{Filters: []domainQuery.Filter{
			domainQuery.NewFilter("competency_id", domainQuery.OperatorEqual, "comp-ai"),
		}}
		got, _, err := repo.Get(ctx, q)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "c1", got[0].ID)
	})

	t.Run("holder_unit_name LIKE", func(t *testing.T) {
		q := &domainQuery.Query{Filters: []domainQuery.Filter{
			domainQuery.NewFilter("holder_unit_name", domainQuery.OperatorLike, "fac"),
		}}
		got, _, err := repo.Get(ctx, q)
		require.NoError(t, err)
		require.Len(t, got, 2)
	})

	t.Run("holder_unit_id equal", func(t *testing.T) {
		q := &domainQuery.Query{Filters: []domainQuery.Filter{
			domainQuery.NewFilter("holder_unit_id", domainQuery.OperatorEqual, "unit-fac"),
		}}
		got, _, err := repo.Get(ctx, q)
		require.NoError(t, err)
		require.Len(t, got, 2)
	})
}

func TestGormCredentialRepositoryFindWithCompetencies(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	require.NoError(t, repo.db.Create(&model.User{
		Id: "h1", Email: "h1@x.com", Name: lo.ToPtr("Holder"), Role: "holder",
		WalletAddress: "0xa", EncryptedWalletPrivateKey: "k",
	}).Error)
	require.NoError(t, repo.db.Create(&model.Competency{Id: "comp-ai", Name: "Artificial Intelligence", Active: true}).Error)
	require.NoError(t, repo.db.Create(&model.Competency{Id: "comp-db", Name: "Databases", Active: true}).Error)

	_, err := repo.Store(ctx, domain.Credential{
		ID: "c1", HolderUserID: "h1", IssuerUserID: "i1",
		IssuerOrganizationID: strPtr("o1"), TypeID: strPtr("t1"),
		Name: "A", FileHash: "0x1",
	})
	require.NoError(t, err)

	require.NoError(t, repo.db.Create(&[]model.CompetencyCredential{
		{CompetencyId: "comp-ai", CredentialId: "c1"},
		{CompetencyId: "comp-db", CredentialId: "c1"},
	}).Error)

	c, err := repo.Find(ctx, "c1", &domainQuery.Query{Includes: []string{"competencies"}})
	require.NoError(t, err)
	require.NotNil(t, c)
	require.Len(t, c.Competencies, 2)

	names := lo.Map(c.Competencies, func(x domain.Competency, _ int) string { return x.Name })
	assert.ElementsMatch(t, []string{"Artificial Intelligence", "Databases"}, names)
}
