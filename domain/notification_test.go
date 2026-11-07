package domain

import (
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestNotificationValidateSubject(t *testing.T) {
	id := ulid.Make().String()
	credential := NotificationSubjectCredential
	competency := NotificationSubjectCompetency
	for _, tc := range []struct {
		name  string
		n     Notification
		valid bool
	}{
		{"subjectless", Notification{Type: NotificationTypeCredentialRegistered}, true},
		{"credential", Notification{Type: NotificationTypeCredentialRegistered, SubjectType: &credential, SubjectID: &id}, true},
		{"missing ID", Notification{Type: NotificationTypeCredentialRegistered, SubjectType: &credential}, false},
		{"missing type", Notification{Type: NotificationTypeCredentialRegistered, SubjectID: &id}, false},
		{"wrong subject", Notification{Type: NotificationTypeCredentialRegistered, SubjectType: &competency, SubjectID: &id}, false},
		{"unknown type", Notification{Type: "unknown"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.valid {
				require.NoError(t, tc.n.Validate())
			} else {
				require.Error(t, tc.n.Validate())
			}
		})
	}
}
