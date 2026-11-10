// Package watermark menggambar watermark naskah resmi: teks perusahaan miring 45 derajat, abu-abu
// transparan, sebagai gambar PNG. Dipakai naskah surat terbit (surat) dan pratinjau di pengaturan
// perusahaan (perusahaan), supaya keduanya sama persis.
package watermark

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/f64"
	"golang.org/x/image/math/fixed"
)

// MaxRunes adalah panjang maksimal teks watermark (kolom companies.watermark_text).
const MaxRunes = 100

// Rupa watermark.
const (
	fontPx = 160 // ukuran huruf saat digambar (menentukan ketajaman)
	alpha  = 56  // ketebalan abu-abu (0-255), sekitar 22%
)

//go:embed assets/LiberationSans-Bold.ttf
var fontData []byte

// cache menyimpan gambar per teks: teks watermark jarang berubah, menggambarnya relatif mahal.
var cache sync.Map // string -> []byte

// Normalize merapikan teks watermark: spasi di tepi dibuang, spasi berlebih di tengah dirapatkan.
func Normalize(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

// PNG mengembalikan gambar watermark untuk teks (sudah di-Normalize, tidak kosong).
func PNG(text string) ([]byte, error) {
	if text == "" || utf8.RuneCountInString(text) > MaxRunes {
		return nil, fmt.Errorf("teks watermark kosong atau lebih dari %d karakter", MaxRunes)
	}
	if v, ok := cache.Load(text); ok {
		return v.([]byte), nil
	}
	img, err := render(text)
	if err != nil {
		return nil, err
	}
	cache.Store(text, img)
	return img, nil
}

// render menggambar teks miring 45 derajat (naik ke kanan) di atas latar transparan.
func render(text string) ([]byte, error) {
	f, err := opentype.Parse(fontData)
	if err != nil {
		return nil, fmt.Errorf("membaca font watermark: %w", err)
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: fontPx, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil, fmt.Errorf("menyiapkan font watermark: %w", err)
	}
	defer face.Close()

	m := face.Metrics()
	pad := fontPx / 4
	w := font.MeasureString(face, text).Ceil() + 2*pad
	h := (m.Ascent + m.Descent).Ceil() + 2*pad
	flat := image.NewNRGBA(image.Rect(0, 0, w, h))
	(&font.Drawer{
		Dst:  flat,
		Src:  image.NewUniform(color.NRGBA{R: 128, G: 128, B: 128, A: alpha}),
		Face: face,
		Dot:  fixed.P(pad, pad+m.Ascent.Ceil()),
	}).DrawString(text)

	// Putar 45 derajat berlawanan arah jarum jam di sekitar titik tengah
	cos := math.Cos(math.Pi / 4)
	side := int(math.Ceil(float64(w+h) * cos))
	rotated := image.NewNRGBA(image.Rect(0, 0, side, side))
	cx, cy, c := float64(w)/2, float64(h)/2, float64(side)/2
	aff := f64.Aff3{cos, cos, c - cos*cx - cos*cy, -cos, cos, c + cos*cx - cos*cy}
	draw.BiLinear.Transform(rotated, aff, flat, flat.Bounds(), draw.Over, nil)

	var buf bytes.Buffer
	if err := png.Encode(&buf, rotated); err != nil {
		return nil, fmt.Errorf("menyimpan gambar watermark: %w", err)
	}
	return buf.Bytes(), nil
}
