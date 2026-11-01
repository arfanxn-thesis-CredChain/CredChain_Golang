package model

import (
	"testing"
	"time"

	"CredChain_Golang/domain"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
)

func TestCredentialNotificationMapping(t *testing.T) {
	read := time.Now().UTC()
	for _, n := range []domain.CredentialNotification{
		{
			ID:             ulid.Make().String(),
			NotifiedUserID: ulid.Make().String(),
			CredentialID:   ulid.Make().String(),
			Type:           domain.CredentialNotificationTypeRegistered,
			CreatedAt:      read.Add(-time.Minute),
			ReadAt:         &read,
		},
		{
			ID:             ulid.Make().String(),
			NotifiedUserID: ulid.Make().String(),
			CredentialID:   ulid.Make().String(),
			Type:           domain.CredentialNotificationTypeApproved,
		},
		{
			ID:             ulid.Make().String(),
			NotifiedUserID: ulid.Make().String(),
			CredentialID:   ulid.Make().String(),
			Type:           domain.CredentialNotificationTypeRejected,
		},
		{
			ID:             ulid.Make().String(),
			NotifiedUserID: ulid.Make().String(),
			CredentialID:   ulid.Make().String(),
			Type:           domain.CredentialNotificationTypeRevoked,
		},
	} {
		m := FromDomainCredentialNotification(n)
		assert.Equal(t, "credential_notifications", m.TableName())
		assert.Equal(t, n.ID, m.ID)
		assert.Equal(t, n.NotifiedUserID, m.NotifiedUserID)
		assert.Equal(t, n.CredentialID, m.CredentialID)
		assert.Equal(t, string(n.Type), m.Type)
		assert.Equal(t, n.CreatedAt, m.CreatedAt)
		assert.Equal(t, n.ReadAt, m.ReadAt)
		assert.Equal(t, n, m.ToDomain())
	}
}
