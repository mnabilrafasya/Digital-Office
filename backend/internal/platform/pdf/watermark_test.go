package pdf

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"office.backend/internal/platform/watermark"
)

// twoPagePDF membuat PDF dua halaman A4 yang strukturnya valid.
func twoPagePDF() []byte {
	content := "0 0 m 100 100 l S"
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R 4 0 R] /Count 2 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 5 0 R >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 5 0 R >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
	}
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objs))
	for i, o := range objs {
		offsets[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, off := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, xref)
	return b.Bytes()
}

func TestAddWatermark(t *testing.T) {
	ctx := context.Background()
	img, err := watermark.PNG("PUPUK SRIWIDJAJA")
	if err != nil {
		t.Fatal(err)
	}
	src := twoPagePDF()
	out, err := AddWatermark(ctx, src, img)
	if err != nil {
		t.Fatal(err)
	}
	if n, err := PageCount(ctx, out); err != nil || n != 2 {
		t.Fatalf("halaman = %d (%v), want 2", n, err)
	}
	// Gambar watermark dipasang sebagai XObject di setiap halaman (satu gambar, dipakai ulang)
	if !bytes.Contains(out, []byte("/Subtype/Image")) && !bytes.Contains(out, []byte("/Subtype /Image")) {
		t.Error("PDF tidak memuat gambar watermark")
	}
	if _, err := AddWatermark(ctx, []byte("%PDF-1.4 rusak"), img); err == nil {
		t.Error("PDF rusak diterima")
	}
}
