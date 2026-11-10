package watermark

import (
	"bytes"
	"image/png"
	"strings"
	"testing"
)

func TestPNG(t *testing.T) {
	data, err := PNG("PUPUK SRIWIDJAJA")
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("bukan PNG: %v", err)
	}
	b := img.Bounds()
	if b.Dx() != b.Dy() || b.Dx() < 800 {
		t.Fatalf("ukuran gambar = %v, want persegi yang cukup tajam", b)
	}
	alphaAt := func(fx, fy float64) uint32 {
		_, _, _, a := img.At(b.Min.X+int(fx*float64(b.Dx())), b.Min.Y+int(fy*float64(b.Dy()))).RGBA()
		return a >> 8
	}
	// Pojok kiri atas dan kanan bawah kosong; teks melintasi tengah (miring naik ke kanan)
	if alphaAt(0.02, 0.02) != 0 || alphaAt(0.98, 0.98) != 0 {
		t.Error("latar watermark harus transparan")
	}
	var maxAlpha, painted uint32
	for y := b.Min.Y; y < b.Max.Y; y += 4 {
		for x := b.Min.X; x < b.Max.X; x += 4 {
			_, _, _, a := img.At(x, y).RGBA()
			if a>>8 > maxAlpha {
				maxAlpha = a >> 8
			}
			if a > 0 {
				painted++
			}
		}
	}
	if maxAlpha == 0 || maxAlpha > alpha+1 || painted == 0 {
		t.Errorf("transparansi watermark salah: alpha maks %d (want 1..%d)", maxAlpha, alpha)
	}
	if again, _ := PNG("PUPUK SRIWIDJAJA"); !bytes.Equal(again, data) {
		t.Error("gambar untuk teks yang sama harus sama (cache)")
	}
	if other, _ := PNG("PETROKIMIA GRESIK"); bytes.Equal(other, data) {
		t.Error("teks berbeda harus menghasilkan gambar berbeda")
	}
}

func TestPNGRejectsInvalidText(t *testing.T) {
	for _, text := range []string{"", strings.Repeat("A", MaxRunes+1)} {
		if _, err := PNG(text); err == nil {
			t.Errorf("teks %d karakter diterima", len(text))
		}
	}
}

func TestNormalize(t *testing.T) {
	if got := Normalize("  PUPUK \t SRIWIDJAJA \n"); got != "PUPUK SRIWIDJAJA" {
		t.Errorf("Normalize = %q", got)
	}
}
