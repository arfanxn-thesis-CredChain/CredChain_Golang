package seeder

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSeedPDFBytes(t *testing.T) {
	pdf1 := seedPDFBytes("Test Credential 1", []string{"Holder: Alice", "Issued: 2026-01-01"})
	pdf2 := seedPDFBytes("Test Credential 2", []string{"Holder: Bob", "Issued: 2026-02-01"})

	assert.True(t, bytes.HasPrefix(pdf1, []byte("%PDF-1.4\n")))
	assert.True(t, bytes.HasSuffix(pdf1, []byte("%%EOF\n")))

	assert.True(t, bytes.HasPrefix(pdf2, []byte("%PDF-1.4\n")))
	assert.True(t, bytes.HasSuffix(pdf2, []byte("%%EOF\n")))

	assert.NotEqual(t, pdf1, pdf2)
	assert.True(t, bytes.Contains(pdf1, []byte("Test Credential 1")))
	assert.True(t, bytes.Contains(pdf1, []byte("Holder: Alice")))
}
