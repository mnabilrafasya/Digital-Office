package pdf

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestStampDesc memastikan deskripsi posisi QR diterima pdfcpu (versi library bisa mengubah
// aturan nama parameter, seperti singkatan "sc" yang ambigu di v0.16).
func TestStampDesc(t *testing.T) {
	var img bytes.Buffer
	if err := png.Encode(&img, image.NewGray(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatal(err)
	}
	if _, err := api.ImageWatermarkForReader(context.Background(), &img, stampDesc, true, false, types.POINTS); err != nil {
		t.Fatalf("stampDesc ditolak pdfcpu: %v", err)
	}
}
