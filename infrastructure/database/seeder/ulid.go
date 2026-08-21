package seeder

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/oklog/ulid/v2"
)

// seedULIDBase is a fixed timestamp so ULIDs are stable across reseeds.
var seedULIDBase = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

// deterministicULID returns a stable 26-char ULID derived from a fixed
// timestamp and the given index, so re-seeding yields identical primary keys.
func deterministicULID(index uint32) string {
	var entropy [16]byte
	binary.BigEndian.PutUint32(entropy[6:], index)
	return ulid.MustNew(ulid.Timestamp(seedULIDBase), bytes.NewReader(entropy[:])).String()
}
