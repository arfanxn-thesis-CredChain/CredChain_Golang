package seeder

import (
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
)

func TestDeterministicULID(t *testing.T) {
	a := deterministicULID(1)
	b := deterministicULID(1)
	assert.Equal(t, a, b, "same index must yield identical ULID")
	assert.Len(t, a, 26)

	c := deterministicULID(2)
	assert.NotEqual(t, a, c, "different index must yield different ULID")

	_, err := ulid.Parse(a)
	assert.NoError(t, err, "must be a valid ULID")
}
