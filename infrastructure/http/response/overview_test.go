package response

import (
	"testing"

	"CredChain_Golang/domain"

	"github.com/stretchr/testify/assert"
)

func TestFromDomainOverviewCredentialCounts(t *testing.T) {
	d := domain.OverviewCredentialCounts{Total: 10, Active: 8, PendingReview: 2, Rejected: 3, Expired: 4, Revoked: 1, Pending: 1, Failed: 1}
	r := FromDomainOverviewCredentialCounts(d)

	assert.Equal(t, 10, r.Total)
	assert.Equal(t, 8, r.Active)
	assert.Equal(t, 2, r.PendingReview)
	assert.Equal(t, 3, r.Rejected)
	assert.Equal(t, 4, r.Expired)
	assert.Equal(t, 1, r.Revoked)
	assert.Equal(t, 1, r.Pending)
	assert.Equal(t, 1, r.Failed)
}

func TestFromDomainOverviewUserCounts(t *testing.T) {
	d := domain.OverviewUserCounts{Total: 6, Holder: 3, Issuer: 1, Admin: 1, SuperAdmin: 1, Active: 5, Trashed: 1}
	r := FromDomainOverviewUserCounts(d)

	assert.Equal(t, 6, r.Total)
	assert.Equal(t, 3, r.Holder)
	assert.Equal(t, 1, r.Issuer)
	assert.Equal(t, 1, r.Admin)
	assert.Equal(t, 1, r.SuperAdmin)
	assert.Equal(t, 5, r.Active)
	assert.Equal(t, 1, r.Trashed)
}
