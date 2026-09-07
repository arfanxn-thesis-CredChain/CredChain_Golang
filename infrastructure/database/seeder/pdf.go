package seeder

import (
	"bytes"
	"fmt"
	"strings"
)

// escapePDFText escapes parentheses and backslashes for PDF string literals.
func escapePDFText(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '(', ')', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			if r >= 32 && r <= 126 {
				b.WriteRune(r)
			} else {
				b.WriteByte('?')
			}
		}
	}
	return b.String()
}

// seedPDFBytes generates a minimal valid uncompressed PDF 1.4 byte slice.
// Contains 5 objects: Catalog, Pages, Page, Content Stream, Font (Helvetica),
// plus cross-reference table and trailer.
func seedPDFBytes(title string, lines []string) []byte {
	var stream bytes.Buffer
	stream.WriteString("BT\n")
	stream.WriteString("/F1 16 Tf\n")
	stream.WriteString("50 740 Td\n")
	stream.WriteString("22 TL\n")
	fmt.Fprintf(&stream, "(%s) Tj\n", escapePDFText(title))

	stream.WriteString("/F1 11 Tf\n")
	stream.WriteString("16 TL\n")
	stream.WriteString("T*\n")
	for _, line := range lines {
		fmt.Fprintf(&stream, "T*\n(%s) Tj\n", escapePDFText(line))
	}
	stream.WriteString("ET\n")
	streamBytes := stream.Bytes()

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")

	offsets := make([]int, 6) // objects 1..5, index 0 unused

	// 1: Catalog
	offsets[1] = buf.Len()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	// 2: Pages
	offsets[2] = buf.Len()
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	// 3: Page
	offsets[3] = buf.Len()
	buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj\n")

	// 4: Content Stream
	offsets[4] = buf.Len()
	fmt.Fprintf(&buf, "4 0 obj\n<< /Length %d >>\nstream\n", len(streamBytes))
	buf.Write(streamBytes)
	buf.WriteString("\nendstream\nendobj\n")

	// 5: Font
	offsets[5] = buf.Len()
	buf.WriteString("5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n")

	// xref
	xrefOffset := buf.Len()
	buf.WriteString("xref\n0 6\n")
	buf.WriteString("0000000000 65535 f \n")
	for i := 1; i <= 5; i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[i])
	}

	// trailer
	buf.WriteString("trailer\n<< /Size 6 /Root 1 0 R >>\n")
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOffset)

	return buf.Bytes()
}
