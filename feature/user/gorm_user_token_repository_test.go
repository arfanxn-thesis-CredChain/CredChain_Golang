package user

import (
	"context"
	"testing"
	"time"

	"CredChain_Golang/domain"
	"CredChain_Golang/tests/db"
	"CredChain_Golang/tests/fixtures"

	"github.com/stretchr/testify/assert"
)

func newTokenRepo(t *testing.T) domain.UserTokenRepository {
	t.Helper()
	return NewGormUserTokenRepository(db.OpenInMemorySQLite(t))
}

func TestGormUserTokenRepository_Store_Single(t *testing.T) {
	repo := newTokenRepo(t)
	tok := fixtures.NewDomainUserToken(fixtures.TokenWithUserID("u1"))
	stored, err := repo.Store(context.Background(), tok)
	assert.NoError(t, err)
	assert.Len(t, stored, 1)
	assert.Equal(t, tok.Id, stored[0].Id)
}

func TestGormUserTokenRepository_Store_Batch(t *testing.T) {
	repo := newTokenRepo(t)
	t1 := fixtures.NewDomainUserToken(fixtures.TokenWithUserID("u1"))
	t2 := fixtures.NewDomainUserToken(fixtures.TokenWithUserID("u1"))
	stored, err := repo.Store(context.Background(), t1, t2)
	assert.NoError(t, err)
	assert.Len(t, stored, 2)
}

func TestGormUserTokenRepository_Find(t *testing.T) {
	repo := newTokenRepo(t)
	tok := fixtures.NewDomainUserToken()
	_, _ = repo.Store(context.Background(), tok)

	got, err := repo.Find(context.Background(), tok.Id)
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, tok.Token, got.Token)
}

func TestGormUserTokenRepository_FindByToken(t *testing.T) {
	repo := newTokenRepo(t)
	tok := fixtures.NewDomainUserToken(fixtures.TokenWithToken("specific-token-string"))
	_, _ = repo.Store(context.Background(), tok)

	got, err := repo.FindByToken(context.Background(), "specific-token-string")
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, tok.Id, got.Id)
}

func TestGormUserTokenRepository_FindByToken_NotFound(t *testing.T) {
	repo := newTokenRepo(t)
	_, err := repo.FindByToken(context.Background(), "nope")
	assert.Error(t, err)
}

func TestGormUserTokenRepository_FindByUserId_Empty(t *testing.T) {
	repo := newTokenRepo(t)
	got, err := repo.FindByUserId(context.Background(), "u-nobody")
	assert.NoError(t, err)
	assert.Empty(t, got)
}

func TestGormUserTokenRepository_FindByUserId_Multiple(t *testing.T) {
	repo := newTokenRepo(t)
	_, _ = repo.Store(context.Background(),
		fixtures.NewDomainUserToken(fixtures.TokenWithUserID("u1")),
		fixtures.NewDomainUserToken(fixtures.TokenWithUserID("u1")),
		fixtures.NewDomainUserToken(fixtures.TokenWithUserID("u2")),
	)
	got, err := repo.FindByUserId(context.Background(), "u1")
	assert.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestGormUserTokenRepository_Revoke_SetsRevokedAt(t *testing.T) {
	repo := newTokenRepo(t)
	tok := fixtures.NewDomainUserToken()
	_, _ = repo.Store(context.Background(), tok)

	n, err := repo.Revoke(context.Background(), tok.Id)
	assert.NoError(t, err)
	assert.EqualValues(t, 1, n)

	got, _ := repo.Find(context.Background(), tok.Id)
	assert.NotNil(t, got.RevokedAt)
}

func TestGormUserTokenRepository_Revoke_Multiple(t *testing.T) {
	repo := newTokenRepo(t)
	t1 := fixtures.NewDomainUserToken()
	t2 := fixtures.NewDomainUserToken()
	_, _ = repo.Store(context.Background(), t1, t2)

	n, err := repo.Revoke(context.Background(), t1.Id, t2.Id)
	assert.NoError(t, err)
	assert.EqualValues(t, 2, n)
}

func TestGormUserTokenRepository_RevokeByUserIdAndType_OnlyMatchingType(t *testing.T) {
	repo := newTokenRepo(t)
	refresh := fixtures.NewDomainUserToken(
		fixtures.TokenWithUserID("u1"),
		fixtures.TokenWithType(domain.UserTokenTypeRefresh))
	other := fixtures.NewDomainUserToken(
		fixtures.TokenWithUserID("u1"),
		fixtures.TokenWithType(domain.UserTokenType("other")))
	_, _ = repo.Store(context.Background(), refresh, other)

	n, err := repo.RevokeByUserIdAndType(context.Background(), "u1", domain.UserTokenTypeRefresh)
	assert.NoError(t, err)
	assert.EqualValues(t, 1, n)

	gotRefresh, _ := repo.Find(context.Background(), refresh.Id)
	gotOther, _ := repo.Find(context.Background(), other.Id)
	assert.NotNil(t, gotRefresh.RevokedAt)
	assert.Nil(t, gotOther.RevokedAt)
}

func TestGormUserTokenRepository_Update_RejectsRepeatedConsumption(t *testing.T) {
	repo := newTokenRepo(t)
	tok := fixtures.NewDomainUserToken()
	_, err := repo.Store(context.Background(), tok)
	assert.NoError(t, err)
	now := time.Now()
	tok.LastUsedAt, tok.RevokedAt = &now, &now
	_, err = repo.Update(context.Background(), tok)
	assert.NoError(t, err)
	_, err = repo.Update(context.Background(), tok)
	var de *domain.Error
	if assert.ErrorAs(t, err, &de) {
		assert.Equal(t, domain.CodeAuthRefreshTokenRevoked, de.Code)
	}
}

func TestGormUserTokenRepository_Update_RejectsExpiredToken(t *testing.T) {
	repo := newTokenRepo(t)
	tok := fixtures.NewDomainUserToken()
	expired := time.Now().Add(-time.Minute)
	tok.ExpiresAt = &expired
	_, err := repo.Store(context.Background(), tok)
	assert.NoError(t, err)
	now := time.Now()
	tok.LastUsedAt, tok.RevokedAt = &now, &now
	_, err = repo.Update(context.Background(), tok)
	assert.Error(t, err)
}

func TestGormUserTokenRepository_Update_ConcurrentConsumption(t *testing.T) {
	repo := newTokenRepo(t)
	tok := fixtures.NewDomainUserToken()
	_, err := repo.Store(context.Background(), tok)
	if !assert.NoError(t, err) {
		return
	}
	now := time.Now()
	tok.LastUsedAt, tok.RevokedAt = &now, &now
	start := make(chan struct{})
	results := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			_, err := repo.Update(context.Background(), tok)
			results <- err
		}()
	}
	close(start)
	successes := 0
	for range 2 {
		if err := <-results; err == nil {
			successes++
		} else {
			var de *domain.Error
			if assert.ErrorAs(t, err, &de) {
				assert.Equal(t, domain.CodeAuthRefreshTokenRevoked, de.Code)
			}
		}
	}
	assert.Equal(t, 1, successes)
}

func TestGormUserTokenRepository_RevokeByUserIdsAndType_ScopedBatch(t *testing.T) {
	repo := newTokenRepo(t)
	tokens := []domain.UserToken{
		fixtures.NewDomainUserToken(fixtures.TokenWithUserID("u1")),
		fixtures.NewDomainUserToken(fixtures.TokenWithUserID("u2")),
		fixtures.NewDomainUserToken(fixtures.TokenWithUserID("u3")),
		fixtures.NewDomainUserToken(fixtures.TokenWithUserID("u1"), fixtures.TokenWithType("other")),
	}
	_, err := repo.Store(context.Background(), tokens...)
	if !assert.NoError(t, err) {
		return
	}
	n, err := repo.RevokeByUserIdsAndType(context.Background(), nil, domain.UserTokenTypeRefresh)
	assert.NoError(t, err)
	assert.Zero(t, n)
	n, err = repo.RevokeByUserIdsAndType(context.Background(), []string{"u1", "u2"}, domain.UserTokenTypeRefresh)
	assert.NoError(t, err)
	assert.EqualValues(t, 2, n)
	for i, token := range tokens {
		got, err := repo.Find(context.Background(), token.Id)
		if assert.NoError(t, err) {
			assert.Equal(t, i < 2, got.RevokedAt != nil)
		}
	}
}

func TestGormUserTokenRepository_Update_SetsLastUsedAndRevoked(t *testing.T) {
	repo := newTokenRepo(t)
	tok := fixtures.NewDomainUserToken()
	_, _ = repo.Store(context.Background(), tok)

	now := time.Now()
	tok.LastUsedAt = &now
	tok.RevokedAt = &now
	updated, err := repo.Update(context.Background(), tok)
	assert.NoError(t, err)
	assert.NotNil(t, updated.LastUsedAt)
	assert.NotNil(t, updated.RevokedAt)
}
